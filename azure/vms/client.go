// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package vms

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"node-label-operator.io/azure"
)

type Service interface {
	Get(context.Context, string, string) (compute.VirtualMachine, error)
	// I need to implment update method for client
	CreateOrUpdate(context.Context, string, string, compute.VirtualMachine) (compute.VirtualMachine, error)
}

type Client struct {
	group    string
	internal Service
}

func NewClientService(group string, internal Service) *Client {
	return &Client{group: group, internal: internal}
}

func NewClient(subID, group string) (*Client, error) {
	c, err := newClient(subID)
	if err != nil {
		return nil, err
	}

	return &Client{group: group, internal: c}, nil
}

func (c *Client) Get(ctx context.Context, name string) (*Spec, error) {
	vm, err := c.internal.Get(ctx, c.group, name)
	if err != nil && azure.IsNotFound(err) {
		// return defaultSpec(), nil
		return nil, err
	} else if err != nil {
		return nil, err
	}

	return &Spec{internal: &vm}, nil
}

func (c *Client) Update(ctx context.Context, name string, spec *Spec) error {
	result, err := c.internal.CreateOrUpdate(ctx, c.group, name, *spec.internal)
	if err != nil {
		return err
	}
	spec.internal = &result
	return nil
}
