package driver

import (
	"encoding/json"
	"os"
	"path"
	"strconv"
	"time"
)

type Recorder struct {
	base string
	t    time.Time
}

func NewRecorder() (*Recorder, error) {
	base := os.Getenv("PWD")
	r := &Recorder{base: base, t: time.Now()}
	if err := r.Mkdir(); err != nil {
		return nil, err
	}
	return r, nil
}
func (r *Recorder) Dir() string {
	return path.Join(r.base, strconv.FormatInt(r.t.UTC().UnixNano(), 10))
}

func (r *Recorder) Mkdir() error {
	return os.MkdirAll(r.Dir(), 0755)
}

func (r *Recorder) RecordRequest(args []string, req *DriverRequest) error {
	b, err := json.Marshal(struct {
		Args    []string
		Request *DriverRequest
	}{
		Args:    args,
		Request: req,
	})
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(r.Dir(), "request.json"), b, 0644)
}

func (r *Recorder) RecordResponse(resp *driverResponse) error {
	b, err := json.Marshal(resp)
	if err != nil {
		return err
	}
	return os.WriteFile(path.Join(r.Dir(), "response.json"), b, 0644)
}
