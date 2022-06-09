package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	fssz "github.com/ferranbt/fastssz"
	"github.com/golang/snappy"
	"github.com/urfave/cli/v2"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime/pprof"
	"strings"
	"time"

	pbbeacon "github.com/prysmaticlabs/prysm/proto/beacon/p2p/v1"
	pbethv1 "github.com/prysmaticlabs/prysm/proto/eth/v1"
	pbethv1alpha1 "github.com/prysmaticlabs/prysm/proto/eth/v1alpha1"
)

const methodsetMethodical = "methodical"
const methodsetFast = "fastssz"

var methodset string
var benchmarkRepeat int
var skipList string
var benchmark = &cli.Command{
	Name:    "benchmark",
	ArgsUsage: "<path to spectest repository>",
	Aliases: []string{"bench"},
	Usage:   "Benchmark for comparing fastssz with methodical to generate profiling data",
	Flags: []cli.Flag{
		&cli.StringFlag{
			Name:        "methodset",
			Value:       "",
			Usage:       "which methodset to evaluate, \"fastssz\" or \"methodical\"",
			Destination: &methodset,
		},
		&cli.StringFlag{
			Name:        "skip-list",
			Value:       "",
			Usage:       "comma-separated list of types to skip (useful for excluding that big ole BeaconState).",
			Destination: &skipList,
		},
		&cli.IntFlag{
			Name:        "repeat",
			Usage:       "how many times to repeat each unmarshal/marshal operation (increase for more stability)",
			Destination: &benchmarkRepeat,
		},
	},
	Action: func(c *cli.Context) error {
		// validate args
		spectestPath := c.Args().Get(0)
		if spectestPath == "" {
			cli.ShowCommandHelp(c, "benchmark")
			return fmt.Errorf("error: missing required <path to spectest repository> argument")
		}
		if methodset != methodsetMethodical && methodset != methodsetFast {
			cli.ShowCommandHelp(c, "benchmark")
			return fmt.Errorf("error: --methodset must be equal to \"fastssz\" or \"methodical\"")
		}

		// initialize profiling, profilePath will fail if spectest path is weird
		ppath, err := profilePath(spectestPath, methodset)
		f, err := os.Create(ppath)
		if err != nil {
			return err
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()

		skip := make(map[string]struct{})
		if skipList != "" {
			skipNames := strings.Split(skipList, ",")
			for _, s := range skipNames {
				skip[s] = struct{}{}
			}
		}
		// use regex to parse test cases out of a dirwalk
		tcs, err := findTestCases(spectestPath, skip)
		if err != nil {
			return err
		}
		fmt.Printf("Found %d test cases", len(tcs))
		for _, tc := range tcs {
			err := executeTestCase(tc, methodset, benchmarkRepeat)
			if err != nil {
				return err
			}
		}
		return nil
	},
}

func profilePath(path string, methodset string) (string, error) {
	pre := regexp.MustCompile(`.*\/tests\/(mainnet|minimal)\/(altair|merge|phase0)\/ssz_static`)
	parts := pre.FindStringSubmatch(path)
	if len(parts) != 3 {
		return "", fmt.Errorf("unfamiliar spectest path, can't determine test configuration and phase")
	}
	return fmt.Sprintf("cpu-%s-%s-%s.%s.pprof", methodset, parts[1], parts[2], time.Now().Format("20060102-150405")), nil
}

func executeTestCase(tc *TestCase, methodset string, repeat int) error {
	b, err := tc.MarshaledBytes()
	if err != nil {
		return err
	}
	tys := make([]pbinit, 0)
	for _, c := range []map[string]pbinit{casesBeaconP2pV1,casesV1,casesV1Alpha1} {
		pi, ok := c[tc.typeName]
		if !ok {
			continue
		}
		tys = append(tys, pi)
	}
	for i := 0; i <= repeat; i++ {
		for _, fn := range tys {
			essz := fn()
			if methodset == methodsetFast {
				err := essz.UnmarshalSSZ(b)
				if err != nil {
					return err
				}
				_, err = essz.MarshalSSZ()
				if err != nil {
					return err
				}

				_, err = essz.HashTreeRoot()
				if err != nil {
					return err
				}
			}
			if methodset == methodsetMethodical {
				err := essz.XXUnmarshalSSZ(b)
				if err != nil {
					return err
				}
				_, err = essz.XXMarshalSSZ()
				if err != nil {
					return err
				}
				_, err = essz.XXHashTreeRoot()
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func findTestCases(path string, skip map[string]struct{}) ([]*TestCase, error) {
	var re = regexp.MustCompile(`.*\/tests\/(mainnet|minimal)\/(altair|merge|phase0)\/ssz_static\/(.*)\/ssz_random\/(case_\d+)`)
	tcs := make([]*TestCase, 0)
	testCaseFromPath := func (path string, d fs.DirEntry, err error) error {
		if !d.IsDir() {
			return nil
		}
		parts := re.FindStringSubmatch(path)
		if len(parts) != 5 {
			return nil
		}
		tc := &TestCase{
			path: path,
			config: parts[1],
			phase: parts[2],
			typeName: parts[3],
			caseId: parts[4],
		}
		if tc.config == "" || tc.phase == "" || tc.typeName == "" || tc.caseId == "" {
			return nil
		}
		if _, ok := skip[tc.typeName]; ok {
			return nil
		}
		tcs = append(tcs, tc)
		return nil
	}
	err := filepath.WalkDir(path, testCaseFromPath)

	return tcs, err
}

type SSZRoots struct {
	Root        string `json:"root"`
	SigningRoot string `json:"signing_root"`
}

type SSZValue struct {
	Message json.RawMessage `json:"message"`
	Signature string `json:"signature"`// hex encoded '0x...'
}

type TestCase struct {
	path string
	config string
	phase string
	typeName string
	caseId string
}

func (tc *TestCase) MarshaledBytes() ([]byte, error) {
	fh, err := os.Open(path.Join(tc.path, "serialized.ssz_snappy"))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	buf := bytes.NewBuffer(nil)
	_, err = buf.ReadFrom(fh)
	return snappy.Decode(nil, buf.Bytes())
}

func (tc *TestCase) Value() (*SSZValue, error) {
	fh, err := os.Open(path.Join(tc.path, "value.yaml"))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	d := json.NewDecoder(fh)
	v := &SSZValue{}
	err = d.Decode(v)
	return v, err
}

func (tc *TestCase) Roots() (*SSZRoots, error) {
	fh, err := os.Open(path.Join(tc.path, "roots.yaml"))
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	d := json.NewDecoder(fh)
	r := &SSZRoots{}
	err = d.Decode(r)
	return r, err
}

//rootBytes, err := hex.DecodeString(rootsYaml.Root[2:])
//require.NoError(t, err)
//require.DeepEqual(t, rootBytes, root[:], "Did not receive expected hash tree root")

type ExperimentalSSZ interface {
	XXUnmarshalSSZ(buf []byte) error
	XXMarshalSSZ() ([]byte, error)
	XXHashTreeRoot() ([32]byte, error)
	fssz.Unmarshaler
	fssz.Marshaler
	fssz.HashRoot
}

type pbinit func() ExperimentalSSZ

var casesBeaconP2pV1 = map[string]pbinit{
	"BeaconState": func() ExperimentalSSZ { return &pbbeacon.BeaconState{} },
	"DepositMessage": func() ExperimentalSSZ { return &pbbeacon.DepositMessage{} },
	"Fork": func() ExperimentalSSZ { return &pbbeacon.Fork{} },
	"ForkData": func() ExperimentalSSZ { return &pbbeacon.ForkData{} },
	"HistoricalBatch": func() ExperimentalSSZ { return &pbbeacon.HistoricalBatch{} },
	"PendingAttestation": func() ExperimentalSSZ { return &pbbeacon.PendingAttestation{} },
	"SigningData": func() ExperimentalSSZ { return &pbbeacon.SigningData{} },
}

var casesV1 map[string]pbinit = map[string]pbinit{
	"AggregateAndProof": func() ExperimentalSSZ { return &pbethv1.AggregateAttestationAndProof{} },
	"Attestation": func() ExperimentalSSZ { return &pbethv1.Attestation{} },
	"AttestationData": func() ExperimentalSSZ { return &pbethv1.AttestationData{} },
	"AttesterSlashing": func() ExperimentalSSZ { return &pbethv1.AttesterSlashing{} },
	"BeaconBlock": func() ExperimentalSSZ { return &pbethv1.BeaconBlock{} },
	"BeaconBlockBody": func() ExperimentalSSZ { return &pbethv1.BeaconBlockBody{} },
	"BeaconBlockHeader": func() ExperimentalSSZ { return &pbethv1.BeaconBlockHeader{} },
	// exists in proto/eth/v1, but fastssz methods are not genrated for it
	//"BeaconState": func() ExperimentalSSZ { return &pbethv1.BeaconState{} },
	"Checkpoint": func() ExperimentalSSZ { return &pbethv1.Checkpoint{} },
	"Deposit": func() ExperimentalSSZ { return &pbethv1.Deposit{} },
	"DepositData": func() ExperimentalSSZ { return &pbethv1.Deposit_Data{} },
	"Eth1Data": func() ExperimentalSSZ { return &pbethv1.Eth1Data{} },
	// Fork is defined in proto/eth/v1 package, but fastssz methods are not generated
	//"Fork": func() ExperimentalSSZ { return &pbethv1.Fork{} },
	"IndexedAttestation": func() ExperimentalSSZ { return &pbethv1.IndexedAttestation{} },
	// PendingAttestation is defined in proto/eth/v1 package, but fastssz methods are not generated
	//"PendingAttestation": func() ExperimentalSSZ { return &pbethv1.PendingAttestation{} },
	"ProposerSlashing": func() ExperimentalSSZ { return &pbethv1.ProposerSlashing{} },
	"SignedAggregateAndProof": func() ExperimentalSSZ { return &pbethv1.SignedAggregateAttestationAndProof{} },
	"SignedBeaconBlock": func() ExperimentalSSZ { return &pbethv1.SignedBeaconBlock{} },
	"SignedBeaconBlockHeader": func() ExperimentalSSZ { return &pbethv1.SignedBeaconBlockHeader{} },
	"SignedVoluntaryExit": func() ExperimentalSSZ { return &pbethv1.SignedVoluntaryExit{} },
	"Validator": func() ExperimentalSSZ { return &pbethv1.Validator{} },
	"VoluntaryExit": func() ExperimentalSSZ { return &pbethv1.VoluntaryExit{} },
}

var casesV1Alpha1 map[string]pbinit = map[string]pbinit{
	"AggregateAndProof": func() ExperimentalSSZ { return &pbethv1alpha1.AggregateAttestationAndProof{} },
	"Attestation": func() ExperimentalSSZ { return &pbethv1alpha1.Attestation{} },
	"AttestationData": func() ExperimentalSSZ { return &pbethv1alpha1.AttestationData{} },
	"AttesterSlashing": func() ExperimentalSSZ { return &pbethv1alpha1.AttesterSlashing{} },
	"BeaconBlock": func() ExperimentalSSZ { return &pbethv1alpha1.BeaconBlock{} },
	"BeaconBlockBody": func() ExperimentalSSZ { return &pbethv1alpha1.BeaconBlockBody{} },
	"BeaconBlockHeader": func() ExperimentalSSZ { return &pbethv1alpha1.BeaconBlockHeader{} },
	"Checkpoint": func() ExperimentalSSZ { return &pbethv1alpha1.Checkpoint{} },
	"Deposit": func() ExperimentalSSZ { return &pbethv1alpha1.Deposit{} },
	"DepositData": func() ExperimentalSSZ { return &pbethv1alpha1.Deposit_Data{} },
	"Eth1Data": func() ExperimentalSSZ { return &pbethv1alpha1.Eth1Data{} },
	"IndexedAttestation": func() ExperimentalSSZ { return &pbethv1alpha1.IndexedAttestation{} },
	"ProposerSlashing": func() ExperimentalSSZ { return &pbethv1alpha1.ProposerSlashing{} },
	"SignedAggregateAndProof": func() ExperimentalSSZ { return &pbethv1alpha1.SignedAggregateAttestationAndProof{} },
	"SignedBeaconBlock": func() ExperimentalSSZ { return &pbethv1alpha1.SignedBeaconBlock{} },
	"SignedBeaconBlockHeader": func() ExperimentalSSZ { return &pbethv1alpha1.SignedBeaconBlockHeader{} },
	"SignedVoluntaryExit": func() ExperimentalSSZ { return &pbethv1alpha1.SignedVoluntaryExit{} },
	"Validator": func() ExperimentalSSZ { return &pbethv1alpha1.Validator{} },
	"VoluntaryExit": func() ExperimentalSSZ { return &pbethv1alpha1.VoluntaryExit{} },
}