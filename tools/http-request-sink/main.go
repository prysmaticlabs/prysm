// Package main implements a simple, http-request-sink which writes
// incoming http request bodies to an append-only text file at a specified directory.
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/prysmaticlabs/prysm/shared/params"
)

func main() {
	port := flag.Int("port", 8080, "port to listen on")
	writeDirPath := flag.String("write-dir", "", "directory to write an append-only file")
	podName := flag.String("pod-name", "", "The name of the pod running this tool")
	flag.Parse()

	f, err := os.OpenFile(
		filepath.Join(*writeDirPath, fmt.Sprintf("%s.log", *podName)),
		os.O_CREATE|os.O_RDWR,
		params.BeaconIoConfig().ReadWritePermissions,
	)
	if err != nil {
		log.Println(err)
	}
	defer func() {
		if err = f.Close(); err != nil {
			log.Fatal(err)
		}
	}()

	http.HandleFunc("/", func(writer http.ResponseWriter, r *http.Request) {
		reqContent := map[string]interface{}{}
		if err = parseRequest(r, &reqContent); err != nil {
			log.Println(err)
		}
		log.Printf("Capturing request from %s", r.RemoteAddr)
		if err = captureRequest(f, reqContent); err != nil {
			log.Println(err)
		}
	})
	log.Printf("Listening on port %d", *port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
}

func captureRequest(f *os.File, m map[string]interface{}) error {
	enc, err := json.Marshal(m)
	if err != nil {
		return err
	}
	_, err = f.WriteString(fmt.Sprintf("%s\n", enc))
	return err
}

func parseRequest(req *http.Request, unmarshalStruct interface{}) error {
	body, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	if err = req.Body.Close(); err != nil {
		return err
	}
	req.Body = ioutil.NopCloser(bytes.NewBuffer(body))
	return json.Unmarshal(body, unmarshalStruct)
}
