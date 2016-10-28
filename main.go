// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//     http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/vault/api"
)

var (
	vaultClient *api.Client
)

func main() {
	log.Println("Starting vault-controller app...")

	if os.Getenv("VAULT_TOKEN") == "" {
		log.Fatal("VAULT_TOKEN must be set and non-empty")
	}
	if os.Getenv("VAULT_WRAP_TTL") == "" {
		os.Setenv("VAULT_WRAP_TTL", "120")
	}

	var err error
	config := api.DefaultConfig()
	vaultClient, err = api.NewClient(config)
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/token", tokenRequestHandler)
	go func() {
		http.ListenAndServe(":80", nil)
	}()

	log.Println("Listening for token requests.")
	signalChan := make(chan os.Signal, 1)
	signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
	for {
		select {
		case <-signalChan:
			log.Printf("Shutdown signal received, exiting...")
			os.Exit(0)
		}
	}
}
