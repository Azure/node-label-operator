// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package identities

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"

	"node-label-operator.io/azure"
)

// ClientFactory is used to inject the client as dependency into the phases
type ClientFactory func(string, string) (Client, error)

type identitiesClient interface {
	Get(context.Context, string, string) (msi.Identity, error)
	CreateOrUpdate(context.Context, string, string, msi.Identity) (msi.Identity, error)
}

type Client interface {
	Get(ctx context.Context, name string) (*Spec, error)
	Ensure(ctx context.Context, name string, spec *Spec) error
}

type client struct {
	group      string
	identities identitiesClient
}

var _ Client = &client{}

func NewClient(subID, group string) (Client, error) {
	c, err := azure.NewIdentityClient(subID)
	if err != nil {
		return nil, err
	}

	return &client{group: group, identities: c}, nil
}

func (c *client) Get(ctx context.Context, name string) (*Spec, error) {
	id, err := c.identities.Get(ctx, c.group, name)
	if err != nil && azure.IsNotFound(err) {
		return &Spec{&msi.Identity{}}, nil
	} else if err != nil {
		return nil, err
	}

	return &Spec{&id}, nil
}

func (c *client) Ensure(ctx context.Context, name string, spec *Spec) error {
	result, err := c.identities.CreateOrUpdate(ctx, c.group, name, *spec.identity)
	if err != nil {
		return err
	}
	spec.identity = &result
	return nil
}
