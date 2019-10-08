package computeresource

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/node-label-operator/azure"
)

type VirtualMachineScaleSet struct {
	group  string
	name   string
	client *compute.VirtualMachineScaleSetsClient
	vmss   *compute.VirtualMachineScaleSet
}

func NewVMSS(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (*VirtualMachineScaleSet, error) {
	client, err := azure.NewScaleSetClient(subscriptionID)
	if err != nil {
		return nil, err
	}
	vmss, err := client.Get(ctx, resourceGroup, resourceName)
	if err != nil {
		return nil, err
	}

	vmss = VMSSUserAssignedIdentity(vmss)

	return &VirtualMachineScaleSet{group: resourceGroup, name: resourceName, client: &client, vmss: &vmss}, nil
}

func NewVMSSInitialized(ctx context.Context, resourceGroup string, c *compute.VirtualMachineScaleSetsClient, v *compute.VirtualMachineScaleSet) *VirtualMachineScaleSet {
	return &VirtualMachineScaleSet{group: resourceGroup, name: *v.Name, client: c, vmss: v}
}

func (m VirtualMachineScaleSet) Update(ctx context.Context) error {
	f, err := m.client.CreateOrUpdate(ctx, m.group, m.name, *m.vmss)
	if err != nil {
		return err
	}

	if err := f.WaitForCompletionRef(ctx, m.client.Client); err != nil {
		return err
	}

	vmss, err := f.Result(*m.client)
	if err != nil {
		return err
	}

	m.vmss = &vmss
	return nil
}

func (m VirtualMachineScaleSet) Name() string {
	return m.name
}

func (m VirtualMachineScaleSet) ID() string {
	return *m.vmss.ID
}

func (m VirtualMachineScaleSet) Tags() map[string]*string {
	return m.vmss.Tags
}

func (m VirtualMachineScaleSet) SetTag(name string, value *string) {
	m.vmss.Tags[name] = value
}

func VMSSUserAssignedIdentity(vmss compute.VirtualMachineScaleSet) compute.VirtualMachineScaleSet {
	if vmss.Identity != nil { // is this an error otherwise?
		vmss.Identity.Type = compute.ResourceIdentityTypeUserAssigned
		for id := range vmss.Identity.UserAssignedIdentities {
			vmss.Identity.UserAssignedIdentities[id] = &compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
		}
	}
	return vmss
}
