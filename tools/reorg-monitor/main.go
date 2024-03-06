package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/smtp"
	"os"
	"path"
	"strconv"
	"strings"
	"time"

	"cloud.google.com/go/storage"
	joonix "github.com/joonix/log"
	"github.com/prysmaticlabs/prysm/v5/api/server/structs"
	"github.com/prysmaticlabs/prysm/v5/io/file"
	"github.com/r3labs/sse/v2"
	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v2"
)

var (
	forkchoiceDebugMethod = "/eth/v1/debug/fork_choice"
	monitorFlags          = struct {
		fluentd            bool
		httpEndpoint       string
		storeDumpsInterval time.Duration
		purgeDumpsInterval time.Duration
		useSendgrid        bool
		sendTo             cli.StringSlice
		sendFrom           string
		smtpHost           string
		smtpPort           string
		smtpPasswordFile   string
		smtpUsername       string
		projectId          string
		bucketName         string
		onReorgDepth       uint64
	}{}
)

func main() {
	app := &cli.App{
		Name: "events-monitor",
		Usage: "Ethereum beacon node events monitoring tool with the ability to send emails " +
			"with the event data as JSON attachments",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "http-endpoint",
				Destination: &monitorFlags.httpEndpoint,
				Value:       "http://localhost:3500",
				Usage:       "HTTP standard API endpoint for an Ethereum beacon node",
			},
			&cli.DurationFlag{
				Name:        "store-dumps-interval",
				Destination: &monitorFlags.storeDumpsInterval,
				Value:       time.Minute * 5,
				Usage:       "Interval to store forkchoice dumps (default 5m)",
			},
			&cli.StringSliceFlag{
				Name:        "send-to",
				Destination: &monitorFlags.sendTo,
				Usage:       "Recipient email address for events",
				Required:    true,
			},
			&cli.StringFlag{
				Name:        "send-from",
				Destination: &monitorFlags.sendFrom,
				Usage:       "Sender email address for events",
				Required:    true,
			},
			&cli.BoolFlag{
				Name:        "sendgrid",
				Destination: &monitorFlags.useSendgrid,
				Usage:       "Whether or not to use sendgrid to send emails. Requires SENDGRID_API_KEY env var set",
			},
			&cli.StringFlag{
				Name:        "smtp-host",
				Destination: &monitorFlags.smtpHost,
				Usage:       "Smtp host for sending emails",
			},
			&cli.StringFlag{
				Name:        "smtp-username",
				Destination: &monitorFlags.smtpUsername,
				Usage:       "Smtp username for sending emails",
			},
			&cli.StringFlag{
				Name:        "smtp-port",
				Destination: &monitorFlags.smtpPort,
				Usage:       "Smtp port for sending emails",
			},
			&cli.StringFlag{
				Name:        "smtp-password-file",
				Destination: &monitorFlags.smtpPasswordFile,
				Usage:       "File path to an smtp password for sending emails",
			},
			&cli.StringFlag{
				Name:        "project-id",
				Destination: &monitorFlags.projectId,
				Value:       "",
				Usage:       "Project id on gcp",
			},
			&cli.StringFlag{
				Name:        "bucket-name",
				Destination: &monitorFlags.bucketName,
				Value:       "",
				Usage:       "Bucket name for gcp uploads",
			},
			&cli.BoolFlag{
				Name:        "fluentd",
				Destination: &monitorFlags.fluentd,
				Usage:       "Fluentd log formatting",
			},
			&cli.Uint64Flag{
				Name:        "on-reorg-depth",
				Destination: &monitorFlags.onReorgDepth,
				Usage:       "Only send emails if reorg depth is >=",
				Value:       3,
			},
		},
		Action: func(cliCtx *cli.Context) error {
			if monitorFlags.fluentd {
				f := joonix.NewFormatter()
				if err := joonix.DisableTimestampFormat(f); err != nil {
					log.Fatal(err)
				}
				log.SetFormatter(f)
			}
			var sender emailSender
			if monitorFlags.useSendgrid {
				sender = newSendgridSender(os.Getenv("SENDGRID_API_KEY"))
			} else {
				smtpPassword, err := file.ReadFileAsBytes(monitorFlags.smtpPasswordFile)
				if err != nil {
					return err
				}
				pw := strings.TrimSpace(string(smtpPassword))
				auth := smtp.PlainAuth(
					"",
					monitorFlags.smtpUsername,
					pw,
					monitorFlags.smtpHost,
				)
				sender = newBasicSmtpSender(auth, monitorFlags.smtpHost, monitorFlags.smtpPort)
			}
			return monitorEvents(cliCtx.Context, sender)
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func monitorEvents(ctx context.Context, sender emailSender) error {
	log.Info("Starting reorg monitor")
	storageClient, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	events := make(chan *sse.Event)
	sseClient := sse.NewClient(monitorFlags.httpEndpoint + "/eth/v1/events?topics=chain_reorg")
	if err := sseClient.SubscribeChan("chain_reorg", events); err != nil {
		return err
	}

	go storeForkchoiceDumps(ctx, storageClient)

	for {
		select {
		case <-ctx.Done():
			log.Info("Context canceled")
			return nil
		case ev := <-events:
			log.Info("Received a head event")
			log.Infof("%+v", ev)
			resp := &structs.ChainReorgEvent{}
			if err := json.Unmarshal(ev.Data, &resp); err != nil {
				log.WithError(err).Error("Could not unmarshal head response")
				continue
			}
			log.Infof("Received chain reorg event: %+v", resp)
			depth, err := strconv.ParseUint(resp.Depth, 10, 64)
			if err != nil {
				log.WithError(err).Errorf("Could not parse reorg event depth: %+v", resp)
				continue
			}
			if depth < monitorFlags.onReorgDepth {
				continue
			}
			var dumpFilename string
			obj, err := writeForkchoiceDump(ctx, storageClient)
			if err != nil {
				log.WithError(err).Error("Could not write forkchoice dump")
			} else {
				dumpFilename = "gs://" + path.Join(obj.BucketName(), obj.ObjectName())
			}

			if err := sendJSONEmail(sender, "chain_reorg", resp, dumpFilename); err != nil {
				log.WithError(err).Error("Could not send chain_reorg event as email attachment")
			}
		}
	}
}

func storeForkchoiceDumps(ctx context.Context, storageClient *storage.Client) {
	log.Info("Now starting goroutine for forkchoice dumps")
	timer := time.NewTicker(monitorFlags.storeDumpsInterval)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			if _, err := writeForkchoiceDump(ctx, storageClient); err != nil {
				log.WithError(err).Error("Could not write forkchoice dump")
			}
		case <-ctx.Done():
			return
		}
	}
}

func writeForkchoiceDump(ctx context.Context, storageClient *storage.Client) (*storage.ObjectHandle, error) {
	log.Info("Attempting to write forkchoice dump")
	var forkchoiceDump map[string]interface{}
	url := monitorFlags.httpEndpoint + forkchoiceDebugMethod
	// #nosec G107
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		log.Errorf("Request to URL: %s", url)
		log.Errorf("Response: %+v", resp)
		return nil, fmt.Errorf("did not receive OK HTTP status: %d", resp.StatusCode)
	}
	defer func() {
		if err := resp.Body.Close(); err != nil {
			log.WithError(err).Error("Could not close body")
		}
	}()
	enc, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	log.Infof("Got %s response from debug endpoint", enc)
	if err := json.Unmarshal(enc, &forkchoiceDump); err != nil {
		return nil, err
	}
	fileName := forkchoiceFileName()
	log.WithField("bucket", monitorFlags.bucketName).Infof("Attempting to write %s to cloud bucket", fileName)
	obj := storageClient.Bucket(monitorFlags.bucketName).Object(fileName)
	wc := obj.NewWriter(ctx)
	defer func() {
		if err := wc.Close(); err != nil {
			log.WithError(err).Error("Could not close")
		}
	}()

	return obj, json.NewEncoder(wc).Encode(forkchoiceDump)
}

type reorgEvent struct {
	ev *structs.ChainReorgEvent
}

func (r *reorgEvent) String() string {
	var b strings.Builder
	fmt.Fprintf(&b, "slot=%s\n", r.ev.Slot)
	fmt.Fprintf(&b, "depth=%s\n", r.ev.Depth)
	fmt.Fprintf(&b, "old_head_block=%#x\n", r.ev.OldHeadBlock)
	fmt.Fprintf(&b, "new_head_block=%#x\n", r.ev.NewHeadBlock)
	fmt.Fprintf(&b, "old_head_state=%#x\n", r.ev.OldHeadState)
	fmt.Fprintf(&b, "old_head_state=%#x\n", r.ev.NewHeadState)
	fmt.Fprintf(&b, "epoch=%s\n", r.ev.Epoch)
	fmt.Fprintf(&b, "execution_optimistic=%v\n", r.ev.ExecutionOptimistic)
	return b.String()
}

func sendJSONEmail(sender emailSender, eventName string, ev *structs.ChainReorgEvent, dumpFilename string) error {
	rev := &reorgEvent{ev: ev}
	body := fmt.Sprintf("Detected %s event at %v with data %s\n", eventName, time.Now(), rev.String())
	if dumpFilename != "" {
		body += fmt.Sprintf("\n\nForkchoice dump written: %s\n", dumpFilename)
	}
	m := newEmailMessage(
		fmt.Sprintf("New %s event detected", eventName),
		body,
	)
	m.from = monitorFlags.sendFrom
	m.to = monitorFlags.sendTo.Value()
	return sender.send(m)
}

func forkchoiceFileName() string {
	// Use layout string for time format.
	const layout = "2006-01-02-03:04:05-pm"
	// Place now in the string.
	t := time.Now()
	return "forkchoicedump-" + t.Format(layout) + ".json"
}
