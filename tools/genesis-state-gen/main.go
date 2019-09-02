package main

import (
	"gopkg.in/yaml.v2"
)

type keyPair struct {
	PrivateKey []byte `yaml:"privkey"`
	PublicKey  []byte `yaml:"pubkey"`
}

type validatorKeys struct {
	Keys []*keyPair
}

func main() {
	enc := []byte{}
	var ks *validatorKeys
	if err := yaml.Unmarshal(enc, ks); err != nil {
		panic(err)
	}
}
