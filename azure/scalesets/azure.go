// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package scalesets

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"

	"node-label-operator.io/azure"
)

type client struct {
	compute.VirtualMachineScaleSetsClient
}

func newClient(subID string) (*client, error) {
	c, err := azure.NewScaleSetClient(subID)
	if err != nil {
		return nil, err
	}
	return &client{c}, nil
}

func (c *client) Get(ctx context.Context, group, name string) (compute.VirtualMachineScaleSet, error) {
	return c.VirtualMachineScaleSetsClient.Get(ctx, group, name)
}

func (c *client) CreateOrUpdate(ctx context.Context, group, name string, sg compute.VirtualMachineScaleSet) (compute.VirtualMachineScaleSet, error) {
	f, err := c.VirtualMachineScaleSetsClient.CreateOrUpdate(ctx, group, name, sg)
	if err != nil {
		return compute.VirtualMachineScaleSet{}, err
	}

	err = f.WaitForCompletionRef(ctx, c.Client)
	if err != nil {
		return compute.VirtualMachineScaleSet{}, err
	}

	return f.Result(c.VirtualMachineScaleSetsClient)
}
