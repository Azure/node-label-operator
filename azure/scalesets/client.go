// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package scalesets

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"

	"node-label-operator.io/azure"
)

type Service interface {
	Get(context.Context, string, string) (compute.VirtualMachineScaleSet, error)
	CreateOrUpdate(context.Context, string, string, compute.VirtualMachineScaleSet) (compute.VirtualMachineScaleSet, error)
}

type Client struct {
	group    string
	internal Service
}

func NewClient(subID, group string) (*Client, error) {
	c, err := newClient(subID)
	if err != nil {
		return nil, err
	}

	return &Client{group: group, internal: c}, nil
}

func (c *Client) Get(ctx context.Context, name string) (*Spec, error) {
	id, err := c.internal.Get(ctx, c.group, name)
	// is the problem that the cluster isn't found and I'm getting a default spec?
	if err != nil && azure.IsNotFound(err) {
		return nil, err
		// return defaultSpec(), nil
	} else if err != nil {
		return nil, err
	}

	return &Spec{&id}, nil
}

func (c *Client) Update(ctx context.Context, name string, spec *Spec) error {
	result, err := c.internal.CreateOrUpdate(ctx, c.group, name, *spec.internal)
	if err != nil {
		return err
	}
	spec.internal = &result
	return nil
}
