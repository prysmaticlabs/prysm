package openapi

import (
	"encoding/base64"
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
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/pkg/errors"
	types "github.com/prysmaticlabs/eth2-types"
	"github.com/prysmaticlabs/prysm/config/params"
	"github.com/prysmaticlabs/prysm/encoding/bytesutil"
	ethpb "github.com/prysmaticlabs/prysm/proto/prysm/v1alpha1"
	log "github.com/sirupsen/logrus"
)

const GET_WEAK_SUBJECTIVITY_CHECKPOINT_EPOCH_PATH = "/eth/v1alpha1/beacon/weak_subjectivity_checkpoint_epoch"
const GET_WEAK_SUBJECTIVITY_CHECKPOINT_PATH = "/eth/v1alpha1/beacon/weak_subjectivity_checkpoint"
const GET_SIGNED_BLOCK_PATH = "/eth/v2/beacon/blocks"
const GET_STATE_PATH = "/eth/v2/debug/beacon/states"
const GET_FORK_SCHEDULE_PATH = "/eth/v1/config/fork_schedule"

// ClientOpt is a functional option for the Client type (http.Client wrapper)
type ClientOpt func(*Client)

// WithTimeout sets the .Timeout attribute of the wrapped http.Client
func WithTimeout(timeout time.Duration) ClientOpt {
	return func(c *Client) {
		c.c.Timeout = timeout
	}
}

// Client provides a collection of helper methods for calling the beacon node OpenAPI endpoints
type Client struct {
	c      *http.Client
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
// host is the base host + port used to construct request urls. This value can be
// a URL string, or NewClient will assume an http endpoint if just `host:port` is used.
func NewClient(host string, opts ...ClientOpt) (*Client, error) {
	host, err := validHostname(host)
	if err != nil {
		return nil, err
	}
	c := &Client{
		c:      &http.Client{},
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

type checkpointEpochResponse struct {
	Epoch string
}

// GetWeakSubjectivityCheckpointEpoch retrieves the epoch for a finalized block within the weak subjectivity period.
// The main use case for method is to find the slot that can be used to fetch a block within the subjectivity period
// which can be used to sync (as an alternative to syncing from genesis).
func (c *Client) GetWeakSubjectivityCheckpointEpoch() (uint64, error) {
	u := c.urlForPath(GET_WEAK_SUBJECTIVITY_CHECKPOINT_EPOCH_PATH)
	r, err := c.c.Get(u.String())
	if err != nil {
		return 0, err
	}
	if r.StatusCode != http.StatusOK {
		return 0, non200Err(r)
	}
	jsonr := &checkpointEpochResponse{}
	err = json.NewDecoder(r.Body).Decode(jsonr)
	if err != nil {
		return 0, err
	}
	return strconv.ParseUint(jsonr.Epoch, 10, 64)
}

type wscResponse struct {
	BlockRoot string
	StateRoot string
	Epoch     string
}

// GetWeakSubjectivityCheckpoint calls an entirely different rpc method than GetWeakSubjectivityCheckpointEpoch
// This endpoint is much slower, because it uses stategen to generate the BeaconState at the beginning of the
// weak subjectivity epoch. This also results in a different set of state+block roots, because this endpoint currently
// uses the state's latest_block_header for the block hash, while the checkpoint sync code assumes that the block
// is from the first slot in the epoch and the state is from the subesequent slot.
func (c *Client) GetWeakSubjectivityCheckpoint() (*ethpb.WeakSubjectivityCheckpoint, error) {
	u := c.urlForPath(GET_WEAK_SUBJECTIVITY_CHECKPOINT_PATH)
	r, err := c.c.Get(u.String())
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	v := &wscResponse{}
	err = json.NewDecoder(r.Body).Decode(v)
	if err != nil {
		return nil, err
	}
	epoch, err := strconv.ParseUint(v.Epoch, 10, 64)
	if err != nil {
		return nil, err
	}
	blockRoot, err := base64.StdEncoding.DecodeString(v.BlockRoot)
	if err != nil {
		return nil, err
	}
	stateRoot, err := base64.StdEncoding.DecodeString(v.StateRoot)
	if err != nil {
		return nil, err
	}
	return &ethpb.WeakSubjectivityCheckpoint{
		Epoch:     types.Epoch(epoch),
		BlockRoot: blockRoot,
		StateRoot: stateRoot,
	}, nil
}

// GetBlockBySlot queries the beacon node API for the SignedBeaconBlockAltair for the given slot
func (c *Client) GetBlockBySlot(slot uint64) (io.Reader, error) {
	blockPath := path.Join(GET_SIGNED_BLOCK_PATH, strconv.FormatUint(slot, 10))
	u := c.urlForPath(blockPath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	return r.Body, nil
}

// GetBlockByRoot retrieves a SignedBeaconBlockAltair with the given root via the beacon node API
func (c *Client) GetBlockByRoot(blockHex string) (io.Reader, error) {
	blockPath := path.Join(GET_SIGNED_BLOCK_PATH, blockHex)
	u := c.urlForPath(blockPath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	return r.Body, nil
}

// GetStateByRoot retrieves a BeaconStateAltair with the given root via the beacon node API
func (c *Client) GetStateByRoot(stateHex string) (io.Reader, error) {
	statePath := path.Join(GET_STATE_PATH, stateHex)
	u := c.urlForPath(statePath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	return r.Body, nil
}

// GetStateBySlot retrieves a BeaconStateAltair at the given slot via the beacon node API
func (c *Client) GetStateBySlot(slot uint64) (io.Reader, error) {
	statePath := path.Join(GET_STATE_PATH, strconv.FormatUint(slot, 10))
	u := c.urlForPath(statePath)
	log.Printf("requesting %s", u.String())
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/octet-stream")
	r, err := c.c.Do(req)
	if err != nil {
		return nil, err
	}
	if r.StatusCode != http.StatusOK {
		return nil, non200Err(r)
	}
	return r.Body, nil
}

type forkScheduleResponse struct {
	Data []struct {
		PreviousVersion string `json:"previous_version"`
		CurrentVersion  string `json:"current_version"`
		Epoch           string `json:"epoch"`
	}
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

func (c *Client) GetForkSchedule() (params.OrderedForkSchedule, error) {
	u := c.urlForPath(GET_FORK_SCHEDULE_PATH)
	log.Printf("requesting %s", u.String())
	r, err := c.c.Get(u.String())
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
		return nil, errors.Wrap(err, fmt.Sprintf("problem unmarshaling %s response", GET_FORK_SCHEDULE_PATH))
	}
	return ofs, nil
}

func non200Err(response *http.Response) error {
	bodyBytes, err := ioutil.ReadAll(response.Body)
	var body string
	if err != nil {
		body = "(Unable to read response body.)"
	} else {
		body = "response body:\n" + string(bodyBytes)
	}
	return fmt.Errorf("Got non-200 status code = %d requesting %s. %s", response.StatusCode, response.Request.URL, body)
}
