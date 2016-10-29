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
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/hashicorp/vault/api"
)

var vaultClient *api.Client

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

	http.Handle("/token", handler{tokenRequestHandler})
	go func() {
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	log.Println("Listening for token requests.")
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Printf("Shutdown signal received, exiting...")
}

type handler struct {
	f func(io.Writer, *http.Request) (int, error)
}

func (h handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	code, err := h.f(w, r)
	w.WriteHeader(code)
	if err != nil {
		log.Printf("%v", err)
		fmt.Fprintf(w, "%v", err)
	}
}
