package beacon

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strconv"
	"text/template"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/beacon-chain/rpc/apimiddleware"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

const (
	get_signed_block_path      = "/eth/v2/beacon/blocks"
	get_block_root_path        = "/eth/v1/beacon/blocks/{{.Id}}/root"
	get_fork_for_state_path    = "/eth/v1/beacon/states/{{.Id}}/fork"
	get_weak_subjectivity_path = "/eth/v1/beacon/weak_subjectivity"
	get_fork_schedule_path     = "/eth/v1/config/fork_schedule"
	get_state_path             = "/eth/v2/debug/beacon/states"
	get_node_version_path      = "/eth/v1/node/version"
)

// StateOrBlockId represents the block_id / state_id parameters that several of the Eth Beacon API methods accept.
// StateOrBlockId constants are defined for named identifiers, and helper methods are provided
// for slot and root identifiers. Example text from the Eth Beacon Node API documentation:
//
// "Block identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded blockRoot with 0x prefix>."
type StateOrBlockId string

const (
	IdFinalized StateOrBlockId = "finalized"
	IdGenesis   StateOrBlockId = "genesis"
	IdHead      StateOrBlockId = "head"
	IdJustified StateOrBlockId = "finalized"
)

// IdFromRoot encodes a block root in the format expected by the API in places where a root can be used to identify
// a BeaconState or SignedBeaconBlock.
func IdFromRoot(r [32]byte) StateOrBlockId {
	return StateOrBlockId(fmt.Sprintf("%#x", r))
}

// IdFromRoot encodes a Slot in the format expected by the API in places where a slot can be used to identify
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
	hc     *http.Client
	host   string
	scheme string
}

// NewClient constructs a new client with the provided options (ex WithTimeout).
// `host` is the base host + port used to construct request urls. This value can be
// a URL string, or NewClient will assume an http endpoint if just `host:port` is used.
func NewClient(host string, opts ...ClientOpt) (*Client, error) {
	host, err := validHostname(host)
	if err != nil {
		return nil, err
	}
	c := &Client{
		hc:     &http.Client{},
		scheme: "http",
		host:   host,
	}
	for _, o := range opts {
		o(c)
	}
	return c, nil
}

func validHostname(h string) (string, error) {
	// try to parse as url (being permissive)
	u, err := url.Parse(h)
	if err == nil && u.Host != "" {
		return u.Host, nil
	}
	// try to parse as host:port
	host, port, err := net.SplitHostPort(h)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", host, port), nil
}

func (c *Client) urlForPath(methodPath string) *url.URL {
	u := &url.URL{
		Scheme: c.scheme,
		Host:   c.host,
	}
	u.Path = path.Join(u.Path, methodPath)
	return u
}

type reqOption func(*http.Request)

func withSSZEncoding() reqOption {
	return func(req *http.Request) {
		req.Header.Set("Accept", "application/octet-stream")
	}
}

// get is a generic, opinionated GET function to reduce boilerplate amongst the getters in this package.
func (c *Client) get(ctx context.Context, path string, opts ...reqOption) ([]byte, error) {
	u := c.urlForPath(path)
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

// GetBlock retrieves the SignedBeaconBlock for the given block id.
// Block identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded blockRoot with 0x prefix>. Variables of type StateOrBlockId are exported by this package
// for the named identifiers.
// The return value contains the ssz-encoded bytes.
func (c *Client) GetBlock(ctx context.Context, blockId StateOrBlockId) ([]byte, error) {
	blockPath := path.Join(get_signed_block_path, string(blockId))
	b, err := c.get(ctx, blockPath, withSSZEncoding())
	if err != nil {
		return nil, errors.Wrapf(err, "error requesting state by id = %s", blockId)
	}
	return b, nil
}

var getBlockRootTpl = idTemplate(get_block_root_path)

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

var getForkTpl = idTemplate(get_fork_for_state_path)

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
func (c *Client) GetForkSchedule(ctx context.Context) (params.OrderedForkSchedule, error) {
	body, err := c.get(ctx, get_fork_schedule_path)
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
		return nil, errors.Wrap(err, fmt.Sprintf("problem unmarshaling %s response", get_fork_schedule_path))
	}
	return ofs, nil
}

type NodeVersion struct {
	implementation string
	semver         string
	systemInfo     string
}

var versionRE = regexp.MustCompile(`^(\w+)\/(v\d+\.\d+\.\d+) \((.*)\)$`)

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
	b, err := c.get(ctx, get_node_version_path)
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

// GetState retrieves the BeaconState for the given state id.
// State identifier can be one of: "head" (canonical head in node's view), "genesis", "finalized",
// <slot>, <hex encoded stateRoot with 0x prefix>. Variables of type StateOrBlockId are exported by this package
// for the named identifiers.
// The return value contains the ssz-encoded bytes.
func (c *Client) GetState(ctx context.Context, stateId StateOrBlockId) ([]byte, error) {
	statePath := path.Join(get_state_path, string(stateId))
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
	body, err := c.get(ctx, get_weak_subjectivity_path)
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
	bodyBytes, err := ioutil.ReadAll(response.Body)
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

func (fsr *forkScheduleResponse) OrderedForkSchedule() (params.OrderedForkSchedule, error) {
	ofs := make(params.OrderedForkSchedule, 0)
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
		ofs = append(ofs, params.ForkScheduleEntry{
			Version: version,
			Epoch:   types.Epoch(uint64(epoch)),
		})
	}
	sort.Sort(ofs)
	return ofs, nil
}
