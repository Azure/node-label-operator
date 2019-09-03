// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package azure

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/azure-sdk-for-go/services/graphrbac/1.6/graphrbac"
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
)

const userAgent string = "genesys"

func NewAvailabilitySetClient(subID string) (compute.AvailabilitySetsClient, error) {
	a, err := injectAuthorizer()
	if err != nil {
		return compute.AvailabilitySetsClient{}, err
	}
	client := compute.NewAvailabilitySetsClient(subID)
	client.Authorizer = a
	if err := client.AddToUserAgent(userAgent); err != nil {
		return compute.AvailabilitySetsClient{}, err
	}
	return client, nil
}

func NewVMClient(subID string) (compute.VirtualMachinesClient, error) {
	a, err := injectAuthorizer()
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

func NewIdentityClient(subID string) (msi.UserAssignedIdentitiesClient, error) {
	a, err := injectAuthorizer()
	if err != nil {
		return msi.UserAssignedIdentitiesClient{}, err
	}
	client := msi.NewUserAssignedIdentitiesClient(subID)
	client.Authorizer = a
	if err := client.AddToUserAgent(userAgent); err != nil {
		return msi.UserAssignedIdentitiesClient{}, err
	}
	return client, nil
}

func NewServicePrincipalClient(tenantID string) (graphrbac.ServicePrincipalsClient, error) {
	a, err := injectGraphAuthorizer()
	if err != nil {
		return graphrbac.ServicePrincipalsClient{}, err
	}
	client := graphrbac.NewServicePrincipalsClient(tenantID)
	client.Authorizer = a
	if err := client.AddToUserAgent(userAgent); err != nil {
		return graphrbac.ServicePrincipalsClient{}, err
	}
	return client, nil
}

func NewApplicationClient(tenantID string) (graphrbac.ApplicationsClient, error) {
	a, err := injectGraphAuthorizer()
	if err != nil {
		return graphrbac.ApplicationsClient{}, err
	}
	client := graphrbac.NewApplicationsClient(tenantID)
	client.Authorizer = a
	if err := client.AddToUserAgent(userAgent); err != nil {
		return graphrbac.ApplicationsClient{}, err
	}
	return client, nil
}

func NewScaleSetClient(subID string) (compute.VirtualMachineScaleSetsClient, error) {
	a, err := injectAuthorizer()
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
