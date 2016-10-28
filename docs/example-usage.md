# Example Usage

This tutorial will walk you through deploying the following components:

* Vault Example 0.0.1

The `vault-example` Pod utilizes the `vault-init` container to request a wrapped token from the Vault Controller. The `vault-init` container performs the following actions during the Pod initialization phase:

* Requests a wrapped token from the Vault Controller
* Unwraps the token by communicating with a Vault server
* Writes the unwrapped token to a shared volume at `/var/run/secrets/vaultproject.io/secret.json`
* Exits so the Pod creation process can continue

## Deploy the Vault Example application

```
kubectl -n vault-controller create -f replicasets/vault-example.yaml
```

Once a `vault-example` Pod is scheduled to a node the `vault-init` container will run during the Pod initialization phase:

```
kubectl -n vault-controller get pods -l app=vault-example
```
```
NAME                  READY     STATUS     RESTARTS   AGE
vault-example-wif7l   0/1       Init:0/1   0          6s
```

View the logs for the `vault-init` container:

```
kubectl -n vault-controller logs \
  $(kubectl -n vault-controller \
    get pods -l app=vault-example \
    -o jsonpath='{.items[0].metadata.name}') \
  -c vault-init
```

Log output:

```
2016/10/28 19:08:28 Starting vault-init...
2016/10/28 19:08:28 Requesting a new wrapped token from http://vault-controller
2016/10/28 19:08:28 token request: Request error error missing or empty pod IP; retrying in 5s
2016/10/28 19:08:33 Requesting a new wrapped token from http://vault-controller
2016/10/28 19:08:33 Token request complete; waiting for callback...
2016/10/28 19:08:33 wrote /var/run/secrets/vaultproject.io/secret.json
2016/10/28 19:08:33 Successfully obtained and unwrapped the vault token, exiting...
```

At this point the init process has completed and the `vault-example` Pod should be running:

```
kubectl -n vault-controller get pods -l app=vault-example
```
```
NAME                  READY     STATUS    RESTARTS   AGE
vault-example-wif7l   1/1       Running   0          1m
```

View the logs for the `vault-example` container:

```
kubectl -n vault-controller logs \
  $(kubectl -n vault-controller get pods \
    -l app=vault-example \
    -o jsonpath='{.items[0].metadata.name}') \
  -c vault-example
```

Log output:

```
2016/10/28 19:08:35 Starting vault-example app...
2016/10/28 19:08:35 Reading vault secret file from /var/run/secrets/vaultproject.io/secret.json
==> WARNING: Don't ever write secrets to logs!

The secret is being printed here for demonstration purposes.
Use the secret details below with the vault cli
to get more info about the token.

{
  "request_id": "7c850f7e-bcad-ba8b-882f-927f868ec928",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": null,
  "warnings": null,
  "auth": {
    "client_token": "7f49f049-cabf-c1e3-906d-2817f68d403e",
    "accessor": "169b473d-e80e-5021-fc02-6e7209aa2235",
    "policies": [
      "default"
    ],
    "metadata": {
      "host_ip": "10.240.0.7",
      "namespace": "vault-controller",
      "pod_ip": "10.224.2.42",
      "pod_name": "vault-example-wif7l",
      "pod_uid": "e7a3df17-9d41-11e6-a0f3-42010a8a00ab"
    },
    "lease_duration": 86400,
    "renewable": true
  }
}

==> vault-example started! Log data will stream in below:
2016/10/28 19:08:35 Successfully renewed the client token; next renewal in 43200 seconds
```
