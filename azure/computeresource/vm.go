package computeresource

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/Azure/node-label-operator/azure"
)

type VirtualMachine struct {
	group  string
	name   string
	client *compute.VirtualMachinesClient
	vm     *compute.VirtualMachine
}

func NewVM(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (*VirtualMachine, error) {
	client, err := azure.NewVMClient(subscriptionID)
	if err != nil {
		return nil, err
	}
	vm, err := client.Get(ctx, resourceGroup, resourceName, compute.InstanceView)
	if err != nil {
		return nil, err
	}

	vm = VMUserAssignedIdentity(vm)

	return &VirtualMachine{group: resourceGroup, name: resourceName, client: &client, vm: &vm}, nil
}

func NewVMInitialized(ctx context.Context, resourceGroup string, c *compute.VirtualMachinesClient, v *compute.VirtualMachine) *VirtualMachine {
	return &VirtualMachine{group: resourceGroup, name: *v.Name, client: c, vm: v}
}

func (m VirtualMachine) Update(ctx context.Context) error {
	f, err := m.client.CreateOrUpdate(ctx, m.group, m.name, *m.vm)
	if err != nil {
		return err
	}

	if err := f.WaitForCompletionRef(ctx, m.client.Client); err != nil {
		return err
	}

	vm, err := f.Result(*m.client)
	if err != nil {
		return err
	}

	m.vm = &vm
	return nil
}

func (m VirtualMachine) Name() string {
	return m.name
}

func (m VirtualMachine) ID() string {
	return *m.vm.ID
}

func (m VirtualMachine) Tags() map[string]*string {
	return m.vm.Tags
}

func (m VirtualMachine) SetTag(name string, value *string) {
	m.vm.Tags[name] = value
}

func VMUserAssignedIdentity(vm compute.VirtualMachine) compute.VirtualMachine {
	if vm.Identity != nil {
		vm.Identity.Type = compute.ResourceIdentityTypeUserAssigned
		for id := range vm.Identity.UserAssignedIdentities {
			vm.Identity.UserAssignedIdentities[id] = &compute.VirtualMachineIdentityUserAssignedIdentitiesValue{}
		}
	}
	return vm
}
