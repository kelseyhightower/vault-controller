# Allow a token to generate TLS certificates from the PKI secret backend
# for the client role.
path "pki/issue/client" {
  capabilities = ["create", "read", "update", "delete", "list"]
}

# Allow a token to generate TLS certificates from the PKI secret backend
# for the server role.
path "pki/issue/server" {
  capabilities = ["create", "read", "update", "delete", "list"]
}
