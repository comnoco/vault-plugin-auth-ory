package auth

import (
	"context"
	"fmt"

	"github.com/hashicorp/vault/api"
	"github.com/pkg/errors"
)

// NewOryAuth creates an OryAuth struct that can be passed into the client.Auth().Login
// method to authenticate with Vault via the Ory auth plugin.
//
// The mount path should be the path that the plugin was mounted at (probably `ory`)
// The namespace, object, and relation should reflect the resource being requested
// The cookie should be the full cookie string, including the name and `=`
// e.g. `ory_kratos_session=MyReallyLongSessionCookieString`
func NewOryAuth(
	mountPath, namespace, object, relation, cookie string,
) (*OryAuth, error) {
	switch {
	case mountPath == "":
		return nil, errors.New("no mount path provided")
	case namespace == "":
		return nil, errors.New("no namespace provided")
	case object == "":
		return nil, errors.New("no object provided")
	case relation == "":
		return nil, errors.New("no relation provided")
	case cookie == "":
		return nil, errors.New("no cookie provided")
	}

	return &OryAuth{
		mountPath: mountPath,
		namespace: namespace,
		object:    object,
		relation:  relation,
		cookie:    cookie,
	}, nil
}

// OryAuth is a Vault AuthMethod that can communicate to the Ory plugin
type OryAuth struct {
	mountPath string

	namespace string
	object    string
	relation  string
	cookie    string
}

// Login performs a login request to the Ory Vault auth plugin.
func (a *OryAuth) Login(ctx context.Context, client *api.Client) (*api.Secret, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	loginData := map[string]interface{}{
		"namespace":             a.namespace,
		"object":                a.object,
		"relation":              a.relation,
		"kratos_session_cookie": a.cookie,
	}
	path := fmt.Sprintf("auth/%s/login", a.mountPath)
	resp, err := client.Logical().WriteWithContext(ctx, path, loginData)
	if err != nil {
		return nil, errors.Wrap(err, "unable to log in with Ory auth")
	}

	return resp, nil
}
