package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
)

var url = "http://localhost:8545"

// This prober serves a /healthz method which acts as a proxy to a locally
// running ethereum node. If the node is still syncing, /healthz will return a
// non-200 error.
func main() {
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {

		var jsonStr = []byte(`{"jsonrpc":"2.0","method":"eth_syncing","params":[],"id":1}`)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonStr))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {

			w.WriteHeader(500)
			_, err := fmt.Fprintf(w, "Error probing node: %v", err)
			if err != nil {
				panic(err)
			}
			return
		}
		defer resp.Body.Close()

		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			w.WriteHeader(500)
			_, err := fmt.Fprintf(w, "Error reading response: %v", err)
			if err != nil {
				panic(err)
			}
			return
		}

		jsonMap := make(map[string]interface{})
		err = json.Unmarshal(body, &jsonMap)
		if err != nil {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Failed to unmarshal json: %v", err)
			return
		}
		ok := jsonMap["result"] == false
		if !ok {
			w.WriteHeader(500)
			fmt.Fprintf(w, "Not synced: %v", jsonMap)
			return
		}

		fmt.Fprint(w, "ok")
		return
	})

	fmt.Println("Serving requests on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
