package plugin

import (
	"context"
	"net/http"

	"github.com/hashicorp/vault/sdk/logical"
	kratos "github.com/ory/kratos-client-go"
	"github.com/pkg/errors"
)

// getKratosClient returns a client for the Ory Kratos API.
func (b *OryAuthBackend) getKratosClient(
	ctx context.Context,
	s logical.Storage,
) (*kratos.APIClient, error) {
	b.Logger().Debug("getting kratos client")

	b.kratosClientMutex.RLock()
	defer b.kratosClientMutex.RUnlock()

	if b.kratosClient != nil {
		b.Logger().Debug("returning existing kratos client")

		return b.kratosClient, nil
	}

	b.Logger().Debug("could not find existing kratos client, creating new one")

	config, err := b.readConfig(ctx, s)
	if err != nil {
		return nil, errors.Wrap(err, "failed to read config")
	}

	kratosConfig := b.configToKratosConfig(config)

	b.Logger().Debug("creating kratos client")
	b.kratosClient = kratos.NewAPIClient(kratosConfig)

	b.Logger().Debug("returning new kratos client", "url", kratosConfig.Servers[0].URL)

	return b.kratosClient, nil
}

// closeKratosClient closes the client for the Ory Kratos API.
func (b *OryAuthBackend) closeKratosClient() {
	b.Logger().Debug("closing kratos client")

	b.kratosClientMutex.Lock()
	defer b.kratosClientMutex.Unlock()

	if b.kratosClient == nil {
		return
	}

	b.kratosClient = nil

	b.Logger().Debug("closed kratos client")
}

// checkKratosHealth checks the health of the Ory Kratos API.
func (b *OryAuthBackend) checkKratosHealth(ctx context.Context, s logical.Storage) error {
	b.Logger().Debug("checking kratos health")

	kratosClient, err := b.getKratosClient(ctx, s)
	if err != nil {
		return errors.Wrap(err, "failed to get kratos client during health check")
	}

	_, res, err := kratosClient.MetadataApi.IsAliveExecute(
		kratos.MetadataApiApiIsAliveRequest{},
	)
	if err != nil {
		return errors.Wrap(err, "kratos health check failed")
	}
	if res.StatusCode != http.StatusOK {
		return errors.Errorf("kratos health check failed: %v", res.StatusCode)
	}

	b.Logger().Debug("kratos health check passed")

	return nil
}
