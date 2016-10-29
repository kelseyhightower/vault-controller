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
	"bytes"
	"encoding/json"
	"io/ioutil"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hashicorp/vault/api"
)

const vaultSecretFile = "/var/run/secrets/vaultproject.io/secret.json"

func main() {
	log.Println("Starting vault-example app...")
	addr := os.Getenv("VAULT_ADDR")
	if addr == "" {
		addr = "http://vault:8200"
	}

	log.Printf("Reading vault secret file from %s", vaultSecretFile)
	data, err := ioutil.ReadFile(vaultSecretFile)
	if err != nil {
		log.Fatalf("could not read secret file: %v", err)
	}
	secret, err := api.ParseSecret(bytes.NewReader(data))
	if err != nil {
		log.Fatalf("could not parse secret: %v", err)
	}

	logSecret(secret)

	client, err := api.NewClient(api.DefaultConfig())
	if err != nil {
		log.Fatal(err)
	}

	client.SetAddress(addr)
	client.SetToken(secret.Auth.ClientToken)

	// The Pod is responsible for renewing the client token.
	retryDelay := 5 * time.Second
	go func() {
		for {
			s, err := client.Auth().Token().RenewSelf(secret.Auth.LeaseDuration)
			if err != nil {
				log.Printf("token renew: Renew client token error: %v; retrying in %v", err, retryDelay)
				time.Sleep(retryDelay)
				continue
			}

			nextRenew := s.Auth.LeaseDuration / 2
			log.Printf("Successfully renewed the client token; next renewal in %d seconds", nextRenew)
			time.Sleep(time.Duration(nextRenew) * time.Second)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutdown signal received, exiting...")
}

// logSecret logs your secret, don't ever do this!
func logSecret(secret *api.Secret) {
	oldLogFlags := log.Flags()
	defer log.SetFlags(oldLogFlags)

	log.SetFlags(0)
	log.Println("==> WARNING: Don't ever write secrets to logs!")
	log.Println("")
	log.Println("The secret is being printed here for demonstration purposes.")
	log.Println("Use the secret details below with the vault cli\nto get more info about the token.")
	log.Println("")
	j, err := json.MarshalIndent(&secret, "", "  ")
	if err != nil {
		log.Fatal(err)
	}
	log.Println(string(j))
	log.Println("")
	log.Println("==> vault-example started! Log data will stream in below:")
}
