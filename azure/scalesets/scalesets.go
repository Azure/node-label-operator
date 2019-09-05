// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package scalesets

import (
	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/go-autorest/autorest/to"
)

type SpecOption func(*Spec) *Spec

type Spec struct {
	internal *compute.VirtualMachineScaleSet
}

func (o *Spec) Spec() *compute.VirtualMachineScaleSet {
	return o.internal
}

func (o Spec) Name() string {
	return *o.internal.Name
}

// is this rep exposure? :o
func (o Spec) Tags() map[string]*string {
	return o.internal.Tags
}

func Name(name string) SpecOption {
	return func(o *Spec) *Spec {
		o.internal.Name = &name
		return o
	}
}

func Location(location string) SpecOption {
	return func(o *Spec) *Spec {
		o.internal.Location = &location
		return o
	}
}

func Prefix(namePrefix string) SpecOption {
	return func(o *Spec) *Spec {
		o.internal.VirtualMachineScaleSetProperties.VirtualMachineProfile.OsProfile.ComputerNamePrefix = &namePrefix
		return o
	}
}

func DefaultSpec() *Spec {
	return &Spec{&compute.VirtualMachineScaleSet{
		Sku: &compute.Sku{},
		Identity: &compute.VirtualMachineScaleSetIdentity{
			UserAssignedIdentities: map[string]*compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{},
		},
		VirtualMachineScaleSetProperties: &compute.VirtualMachineScaleSetProperties{
			Overprovision: to.BoolPtr(false),
			UpgradePolicy: &compute.UpgradePolicy{
				Mode: compute.Manual,
			},
			VirtualMachineProfile: &compute.VirtualMachineScaleSetVMProfile{
				Priority: compute.Regular,
				OsProfile: &compute.VirtualMachineScaleSetOSProfile{
					AdminUsername: to.StringPtr("azureuser"),
					LinuxConfiguration: &compute.LinuxConfiguration{
						DisablePasswordAuthentication: to.BoolPtr(true),
						SSH: &compute.SSHConfiguration{
							PublicKeys: &[]compute.SSHPublicKey{},
						},
					},
				},
				DiagnosticsProfile: &compute.DiagnosticsProfile{
					BootDiagnostics: &compute.BootDiagnostics{},
				},
				StorageProfile: &compute.VirtualMachineScaleSetStorageProfile{
					ImageReference: &compute.ImageReference{
						Offer:     to.StringPtr("UbuntuServer"),
						Publisher: to.StringPtr("Canonical"),
						Sku:       to.StringPtr("18.04-LTS"),
						Version:   to.StringPtr("latest"),
					},
					OsDisk: &compute.VirtualMachineScaleSetOSDisk{},
				},
				NetworkProfile: &compute.VirtualMachineScaleSetNetworkProfile{
					NetworkInterfaceConfigurations: &[]compute.VirtualMachineScaleSetNetworkConfiguration{},
				},
			},
		},
	}}
}
