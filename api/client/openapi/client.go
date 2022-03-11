package openapi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"net/http"
	"net/url"
	"path"
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
	get_weak_subjectivity_path = "/eth/v1/beacon/weak_subjectivity"
	get_signed_block_path      = "/eth/v2/beacon/blocks"
	get_state_path             = "/eth/v2/debug/beacon/states"
	get_fork_schedule_path     = "/eth/v1/config/fork_schedule"
	get_fork_for_state_path    = "/eth/v1/beacon/states/{{.StateId}}/fork"
	get_block_root_path        = "/eth/v1/beacon/blocks/{{.BlockId}}/root"
)

type StateOrBlockId string

const (
	IdHead      StateOrBlockId = "head"
	IdGenesis   StateOrBlockId = "genesis"
	IdFinalized StateOrBlockId = "finalized"
	IdJustified StateOrBlockId = "finalized"
)

// ClientOpt is a functional option for the Client type (http.Client wrapper)
type ClientOpt func(*Client)

// WithTimeout sets the .Timeout attribute of the wrapped http.Client.
func WithTimeout(timeout time.Duration) ClientOpt {
	return func(c *Client) {
		c.Client.Timeout = timeout
	}
}

// Client provides a collection of helper methods for calling the Eth Beacon Node API endpoints.
type Client struct {
	Client *http.Client
	host   string
	scheme string
}

func (c *Client) urlForPath(methodPath string) *url.URL {
	u := &url.URL{
		Scheme: c.scheme,
		Host:   c.host,
	}
	u.Path = path.Join(u.Path, methodPath)
	return u
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
		Client: &http.Client{},
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

// GetBlockByRoot retrieves a SignedBeaconBlock with the given root via the beacon node API.
// The value is an io.Reader for the raw ssz-encoded bytes.
func (c *Client) GetBlockByRoot(blockHex string) (io.Reader, error) {
	blockPath := path.Join(get_signed_block_path, blockHex)
	u := c.urlForPath(blockPath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.Client.Do(req)
	defer func() {
		err = r.Body.Close()
	}()
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	b := bytes.NewBuffer(nil)
	_, err = io.Copy(b, r.Body)
	return b, nil
}

// GetBlockBySlot queries the beacon node API for the SignedBeaconBlockAltair for the given slot.
// The value is an io.Reader for the raw ssz-encoded bytes.
func (c *Client) GetBlockBySlot(slot types.Slot) (io.Reader, error) {
	blockPath := path.Join(get_signed_block_path, strconv.FormatUint(uint64(slot), 10))
	u := c.urlForPath(blockPath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.Client.Do(req)
	defer func() {
		err = r.Body.Close()
	}()
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	b := bytes.NewBuffer(nil)
	_, err = io.Copy(b, r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading http response body from GetBlockBySlot")
	}
	return b, nil
}

// Can be one of: "head" (canonical head in node's view), "genesis", "finalized", "justified", <slot>, <hex encoded blockRoot with 0x prefix>.
func (c *Client) GetBlockRoot(blockId string) ([32]byte, error) {
	var root [32]byte
	t := template.Must(template.New("get-block-root").Parse(get_block_root_path))
	b := bytes.NewBuffer(nil)
	err := t.Execute(b, struct{ BlockId string }{BlockId: blockId})
	if err != nil {
		return root, errors.Wrap(err, fmt.Sprintf("unable to generate path w/ blockId=%s", blockId))
	}
	rootPath := b.String()
	u := c.urlForPath(rootPath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return root, err
	}
	r, err := c.Client.Do(req)
	defer func() {
		err = r.Body.Close()
	}()
	if err != nil {
		return root, err
	}
	if r.StatusCode != http.StatusOK {
		return root, non200Err(r)
	}
	jsonr := &struct{ Data struct{ Root string } }{}
	err = json.NewDecoder(r.Body).Decode(jsonr)
	if err != nil {
		return root, errors.Wrap(err, "error decoding json data from get block root response")
	}
	rs, err := hexutil.Decode(jsonr.Data.Root)
	if err != nil {
		return root, errors.Wrap(err, fmt.Sprintf("error decoding hex-encoded value %s", jsonr.Data.Root))
	}
	return bytesutil.ToBytes32(rs), nil
}

// GetForkForState queries the Beacon Node API for the Fork from the state identified by stateId.
func (c *Client) GetForkForState(stateId string) (*ethpb.Fork, error) {
	t := template.Must(template.New("get-fork-for-state").Parse(get_fork_for_state_path))
	b := bytes.NewBuffer(nil)
	err := t.Execute(b, struct{ StateId string }{StateId: stateId})
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("unable to generate path w/ stateId=%s", stateId))
	}
	u := c.urlForPath(b.String())
	r, err := c.Client.Get(u.String())
	defer func() {
		err = r.Body.Close()
	}()
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	fr := &forkResponse{}
	dataWrapper := &struct{ Data *forkResponse }{Data: fr}
	err = json.NewDecoder(r.Body).Decode(dataWrapper)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding json response in GetForkForState")
	}

	return fr.Fork()
}

// GetStateByRoot retrieves a BeaconStateAltair with the given root via the beacon node API
func (c *Client) GetStateByRoot(stateHex string) (io.Reader, error) {
	return c.GetStateById(StateOrBlockId(stateHex))
}

// GetStateBySlot retrieves a BeaconStateAltair at the given slot via the beacon node API
func (c *Client) GetStateBySlot(slot types.Slot) (io.Reader, error) {
	return c.GetStateById(IdFromSlot(types.Slot(slot)))
}

func IdFromSlot(s types.Slot) StateOrBlockId {
	return StateOrBlockId(strconv.FormatUint(uint64(s), 10))
}

func (c *Client) GetStateById(stateId StateOrBlockId) (io.Reader, error) {
	statePath := path.Join(get_state_path, string(stateId))
	u := c.urlForPath(statePath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.Client.Do(req)
	defer func() {
		err = r.Body.Close()
	}()
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	b := bytes.NewBuffer(nil)
	_, err = io.Copy(b, r.Body)
	if err != nil {
		return nil, errors.Wrap(err, "error reading http response body from GetStateById")
	}
	return b, nil
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

// GetForkSchedule retrieve all forks, past present and future, of which this node is aware.
func (c *Client) GetForkSchedule() (params.OrderedForkSchedule, error) {
	u := c.urlForPath(get_fork_schedule_path)
	log.Printf("requesting %s", u.String())
	r, err := c.Client.Get(u.String())
	defer func() {
		err = r.Body.Close()
	}()
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	fsr := &forkScheduleResponse{}
	err = json.NewDecoder(r.Body).Decode(fsr)
	if err != nil {
		return nil, err
	}
	ofs, err := fsr.OrderedForkSchedule()
	if err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("problem unmarshaling %s response", get_fork_schedule_path))
	}
	return ofs, nil
}

// GetWeakSubjectivity calls a proposed API endpoint that is unique to prysm
// This api method does the following:
// - computes weak subjectivity epoch
// - finds the highest non-skipped block preceding the epoch
// - returns the htr of the found block and returns this + the value of state_root from the block
func (c *Client) GetWeakSubjectivity() (*WeakSubjectivityData, error) {
	u := c.urlForPath(get_weak_subjectivity_path)
	r, err := c.Client.Get(u.String())
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	v := &apimiddleware.WeakSubjectivityResponse{}
	err = json.NewDecoder(r.Body).Decode(v)
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
