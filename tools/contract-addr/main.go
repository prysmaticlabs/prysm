/**
 * This tool exists to serve currently configured contract address in k8s.
 * It reads the contract address from a plain text file as provided by etcd.
 */
package main

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

var address = flag.String("address-path", "", "The file path to the plain text file with the contract address")

func main() {
	flag.Parse()
	if *address == "" {
		panic("Contract address filepath not set")
	}

	fmt.Println("Starting on port 8080")

	if err := http.ListenAndServe(":8080", &handler{}); err != nil {
		panic("Failed to run server: " + err.Error())
	}
}

type handler struct{}

func (h *handler) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	dat, err := ioutil.ReadFile(*address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	_, err = io.WriteString(w, string(dat))
	if err != nil {
		fmt.Printf("Failed to write response: %v", err)
	}
}
