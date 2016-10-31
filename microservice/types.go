package main

import "time"

type Secret struct {
	RequestID     string          `json:"request_id"`
	LeaseID       string          `json:"lease_id"`
	LeaseDuration int             `json:"lease_duration"`
	Renewable     bool            `json:"renewable"`
	Warnings      []string        `json:"warnings"`
	WrapInfo      *SecretWrapInfo `json:"wrap_info,omitempty"`
	Auth          *SecretAuth     `json:"auth,omitempty"`
}

type SecretAuth struct {
	ClientToken   string            `json:"client_token"`
	Accessor      string            `json:"accessor"`
	Policies      []string          `json:"policies"`
	Metadata      map[string]string `json:"metadata"`
	LeaseDuration int               `json:"lease_duration"`
	Renewable     bool              `json:"renewable"`
}

type SecretWrapInfo struct {
	Token           string    `json:"token"`
	TTL             int       `json:"ttl"`
	CreationTime    time.Time `json:"creation_time"`
	WrappedAccessor string    `json:"wrapped_accessor"`
}

type PKIIssueSecret struct {
	Secret `json:",inline"`
	Data   PKIData `json:"data"`
}

type PKIData struct {
	Certificate    string `json:"certificate"`
	IssuingCA      string `json:"issuing_ca"`
	PrivateKey     string `json:"private_key"`
	PrivateKeyType string `json:"private_key_type"`
	SerialNumber   string `json:"serial_number"`
}
