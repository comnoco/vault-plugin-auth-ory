package plugin

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/logical"
	keto "github.com/ory/keto-client-go/client"
	kratos "github.com/ory/kratos-client-go"
	"github.com/pkg/errors"
)

// Config is the configuration for the plugin.
type Config struct {
	// plugin
	UseSessionExpiryTTL bool `json:"use_session_expiry_ttl,omitempty"`
	TTLSeconds          int  `json:"ttl_seconds,omitempty"`
	MaxTTLSeconds       int  `json:"max_ttl_seconds,omitempty"`

	// Keto encapsulates the keto config (not currently supported)
	KetoHost string `json:"keto_host,omitempty"`
	// TODO implement full keto config
	// Keto     *KetoConfig `json:"keto,omitempty"`

	// Kratos encapsulates the kratos config (not currently supported)
	KratosURL           string            `json:"kratos_url,omitempty"`
	KratosDescription   string            `json:"kratos_description,omitempty"`
	KratosUserAgent     string            `json:"kratos_user_agent,omitempty"`
	KratosDefaultHeader map[string]string `json:"kratos_default_header,omitempty"`
	KratosDebug         bool              `json:"kratos_debug,omitempty"`
	// TODO implement full kratos config
	// Kratos              *KratosConfig     `json:"kratos,omitempty"`
}

// ServerVariable stores the information about a server variable.
type ServerVariable struct {
	Description  string   `json:"description,omitempty"`
	DefaultValue string   `json:"default_value,omitempty"`
	EnumValues   []string `json:"enum_values,omitempty"`
}

// ServerConfiguration stores the information about a server.
type ServerConfiguration struct {
	URL         string                    `json:"url,omitempty"`
	Description string                    `json:"description.omitempty"`
	Variables   map[string]ServerVariable `json:"variables.omitempty"`
}

// ServerConfigurations stores multiple ServerConfiguration items.
type ServerConfigurations []ServerConfiguration

// KratosConfig stores the configuration of the Kratos API client.
type KratosConfig struct {
	Host             string                          `json:"host,omitempty"`
	Scheme           string                          `json:"scheme,omitempty"`
	DefaultHeader    map[string]string               `json:"default_header,omitempty"`
	UserAgent        string                          `json:"user_agent,omitempty"`
	Debug            bool                            `json:"debug,omitempty"`
	Servers          ServerConfigurations            `json:"servers,omitempty"`
	OperationServers map[string]ServerConfigurations `json:"operation_servers,omitempty"`
}

// KetoConfig stores the configuration of the Keto API client.
type KetoConfig struct {
	TransportConfig *keto.TransportConfig `json:"transport_config,omitempty"`
}

// TransportConfig contains the transport related info.
type TransportConfig struct {
	Host     string   `json:"host,omitempty"`
	BasePath string   `json:"base_path,omitempty"`
	Schemes  []string `json:"schemes,omitempty"`
}

// readConfig reads the configuration from the storage.
func (b *OryAuthBackend) readConfig(ctx context.Context, s logical.Storage) (*Config, error) {
	b.Logger().Debug("reading config")

	entry, err := s.Get(ctx, "config")
	if err != nil {
		return nil, errors.Wrap(err, "error getting config from storage")
	}

	if entry == nil {
		b.Logger().Debug("config entry was nil")
		return nil, nil
	}

	config := &Config{}
	err = entry.DecodeJSON(config)
	if err != nil {
		return nil, errors.Wrap(err, "error decoding config JSON")
	}

	b.Logger().Debug("successfully read config")

	return config, nil
}

// setConfig stores the config in the storage.
func (b *OryAuthBackend) setConfig(ctx context.Context, s logical.Storage, config *Config) error {
	b.Logger().Debug("setting config")

	if config == nil {
		return errors.New("config is not found")
	}

	entry, err := logical.StorageEntryJSON("config", config)
	if err != nil {
		return errors.Wrap(err, "could not create JSON storage entry")
	}

	if err := s.Put(ctx, entry); err != nil {
		return errors.Wrap(err, "could not store config in storage")
	}

	b.Logger().Debug("successfully set configuration")

	return nil
}

// configToKratosConfig converts the plugin configuration to the Kratos API client configuration.
func (b *OryAuthBackend) configToKratosConfig(config *Config) *kratos.Configuration {
	b.Logger().Debug("converting to kratos config")

	if config == nil {
		b.Logger().Error("nil config passed to kratos config transformer, using default")
		return kratos.NewConfiguration()
	}

	kratosConfig := kratos.NewConfiguration()

	kratosConfig.Debug = config.KratosDebug
	kratosConfig.DefaultHeader = config.KratosDefaultHeader

	if config.KratosUserAgent != "" {
		kratosConfig.UserAgent = config.KratosUserAgent
	}

	kratosConfig.Servers = kratos.ServerConfigurations{
		kratos.ServerConfiguration{
			URL:         config.KratosURL,
			Description: config.KratosDescription,
			Variables:   make(map[string]kratos.ServerVariable),
		},
	}

	kratosConfig.HTTPClient = &http.Client{}

	return kratosConfig
}
