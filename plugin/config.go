package plugin

import (
	"context"
	"fmt"
	"net/http"

	"github.com/hashicorp/vault/sdk/logical"
	keto "github.com/ory/keto-client-go/client"
	kratos "github.com/ory/kratos-client-go"
)

// Config is the configuration for the plugin.
type Config struct {
	Debug               bool              `json:"debug,omitempty"`
	KetoHost            string            `json:"keto_host"`
	KratosHost          string            `json:"kratos_host"`
	KratosScheme        string            `json:"kratos_scheme"`
	KratosDesc          string            `json:"kratos_desc`
	KratosDefaultHeader map[string]string `json:"kratos_default_header`
	KratosUserAgent     string            `json:"kratos_user_agent`
	// TODO (TW) expose Kratos and Keto configuration options.
	Kratos *KratosConfig `json:"kratos"          structs:"kratos" mapstructure:"kratos"`
	Keto   *KetoConfig   `json:"keto"            structs:"keto"   mapstructure:"keto"`
}

// ServerVariable stores the information about a server variable
type ServerVariable struct {
	Description  string   `json:"description,omitempty"   yaml:"description,omitempty"`
	DefaultValue string   `json:"default_value,omitempty" yaml:"default_value,omitempty"`
	EnumValues   []string `json:"enum_values,omitempty"   yaml:"enum_values,omitempty"`
}

// ServerConfiguration stores the information about a server
type ServerConfiguration struct {
	URL         string                    `json:"url"         structs:"url"         mapstructure:"url"`
	Description string                    `json:"description" structs:"description" mapstructure:"description"`
	Variables   map[string]ServerVariable `json:"variables"   structs:"variables"   mapstructure:"variables"`
}

// ServerConfigurations stores multiple ServerConfiguration items
type ServerConfigurations []ServerConfiguration

// KratosConfig stores the configuration of the Kratos API client
type KratosConfig struct {
	Host             string            `json:"host,omitempty"          structs:"host,omitempty"          mapstructure:"host,omitempty"`
	Scheme           string            `json:"scheme,omitempty"        structs:"scheme,omitempty"        mapstructure:"scheme,omitempty"`
	DefaultHeader    map[string]string `json:"defaultHeader,omitempty" structs:"defaultHeader,omitempty" mapstructure:"defaultHeader,omitempty"`
	UserAgent        string            `json:"userAgent,omitempty"     structs:"userAgent,omitempty"     mapstructure:"userAgent,omitempty"`
	Debug            bool              `json:"debug,omitempty"         structs:"debug,omitempty"         mapstructure:"debug,omitempty"`
	Servers          ServerConfigurations
	OperationServers map[string]ServerConfigurations
	HTTPClient       *http.Client
}

// KetoConfig stores the configuration of the Keto API client
type KetoConfig struct {
	TransportConfig *keto.TransportConfig `json:"transportConfig,omitempty" structs:"transportConfig,omitempty" mapstructure:"transportConfig,omitempty"`
}

// TransportConfig contains the transport related info,
// found in the meta section of the spec file.
type TransportConfig struct {
	Host     string   `json:"host,omitempty"     structs:"host,omitempty"     mapstructure:"host,omitempty"`
	BasePath string   `json:"basePath,omitempty" structs:"basePath,omitempty" mapstructure:"basePath,omitempty"`
	Schemes  []string `json:"schemes,omitempty"  structs:"schemes,omitempty"  mapstructure:"schemes,omitempty"`
}

// readConfig reads the configuration from the storage.
func (b *OryAuthBackend) readConfig(ctx context.Context, s logical.Storage) (*Config, error) {
	b.Logger().Debug("reading configuration")

	entry, err := s.Get(ctx, "config")
	if err != nil {
		b.Logger().Debug("error getting config from storage", "error", err)
		return nil, err
	}

	if entry == nil {
		b.Logger().Debug("entry was nil")

		return nil, nil
	}

	b.Logger().Debug("got entry")

	b.Logger().Debug("decoding entry")

	config := &Config{}
	err = entry.DecodeJSON(config)
	if err != nil {
		return nil, err
	}

	b.Logger().Debug("successfully decoded entry")

	return config, nil
}

// configToKratosConfig converts the plugin configuration to the Kratos API client configuration.
func (b *OryAuthBackend) configToKratosConfig(config *Config) *kratos.Configuration {
	if config == nil {
		b.Logger().Error("Config nil")
		return nil
	}

	// if config.Kratos == nil {
	// 	b.Logger().Error("Kratos configuration missing in config")
	// 	return nil
	// }

	kratosConfig := &kratos.Configuration{
		Host:          config.KratosHost,
		Scheme:        config.KratosScheme,
		DefaultHeader: config.KratosDefaultHeader,
		UserAgent:     config.KratosUserAgent,
		Debug:         config.Debug,
		Servers: kratos.ServerConfigurations{
			kratos.ServerConfiguration{
				URL:         fmt.Sprintf("%s://%s", config.KratosScheme, config.KratosHost),
				Description: config.KratosDesc,
				Variables:   map[string]kratos.ServerVariable{},
			},
		},
		OperationServers: make(map[string]kratos.ServerConfigurations, 0),
		HTTPClient:       &http.Client{},
	}

	// for _, server := range config.Kratos.Servers {
	// 	variables := make(map[string]kratos.ServerVariable)

	// 	for _, variable := range server.Variables {
	// 		variables[variable.DefaultValue] = kratos.ServerVariable{
	// 			EnumValues:   variable.EnumValues,
	// 			DefaultValue: variable.DefaultValue,
	// 			Description:  variable.Description,
	// 		}
	// 	}

	// 	kratosConfig.Servers = append(kratosConfig.Servers, kratos.ServerConfiguration{
	// 		URL:         server.URL,
	// 		Description: server.Description,
	// 		Variables:   variables,
	// 	})
	// }

	return kratosConfig
}

// configToKetoConfig converts the plugin configuration to the Keto API client configuration.
func (b *OryAuthBackend) configToKetoConfig(config *Config) *keto.TransportConfig {
	if config == nil {
		b.Logger().Error("Config nil")
		return nil
	}

	if config.Keto == nil {
		b.Logger().Error("Keto configuration missing in config")
		return nil
	}

	if config.Keto.TransportConfig == nil {
		return nil
	}

	ketoConfig := &keto.TransportConfig{
		Host:     config.Keto.TransportConfig.Host,
		BasePath: config.Keto.TransportConfig.BasePath,
		Schemes:  config.Keto.TransportConfig.Schemes,
	}

	return ketoConfig
}
