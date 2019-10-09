package computeresource

import (
	"context"
)

const (
	VM   string = "virtualMachines"
	VMSS string = "virtualMachineScaleSets"
)

// ComputeResource is a compute resource such as a Virtual Machine that
// should have its labels propagated to nodes running on the compute resource
type ComputeResource interface {
	Name() string
	ID() string
	Tags() map[string]*string
	SetTag(name string, value *string)
	Update(ctx context.Context) error
}
