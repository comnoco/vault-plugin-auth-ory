package main

import (
	"context"
	"fmt"
	"os"

	ory "github.com/comnoco/vault-plugin-auth-ory/auth"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

func main() {
	if err := run(); err != nil {
		fmt.Println(err)
		os.Exit(0)
	}
}

func run() error {
	secretName := "top_secret"
	mountPath := "ory"
	namespace := "secret"
	object := "e00de7c2-1ee2-4508-b95c-c3c0d881d0cc"
	relation := "viewer"
	cookie := "ory_kratos_session=MTY3NzA4MzI4M3xQRTV3RUp2MHdWblNSaWtQcDhKMjdtLXBqSDJ2eGFQTm9NbktFX0dlNWlJaUV4U1hFdmhscVM0ZEIyU3dzMHJROVUxTjk2NkdqM0xoR3dJeXNmNEpFUFhHMC1Rc0hJa0VldEdnazk0OFlfaVpqc0J2Rng3cDNOVHZ5a0xnbDVRWF9haHZjMjMxRk9iQmhJYUVZMUdrZFNSN29NQmVzOGVvWGpBaVNVVk9ReHl0aGtWSlFjZ1FtZldvand2cjVSbndDWEtqVENOVkJJMGVQMHRfYXBoTE0wTXk5SXVDM2UyOFo2TDEtdENxWTRkb01KV2NwLV83bG10WTVEaXhBa0FENy1UOUxrSjE4TU0tfGvmo31B3VG9JarStNbFB7TXOV8dhbvTq1mhZaEI8HEN"
	vaultAddr := "https://localhost:8200/"

	vaultCfg := api.DefaultConfig()
	vaultCfg.Address = vaultAddr
	vault, err := api.NewClient(vaultCfg)
	if err != nil {
		return errors.Wrap(err, "unable to create Vault API Client")
	}

	auth, err := ory.NewOryAuth(
		mountPath,
		namespace,
		object,
		relation,
		cookie,
	)
	if err != nil {
		return errors.Wrap(err, "failed to make ory auth method")
	}

	sec, err := vault.Auth().Login(context.Background(), auth)
	if err != nil {
		return errors.Wrap(err, "failed to auth with ory")
	}

	fmt.Println("Token:", sec.Auth.ClientToken)
	fmt.Println("Policies:", sec.Auth.Policies)

	secrets, err := listSecrets(vault, object)
	if err != nil {
		return errors.Wrap(err, "failed to list secrets")
	}

	for _, secret := range secrets.Data {
		fmt.Println("Secret metadata:", secret)
	}

	secret, err := readSecret(vault, object, secretName)
	if err != nil {
		return errors.Wrap(err, "failed to fetch secret")
	}

	fmt.Println("Secret:", secret)

	return nil
}

func listSecrets(vault *api.Client, path string) (*api.Secret, error) {
	apiCallPath := fmt.Sprintf("secret/metadata/%s", path)
	return vault.Logical().List(apiCallPath)
}

func readSecret(vault *api.Client, path, secretName string) (*api.Secret, error) {
	apiCallPath := fmt.Sprintf("secret/data/%s/%s", path, secretName)
	return vault.Logical().Read(apiCallPath)
}
