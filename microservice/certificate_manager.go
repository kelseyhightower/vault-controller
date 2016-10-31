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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

type PKIConfig struct {
	Addr        string
	CommonName  string
	DNSNames    []string
	IPAddresses []string
	IssuePath   string
	Token       string
	TTL         string
}

type CertificateManager struct {
	PKIConfig     *PKIConfig
	CACertificate []byte

	sync.RWMutex
	certificate *tls.Certificate
}

func NewCertificateManager(config *PKIConfig) (*CertificateManager, error) {
	cm := &CertificateManager{
		PKIConfig: config,
	}
	err := cm.SetCertificate()
	if err != nil {
		return nil, err
	}
	return cm, nil
}

func (cm *CertificateManager) Certificates() []tls.Certificate {
	cm.RLock()
	defer cm.RUnlock()
	return []tls.Certificate{*cm.certificate}
}

func (cm *CertificateManager) GetCertificate(clientHello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	cm.RLock()
	defer cm.RUnlock()
	return cm.certificate, nil
}

func (cm *CertificateManager) SetCertificate() error {
	u := fmt.Sprintf("%s/v1%s", cm.PKIConfig.Addr, cm.PKIConfig.IssuePath)
	parameters := map[string]string{
		"common_name": cm.PKIConfig.CommonName,
		"ttl":         cm.PKIConfig.TTL,
	}

	if cm.PKIConfig.DNSNames != nil {
		parameters["alt_names"] = strings.Join(cm.PKIConfig.DNSNames, ",")
	}
	if cm.PKIConfig.IPAddresses != nil {
		parameters["ip_sans"] = strings.Join(cm.PKIConfig.IPAddresses, ",")
	}

	var body bytes.Buffer
	err := json.NewEncoder(&body).Encode(&parameters)
	if err != nil {
		return fmt.Errorf("certificate manager: error encoding request body: %v", err)
	}

	request, err := http.NewRequest("POST", u, &body)
	if err != nil {
		return fmt.Errorf("certificate manager: error creating pki request: %v", err)
	}
	request.Header.Add("X-Vault-Token", cm.PKIConfig.Token)

	resp, err := http.DefaultClient.Do(request)
	if err != nil {
		return fmt.Errorf("certificate manager: error during pki request: %v", err)
	}
	defer resp.Body.Close()

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("certificate manager: error reading pki response: %v", err)
	}
	if resp.StatusCode != 200 {
		return fmt.Errorf(string(data))
	}

	var secret PKIIssueSecret
	err = json.Unmarshal(data, &secret)
	if err != nil {
		return fmt.Errorf("certificate manager: error parsing pki secret: %v", err)
	}

	var certPEMBlock bytes.Buffer
	certPEMBlock.WriteString(secret.Data.Certificate)
	certPEMBlock.WriteString("\n")
	certPEMBlock.WriteString(secret.Data.IssuingCA)

	c, err := tls.X509KeyPair(certPEMBlock.Bytes(), []byte(secret.Data.PrivateKey))
	if err != nil {
		return fmt.Errorf("certificate manager: error parsing pki certificates: %v", err)
	}

	cm.Lock()
	cm.certificate = &c
	cm.CACertificate = []byte(secret.Data.IssuingCA)
	cm.Unlock()
	return nil
}

func (cm *CertificateManager) StartRenewCertificate() {
	go func() {
		for {
			if cm.certificate == nil {
				time.Sleep(10 * time.Second)
				continue
			}
			x509Cert, err := x509.ParseCertificate(cm.certificate.Certificate[0])
			if err != nil {
				log.Println(err)
				time.Sleep(10 * time.Second)
			}
			renew := x509Cert.NotAfter.Sub(time.Now()).Seconds() / 2
			log.Println("renewing cert in", renew)

			select {
			case <-time.After(time.Second * time.Duration(int64(renew))):
				err := cm.SetCertificate()
				if err != nil {
					log.Println(err)
				}
			}
		}
	}()
}
