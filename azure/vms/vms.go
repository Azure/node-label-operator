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

func (spec *Spec) Spec() *compute.VirtualMachine {
	return spec.internal
}

func defaultSpec() *Spec {
	// shouild I fill this out?
	return &Spec{&compute.VirtualMachine{}}
}
