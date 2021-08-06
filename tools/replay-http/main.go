package main

import (
	"bufio"
	"bytes"
	"encoding/base64"
	"flag"
	"fmt"
	"net/http"
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
		requests = append(requests, scanner.Bytes())
	}
	if err := scanner.Err(); err != nil {
		panic(err)
	}
	client := &http.Client{
		Timeout: time.Second * 10,
	}

	for _, r := range requests {
		buf := bytes.NewBuffer(make([]byte, base64.StdEncoding.DecodedLen(len(r))))
		if _, err := base64.StdEncoding.Decode(buf.Bytes(), r); err != nil {
			panic(err)
		}
		fmt.Println(buf.String())
		r, err := http.ReadRequest(bufio.NewReader(buf))
		if err != nil {
			panic(err)
		}
		if _, err := client.Do(r); err != nil {
			panic(err)
		}
	}
}
