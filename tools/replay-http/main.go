/**
Tool for replaying http request from a gzipped file of base64, line-delimited
http raw requests. Credits to https://gist.github.com/kasey/c9e663eae5baebbf8fbe548c2b1d961b.
*/
package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"flag"
	"io"
	"net/http"
	"os"

	log "github.com/sirupsen/logrus"
)

var (
	filePath = flag.String("file", "", "http base64 encoded requests file")
	endpoint = flag.String("endpoint", "", "jaeger endpoint")
)

func main() {
	flag.Parse()
	f, err := os.Open(*filePath)
	if err != nil {
		panic(err)
	}
	gzreader, err := gzip.NewReader(f)
	defer func() {
		if err := gzreader.Close(); err != nil {
			log.WithError(err).Error("Could not close gzip")
		}
		if err := f.Close(); err != nil {
			log.WithError(err).Error("Could not close stdout file")
		}
	}()
	lr := bufio.NewReader(gzreader)
	for {
		line, err := lr.ReadBytes([]byte("\n")[0])
		if err == io.EOF {
			os.Exit(0)
		}
		if err != nil {
			log.Fatal(err)
		}
		line = line[0 : len(line)-1]
		decoded := make([]byte, base64.StdEncoding.DecodedLen(len(line)))
		_, err = base64.StdEncoding.Decode(decoded, line)
		if err != nil {
			log.Fatal(err)
		}
		dbuf := bytes.NewBuffer(decoded)
		req, err := http.ReadRequest(bufio.NewReader(dbuf))
		if err != nil {
			log.Fatal(err)
		}
		req.URL.Scheme = "http"
		req.URL.Host = *endpoint
		req.RequestURI = ""
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			log.Fatal(err)
		}
		respBuf := bytes.NewBuffer(nil)
		if err := resp.Write(respBuf); err != nil {
			log.Fatal(err)
		}
		log.Print(respBuf.String())
	}
}
