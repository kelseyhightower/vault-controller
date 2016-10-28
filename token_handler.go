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
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"strings"

	"github.com/hashicorp/vault/api"
)

func tokenRequestHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("token request from %s", r.RemoteAddr)
	name := r.FormValue("name")
	if name == "" {
		log.Printf("token request: empty name from %s\n", r.RemoteAddr)
		w.WriteHeader(400)
		io.WriteString(w, "missing or empty name parameter.")
		return
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
		log.Printf("token request: error during pod (%s) lookup: %s", name, err)
		w.WriteHeader(500)
		io.WriteString(w, fmt.Sprintf("error during pod (%s) lookup", name))
		return
	}
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Printf("token request: error parsing pod (%s) details: %s", name, err)
		w.WriteHeader(500)
		io.WriteString(w, fmt.Sprintf("error during pod (%s) lookup", name))
		return
	}

	var pod Pod
	err = json.Unmarshal(data, &pod)
	if err != nil {
		log.Printf("token request: error parsing pod (%s) details: %s", name, err)
		w.WriteHeader(500)
		io.WriteString(w, fmt.Sprintf("error during pod (%s) lookup", name))
		return
	}

	if pod.Status.PodIP == "" {
		log.Printf("token request: error missing or empty pod (%s) IP", name)
		w.WriteHeader(412)
		io.WriteString(w, "error missing or empty pod IP")
		return
	}

	policies := pod.Metadata.Annotations["vaultproject.io/policies"]
	if policies == "" {
		log.Printf("token request: error missing or empty pod (%s) vaultproject.io/role annotation", name)
		w.WriteHeader(412)
		io.WriteString(w, "error missing or empty vaultproject.io/role annotation")
		return
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
		log.Printf("token request: error creating wrapped token for pod (%s)", name)
		w.WriteHeader(500)
		io.WriteString(w, "error creating wrapped token")
		return
	}

	var wrappedToken bytes.Buffer
	err = json.NewEncoder(&wrappedToken).Encode(&secret.WrapInfo)
	if err != nil {
		log.Printf("token request: error parsing wrapped token for pod (%s)", name)
		w.WriteHeader(500)
		io.WriteString(w, "error creating wrapped token")
		return
	}

	// Push the wrapped token to the pod using the pod IP.
	go func() {
		podURL := fmt.Sprintf("http://%s", pod.Status.PodIP)
		resp, err := http.Post(podURL, "", &wrappedToken)
		if err != nil {
			log.Printf("error pushing wrapped token to %s: %s", podURL, err)
			return
		}
		if resp.StatusCode != 200 {
			log.Printf("error pushing wrapped token to %s: %s", podURL, resp.Status)
			return
		}
		log.Printf("successfully pushed wrapped token to %s", podURL)
		return
	}()

	w.WriteHeader(202)
	return
}
