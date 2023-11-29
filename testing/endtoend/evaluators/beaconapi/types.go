package beaconapi

import "github.com/prysmaticlabs/prysm/v4/consensus-types/primitives"

type meta interface {
	getBasePath() string
	sszEnabled() bool
	enableSsz()
	getSszResp() []byte
	setSszResp(resp []byte)
	getStart() primitives.Epoch
	setStart(start primitives.Epoch)
	getReq() interface{}
	setReq(req interface{})
	getPResp() interface{}
	getLResp() interface{}
	getParams(epoch primitives.Epoch) []string
	setParams(f func(primitives.Epoch) []string)
	getCustomEval() func(interface{}, interface{}) error
	setCustomEval(f func(interface{}, interface{}) error)
}

type metadata[Resp any] struct {
	basePath   string
	ssz        bool
	start      primitives.Epoch
	req        interface{}
	pResp      *Resp
	lResp      *Resp
	sszResp    []byte
	params     func(currentEpoch primitives.Epoch) []string
	customEval func(interface{}, interface{}) error
}

func (m *metadata[Resp]) getBasePath() string {
	return m.basePath
}

func (m *metadata[Resp]) sszEnabled() bool {
	return m.ssz
}

func (m *metadata[Resp]) enableSsz() {
	m.ssz = true
}

func (m *metadata[Resp]) getSszResp() []byte {
	return m.sszResp
}

func (m *metadata[Resp]) setSszResp(resp []byte) {
	m.sszResp = resp
}

func (m *metadata[Resp]) getStart() primitives.Epoch {
	return m.start
}

func (m *metadata[Resp]) setStart(start primitives.Epoch) {
	m.start = start
}

func (m *metadata[Resp]) getReq() interface{} {
	return m.req
}

func (m *metadata[Resp]) setReq(req interface{}) {
	m.req = req
}

func (m *metadata[Resp]) getPResp() interface{} {
	return m.pResp
}

func (m *metadata[Resp]) getLResp() interface{} {
	return m.lResp
}

func (m *metadata[Resp]) getParams(epoch primitives.Epoch) []string {
	if m.params == nil {
		return nil
	}
	return m.params(epoch)
}

func (m *metadata[Resp]) setParams(f func(currentEpoch primitives.Epoch) []string) {
	m.params = f
}

func (m *metadata[Resp]) getCustomEval() func(interface{}, interface{}) error {
	return m.customEval
}

func (m *metadata[Resp]) setCustomEval(f func(interface{}, interface{}) error) {
	m.customEval = f
}

func newMetadata[Resp any](basePath string, opts ...metadataOpt) *metadata[Resp] {
	m := &metadata[Resp]{
		basePath: basePath,
		pResp:    new(Resp),
		lResp:    new(Resp),
	}
	for _, o := range opts {
		o(m)
	}
	return m
}

type metadataOpt func(meta)

func withSsz() metadataOpt {
	return func(m meta) {
		m.enableSsz()
	}
}

func withStart(start primitives.Epoch) metadataOpt {
	return func(m meta) {
		m.setStart(start)
	}
}

func withReq(req interface{}) metadataOpt {
	return func(m meta) {
		m.setReq(req)
	}
}

func withParams(f func(currentEpoch primitives.Epoch) []string) metadataOpt {
	return func(m meta) {
		m.setParams(f)
	}
}

func withCustomEval(f func(interface{}, interface{}) error) metadataOpt {
	return func(m meta) {
		m.setCustomEval(f)
	}
}
