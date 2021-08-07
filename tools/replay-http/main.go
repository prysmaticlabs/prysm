package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"time"
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
	scanner := bufio.NewScanner(f)
	requests := make([][]byte, 0)
	for scanner.Scan() {
		scanned := scanner.Bytes()
		buf := bytes.NewBuffer(make([]byte, base64.StdEncoding.DecodedLen(len(scanned))))
		if _, err := base64.StdEncoding.Decode(buf.Bytes(), scanned); err != nil {
			panic(err)
		}
		requests = append(requests, buf.Bytes())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	client := &http.Client{
		Timeout: time.Second * 10,
	}
	u, err := url.Parse(*endpoint)
	if err != nil {
		panic(err)
	}
	fmt.Println(len(requests))
	for _, rawReq := range requests {
		r, err := http.ReadRequest(bufio.NewReader(bytes.NewBuffer(rawReq)))
		if err != nil {
			panic(err)
		}
		r.RequestURI = ""
		r.URL = u
		// fmt.Println(r)
		if _, err := client.Do(r); err != nil {
			panic(err)
		}
		time.Sleep(time.Millisecond * 200)
	}
}
