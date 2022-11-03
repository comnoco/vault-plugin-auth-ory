# Ory Kratos + Keto Auth Method Plugin for HashiCorp Vault

This repository contains code for a HashiCorp Vault Auth Plugin that authenticates with Ory Kratos and Keto APIs.

## Setup

The setup guide assumes some familiarity with Vault and Vault's plugin
ecosystem. You must have a Vault server already running, unsealed, and
authenticated.

1. Download and decompress the latest plugin binary from the Releases tab on
GitHub. Alternatively you can compile the plugin from source.

1. Move the compiled plugin into Vault's configured `plugin_directory`:

  ```sh
  $ mv vault-auth-plugin-ory /etc/vault/plugins/vault-auth-plugin-ory
  ```

1. Calculate the SHA256 of the plugin and register it in Vault's plugin catalog.
If you are downloading the pre-compiled binary, it is highly recommended that
you use the published checksums to verify integrity.

  ```sh
  $ export SHA256=$(shasum -a 256 "/etc/vault/plugins/vault-auth-plugin-ory" | cut -d' ' -f1)

  $ vault plugin register \
      -sha256="${SHA256}" \
      -command="vault-auth-plugin-ory" \
      auth vault-plugin-auth-ory
  ```

1. Mount the auth method:

  ```sh
  $ vault auth enable \
      -path="ory" \
      -plugin-name="vault-plugin-auth-ory" plugin
  ```

## Authenticating with Ory Kratos and Keto

To authenticate, the user supplies a valid Ory Kratos session cookie, along with the namespace,
object, and relation to check against Keto.

```sh
$ vault write auth/ory/login namespace=[namespace] object=[object] relation=[relation] kratos_session_cookie=kratos_session_cookie=[...]
```

The response will be a standard auth response with some token metadata:

```text
Key                     Value
--------------------------------
token                   [token]
token_accessor          [accessor]
token_duration          1h
token_renewable         true
token_policies          ["default" "secret/[namespace]/[object]"]
identity_policies       []
policies                ["default" "secret/[namespace]/[object]"]
token_meta_namespace    [namespace]
token_meta_object       [object]
token_meta_relation     [relation]
token_meta_subject      [kratos user ID]
```

## License

This code is licensed under the MPLv2 license.
