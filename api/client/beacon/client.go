package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/prysmaticlabs/prysm/v3/network/forks"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	"github.com/prysmaticlabs/prysm/v3/beacon-chain/rpc/apimiddleware"
	types "github.com/prysmaticlabs/prysm/v3/consensus-types/primitives"
	"github.com/prysmaticlabs/prysm/v3/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/v3/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

const (
	getSignedBlockPath      = "/eth/v2/beacon/blocks"
	getBlockRootPath        = "/eth/v1/beacon/blocks/{{.Id}}/root"
	getForkForStatePath     = "/eth/v1/beacon/states/{{.Id}}/fork"
	getWeakSubjectivityPath = "/eth/v1/beacon/weak_subjectivity"
	getForkSchedulePath     = "/eth/v1/config/fork_schedule"
	getStatePath            = "/eth/v2/debug/beacon/states"
	getNodeVersionPath      = "/eth/v1/node/version"
)

// StateOrBlockId represents the block_id / state_id parameters that several of the Eth Beacon API methods accept.
// StateOrBlockId constants are defined for named identifiers, and helper methods are provided
// for slot and root identifiers. Example text from the Eth Beacon Node API documentation:
//
// "Block identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded blockRoot with 0x prefix>."
type StateOrBlockId string

const (
	IdGenesis   StateOrBlockId = "genesis"
	IdHead      StateOrBlockId = "head"
	IdFinalized StateOrBlockId = "finalized"
)

var ErrMalformedHostname = errors.New("hostname must include port, separated by one colon, like example.com:3500")

// IdFromRoot encodes a block root in the format expected by the API in places where a root can be used to identify
// a BeaconState or SignedBeaconBlock.
func IdFromRoot(r [32]byte) StateOrBlockId {
	return StateOrBlockId(fmt.Sprintf("%#x", r))
}

// IdFromSlot encodes a Slot in the format expected by the API in places where a slot can be used to identify
// a BeaconState or SignedBeaconBlock.
func IdFromSlot(s types.Slot) StateOrBlockId {
	return StateOrBlockId(strconv.FormatUint(uint64(s), 10))
}

// idTemplate is used to create template functions that can interpolate StateOrBlockId values.
func idTemplate(ts string) func(StateOrBlockId) string {
	t := template.Must(template.New("").Parse(ts))
	f := func(id StateOrBlockId) string {
		b := bytes.NewBuffer(nil)
		err := t.Execute(b, struct{ Id string }{Id: string(id)})
		if err != nil {
			panic(fmt.Sprintf("invalid idTemplate: %s", ts))
		}
		return b.String()
	}
	// run the template to ensure that it is valid
	// this should happen load time (using package scoped vars) to ensure runtime errors aren't possible
	_ = f(IdGenesis)
	return f
}

// ClientOpt is a functional option for the Client type (http.Client wrapper)
type ClientOpt func(*Client)

// WithTimeout sets the .Timeout attribute of the wrapped http.Client.
func WithTimeout(timeout time.Duration) ClientOpt {
	return func(c *Client) {
		c.hc.Timeout = timeout
	}
}

// Client provides a collection of helper methods for calling the Eth Beacon Node API endpoints.
type Client struct {
	hc      *http.Client
	baseURL *url.URL
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
		return nil, ErrMalformedHostname
	}
	return &url.URL{Host: fmt.Sprintf("%s:%s", host, port), Scheme: "http"}, nil
}

// NodeURL returns a human-readable string representation of the beacon node base url.
func (c *Client) NodeURL() string {
	return c.baseURL.String()
}

type reqOption func(*http.Request)

func withSSZEncoding() reqOption {
	return func(req *http.Request) {
		req.Header.Set("Accept", "application/octet-stream")
	}
}

// get is a generic, opinionated GET function to reduce boilerplate amongst the getters in this package.
func (c *Client) get(ctx context.Context, path string, opts ...reqOption) ([]byte, error) {
	u := c.baseURL.ResolveReference(&url.URL{Path: path})
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	for _, o := range opts {
		o(req)
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

func renderGetBlockPath(id StateOrBlockId) string {
	return path.Join(getSignedBlockPath, string(id))
}

// GetBlock retrieves the SignedBeaconBlock for the given block id.
// Block identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded blockRoot with 0x prefix>. Variables of type StateOrBlockId are exported by this package
// for the named identifiers.
// The return value contains the ssz-encoded bytes.
func (c *Client) GetBlock(ctx context.Context, blockId StateOrBlockId) ([]byte, error) {
	blockPath := renderGetBlockPath(blockId)
	b, err := c.get(ctx, blockPath, withSSZEncoding())
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting state by id = %s", blockId)
	}
	return b, nil
}

var getBlockRootTpl = idTemplate(getBlockRootPath)

// GetBlockRoot retrieves the hash_tree_root of the BeaconBlock for the given block id.
// Block identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded blockRoot with 0x prefix>. Variables of type StateOrBlockId are exported by this package
// for the named identifiers.
func (c *Client) GetBlockRoot(ctx context.Context, blockId StateOrBlockId) ([32]byte, error) {
	rootPath := getBlockRootTpl(blockId)
	b, err := c.get(ctx, rootPath)
	if err != nil {
		return [32]byte{}, errors.Wrapf(err, "error requesting block root by id = %s", blockId)
	}
	jsonr := &struct{ Data struct{ Root string } }{}
	err = json.Unmarshal(b, jsonr)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, "error decoding json data from get block root response")
	}
	rs, err := hexutil.Decode(jsonr.Data.Root)
	if err != nil {
		return [32]byte{}, errors.Wrap(err, fmt.Sprintf("error decoding hex-encoded value %s", jsonr.Data.Root))
	}
	return bytesutil.ToBytes32(rs), nil
}

var getForkTpl = idTemplate(getForkForStatePath)

// GetFork queries the Beacon Node API for the Fork from the state identified by stateId.
// Block identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded blockRoot with 0x prefix>. Variables of type StateOrBlockId are exported by this package
// for the named identifiers.
func (c *Client) GetFork(ctx context.Context, stateId StateOrBlockId) (*ethpb.Fork, error) {
	body, err := c.get(ctx, getForkTpl(stateId))
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting fork by state id = %s", stateId)
	}
	fr := &forkResponse{}
	dataWrapper := &struct{ Data *forkResponse }{Data: fr}
	err = json.Unmarshal(body, dataWrapper)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding json response in GetFork")
	}

	return fr.Fork()
}

// GetForkSchedule retrieve all forks, past present and future, of which this node is aware.
func (c *Client) GetForkSchedule(ctx context.Context) (forks.OrderedSchedule, error) {
	body, err := c.get(ctx, getForkSchedulePath)
	if err != nil {
		return nil, errors.Wrap(err, "error requesting fork schedule")
	}
	fsr := &forkScheduleResponse{}
	err = json.Unmarshal(body, fsr)
	if err != nil {
		return nil, err
	}
	ofs, err := fsr.OrderedForkSchedule()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("problem unmarshaling %s response", getForkSchedulePath))
	}
	return ofs, nil
}

type NodeVersion struct {
	implementation string
	semver         string
	systemInfo     string
}

var versionRE = regexp.MustCompile(`^(\w+)/(v\d+\.\d+\.\d+[-a-zA-Z0-9]*)\s*/?(.*)$`)

func parseNodeVersion(v string) (*NodeVersion, error) {
	groups := versionRE.FindStringSubmatch(v)
	if len(groups) != 4 {
		return nil, errors.Wrapf(ErrInvalidNodeVersion, "could not be parsed: %s", v)
	}
	return &NodeVersion{
		implementation: groups[1],
		semver:         groups[2],
		systemInfo:     groups[3],
	}, nil
}

// GetNodeVersion requests that the beacon node identify information about its implementation in a format
// similar to a HTTP User-Agent field. ex: Lighthouse/v0.1.5 (Linux x86_64)
func (c *Client) GetNodeVersion(ctx context.Context) (*NodeVersion, error) {
	b, err := c.get(ctx, getNodeVersionPath)
	if err != nil {
		return nil, errors.Wrap(err, "error requesting node version")
	}
	d := struct {
		Data struct {
			Version string `json:"version"`
		} `json:"data"`
	}{}
	err = json.Unmarshal(b, &d)
	if err != nil {
		return nil, errors.Wrapf(err, "error unmarshaling response body: %s", string(b))
	}
	return parseNodeVersion(d.Data.Version)
}

func renderGetStatePath(id StateOrBlockId) string {
	return path.Join(getStatePath, string(id))
}

// GetState retrieves the BeaconState for the given state id.
// State identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded stateRoot with 0x prefix>. Variables of type StateOrBlockId are exported by this package
// for the named identifiers.
// The return value contains the ssz-encoded bytes.
func (c *Client) GetState(ctx context.Context, stateId StateOrBlockId) ([]byte, error) {
	statePath := path.Join(getStatePath, string(stateId))
	b, err := c.get(ctx, statePath, withSSZEncoding())
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting state by id = %s", stateId)
	}
	return b, nil
}

// GetWeakSubjectivity calls a proposed API endpoint that is unique to prysm
// This api method does the following:
// - computes weak subjectivity epoch
// - finds the highest non-skipped block preceding the epoch
// - returns the htr of the found block and returns this + the value of state_root from the block
func (c *Client) GetWeakSubjectivity(ctx context.Context) (*WeakSubjectivityData, error) {
	body, err := c.get(ctx, getWeakSubjectivityPath)
	if err != nil {
		return nil, err
	}
	v := &apimiddleware.WeakSubjectivityResponse{}
	err = json.Unmarshal(body, v)
	if err != nil {
		return nil, err
	}
	epoch, err := strconv.ParseUint(v.Data.Checkpoint.Epoch, 10, 64)
	if err != nil {
		return nil, err
	}
	blockRoot, err := hexutil.Decode(v.Data.Checkpoint.Root)
	if err != nil {
		return nil, err
	}
	stateRoot, err := hexutil.Decode(v.Data.StateRoot)
	if err != nil {
		return nil, err
	}
	return &WeakSubjectivityData{
		Epoch:     types.Epoch(epoch),
		BlockRoot: bytesutil.ToBytes32(blockRoot),
		StateRoot: bytesutil.ToBytes32(stateRoot),
	}, nil
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

type forkResponse struct {
	PreviousVersion string `json:"previous_version"`
	CurrentVersion  string `json:"current_version"`
	Epoch           string `json:"epoch"`
}

func (f *forkResponse) Fork() (*ethpb.Fork, error) {
	epoch, err := strconv.ParseUint(f.Epoch, 10, 64)
	if err != nil {
		return nil, err
	}
	cSlice, err := hexutil.Decode(f.CurrentVersion)
	if err != nil {
		return nil, err
	}
	if len(cSlice) != 4 {
		return nil, fmt.Errorf("got %d byte version for CurrentVersion, expected 4 bytes. hex=%s", len(cSlice), f.CurrentVersion)
	}
	pSlice, err := hexutil.Decode(f.PreviousVersion)
	if err != nil {
		return nil, err
	}
	if len(pSlice) != 4 {
		return nil, fmt.Errorf("got %d byte version, expected 4 bytes. version hex=%s", len(pSlice), f.PreviousVersion)
	}
	return &ethpb.Fork{
		CurrentVersion:  cSlice,
		PreviousVersion: pSlice,
		Epoch:           types.Epoch(epoch),
	}, nil
}

type forkScheduleResponse struct {
	Data []forkResponse
}

func (fsr *forkScheduleResponse) OrderedForkSchedule() (forks.OrderedSchedule, error) {
	ofs := make(forks.OrderedSchedule, 0)
	for _, d := range fsr.Data {
		epoch, err := strconv.Atoi(d.Epoch)
		if err != nil {
			return nil, err
		}
		vSlice, err := hexutil.Decode(d.CurrentVersion)
		if err != nil {
			return nil, err
		}
		if len(vSlice) != 4 {
			return nil, fmt.Errorf("got %d byte version, expected 4 bytes. version hex=%s", len(vSlice), d.CurrentVersion)
		}
		version := bytesutil.ToBytes4(vSlice)
		ofs = append(ofs, forks.ForkScheduleEntry{
			Version: version,
			Epoch:   types.Epoch(uint64(epoch)),
		})
	}
	sort.Sort(ofs)
	return ofs, nil
}
