package config

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func UnmarshalFromURL(ctx context.Context, from string, to interface{}) error {
	u, err := url.ParseRequestURI(from)
	if err != nil {
		return err
	}
	if u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid URL: %s", from)
	}
	req, reqerr := http.NewRequestWithContext(ctx, http.MethodGet, from, nil)
	if reqerr != nil {
		return errors.Wrap(reqerr, "failed to create http request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		return errors.Wrap(resperr, "failed to send http request")
	}
	defer func(Body io.ReadCloser) {
		err = Body.Close()
		if err != nil {
			log.WithError(err).Error("failed to close response body")
		}
	}(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("http request to %v failed with status code %d", from, resp.StatusCode)
	}
	if decodeerr := json.NewDecoder(resp.Body).Decode(&to); decodeerr != nil {
		return errors.Wrap(decodeerr, "failed to decode http response")
	}
	return nil
}

func UnmarshalFromFile(ctx context.Context, from string, to interface{}) error {
	if ctx == nil {
		return errors.New("node: nil context passed to unmarshalFromFile")
	}
	cleanpath := filepath.Clean(from)
	b, err := os.ReadFile(cleanpath)
	if err != nil {
		return errors.Wrap(err, "failed to open file")
	}

	if err := yaml.Unmarshal(b, to); err != nil {
		return errors.Wrap(err, "failed to unmarshal yaml file")
	}

	return nil
}
