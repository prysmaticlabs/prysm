package flagutil

import (
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func UnmarshalFromFileOrURL(ctx context.Context, from string, to interface{}) error {
	u, err := url.ParseRequestURI(from)
	if err == nil && u.Scheme != "" && u.Host != "" {
		return UnmarshalFromURL(ctx, u, to)
	}
	return UnmarshalFromFile(from, to)
}

func UnmarshalFromURL(ctx context.Context, from *url.URL, to interface{}) error {
	req, reqerr := http.NewRequestWithContext(ctx, http.MethodGet, from.RequestURI(), nil)
	if reqerr != nil {
		return errors.Wrap(reqerr, "failed to create http request")
	}
	req.Header.Set("Content-Type", "application/json")
	resp, resperr := http.DefaultClient.Do(req)
	if resperr != nil {
		return errors.Wrap(resperr, "failed to send http request")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("http request failed with status code %d", resp.StatusCode)
	}
	if decodeerr := json.NewDecoder(resp.Body).Decode(&to); decodeerr != nil {
		return errors.Wrap(decodeerr, "failed to decode http response")
	}
	return nil
}

func UnmarshalFromFile(from string, to interface{}) error {
	cleanpath := filepath.Clean(from)
	fileExtension := filepath.Ext(cleanpath)
	if fileExtension == ".json" {
		jsonFile, jsonerr := os.Open(cleanpath)
		if jsonerr != nil {
			return errors.Wrap(jsonerr, "failed to open json file")
		}
		// defer the closing of our jsonFile so that we can parse it later on
		defer jsonFile.Close()
		byteValue, readerror := ioutil.ReadAll(jsonFile)
		if readerror != nil {
			return errors.Wrap(readerror, "failed to read json file")
		}
		if unmarshalerr := json.Unmarshal(byteValue, &to); unmarshalerr != nil {
			return errors.Wrap(unmarshalerr, "failed to unmarshal json file")
		}
		return nil
	} else {
		return errors.Errorf("unsupported file extension %s , (ex. '.json')", fileExtension)
	}
}
