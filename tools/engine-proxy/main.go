package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
)

func main() {
	port := flag.Int("port", 8546, "")
	executionEndpoint := flag.String("execution-endpoint", "http://localhost:8545", "")
	http.HandleFunc("/", func(writer http.ResponseWriter, r *http.Request) {
		// Intercept the request and log it.
		reqContent := map[string]interface{}{}
		if err := parseRequest(r, &reqContent); err != nil {
			log.Println(err)
			return
		}
		log.Printf("Capturing request from %s: %v", r.RemoteAddr, reqContent)

		url := r.URL
		url.Host = *executionEndpoint
		proxyReq, err := http.NewRequest(r.Method, url.String(), r.Body)
		if err != nil {
			log.Println(err)
			return
		}
		proxyReq.Header.Set("Host", r.Host)
		proxyReq.Header.Set("X-Forwarded-For", r.RemoteAddr)

		for header, values := range r.Header {
			for _, value := range values {
				proxyReq.Header.Add(header, value)
			}
		}
		client := &http.Client{}
		proxyRes, err := client.Do(proxyReq)
		if err != nil {
			log.Println(err)
			return
		}
		if _, err = io.Copy(writer, proxyRes.Body); err != nil {
			log.Println(err)
			return
		}
	})
	log.Printf("Listening on port %d", *port)
	log.Fatal(http.ListenAndServe(":"+strconv.Itoa(*port), nil))
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
