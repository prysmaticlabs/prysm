package builder

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"text/template"
	"time"

	v1 "github.com/prysmaticlabs/prysm/proto/engine/v1"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"

	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/prysm/consensus-types/primitives"
	log "github.com/sirupsen/logrus"
)

const (
	getExecHeaderPath          = "/eth/v1/builder/header/{{.Slot}}/{{.ParentHash}}/{{.Pubkey}}"
	getStatus                  = "/eth/v1/builder/status"
	postBlindedBeaconBlockPath = "/eth/v1/builder/blinded_blocks"
	postRegisterValidatorPath  = "/eth/v1/builder/validators"
)

var errMalformedHostname = errors.New("hostname must include port, separated by one colon, like example.com:3500")

// ClientOpt is a functional option for the Client type (http.Client wrapper)
type ClientOpt func(*Client)

// WithTimeout sets the .Timeout attribute of the wrapped http.Client.
func WithTimeout(timeout time.Duration) ClientOpt {
	return func(c *Client) {
		c.hc.Timeout = timeout
	}
}

type observer interface {
	observe(r *http.Request) error
}

func WithObserver(m observer) ClientOpt {
	return func(c *Client) {
		c.obvs = append(c.obvs, m)
	}
}

type requestLogger struct{}

func (*requestLogger) observe(r *http.Request) (e error) {
	b := bytes.NewBuffer(nil)
	if r.Body == nil {
		log.WithFields(log.Fields{
			"body-base64": "(nil value)",
			"url":         r.URL.String(),
		}).Info("builder http request")
		return nil
	}
	t := io.TeeReader(r.Body, b)
	defer func() {
		if r.Body != nil {
			e = r.Body.Close()
		}
	}()
	body, err := io.ReadAll(t)
	if err != nil {
		return err
	}
	r.Body = io.NopCloser(b)
	log.WithFields(log.Fields{
		"body-base64": string(body),
		"url":         r.URL.String(),
	}).Info("builder http request")

	return nil
}

var _ observer = &requestLogger{}

// Client provides a collection of helper methods for calling Builder API endpoints.
type Client struct {
	hc      *http.Client
	baseURL *url.URL
	obvs    []observer
}

// NewClient constructs a new client with the provided options (ex WithTimeout).
// `host` is the base host + port used to construct request urls. This value can be
// a URL string, or NewClient will assume an http endpoint if just `host:port` is used.
func NewClient(host string, opts ...ClientOpt) (*Client, error) {
	u, err := urlForHost(host)
	if err != nil {
		return nil, err
	}
	c := &Client{
		hc:      &http.Client{},
		baseURL: u,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

func urlForHost(h string) (*url.URL, error) {
	// try to parse as url (being permissive)
	u, err := url.Parse(h)
	if err == nil && u.Host != "" {
		return u, nil
	}
	// try to parse as host:port
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		return nil, errMalformedHostname
	}
	return &url.URL{Host: net.JoinHostPort(host, port), Scheme: "http"}, nil
}

// NodeURL returns a human-readable string representation of the beacon node base url.
func (c *Client) NodeURL() string {
	return c.baseURL.String()
}

type reqOption func(*http.Request)

// do is a generic, opinionated GET function to reduce boilerplate amongst the getters in this packageapi/client/builder/types.go.
func (c *Client) do(ctx context.Context, method string, path string, body io.Reader, opts ...reqOption) ([]byte, error) {
	u := c.baseURL.ResolveReference(&url.URL{Path: path})
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		o(req)
	}
	for _, o := range c.obvs {
		if err := o.observe(req); err != nil {
			return nil, err
		}
	}
	r, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() {
		err = r.Body.Close()
	}()
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	b, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading http response body from GetBlock")
	}
	return b, nil
}

var execHeaderTemplate = template.Must(template.New("").Parse(getExecHeaderPath))

func execHeaderPath(slot types.Slot, parentHash [32]byte, pubkey [48]byte) (string, error) {
	v := struct {
		Slot       types.Slot
		ParentHash string
		Pubkey     string
	}{
		Slot:       slot,
		ParentHash: fmt.Sprintf("%#x", parentHash),
		Pubkey:     fmt.Sprintf("%#x", pubkey),
	}
	b := bytes.NewBuffer(nil)
	err := execHeaderTemplate.Execute(b, v)
	if err != nil {
		return "", errors.Wrapf(err, "error rendering exec header template with slot=%d, parentHash=%#x, pubkey=%#x", slot, parentHash, pubkey)
	}
	return b.String(), nil
}

// GetHeader is used by a proposing validator to request an ExecutionPayloadHeader from the Builder node.
func (c *Client) GetHeader(ctx context.Context, slot types.Slot, parentHash [32]byte, pubkey [48]byte) (*ethpb.SignedBuilderBid, error) {
	path, err := execHeaderPath(slot, parentHash, pubkey)
	if err != nil {
		return nil, err
	}
	hb, err := c.do(ctx, http.MethodGet, path, nil)
	if err != nil {
		return nil, err
	}
	hr := &ExecHeaderResponse{}
	if err := json.Unmarshal(hb, hr); err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling the builder GetHeader response, using slot=%d, parentHash=%#x, pubkey=%#x", slot, parentHash, pubkey)
	}
	return hr.ToProto()
}

// RegisterValidator encodes the SignedValidatorRegistrationV1 message to json (including hex-encoding the byte
// fields with 0x prefixes) and posts to the builder validator registration endpoint.
func (c *Client) RegisterValidator(ctx context.Context, svr *ethpb.SignedValidatorRegistrationV1) error {
	v := &SignedValidatorRegistration{SignedValidatorRegistrationV1: svr}
	body, err := json.Marshal(v)
	if err != nil {
		return errors.Wrap(err, "error encoding the SignedValidatorRegistration value body in RegisterValidator")
	}
	_, err = c.do(ctx, http.MethodPost, postRegisterValidatorPath, bytes.NewBuffer(body))
	return err
}

// SubmitBlindedBlock calls the builder API endpoint that binds the validator to the builder and submits the block.
// The response is the full ExecutionPayload used to create the blinded block.
func (c *Client) SubmitBlindedBlock(ctx context.Context, sb *ethpb.SignedBlindedBeaconBlockBellatrix) (*v1.ExecutionPayload, error) {
	v := &SignedBlindedBeaconBlockBellatrix{SignedBlindedBeaconBlockBellatrix: sb}
	body, err := json.Marshal(v)
	if err != nil {
		return nil, errors.Wrap(err, "error encoding the SignedBlindedBeaconBlockBellatrix value body in SubmitBlindedBlock")
	}
	rb, err := c.do(ctx, http.MethodPost, postBlindedBeaconBlockPath, bytes.NewBuffer(body))
	if err != nil {
		return nil, errors.Wrap(err, "error posting the SignedBlindedBeaconBlockBellatrix to the builder api")
	}
	ep := &ExecPayloadResponse{}
	if err := json.Unmarshal(rb, ep); err != nil {
		return nil, errors.Wrap(err, "error unmarshaling the builder SubmitBlindedBlock response")
	}
	return ep.ToProto()
}

// Status asks the remote builder server for a health check. A response of 200 with an empty body is the success/healthy
// response, and an error response may have an error message. This method will return a nil value for error in the
// happy path, and an error with information about the server response body for a non-200 response.
func (c *Client) Status(ctx context.Context) error {
	_, err := c.do(ctx, http.MethodGet, getStatus, nil)
	return err
}

func non200Err(response *http.Response) error {
	bodyBytes, err := io.ReadAll(response.Body)
	var body string
	if err != nil {
		body = "(Unable to read response body.)"
	} else {
		body = "response body:\n" + string(bodyBytes)
	}
	msg := fmt.Sprintf("code=%d, url=%s, body=%s", response.StatusCode, response.Request.URL, body)
	switch response.StatusCode {
	case 404:
		return errors.Wrap(ErrNotFound, msg)
	default:
		return errors.Wrap(ErrNotOK, msg)
	}
}
