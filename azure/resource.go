package azure

import (
	"context"

	"github.com/Azure/azure-sdk-for-go/profiles/latest/compute/mgmt/compute"
)

type Service interface {
	Get(context.Context, string, string) (compute.VirtualMachine, error)
	CreateOrUpdate(context.Context, string, string, compute.VirtualMachine) (compute.VirtualMachine, error)
}

type Client interface {
	Get(context.Context, string) (*Spec, error)
	Update(context.Context, string, *Spec) error
}

type Spec interface {
	// what goes here?
}
