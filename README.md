# Vault Controller

The Vault Controller automates the creation of Vault tokens for Pods.

## Status

This is a prototype. Do not use this in production.

## Use Case

* Each Pod requires a dedicated Vault token tied its life-cycle
* Each Pod will use a dedicated Vault token to request secrets from a Vault server

## Documentation

* [How it Works](docs/how-it-works.md)

## Usage

* [Deployment Guide](docs/deployment-guide.md)
* [Example Usage](docs/example-usage.md)
