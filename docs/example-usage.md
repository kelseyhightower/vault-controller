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
vault-example-f0cip   0/1       Init:0/1   0          1s
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
2016/10/28 18:27:34 Starting vault-init...
2016/10/28 18:27:34 Requesting a new wrapped token from http://vault-controller:80
2016/10/28 18:27:34 token request: Request error error missing or empty pod IP; retrying in 5s
2016/10/28 18:27:39 Requesting a new wrapped token from http://vault-controller:80
2016/10/28 18:27:39 Token request complete; waiting for callback...
2016/10/28 18:27:39 Successfully obtained and unwrapped the vault token, exiting...
```

At this point the init process has completed and the `vault-example` Pod should be running:

```
kubectl -n vault-controller get pods -l app=vault-example
```
```
NAME                  READY     STATUS    RESTARTS   AGE
vault-example-f0cip   1/1       Running   0          2m
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
2016/10/28 18:27:41 Starting vault-example app...
2016/10/28 18:27:41 Reading vault secret file from /var/run/secrets/vaultproject.io/secret.json
==> WARNING: Don't ever write secrets to logs!

The secret is being printed here for demonstration purposes.
Use the secret details below with the vault cli
to get more info about the token.

{
  "request_id": "e7326c2a-c9be-bdfc-aa10-17f7cd56d296",
  "lease_id": "",
  "lease_duration": 0,
  "renewable": false,
  "data": null,
  "warnings": null,
  "auth": {
    "client_token": "b81ffe89-264c-8538-f6f7-fd7e4d67cb17",
    "accessor": "abacdce0-c2db-d0c3-7e0c-b43208f45157",
    "policies": [
      "default"
    ],
    "metadata": {
      "host_ip": "10.240.0.7",
      "namespace": "vault-controller",
      "pod_ip": "10.224.2.39",
      "pod_name": "vault-example-f0cip",
      "pod_uid": "312e3c8a-9d3c-11e6-a0f3-42010a8a00ab"
    },
    "lease_duration": 86400,
    "renewable": true
  }
}

==> vault-example started! Log data will stream in below:
2016/10/28 18:27:41 Successfully renewed the client token; next renewal in 43200 seconds
```
