# Vault Controller

The Vault Controller automates the creation of Vault tokens for Kubernetes Pods.

## Status

This is a prototype. Do not use this in production.

## Use Case

* Each Pod requires a dedicated Vault token tied to the Pod's life-cycle
* Each Pod will use a dedicated Vault token to request secrets from a Vault server

## Documentation

* [How it Works](docs/how-it-works.md)

## Usage

The following tutorials will guide you through the deployment of the `vault-controller` and an example application to see how it all works.

### Prerequisites

Clone this repository:

```
git clone https://github.com/kelseyhightower/vault-controller.git
```

```
cd vault-controller
```

Before you can complete the tutorials you'll need access to a Kubernetes clusters. [Google Container Engine (GKE)](https://cloud.google.com/container-engine/) or [minikube](https://github.com/kubernetes/minikube) should work.

### Tutorials

* [Deployment Guide](docs/deployment-guide.md)
* [Example Usage](docs/example-usage.md)

## Cleanup

Once you are done with the tutorials run the following command to clean up:

```
kubectl delete namespace vault-controller
```
