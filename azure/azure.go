// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/azure/auth"
)

const userAgent string = "node-label-operator"

func NewVMClient(subID string) (compute.VirtualMachinesClient, error) {
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return compute.VirtualMachinesClient{}, err
	}
	client := compute.NewVirtualMachinesClient(subID)
	client.Authorizer = a
	if err := client.AddToUserAgent(userAgent); err != nil {
		return compute.VirtualMachinesClient{}, err
	}
	return client, nil
}

func NewScaleSetClient(subID string) (compute.VirtualMachineScaleSetsClient, error) {
	a, err := auth.NewAuthorizerFromEnvironment()
	if err != nil {
		return compute.VirtualMachineScaleSetsClient{}, err
	}
	client := compute.NewVirtualMachineScaleSetsClient(subID)
	client.Authorizer = a
	if err := client.AddToUserAgent(userAgent); err != nil {
		return compute.VirtualMachineScaleSetsClient{}, err
	}
	return client, nil
}
