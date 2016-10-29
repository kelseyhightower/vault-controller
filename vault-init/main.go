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
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path"
	"syscall"
	"time"

	"github.com/fsnotify/fsnotify"
)

const tokenFile = "/var/run/secrets/vaultproject.io/secret.json"

func main() {
	log.Println("Starting vault-init...")

	name := os.Getenv("POD_NAME")
	if name == "" {
		log.Fatal("POD_NAME must be set and non-empty")
	}

	namespace := os.Getenv("POD_NAMESPACE")
	if namespace == "" {
		log.Fatal("POD_NAMESPACE must be set and non-empty")
	}

	vaultAddr := os.Getenv("VAULT_ADDR")
	if vaultAddr == "" {
		vaultAddr = "http://vault:8200"
	}

	vaultControllerAddr := os.Getenv("VAULT_CONTROLLER_ADDR")
	if vaultControllerAddr == "" {
		vaultControllerAddr = "http://vault-controller"
	}

	http.Handle("/", tokenHandler{vaultAddr})
	go func() {
		log.Fatal(http.ListenAndServe(":80", nil))
	}()

	// Ensure the token handler is ready.
	time.Sleep(time.Millisecond * 300)

	// Remove exiting token files before requesting a new one.
	if err := os.Remove(tokenFile); err != nil {
		log.Printf("could not remove token file at %s: %s", tokenFile, err)
	}

	// Set up a file watch on the wrapped vault token.
	tokenWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("could not create watcher: %s", err)
	}
	err = tokenWatcher.Add(path.Dir(tokenFile))
	if err != nil {
		log.Fatalf("could not add watcher: %v", err)
	}

	done := make(chan bool)
	retryDelay := 5 * time.Second
	go func() {
		for {
			err := requestToken(vaultControllerAddr, name, namespace)
			if err != nil {
				log.Printf("token request: Request error %v; retrying in %v", err, retryDelay)
				time.Sleep(retryDelay)
				continue
			}
			log.Println("Token request complete; waiting for callback...")
			select {
			case <-time.After(time.Second * 30):
				log.Println("token request: Timeout waiting for callback")
				break
			case <-tokenWatcher.Events:
				tokenWatcher.Close()
				close(done)
				return
			case err := <-tokenWatcher.Errors:
				log.Println("token request: error watching the token file", err)
			}
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-quit:
		log.Printf("Shutdown signal received, exiting...")
	case <-done:
		log.Println("Successfully obtained and unwrapped the vault token, exiting...")
	}
}

func requestToken(vaultControllerAddr, name, namespace string) error {
	u := fmt.Sprintf("%s/token?name=%s&namespace=%s", vaultControllerAddr, name, namespace)
	log.Printf("Requesting a new wrapped token from %s", vaultControllerAddr)
	resp, err := http.Post(u, "", nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 202 {
		return nil
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	return fmt.Errorf("%s", data)
}
