# Ory Kratos + Keto Auth Plugin for HashiCorp Vault

This repository contains code for a [HashiCorp Vault](https://github.com/hashicorp/vault) Auth [Plugin](https://developer.hashicorp.com/vault/docs/plugins) that authenticates with [Ory Kratos](https://github.com/ory/kratos) and [Ory Keto](https://github.com/ory/keto) APIs.

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

## Development Setup

1. Build the plugin for your platform:

  ```sh
  $ make darwin/arm64
  ```

  or build for all platforms:

  ```sh
  $ make build
  ```

1. Start a Vault server in dev mode pointing to the plugin directory:

  ```sh
  $ make start
  ```

1. Enable the plugin in Vault:

  ```sh
  $ make enable
  ```

1. Authenticate with the plugin:

  ```sh
  $ vault write auth/ory/login \
namespace=workspace \
object=c5cc3e28-e3c3-45ca-be86-a0a55953bfca \
relation=editor \
kratos_session_cookie=kratos_session_cookie=MTY2NzgyMjg2M3xBYVJxa2hmNFlOOFAyZnc3U3VidnZKd1A0VmdyWFgyU3ozbUNvRG4zeC1oNU1DS3Z6dkc1ODllTHdua0s5aFdpcW1ZZ0pveVNBVVM3ZXBIRWdQdlJGWXN0aS1iVU5tenVFbUw1WE1QNDRVcms5eWZZRk52R3dOdTJKLVcxYVlFWFU4ajNFUmc0bnc9PXyq29KzMQjNDdZLeJAuNLUBeU1g1-iD7l31nahltn4mZg==
  ```

## Authenticating with Ory Kratos and Keto

To authenticate, the user supplies a valid Ory Kratos session cookie, along with the namespace,
object, and relation to check against Keto.

```sh
$ vault write auth/ory/login namespace=[namespace] object=[object] relation=[relation] kratos_session_cookie=[full kratos_session_cookie=[...] string]
```

The response will be a standard auth response with some token metadata:

```text
Key                     Value
--------------------------------
token                   [token]
token_accessor          [accessor]
token_duration          [TTL]
token_renewable         false
token_policies          ["default" "[namespace]_[object]"]
identity_policies       []
policies                ["default" "[namespace]_[object]"]
```

## License

This code is licensed under the MPLv2 license.
