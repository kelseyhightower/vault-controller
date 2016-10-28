# Deployment Guide

This tutorial will walk you through deploying the following components:

* Vault 0.6.2
* Vault Controller 0.0.1

## Create the vault-controller namespace

Create a new Kubernetes namespace for this tutorial: 

```
kubectl create namespace vault-controller
```

## Deploy the Vault Server

The following command will create a Vault 0.6.2 server running in [dev mode](https://www.vaultproject.io/intro/getting-started/dev-server.html). This configuration should not be used in production or exposed to the outside world.

```
kubectl -n vault-controller create -f replicasets/vault.yaml
```

### Create the vault service

Expose the Vault server internally to the cluster:

```
kubectl -n vault-controller create -f services/vault.yaml
```

At this point the Vault server will be accessible to other Pods in the `vault-controller` namespace at the following address:

```
http://vault:8200
```

To streamline this tutorial the Vault root key has been set to:

```
3e4a5ba1-kube-422b-d1db-844979cab098
```

## Deploy the Vault Controller

### Create the vault-controller Secret

Store the Vault root key in a Kubernetes secret. This will allow the Vault Controller to reference the Vault root key at deployment time, authenticate with the Vault server, and generate tokens for Pods. Only the Vault Controller should have access to the root key.

```
kubectl -n vault-controller \
  create secret generic vault-controller \
  --from-literal "vault_token=3e4a5ba1-kube-422b-d1db-844979cab098"
```

### Deploy the Vault Controller:

```
kubectl -n vault-controller create -f replicasets/vault-controller.yaml 
```

### Create the vault-controller service

Expose the Vault Controller internally to the cluster:

```
kubectl -n vault-controller create -f services/vault-controller.yaml
```

At this point the Vault Controller will be accessible to other Pods in the `vault-controller` namespace at the following address:

```
http://vault-controller
```

## Next Steps

A Vault server and Vault Controller are now running in the `vault-controller` namespace. Now it's time to [deploy a Pod that can request tokens from the vault-controller](example-usage.md).
