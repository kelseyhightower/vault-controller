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
	"crypto/tls"
	"crypto/x509"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

const tokenFile = "/var/run/secrets/vaultproject.io/secret.json"

var (
	addr          string
	clientPKIPath string
	clientPKITTL  string
	clusterDomain string
	hostname      string
	ip            string
	name          string
	namespace     string
	remoteAddr    string
	serverPKIPath string
	serverPKITTL  string
	serviceName   string
	subdomain     string
	vaultAddr     string
	vaultToken    string
)

func main() {
	flag.StringVar(&addr, "addr", "0.0.0.0:443", "HTTPS service address")
	flag.StringVar(&clientPKIPath, "client-pki-path", "", "PKI secret backend issue path (e.g., '/pki/issue/<role name>')")
	flag.StringVar(&clientPKITTL, "client-pki-ttl", "60s", "certificate time to live")
	flag.StringVar(&clusterDomain, "cluster-domain", "cluster.local", "Kubernetes cluster domain")
	flag.StringVar(&hostname, "hostname", "", "hostname as defined by pod.spec.hostname")
	flag.StringVar(&ip, "ip", "", "IP address as defined by pod.status.podIP")
	flag.StringVar(&name, "name", "", "name as defined by pod.metadata.name")
	flag.StringVar(&namespace, "namespace", "default", "namespace as defined by pod.metadata.namespace")
	flag.StringVar(&remoteAddr, "remote-addr", "", "remote server address (e.g., 'service-name:443')")
	flag.StringVar(&serverPKIPath, "server-pki-path", "", "PKI secret backend issue path (e.g., '/pki/issue/<role name>')")
	flag.StringVar(&serverPKITTL, "server-pki-ttl", "60s", "server certificate time to live")
	flag.StringVar(&serviceName, "service-name", "", "Kubernetes service name that resolves to this Pod")
	flag.StringVar(&subdomain, "subdomain", "", "subdomain as defined by pod.spec.subdomain")
	flag.StringVar(&vaultAddr, "vault-addr", "https://vault:8200", "Vault service address")
	flag.Parse()

	var wg sync.WaitGroup
	done := make(chan bool)

	tm, err := NewTokenManager(vaultAddr, tokenFile)
	if err != nil {
		log.Fatal(err)
	}
	vaultToken = tm.Token
	go tm.StartRenewToken()

	if serviceName != "" {
		go startServer()
	}

	if remoteAddr != "" {
		go startClient(done, &wg)
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	close(done)
	log.Printf("Shutdown signal received, exiting...")
	tm.StopRenewToken()
	wg.Wait()
}

func startServer() {
	ipAddresses := []string{ip, "127.0.0.1"}
	c := &PKIConfig{
		Addr:        vaultAddr,
		CommonName:  serviceDomainName(serviceName, namespace, clusterDomain),
		DNSNames:    dnsNames(serviceName, ip, hostname, subdomain, namespace, clusterDomain),
		IPAddresses: ipAddresses,
		IssuePath:   serverPKIPath,
		Token:       vaultToken,
		TTL:         serverPKITTL,
	}
	cm, err := NewCertificateManager(c)
	if err != nil {
		log.Fatal(err)
	}

	go cm.StartRenewCertificate()

	clientCAPool := x509.NewCertPool()
	if ok := clientCAPool.AppendCertsFromPEM(cm.CACertificate); !ok {
		log.Fatal("missing CA certificate")
	}

	server := http.Server{
		Addr: addr,
		TLSConfig: &tls.Config{
			ClientAuth:     tls.RequireAndVerifyClientCert,
			ClientCAs:      clientCAPool,
			GetCertificate: cm.GetCertificate,
		},
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "Hello from %s service", serviceName)
	})

	log.Fatal(server.ListenAndServeTLS("", ""))
}

func startClient(done <-chan bool, wg *sync.WaitGroup) {
	c := &PKIConfig{
		Addr:       vaultAddr,
		CommonName: podDomainName(ip, namespace, clusterDomain),
		IssuePath:  clientPKIPath,
		Token:      vaultToken,
		TTL:        clientPKITTL,
	}
	cm, err := NewCertificateManager(c)
	if err != nil {
		log.Fatal(err)
	}

	go cm.StartRenewCertificate()

	for {
		select {
		case <-time.After(5 * time.Second):
			rootCAPool := x509.NewCertPool()
			if ok := rootCAPool.AppendCertsFromPEM(cm.CACertificate); !ok {
				log.Fatal("missing CA certificate")
			}

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{
					Certificates:       cm.Certificates(),
					InsecureSkipVerify: false,
					RootCAs:            rootCAPool,
				},
			}
			client := &http.Client{Transport: tr}

			resp, err := client.Get(remoteAddr)
			if err != nil {
				log.Println(err)
				continue
			}

			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				log.Println(err)
				continue
			}
			log.Println(string(body))
		case <-done:
			wg.Done()
			return
		}
	}
}

func dnsNames(serverName, ip, hostname, subdomain, namespace, clusterDomain string) []string {
	ns := []string{podDomainName(ip, namespace, clusterDomain)}

	if serverName != "" {
		ns = append(ns, serverName)
	}

	if hostname != "" && subdomain != "" {
		ns = append(ns, podHeadlessDomainName(hostname, subdomain, namespace, clusterDomain))
	}
	return ns
}

func serviceDomainName(name, namespace, domain string) string {
	return fmt.Sprintf("%s.%s.svc.%s", name, namespace, domain)
}

func podDomainName(ip, namespace, domain string) string {
	return fmt.Sprintf("%s.%s.pod.%s", ip, namespace, domain)
}

func podHeadlessDomainName(hostname, subdomain, namespace, domain string) string {
	if hostname == "" || subdomain == "" {
		return ""
	}
	return fmt.Sprintf("%s.%s.%s.svc.%s", hostname, subdomain, namespace, domain)
}
