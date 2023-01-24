package plugin

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/vault/sdk/framework"
	"github.com/hashicorp/vault/sdk/logical"

	keto "github.com/ory/keto/proto/ory/keto/relation_tuples/v1alpha2"
	kratos "github.com/ory/kratos-client-go"

	"github.com/pkg/errors"
)

const (
	// pathLoginSynopsis is used to generate the help text for the login path.
	pathLoginSynopsis = `
Authenticates Ory Kratos users with Vault and authorises a policy with Keto.
`

	// pathLoginDesc is used to generate the help text for the login path.
	pathLoginDescription = `
Authenticate Ory Kratos identities using a Kratos session cookie.
Authorise the identity with Keto using a namespace, object and relation.
Resulting policy is named after the namespace and relation in the format
namespace_relation.
`
)

// NewPathLogin returns the path for the login endpoint.
func NewPathLogin(b *OryAuthBackend) []*framework.Path {
	return []*framework.Path{
		{
			Pattern: "login$",
			Fields: map[string]*framework.FieldSchema{
				"kratos_session_cookie": {
					Type: framework.TypeString,
					Description: `The Kratos session cookie.
This is the value of the Kratos session cookie.`,
				},
				"namespace": {
					Type: framework.TypeString,
					Description: `Keto namespace of the resource being authenticated against.
If 'namespace' is not specified, login fails.`,
				},
				"object": {
					Type: framework.TypeString,
					Description: `Keto object being authenticated against.
If 'object' is not specified, login fails.`,
				},
				"relation": {
					Type: framework.TypeString,
					Description: `Keto relation between subject and object being authenticated against.
If 'relation' is not specified, login fails.`,
				},
			},
			Callbacks: map[logical.Operation]framework.OperationFunc{
				logical.UpdateOperation: b.loginUpdateHandler,
			},
			HelpSynopsis:    pathLoginSynopsis,
			HelpDescription: pathLoginDescription,
		},
	}
}

// loginUpdateHandler is the handler for the login path.
func (b *OryAuthBackend) loginUpdateHandler(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*logical.Response, error) {
	b.Logger().Debug("pathLoginUpdate called")

	kratosSession, err := b.getKratosSession(ctx, req, data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	namespace, err := b.getNamespace(data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	object, err := b.getObject(data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	relation, err := b.getRelation(data)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	subject, err := b.getSubject(kratosSession)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	// TODO do we replace with List call and create policies for all relations?
	allowed, err := b.checkRelation(ctx, req, namespace, object, relation, subject)
	if err != nil {
		return logical.ErrorResponse(err.Error()), nil
	}

	if !allowed {
		return logical.ErrorResponse(
			"subject does not have the relation to the object in the namespace",
		), nil
	}

	policy := strings.Join([]string{namespace, relation}, "_")
	policies := []string{policy}

	metadata := map[string]string{
		"namespace": namespace,
		"object":    object,
		"relation":  relation,
		"subject":   subject,
	}

	internalData := map[string]interface{}{
		"namespace": namespace,
		"object":    object,
		"relation":  relation,
		"subject":   subject,
	}

	config, err := b.readConfig(ctx, req.Storage)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch config")
	}

	var ttl time.Duration
	if config.UseSessionExpiryTTL {
		ttl = time.Until(*kratosSession.ExpiresAt)
	} else {
		ttl = time.Duration(config.TTLSeconds) * time.Second
	}

	res := &logical.Response{
		Auth: &logical.Auth{
			Period: ttl,
			Alias: &logical.Alias{
				Name:     "ory-auth",
				Metadata: metadata,
			},
			Policies:     policies,
			InternalData: internalData,
			DisplayName:  "kratos-keto",
			LeaseOptions: logical.LeaseOptions{
				Renewable: false,
				TTL:       ttl,
				MaxTTL:    time.Duration(config.MaxTTLSeconds) * time.Second,
			},
		},
	}

	return res, nil
}

// getKratosSession returns the Kratos session from the request.
func (b *OryAuthBackend) getKratosSession(
	ctx context.Context,
	req *logical.Request,
	data *framework.FieldData,
) (*kratos.Session, error) {
	val, ok := data.GetOk("kratos_session_cookie")
	if !ok {
		return nil, errors.New("kratos_session_cookie is required")
	}

	kratosSessionCookie, ok := val.(string)
	if !ok || kratosSessionCookie == "" {
		return nil, errors.New("missing kratos_session_cookie")
	}
	b.Logger().Debug("found kratos session cookie", "kratos_session_cookie", kratosSessionCookie)

	client, err := b.getKratosClient(ctx, req.Storage)
	if err != nil {
		return nil, errors.Wrap(err, "could not get Kratos client")
	}

	session, _, err := b.validateSessionCookie(ctx, client, kratosSessionCookie)
	if err != nil {
		return nil, errors.Wrap(err, "could not validate kratos session cookie")
	}

	b.Logger().Debug("found kratos session", "session", session)

	return session, nil
}

// getNamespace returns the namespace from the request.
func (b *OryAuthBackend) getNamespace(
	data *framework.FieldData,
) (string, error) {
	b.Logger().Debug("getting namespace from data")

	val, ok := data.GetOk("namespace")
	if !ok {
		return "", errors.New("namespace is required")
	}

	namespace, ok := val.(string)
	if !ok || namespace == "" {
		return "", errors.New("missing namespace")
	}

	return namespace, nil
}

// getObject returns the object from the request.
func (b *OryAuthBackend) getObject(
	data *framework.FieldData,
) (string, error) {
	b.Logger().Debug("getting object from data")

	val, ok := data.GetOk("object")
	if !ok {
		return "", errors.New("object is required")
	}

	object, ok := val.(string)
	if !ok || object == "" {
		return "", errors.New("missing object")
	}

	return object, nil
}

// getRelation returns the relation from the request.
func (b *OryAuthBackend) getRelation(
	data *framework.FieldData,
) (string, error) {
	b.Logger().Debug("getting relation from data")

	val, ok := data.GetOk("relation")
	if !ok {
		return "", errors.New("relation is required")
	}

	relation, ok := val.(string)
	if !ok || relation == "" {
		return "", errors.New("missing relation")
	}

	return relation, nil
}

// getSubject returns the subject from the Kratos session.
func (b *OryAuthBackend) getSubject(session *kratos.Session) (string, error) {
	b.Logger().Debug("getting subject from Kratos session")

	if session == nil {
		return "", errors.New("session is nil")
	}

	return session.Identity.Id, nil
}

// checkRelation checks if the subject has the relation to the object in the namespace.
func (b *OryAuthBackend) checkRelation(
	ctx context.Context,
	req *logical.Request,
	namespace string,
	object string,
	relation string,
	subject string,
) (bool, error) {
	b.Logger().Debug("checking if subject has relation to object in namespace")

	if namespace == "" {
		return false, errors.New("namespace is empty")
	}

	if object == "" {
		return false, errors.New("object is empty")
	}

	if relation == "" {
		return false, errors.New("relation is empty")
	}

	if subject == "" {
		return false, errors.New("subject is empty")
	}

	ketoClient, err := b.getKetoClient(ctx, req.Storage)
	if err != nil {
		return false, errors.Wrap(err, "failed to get keto client")
	}

	res, err := ketoClient.CheckServiceClient.Check(
		ctx,
		&keto.CheckRequest{
			Namespace: namespace,
			Object:    object,
			Relation:  relation,
			Subject:   keto.NewSubjectID(subject),
		},
	)
	if err != nil {
		return false, errors.Wrap(err, "failed keto check")
	}

	return res.GetAllowed(), nil
}

// validateSessionCookie validates the session cookie by making a request to the Kratos API.
func (b *OryAuthBackend) validateSessionCookie(
	ctx context.Context,
	client *kratos.APIClient,
	kratosSessionCookie string,
) (*kratos.Session, int, error) {
	session, res, err := client.V0alpha2Api.ToSessionExecute(
		kratos.V0alpha2ApiApiToSessionRequest{}.Cookie(kratosSessionCookie),
	)
	if err != nil {
		b.Logger().Error("error while trying to get kratos session", "err", err)
		return nil, http.StatusInternalServerError, errors.Wrap(err, "failed to get kratos session")
	}

	if res.StatusCode != http.StatusOK {
		b.Logger().Debug("status was not 200", "status", res.StatusCode)
		return nil, res.StatusCode, errors.Wrap(err, "failed to get kratos session")
	}

	return session, http.StatusOK, nil
}
