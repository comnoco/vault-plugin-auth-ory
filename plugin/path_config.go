package plugin

import (
	"context"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
)

const (
	// configSynopsis is used to provide a short summary of the config path.
	configSynopsis = `Configures the Ory services to use for authentication.`

	// configDescription is used to provide a detailed description of the config path.
	configDescription = `This endpoint configures the details for accessing Ory APIs.`
)

var configFields map[string]*framework.FieldSchema = map[string]*framework.FieldSchema{
	// TODO (TW) Add fields for configuring the Ory services.
	// "kratos_auth_url": {
	// 	Type:        framework.TypeString,
	// 	Description: "Kratos authentication URL",
	// },
}

// NewPathConfig creates a new path for configuring the backend.
func NewPathConfig(b *OryAuthBackend) []*framework.Path {
	return []*framework.Path{
		&framework.Path{
			Pattern: "config",
			Fields:  configFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.updateConfigHandler,
				logical.ReadOperation:   b.readConfigHandler,
				logical.UpdateOperation: b.updateConfigHandler,
			},
			HelpSynopsis:    configSynopsis,
			HelpDescription: configDescription,
		},
	}
}

// readConfigHandler reads the configuration from the storage.
func (b *OryAuthBackend) readConfigHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	// TODO (TW) https://developer.hashicorp.com/vault/docs/concepts/integrated-storage
	config, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, nil
	}

	res := &logical.Response{
		Data: map[string]interface{}{
			"kratos_auth_url": config.KratosAuthURL,
		},
	}

	return res, nil
}

// updateConfigHandler updates the configuration in the storage.
func (b *OryAuthBackend) updateConfigHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	var (
		val interface{}
		ok  bool
	)

	config, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = &Config{}
	}

	val, ok = data.GetOk("kratos_url")
	if ok {
		config.KratosAuthURL = val.(string)
	}

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return nil, err
	}

	err = req.Storage.Put(ctx, entry)
	if err != nil {
		return nil, err
	}

	b.Close()

	return nil, nil
}
