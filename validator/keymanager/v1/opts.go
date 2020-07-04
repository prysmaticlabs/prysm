package v1

import (
	"encoding/json"
	"io/ioutil"
	"strings"
)

// decodeOpts decodes a value in to an options container.
// The input can be either JSON data or a path to a file containing JSON.
// This function returns an error if there is a problem decoding the input.
func decodeOpts(input string, res interface{}) error {
	if input == "" {
		// Empty input is okay.
		return nil
	}

	var data []byte
	if strings.HasPrefix(input, "{") {
		// Looks like straight JSON.
		data = []byte(input)
	} else {
		// Assume it's a path.
		file, err := ioutil.ReadFile(input)
		if err != nil {
			return err
		}
		data = file
	}

	return json.Unmarshal(data, res)
}
