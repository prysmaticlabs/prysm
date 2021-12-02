package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/prysmaticlabs/prysm/io/file"
	"github.com/prysmaticlabs/prysm/validator/slashing-protection-history/format"
)

type importKeystoresRequestJson struct {
	Keystores          []string `json:"keystores"`
	Passwords          []string `json:"passwords"`
	SlashingProtection string   `json:"slashing_protection"`
}

type importKeystoresResponseJson struct {
	Statuses []*importKeystoresStatusJson `json:"statuses"`
}

type importKeystoresStatusJson struct {
	KeystorePath string `json:"keystore_path"`
	Status       string `json:"status"`
}

func main() {
	entries, err := os.ReadDir(prefix)
	if err != nil {
		panic(err)
	}
	keystoreJsons := make([]string, 0)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		f, err := os.Open(filepath.Join(prefix, entry.Name()))
		if err != nil {
			panic(err)
		}
		defer func() {
			if err := f.Close(); err != nil {
				panic(err)
			}
		}()
		enc, err := ioutil.ReadAll(f)
		if err != nil {
			panic(err)
		}
		k := make(map[string]interface{})
		if err := json.Unmarshal(enc, &k); err != nil {
			panic(err)
		}
		jsonData, err := json.Marshal(k)
		if err != nil {
			panic(err)
		}
		keystoreJsons = append(keystoreJsons, string(jsonData))
	}
	encHistory, err := file.ReadFileAsBytes(filepath.Join(histPrefix, "slashing_protection.json"))
	if err != nil {
		panic(err)
	}
	history := &format.EIPSlashingProtectionFormat{}
	if err := json.Unmarshal(encHistory, history); err != nil {
		panic(err)
	}
	jsonHistory, err := json.Marshal(history)
	if err != nil {
		panic(err)
	}
	passwords := make([]string, len(keystoreJsons))
	for i := range passwords {
		passwords[i] = ""
	}

	postReq := importKeystoresRequestJson{
		Keystores:          keystoreJsons,
		Passwords:          passwords,
		SlashingProtection: string(jsonHistory),
	}
	fmt.Println(len(postReq.Keystores))
	jsonData, err := json.Marshal(postReq)
	if err != nil {
		panic(err)
	}
	url := "http://localhost:7500/eth/v1/keystores"
	request, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		panic(err)
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Add("Authorization", bearer)

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		panic(err)
	}
	defer func() {
		if err := response.Body.Close(); err != nil {
			panic(err)
		}
	}()

	fmt.Println("response Status:", response.Status)
	fmt.Println("response Headers:", response.Header)
	body, _ := ioutil.ReadAll(response.Body)
	fmt.Println("response Body:", string(body))
}
