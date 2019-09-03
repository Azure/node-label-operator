// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package vms

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"

	"node-label-operator.io/azure"
)

type client struct {
	compute.VirtualMachinesClient
}

func newClient(subID string) (*client, error) {
	c, err := azure.NewVMClient(subID)
	if err != nil {
		return nil, err
	}
	return &client{c}, nil
}

func (c *client) Get(ctx context.Context, group, name string) (compute.VirtualMachine, error) {
	return c.VirtualMachinesClient.Get(ctx, group, name, compute.InstanceView)
}

func (c *client) CreateOrUpdate(ctx context.Context, group, name string, sg compute.VirtualMachine) (compute.VirtualMachine, error) {
	f, err := c.VirtualMachinesClient.CreateOrUpdate(ctx, group, name, sg)
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	err = f.WaitForCompletionRef(ctx, c.Client)
	if err != nil {
		return compute.VirtualMachine{}, err
	}

	return f.Result(c.VirtualMachinesClient)
}
