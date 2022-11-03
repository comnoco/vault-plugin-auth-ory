#!/usr/bin/env bash

set -ex

GOOS=linux go build

docker kill vaultplg 2>/dev/null || true
tmpdir=$(mktemp -d vaultplgXXXXXX)
mkdir "$tmpdir/data"
docker run --rm -d -p8200:8200 --name vaultplg -v "$(pwd)/$tmpdir/data":/data -v $(pwd):/example --cap-add=IPC_LOCK -e 'VAULT_LOCAL_CONFIG=
{
  "backend": {"file": {"path": "/data"}},
  "listener": [{"tcp": {"address": "0.0.0.0:8200", "tls_disable": true}}],
  "plugin_directory": "/plugins",
  "log_level": "debug",
  "disable_mlock": true,
  "api_addr": "http://localhost:8200"
}
' vault server
sleep 1

export VAULT_ADDR=http://localhost:8200

initoutput=$(vault operator init -key-shares=1 -key-threshold=1 -format=json)
vault operator unseal $(echo "$initoutput" | jq -r .unseal_keys_hex[0])

export VAULT_TOKEN=$(echo "$initoutput" | jq -r .root_token)

vault write sys/plugins/catalog/auth/vault-plugin-auth-ory \
    sha_256=$(shasum -a 256 vault-plugin-auth-ory | cut -d' ' -f1) \
    command="vault-plugin-auth-ory"

vault auth enable \
    -path="plugins" \
    -plugin-name="vault-plugin-auth-ory" plugin

VAULT_TOKEN=  vault write auth/plugins/login password="super-secret-password"