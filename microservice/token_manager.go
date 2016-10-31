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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"sync"
	"time"
)

type TokenManager struct {
	addr  string
	Token string
	wg    *sync.WaitGroup
	done  chan bool
}

func NewTokenManager(addr, tokenFile string) (*TokenManager, error) {
	log.Printf("Reading vault secret file from %s", tokenFile)
	data, err := ioutil.ReadFile(tokenFile)
	if err != nil {
		return nil, fmt.Errorf("could not read secret file: %v", err)
	}

	var secret Secret
	err = json.Unmarshal(data, &secret)
	if err != nil {
		return nil, err
	}

	tm := &TokenManager{
		addr:  addr,
		Token: secret.Auth.ClientToken,
		done:  make(chan bool),
		wg:    &sync.WaitGroup{},
	}
	return tm, nil
}

func (tm *TokenManager) StopRenewToken() {
	close(tm.done)
	tm.wg.Wait()
}

func (tm *TokenManager) StartRenewToken() {
	tm.wg.Add(1)

	u := fmt.Sprintf("%s/v1/auth/token/renew-self", tm.addr)
	retryDelay := 5 * time.Second
	for {
		request, err := http.NewRequest("POST", u, nil)
		if err != nil {
			log.Printf("token renew: Renew client token error: %v; retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		request.Header.Add("X-Vault-Token", tm.Token)

		resp, err := http.DefaultClient.Do(request)
		if err != nil {
			log.Printf("token renew: Renew client token error: %v; retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		data, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			resp.Body.Close()
			log.Printf("token renew: Renew client token error: %v; retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}
		resp.Body.Close()

		var secret Secret
		err = json.Unmarshal(data, &secret)
		if err != nil {
			log.Printf("token renew: Renew client token error: %v; retrying in %v", err, retryDelay)
			time.Sleep(retryDelay)
			continue
		}

		nextRenew := secret.Auth.LeaseDuration / 2
		log.Printf("Successfully renewed the client token; next renewal in %d seconds", nextRenew)

		select {
		case <-time.After(time.Duration(nextRenew) * time.Second):
			break
		case <-tm.done:
			tm.wg.Done()
			return
		}
	}
}
