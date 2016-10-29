// Copyright 2016 Google Inc. All Rights Reserved.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/api"
)

func tokenRequestHandler(w io.Writer, r *http.Request) (int, error) {
	log.Printf("token request from %s", r.RemoteAddr)
	name := r.FormValue("name")
	if name == "" {
		return 400, fmt.Errorf("missing or empty name parameter from %s", r.RemoteAddr)
	}

	namespace := r.FormValue("namespace")
	if namespace == "" {
		log.Println("token request: namespace missing or empty using default")
		namespace = "default"
	}

	// Use kubectl proxy to lookup the pod details by name
	u := fmt.Sprintf("http://127.0.0.1:8001/api/v1/namespaces/%s/pods/%s", namespace, name)
	resp, err := http.Get(u)
	if err != nil {
		return 500, fmt.Errorf("error during pod (%s) lookup %s", name, err)
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return 500, fmt.Errorf("error parsing pod (%s) details: %s", name, err)
	}

	var pod Pod
	err = json.Unmarshal(data, &pod)
	if err != nil {
		return 500, fmt.Errorf("error parsing pod (%s) details: %s", name, err)
	}

	if pod.Status.PodIP == "" {
		return 412, fmt.Errorf("error missing or empty pod IP (%s)", name)
	}

	policies := pod.Metadata.Annotations["vaultproject.io/policies"]
	if policies == "" {
		return 500, fmt.Errorf("error missing or empty pod vaultproject.io/role annotation (%s)", name)
	}
	ttl := pod.Metadata.Annotations["vaultproject.io/ttl"]
	if ttl == "" {
		ttl = "72h"
	}

	tcr := &api.TokenCreateRequest{
		Policies: strings.Split(policies, ","),
		Metadata: map[string]string{
			"host_ip":   pod.Status.HostIP,
			"namespace": pod.Metadata.Namespace,
			"pod_ip":    pod.Status.PodIP,
			"pod_name":  pod.Metadata.Name,
			"pod_uid":   pod.Metadata.Uid,
		},
		DisplayName: pod.Metadata.Name,
		Period:      ttl,
		NoParent:    true,
		TTL:         ttl,
	}
	secret, err := vaultClient.Auth().Token().Create(tcr)
	if err != nil {
		return 500, fmt.Errorf("error creating wrapped token for pod (%s)", name)
	}

	var wrappedToken bytes.Buffer
	err = json.NewEncoder(&wrappedToken).Encode(&secret.WrapInfo)
	if err != nil {
		return 500, fmt.Errorf("error parsing wrapped token for pod (%s)", name)
	}
	go pushWrappedTokenTo(pod.Status.PodIP, &wrappedToken)

	return 202, nil
}

func pushWrappedTokenTo(ip string, token io.Reader) {
	url := fmt.Sprintf("http://%s", ip)
	resp, err := http.Post(url, "", token)
	if err != nil {
		log.Printf("error pushing wrapped token to %s: %s", url, err)
		return
	}
	if resp.StatusCode != http.StatusOK {
		log.Printf("error pushing wrapped token to %s: %s", url, resp.Status)
		return
	}
	log.Printf("successfully pushed wrapped token to %s", url)
}
