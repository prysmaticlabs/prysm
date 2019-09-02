package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

var (
	inputFile = flag.String("validator-keys-yaml", "", "Input validator keys YAML file")
)

type keyPair struct {
	PrivateKey string `yaml:"privkey"`
	PublicKey  string `yaml:"pubkey"`
}

func main() {
	flag.Parse()
	f, err := os.Open(*inputFile)
	if err != nil {
		panic(err)
	}
	enc, err := ioutil.ReadAll(f)
	if err != nil {
		panic(err)
	}
	var ks []*keyPair
	if err := yaml.Unmarshal(enc, &ks); err != nil {
		panic(err)
	}
	fmt.Println(ks)
}
