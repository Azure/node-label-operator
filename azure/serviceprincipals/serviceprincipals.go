package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"

	"node-label-operator.io/azure"
)

type SpecOption func(*Spec) *Spec

type Spec struct {
	internal *graphrbac.ServicePrincipal
}

type Service interface {
	Get(context.Context, string) (graphrbac.ServicePrincipal, error)
}

type Client struct {
	internal Service
}

type client struct {
	sp  graphrbac.ServicePrincipalsClient
	app graphrbac.ApplicationsClient
}

func (s *Spec) ObjectID() string {
	return *s.internal.ObjectID
}

func NewClient(tenantID string) (*Client, error) {
	c, err := newClient(tenantID)
	if err != nil {
		return nil, err
	}

	return &Client{internal: c}, nil
}

func (c *Client) Get(ctx context.Context, appID string) (*Spec, error) {
	sp, err := c.internal.Get(ctx, appID)
	if err != nil {
		return nil, err
	}
	return &Spec{&sp}, nil
}

func newClient(tenantID string) (*client, error) {
	sp, err := azure.NewServicePrincipalClient(tenantID)
	if err != nil {
		return nil, err
	}
	app, err := azure.NewApplicationClient(tenantID)
	if err != nil {
		return nil, err
	}
	return &client{sp: sp, app: app}, nil
}

func (c *client) Get(ctx context.Context, appID string) (graphrbac.ServicePrincipal, error) {
	result, err := c.app.GetServicePrincipalsIDByAppID(ctx, appID)
	if err != nil {
		return graphrbac.ServicePrincipal{}, err
	}
	return graphrbac.ServicePrincipal{ObjectID: result.Value}, nil
}
