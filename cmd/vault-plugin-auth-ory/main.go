package main

import (
	"os"

	ory "github.com/comnoco/vault-plugin-auth-ory/plugin"

	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/sdk/plugin"
)

func main() {
	logger := hclog.New(&hclog.LoggerOptions{})
	logger.Info("starting vault-plugin-auth-ory")

	apiClientMeta := &api.PluginAPIClientMeta{}
	flags := apiClientMeta.FlagSet()
	flags.Parse(os.Args[1:])

	tlsConfig := apiClientMeta.GetTLSConfig()
	tlsProviderFunc := api.VaultPluginTLSProvider(tlsConfig)

	if err := plugin.Serve(&plugin.ServeOpts{
		BackendFactoryFunc: ory.Factory,
		TLSProviderFunc:    tlsProviderFunc,
	}); err != nil {
		logger.Error("plugin shutting down", "error", err)
		os.Exit(1)
	}
}
