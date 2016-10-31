# Microservice Tutorial

```
kubectl port-forward vault-xinsk 8200:8200
```

```
export VAULT_ADDR=http://127.0.0.1:8200
```

```
export VAULT_TOKEN=3e4a5ba1-kube-422b-d1db-844979cab098
```

```
vault mount pki
```

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

```
vault write pki/roles/client \
  allowed_domains="cluster.local" \
  allow_subdomains="true" \
  client_flag="true" \
  max_ttl="72h" \
  server_flag="false"
```

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

```
vault policy-write microservice policies/microservice.hcl
```
