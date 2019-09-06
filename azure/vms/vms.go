// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package vms

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
)

type SpecOption func(*Spec) *Spec

type Spec struct {
	internal *compute.VirtualMachine
}

func (o *Spec) Spec() *compute.VirtualMachine {
	return o.internal
}

func (o Spec) Name() string {
	return *o.internal.Name
}

func (o Spec) Tags() map[string]*string {
	return o.internal.Tags
}

func UserAssignedIdentity() SpecOption {
	return func(o *Spec) *Spec {
		o.internal.Identity.Type = compute.ResourceIdentityTypeUserAssigned
		for id, _ := range o.internal.Identity.UserAssignedIdentities {
			o.internal.Identity.UserAssignedIdentities[id] = &compute.VirtualMachineIdentityUserAssignedIdentitiesValue{}
		}
		return o
	}
}

func defaultSpec() *Spec {
	// shouild I fill this out?
	return &Spec{&compute.VirtualMachine{}}
}
