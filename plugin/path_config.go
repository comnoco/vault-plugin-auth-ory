package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	"github.com/pkg/errors"
)

const (
	// configSynopsis is used to provide a short summary of the config path.
	configSynopsis = `Configures the Ory services to use for authentication.`

	// configDescription is used to provide a detailed description of the config path.
	configDescription = `This endpoint configures the details for accessing Ory APIs.`
)

var configFields map[string]*framework.FieldSchema = map[string]*framework.FieldSchema{
	// plugin
	"ttl_seconds": {
		Type:        framework.TypeDurationSecond,
		Description: "The TTL of the authorised key in seconds (if `use_session_expiry_ttl` is false)",
		Required:    false,
		Default:     3600,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "TTL Seconds",
			Sensitive: false,
		},
	},
	"max_ttl_seconds": {
		Type:        framework.TypeDurationSecond,
		Description: "The maximum TTL of the authorised key in seconds",
		Required:    false,
		Default:     3600,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Max TTL Seconds",
			Sensitive: false,
		},
	},
	"use_session_expiry_ttl": {
		Type:        framework.TypeBool,
		Description: "Uses the Kratos session expiry as the TTL of the authorised key",
		Required:    false,
		Default:     false,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Use Session Expiry TTL",
			Sensitive: false,
		},
	},

	// keto
	"keto_host": {
		Type:        framework.TypeString,
		Description: "The host of the Keto instance",
		Required:    true,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Keto host",
			Sensitive: false,
		},
	},

	// kratos
	"kratos_url": {
		Type:        framework.TypeString,
		Description: "The URL of the Kratos instance",
		Required:    true,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Kratos URL",
			Sensitive: false,
		},
	},
	"kratos_description": {
		Type:        framework.TypeString,
		Description: "The description of the Kratos instance",
		Required:    true,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Kratos Description",
			Sensitive: false,
		},
	},
	"kratos_user_agent": {
		Type:        framework.TypeString,
		Description: "User Agent used to make requests to Kratos",
		Required:    false,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Kratos User Agent",
			Sensitive: false,
		},
	},
	"kratos_default_header": {
		Type:        framework.TypeKVPairs,
		Description: "Headers to be sent with every Kratos request",
		Required:    false,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Kratos Default Header",
			Sensitive: false,
		},
	},
	"kratos_debug": {
		Type:        framework.TypeBool,
		Description: "Whether or not Kratos is in debug mode",
		Required:    true,
		DisplayAttrs: &framework.DisplayAttributes{
			Name:      "Kratos Debug",
			Sensitive: false,
		},
	},
}

// NewPathConfig creates a new path for configuring the backend.
func NewPathConfig(b *OryAuthBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "config",
			Fields:  configFields,
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.CreateOperation: b.createConfigHandler,
				logical.ReadOperation:   b.readConfigHandler,
				logical.UpdateOperation: b.updateConfigHandler,
				logical.DeleteOperation: b.deleteConfigHandler,
			},
			HelpSynopsis:    configSynopsis,
			HelpDescription: configDescription,
		},
	}
}

// createConfigHandler creates the configuration in the storage.
func (b *OryAuthBackend) createConfigHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	config := &Config{}

	err := b.decodeFieldData(config, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode field data during create")
	}

	err = b.setConfig(ctx, req.Storage, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create config")
	}

	b.closeKratosClient()
	b.closeKetoClient()

	return nil, nil
}

// readConfigHandler reads the configuration from the storage.
func (b *OryAuthBackend) readConfigHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	config, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		return nil, errors.New("config was nil")
	}

	jsonData, err := json.Marshal(config)
	if err != nil {
		return nil, errors.Wrap(err, "could not marshal config to JSON")
	}

	var response map[string]interface{}
	err = json.Unmarshal(jsonData, &response)
	if err != nil {
		return nil, errors.Wrap(err, "could not unmarshal JSON")
	}

	return &logical.Response{
		Data: response,
	}, nil
}

// updateConfigHandler updates the configuration in the storage.
func (b *OryAuthBackend) updateConfigHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	config, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, err
	}

	if config == nil {
		config = &Config{}
	}

	err = b.decodeFieldData(config, data)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode field data during update")
	}

	err = b.setConfig(ctx, req.Storage, config)
	if err != nil {
		return nil, errors.Wrap(err, "failed to update config")
	}

	b.closeKratosClient()
	b.closeKetoClient()

	return nil, nil
}

// deleteConfigHandler deletes the configuration in the storage.
func (b *OryAuthBackend) deleteConfigHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	return nil, req.Storage.Delete(ctx, "config")
}

// decodeFieldData decodes the incoming config field data and sets the values in the config struct
func (b *OryAuthBackend) decodeFieldData(config *Config, data *framework.FieldData) error {
	if config == nil {
		return errors.New("nil config used to decode field data")
	}
	if data == nil {
		return errors.New("nil data used to decode field data")
	}

	// plugin configs
	if val, ok := data.GetOk("use_session_expiry_ttl"); ok {
		b.Logger().Debug("got config value", "use_session_expiry_ttl", val)

		config.UseSessionExpiryTTL = val.(bool)
		if !ok {
			b.Logger().Error(fmt.Sprintf("use_session_expiry_ttl was a %T, expected a bool", val))
		}
	}

	if val, ok := data.GetOk("ttl_seconds"); ok {
		b.Logger().Debug("got config value", "ttl_seconds", val)

		config.TTLSeconds, ok = val.(int)
		if !ok {
			b.Logger().Error(fmt.Sprintf("ttl_seconds was a %T, expected int", val))
		}
	}

	if val, ok := data.GetOk("max_ttl_seconds"); ok {
		b.Logger().Debug("got config value", "max_ttl_seconds", val)

		config.MaxTTLSeconds, ok = val.(int)
		if !ok {
			b.Logger().Error(fmt.Sprintf("max_ttl_seconds was a %T, expected int", val))
		}
	}

	// keto configs
	if val, ok := data.GetOk("keto_host"); ok {
		b.Logger().Debug("got config value", "keto_host", val)
		config.KetoHost, ok = val.(string)
		if !ok {
			b.Logger().Error(fmt.Sprintf("keto_host was a %T, expected a string", val))
		}
	}

	// kratos configs
	if val, ok := data.GetOk("kratos_url"); ok {
		b.Logger().Debug("got config value", "kratos_url", val)

		config.KratosURL, ok = val.(string)
		if !ok {
			b.Logger().Error(fmt.Sprintf("kratos_url was a %T, expected a string", val))
		}
	}

	if val, ok := data.GetOk("kratos_description"); ok {
		b.Logger().Debug("got config value", "kratos_description", val)

		config.KratosDescription, ok = val.(string)
		if !ok {
			b.Logger().Error(fmt.Sprintf("kratos_description was a %T, expected a string", val))
		}
	}

	if val, ok := data.GetOk("kratos_user_agent"); ok {
		b.Logger().Debug("got config value", "kratos_user_agent", val)

		config.KratosUserAgent, ok = val.(string)
		if !ok {
			b.Logger().Error(fmt.Sprintf("kratos_user_agent was a %T, expected a string", val))
		}
	}

	if val, ok := data.GetOk("kratos_default_header"); ok {
		b.Logger().Debug("got config value", "kratos_default_header", val)

		config.KratosDefaultHeader, ok = val.(map[string]string)
		if !ok {
			b.Logger().
				Error(fmt.Sprintf("kratos_default_header was a %T, expected a map[string]string", val))
		}
	}

	if val, ok := data.GetOk("kratos_debug"); ok {
		b.Logger().Debug("got config value", "kratos_debug", val)

		config.KratosDebug, ok = val.(bool)
		if !ok {
			b.Logger().Error(fmt.Sprintf("kratos_debug was a %T, expected a bool", val))
		}
	}

	return nil
}
