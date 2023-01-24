package plugin

import (
	"context"

	keto "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"

	"github.com/hashicorp/vault/sdk/logical"

	"github.com/pkg/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/connectivity"
	"google.golang.org/grpc/credentials/insecure"
)

// getKetoClient returns a client for the Ory Keto API.
func (b *OryAuthBackend) getKetoClient(
	ctx context.Context,
	s logical.Storage,
) (*KetoClient, error) {
	b.Logger().Debug("getting keto client")

	b.ketoClientMutex.RLock()
	defer b.ketoClientMutex.RUnlock()

	if b.ketoClient != nil {
		b.Logger().Debug("returning existing keto client")

		return b.ketoClient, nil
	}

	b.Logger().Debug("could not find existing keto client, creating new one")

	config, err := b.readConfig(ctx, s)
	if err != nil {
		return nil, errors.Wrap(err, "could not read keto config")
	}

	b.Logger().Debug("creating keto client", "host", config.KetoHost)

	conn, err := grpc.Dial(
		config.KetoHost,
		grpc.WithTransportCredentials(insecure.NewCredentials()), // TODO support tls
	)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to keto")
	}

	b.ketoClient = &KetoClient{
		conn:               conn,
		CheckServiceClient: keto.NewCheckServiceClient(conn),
	}

	b.Logger().Debug("returning new keto client")

	return b.ketoClient, nil
}

// closeKetoClient closes the client to the Ory Keto API.
func (b *OryAuthBackend) closeKetoClient() {
	b.ketoClientMutex.Lock()
	defer b.ketoClientMutex.Unlock()

	if b.ketoClient == nil {
		return
	}

	if b.ketoClient.conn != nil {
		b.ketoClient.conn.Close()
	}

	if b.ketoClient.CheckServiceClient != nil {
		b.ketoClient.CheckServiceClient = nil
	}

	b.ketoClient = nil
}

// checkKetoHealth checks the health of the Ory Keto API.
func (b *OryAuthBackend) checkKetoHealth(ctx context.Context, s logical.Storage) error {
	b.Logger().Debug("checking keto health")

	ketoClient, err := b.getKetoClient(ctx, s)
	if err != nil {
		return errors.Wrap(err, "failed to get keto client during health check")
	}

	connState := ketoClient.conn.GetState()
	if connState != connectivity.Ready && connState != connectivity.Idle {
		return errors.Errorf("keto health check failed: %v", connState)
	}

	b.Logger().Debug("keto health check passed")

	return nil
}
