# Use Case: Short-lived TLS Certs, TLS Mutual Auth, and Microservices

This tutorial will walk you through deploying a set of microservices that utilize short-lived TLS certificates for secure communication between microservice and their clients. 

## Prerequisites

Be sure to complete the [Deployment Guide](deployment-guide.md) tutorial first.

This tutorial will require remote access to the Vault server running in your cluster. Set up a proxy from your local machine to the Vault server:

```
kubectl -n vault-controller port-forward \
  $(kubectl -n vault-controller \
    get pods -l app=vault \
    -o jsonpath='{.items[0].metadata.name}') \
  8200:8200
```

```
Forwarding from 127.0.0.1:8200 -> 8200
Forwarding from [::1]:8200 -> 8200
```

At this point you can now use the vault cli in a separate terminal to interact with the remote Vault server.

```
export VAULT_ADDR="http://127.0.0.1:8200"
```

```
export VAULT_TOKEN="3e4a5ba1-kube-422b-d1db-844979cab098"
```

Verify the status of the remote Vault server:

```
vault status
```

Output:

```
Sealed: false
Key Shares: 1
Key Threshold: 1
Unseal Progress: 0
Version: 0.6.2
Cluster Name: vault-cluster-963019e9
Cluster ID: 5e36a011-cc89-0dcb-a261-ae8c3c31dfd4

High-Availability Enabled: false
```

## Setup the PKI secret backend

Each Pod will generate a dynamic set of short-lived TLS certificates, using its unique Vault token, from the Vault [PKI secret backend](https://www.vaultproject.io/docs/secrets/pki/index.html). In this sections you'll complete the following tasks:

* Enable the Vault PKI secret backend
* Configure a CA certificate
* Create a Vault policy that enables clients to generate TLS certificates
* Create PKI roles to limit usage of TLS certificates 

### Enable the Vault PKI secret backend

Mount the PKI secret backend:

```
vault mount pki
```

### Configure a CA certificate

Generate a root certificate:

```
vault mount-tune -max-lease-ttl=87600h pki
```

```
vault write pki/root/generate/internal common_name=cluster.local ttl=87600h
```

```
vault write pki/config/urls issuing_certificates="http://vault:8200/v1/pki/ca"
```

```
vault write pki/config/urls issuing_certificates="http://vault:8200/v1/pki/ca" \
  crl_distribution_points="http://vault:8200/v1/pki/crl"
```

### Configure the PKI roles

In this section you'll setup two PKI roles which allow us to define a policy used when generating TLS certificates.

Create the client PKI role:

```
vault write pki/roles/client \
  allowed_domains="cluster.local" \
  allow_subdomains="true" \
  client_flag="true" \
  max_ttl="72h" \
  server_flag="false"
```

Create the server PKI role:

```
vault write pki/roles/server \
  allow_any_name="true" \
  allowed_domains="cluster.local" \
  allow_subdomains="true" \
  client_flag="false" \
  max_ttl="72h" \
  enforce_hostnames="false" \
  server_flag="true"
```

The `allow_any_name` and `enforce_hostnames` flags are being set here to enable the ability to generate server certificates that contain Kubernetes service names in the DNS Subject Alternative Names field of the issued certificate. This means clients can lookup the service using a DNS short name such as `foo`, which maps to `foo.default.svc.cluster.local`, and still be able to validate the server side certificate. 

## Create a Vault Policy

In this section we need to create a Vault policy that will enable Pods to generate certificates. Pods generate secrets by writing to a PKI issue path based on the role (`/pki/issue/<role name>`).

```
vault policy-write microservice policies/microservice.hcl
```

At this point tokens with the `microservice` policy can generate TLS certs from the following paths:

```
pki/issue/client
pki/issue/server
```

The `vaultproject.io/policies` Pod annonation can be used to generate tokens with a set of policies. To include the microservice policy add the following annotation to a Pod spec:

```
vaultproject.io/policies: "default,microservice"
```

The default policy is required because it enables you to renew tokens and other house keeping tasks.


## Deploy the Server Service

The `server` Pod will utilize the `vault-init` container to obtain a dedicated Vault token from the Vault Controller. Once the `server` container starts it will fetch a short-lived TLS certificate from the Vault PKI secret backend and start serving traffic on port 443.

The TLS server certificate will be validate for the following IPs and DNS names:

* The service name as configured with the `--service-name` flag
* IP SANs: POD IP address and 127.0.0.1
* service-name.namespace.svc.cluster.local
* pod-ip.namespace.pod.cluster.local
* hostname.subdoamin.namespace.svc.cluster.local

> See the [Using DNS Pod and Services](http://kubernetes.io/docs/admin/dns) guide for more information.

Create the `server` ReplicaSet:

```
kubectl -n vault-controller create -f replicasets/server.yaml
```


Expose the `server` Pods internally:

```
kubectl -n vault-controller create -f services/server.yaml
```

View the logs for the `server` Pod:

```
kubectl -n vault-controller logs \
  $(kubectl -n vault-controller \
    get pods -l app=server \
    -o jsonpath='{.items[0].metadata.name}')
```

Log Output:

```
2016/10/31 04:07:44 Reading vault secret file from /var/run/secrets/vaultproject.io/secret.json
2016/10/31 04:07:44 Successfully renewed the client token; next renewal in 43200 seconds
2016/10/31 04:07:45 renewing cert in 29 seconds
```

At this point the other Pods can access the `server` service using the `https://server` address, but each client will require a signed client certificate from the PKI secret backend.

## Deploy the client

In this section you'll deploy a client Pod that will talk to the `server` service. The client Pods will also utilize the `vault-init` container to get a unique Vault token that will be used to generate a set of short-lived client certificates. 

Create the `client` ReplicaSet:

```
kubectl -n vault-controller create -f replicasets/client.yaml
```

Once the `client` container starts it will obtain a short-lived client certificate from the PKI path `pki/issue/client`. For demonstration purposes each TLS certificate is generated with a TTL of 60 seconds and automatically renewed by the `client` and server Pods before they expire. The certificate TTL can be adjusted using the `-client-pki-ttl` flag.

View the logs for the `client` Pod:

```
kubectl -n vault-controller logs \
  $(kubectl -n vault-controller \
    get pods -l app=client \
    -o jsonpath='{.items[0].metadata.name}')
```

Log Output:

```
2016/10/31 04:24:44 Reading vault secret file from /var/run/secrets/vaultproject.io/secret.json
2016/10/31 04:24:44 Successfully renewed the client token; next renewal in 43200 seconds
2016/10/31 04:24:45 renewing cert in 29.726472304
2016/10/31 04:24:50 Hello from server service
2016/10/31 04:24:55 Hello from server service
```

At this point the `client` Pod contacts the `server` service every 5 minutes, and continuously renew it Vault token and client certificate in the background.

## Next Steps

The `kelseyhightower/microservice` container can be used in both client and server mode by setting the appropriate flags. At this point you can test different configurations, service names, and deployments of the `kelseyhightower/microservice` container to get a fell for working with vault and short-lived TLS certificates.

```
microservice -h
```
```
Usage of microservice:
  -addr string
    	HTTPS service address (default "0.0.0.0:443")
  -client-pki-path string
    	PKI secret backend issue path (e.g., '/pki/issue/<role name>')
  -client-pki-ttl string
    	certificate time to live (default "60s")
  -cluster-domain string
    	Kubernetes cluster domain (default "cluster.local")
  -hostname string
    	hostname as defined by pod.spec.hostname
  -ip string
    	IP address as defined by pod.status.podIP
  -name string
    	name as defined by pod.metadata.name
  -namespace string
    	namespace as defined by pod.metadata.namespace (default "default")
  -remote-addr string
    	remote server address (e.g., 'service-name:443')
  -server-pki-path string
    	PKI secret backend issue path (e.g., '/pki/issue/<role name>')
  -server-pki-ttl string
    	server certificate time to live (default "60s")
  -service-name string
    	Kubernetes service name that resolves to this Pod
  -subdomain string
    	subdomain as defined by pod.spec.subdomain
  -vault-addr string
    	Vault service address (default "https://vault:8200")
```
