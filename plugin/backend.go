package plugin

import (
	"context"
	"sync"

	"github.com/comnoco/vault-plugin-auth-ory/version"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"
	keto "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
	kratos "github.com/ory/kratos-client-go"
	"google.golang.org/grpc"
)

const (
	help = `
  The Ory auth plugin allows authentication against Ory services.
  `
)

// Factory returns a new instance of the Ory auth backend.
func Factory(ctx context.Context, conf *logical.BackendConfig) (logical.Backend, error) {
	b := NewBackend()
	err := b.Setup(ctx, conf)
	if err != nil {
		return nil, err
	}

	return b, nil
}

// OryAuthBackend is the backend plugin backed by Ory services.
type OryAuthBackend struct {
	*framework.Backend

	kratosClient      *kratos.APIClient
	kratosClientMutex sync.RWMutex

	ketoClient      *KetoClient
	ketoClientMutex sync.RWMutex
}

// KetoClient is a client for the Ory Keto API.
type KetoClient struct {
	// conn is the gRPC connection to the Keto API.
	conn *grpc.ClientConn

	// CheckServiceClient is the client for the Keto Check API.
	CheckServiceClient keto.CheckServiceClient
}

// NewBackend returns a new instance of the Ory-backed auth backend.
func NewBackend() *OryAuthBackend {
	b := &OryAuthBackend{}

	b.Backend = &framework.Backend{
		RunningVersion: version.RunningVersion,
		BackendType:    logical.TypeCredential,
		Invalidate:     b.invalidateHandler,
		PeriodicFunc:   b.periodicHandler,
		// AuthRenew:    b.authRenewHandler,
		Help: help,
		PathsSpecial: &logical.Paths{
			Unauthenticated: []string{"login"},
			SealWrapStorage: []string{"config"},
		},
		Paths: framework.PathAppend(
			NewPathConfig(b),
			NewPathLogin(b),
		),
	}

	b.Logger().Debug("created backend")

	return b
}

// Close closes the backend.
func (b *OryAuthBackend) Close() {
	b.Logger().Debug("closing backend")

	b.closeKratosClient()
	b.closeKetoClient()

	b.Logger().Debug("closed backend")
}

// invalidateHandler is called when the backend is invalidated.
func (b *OryAuthBackend) invalidateHandler(_ context.Context, key string) {
	b.Logger().Debug("invalidating backend", "key", key)

	switch key {
	case "config":
		b.Close()
	}
}

// periodicHandler is called periodically to perform any backend tasks.
func (b *OryAuthBackend) periodicHandler(ctx context.Context, req *logical.Request) error {
	// TODO implement periodic handler (not necessarily a health check like below)

	// b.Logger().Debug("running periodic healthCheck")

	// err := b.checkKratosHealth(ctx, req.Storage)
	// if err != nil {
	// 	return err
	// }

	// err = b.checkKetoHealth(ctx, req.Storage)
	// if err != nil {
	// 	return err
	// }

	// b.Logger().Debug("periodic health checks passed")

	return nil
}
