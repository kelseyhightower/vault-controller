# Vault Controller: How It Works


## Requesting a Wrapped Token

The Vault Controller creates [wrapped tokens](https://www.hashicorp.com/blog/vault-0.6.html#response-wrapping) for each Pod based
on a simple callback flow. Pods request a wrapped token via an HTTP request to a Vault Controller running in the Kubernetes cluster:

```
http://vault-controller/token?name=vault-example-bx1r8&namespace=default
```

The Pod MUST supply the Pod name and namespace when requesting a wrapped token.

### Verifying the Pod

Since we cannot blindly trust the caller the Vault Controller will look up the Pod details via the Kubernetes API using the Pod name and namespace from the token request:

```
http://127.0.0.1:8001/api/v1/namespaces/default/pods/vault-example-bx1r8
```

After retrieving the Pod details the Vault Controller will extract the following Pod annotations:

```
vaultproject.io/policies
vaultproject.io/ttl
```

These annotations are trusted. If a Kubernetes object is created with those annotations we assume the request to create the object was authenticated and represents the desired state of the administrator.

The `vaultproject.io/policies` annotation is required and must hold a comma separated list of Vault policies to attach to the token:

```
vaultproject.io/policies: "default,web"
```

The `vaultproject.io/ttl` annotation is optional and holds the TTL attached to the token; defaults to 72 hours.

```
vaultproject.io/ttl: "72h"
```

All tokens are generated with the following token configuration:

```
&api.TokenCreateRequest{
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
```

### Pushing the wrapped token to the Pod

Once the wrapped token is created the Vault Controller pushes the token to the Pod using the Pod IP extracted from the Pod details obtained earlier:

```
http://10.224.2.33:8080
```

Request Body:
 
```
{
  "token":"664efd9a-da96-29d2-4d8c-1ea5f6218af6",
  "ttl":120,
  "creation_time":"2016-10-28T05:36:56.772759816Z",
  "wrapped_accessor":"4a85b34c-4baa-0360-bac6-bebf17dedce4"
}
```

If the Pod is able to successfully unwrap the token it MUST respond HTTP 200. Future attempts to push a wrapped token to the Pod MUST fail with an HTTP 409 Conflict if the existing token is still valid.

### Renewing the Token

After the token has been unwrapped it's the responsibility of the Pod to renew the token against a Vault server. No future calls to the Vault Controller are required.
