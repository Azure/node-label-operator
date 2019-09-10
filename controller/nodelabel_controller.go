package controller

import (
	"context"
	"errors"
	"fmt"

	"github.com/Azure/azure-sdk-for-go/services/compute/mgmt/2019-03-01/compute"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"node-label-operator.io/azure"
)

const (
	VM   string = "virtualMachines"
	VMSS string = "virtualMachineScaleSets"
)

type ReconcileNodeLabel struct {
	client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	ctx      context.Context
}

type ComputeResource interface {
	// Get(ctx context.Context, name string) (azure.Spec, error)
	Update(ctx context.Context) error
	Tags() map[string]*string
	SetTag(name string, value *string)
}

type VirtualMachine struct {
	group  string
	client *compute.VirtualMachinesClient
	vm     *compute.VirtualMachine
}

func NewVM(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (VirtualMachine, error) {
	client, err := azure.NewVMClient(subscriptionID)
	if err != nil {
		return VirtualMachine{}, err
	}
	vm, err := client.Get(ctx, resourceGroup, resourceName, compute.InstanceView)
	if err != nil {
		return VirtualMachine{}, err
	}

	vm = VMUserAssignedIdentity(vm)

	return VirtualMachine{group: resourceGroup, client: &client, vm: &vm}, nil
}

func (m VirtualMachine) Get(ctx context.Context, name string) (compute.VirtualMachine, error) {
	vm, err := m.client.Get(ctx, m.group, name, compute.InstanceView)
	if err != nil {
		return vm, err
	}

	vm = VMUserAssignedIdentity(vm)

	return vm, nil
}

func (m VirtualMachine) Update(ctx context.Context) error {
	f, err := m.client.CreateOrUpdate(ctx, m.group, *m.vm.Name, *m.vm)
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

func (m VirtualMachine) Tags() map[string]*string {
	return m.vm.Tags
}

func (m VirtualMachine) SetTag(name string, value *string) {
	m.vm.Tags[name] = value
}

func VMUserAssignedIdentity(vm compute.VirtualMachine) compute.VirtualMachine {
	vm.Identity.Type = compute.ResourceIdentityTypeUserAssigned
	for id, _ := range vm.Identity.UserAssignedIdentities {
		vm.Identity.UserAssignedIdentities[id] = &compute.VirtualMachineIdentityUserAssignedIdentitiesValue{}
	}
	return vm
}

type VirtualMachineScaleSet struct {
	group  string
	client *compute.VirtualMachineScaleSetsClient
	vmss   *compute.VirtualMachineScaleSet
	// client *scalesets.Client
	// vmss   *scalesets.Spec
}

// I'm not sure this is actually modifying the client :(
func NewVMSS(ctx context.Context, subscriptionID, resourceGroup, resourceName string) (VirtualMachineScaleSet, error) {
	client, err := azure.NewScaleSetClient(subscriptionID)
	if err != nil {
		return VirtualMachineScaleSet{}, err
	}
	vmss, err := client.Get(ctx, resourceGroup, resourceName)
	if err != nil {
		return VirtualMachineScaleSet{}, err
	}

	vmss = VMSSUserAssignedIdentity(vmss)

	return VirtualMachineScaleSet{group: resourceGroup, client: &client, vmss: &vmss}, nil
}

// find a way to actually use get??
func (m VirtualMachineScaleSet) Get(ctx context.Context, name string) (compute.VirtualMachineScaleSet, error) {
	vmss, err := m.client.Get(ctx, m.group, name)
	if err != nil {
		return compute.VirtualMachineScaleSet{}, err
	}

	vmss = VMSSUserAssignedIdentity(vmss)
	// assign := scalesets.UserAssignedIdentity()
	// assign(vmss) // spec.Set?

	return vmss, nil
}

// does this work the wayw it's supposed to?
func (m VirtualMachineScaleSet) Update(ctx context.Context) error {
	f, err := m.client.CreateOrUpdate(ctx, m.group, *m.vmss.Name, *m.vmss)
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

func (m VirtualMachineScaleSet) Tags() map[string]*string {
	return m.vmss.Tags
}

// is this actually being set? deep or shallow copy?
func (m VirtualMachineScaleSet) SetTag(name string, value *string) {
	m.vmss.Tags[name] = value
}

func VMSSUserAssignedIdentity(vmss compute.VirtualMachineScaleSet) compute.VirtualMachineScaleSet {
	vmss.Identity.Type = compute.ResourceIdentityTypeUserAssigned
	for id, _ := range vmss.Identity.UserAssignedIdentities {
		vmss.Identity.UserAssignedIdentities[id] = &compute.VirtualMachineScaleSetIdentityUserAssignedIdentitiesValue{}
	}
	return vmss
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get

func (r *ReconcileNodeLabel) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	r.ctx = context.Background()
	log := r.Log.WithValues("node-label-operator", req.NamespacedName)

	var configMap corev1.ConfigMap
	var configOptions ConfigOptions
	optionsNamespacedName := types.NamespacedName{Name: "node-label-operator", Namespace: "default"} // is this okay
	if err := r.Get(r.ctx, optionsNamespacedName, &configMap); err != nil {
		log.V(1).Info("unable to fetch ConfigMap, instead using default configuration settings")
		// should I allow this to continue? It would be unfortunate to have things sync and then clean it up
		configOptions = DefaultConfigOptions()
	} else {
		configOptions, err = NewConfigOptions(configMap) // ConfigMap.Data is string -> string but I don't always want that
		if err != nil {
			log.Error(err, "failed to load options from config file")
			return ctrl.Result{}, err
		}
	}

	var node corev1.Node
	if err := r.Get(r.ctx, req.NamespacedName, &node); err != nil {
		log.Error(err, "unable to fetch Node")
		return ctrl.Result{}, err // what should I return here?
	}
	log.V(1).Info("provider info", "provider ID", node.Spec.ProviderID)
	provider, err := azure.ParseProviderID(node.Spec.ProviderID)
	if err != nil {
		log.Error(err, "invalid provider ID")
	}

	switch provider.ResourceType {
	case VMSS:
		// Add VMSS tags to node
		if err := r.reconcileVMSS(req.NamespacedName, &provider, &node, configOptions); err != nil {
			log.Error(err, "failed to apply tags to nodes")
			return reconcile.Result{}, err
		}
	case VM:
		// Add VM tags to node
		if err := r.reconcileVMs(req.NamespacedName, &provider, &node, configOptions); err != nil {
			log.Error(err, "failed to apply tags to nodes")
			return reconcile.Result{}, err
		}
	default:
		log.V(1).Info("unrecognized resource type", "resource type", provider.ResourceType)
	}

	return ctrl.Result{}, nil
}

// pass VMSS -> tags info and assign to nodes on VMs (unless node already has label)
func (r *ReconcileNodeLabel) reconcileVMSS(namespacedName types.NamespacedName, provider *azure.Resource,
	node *corev1.Node, configOptions ConfigOptions) error {
	vmss, err := NewVMSS(r.ctx, provider.SubscriptionID, provider.ResourceGroup, provider.ResourceName)
	if err != nil {
		return err
	}

	if configOptions.SyncDirection == TwoWay || configOptions.SyncDirection == ARMToNode {
		updatedNode, err := r.applyTagsToNodes(namespacedName, vmss, node, configOptions)
		if err != nil {
			return err
		}
		if err = r.Update(r.ctx, updatedNode); err != nil { // should this be a patch?
			return err
		}
	}

	// assign all labels on Node to VMSS, if not already there
	if configOptions.SyncDirection == TwoWay || configOptions.SyncDirection == NodeToARM {
		updatedVMSS, err := r.applyLabelsToAzureResource(namespacedName, vmss, node, configOptions)
		if err != nil {
			return err
		}
		// I shouldn't just be updating no matter what... not if arm-to-node
		if err = updatedVMSS.Update(r.ctx); err != nil {
			return err
		}
	}

	return nil
}

func (r *ReconcileNodeLabel) reconcileVMs(namespacedName types.NamespacedName, provider *azure.Resource,
	node *corev1.Node, configOptions ConfigOptions) error {
	vm, err := NewVM(r.ctx, provider.SubscriptionID, provider.ResourceGroup, provider.ResourceName)
	if err != nil {
		return err
	}

	if configOptions.SyncDirection == TwoWay || configOptions.SyncDirection == ARMToNode {
		updatedNode, err := r.applyTagsToNodes(namespacedName, vm, node, configOptions)
		if err != nil {
			return err
		}
		if err = r.Update(r.ctx, updatedNode); err != nil { // should this be a patch?
			return err
		}
	}

	if configOptions.SyncDirection == TwoWay || configOptions.SyncDirection == NodeToARM {
		updatedVM, err := r.applyLabelsToAzureResource(namespacedName, vm, node, configOptions)
		if err != nil {
			return err
		}
		if err = updatedVM.Update(r.ctx); err != nil {
			return err
		}
	}

	return nil
}

// is it a problem that I have r.Recorder and I'm trying to unit test this with sorta fake reconciler?
// maybe return bool for updating?
func (r *ReconcileNodeLabel) applyTagsToNodes(namespacedName types.NamespacedName, computeResource ComputeResource, node *corev1.Node, configOptions ConfigOptions) (*corev1.Node, error) {
	log := r.Log.WithValues("node-label-operator", namespacedName)
	log.V(0).Info("configOptions", "sync direction", configOptions.SyncDirection)

	for tagName, tagVal := range computeResource.Tags() {
		if !ValidLabelName(tagName) {
			log.V(0).Info("invalid label name", "tag name", tagName)
			continue
		}
		validLabelName := ConvertTagNameToValidLabelName(tagName, configOptions)
		labelVal, ok := node.Labels[validLabelName]
		if !ok {
			// add tag as label
			log.V(1).Info("applying tags to nodes", "tagName", tagName, "tagVal", *tagVal)
			node.Labels[validLabelName] = *tagVal
		} else if labelVal != *tagVal {
			switch configOptions.ConflictPolicy {
			case ARMPrecedence:
				// set label anyway
				log.V(1).Info("overriding existing node label with ARM tag", "tagName", tagName, "tagVal", tagVal)
				node.Labels[validLabelName] = *tagVal
			case NodePrecedence:
				// do nothing
				log.V(0).Info("name->value conflict found", "node label value", labelVal, "ARM tag value", *tagVal)
			case Ignore:
				// raise k8s event
				r.Recorder.Event(node, "Warning", "ConflictingTagLabelValues",
					fmt.Sprintf("ARM tag was not applied to node because a different value for '%s' already exists (%s != %s).", tagName, *tagVal, labelVal))
				log.V(0).Info("name->value conflict found, leaving unchanged", "label value", labelVal, "tag value", *tagVal)
			default:
				return node, errors.New("unrecognized conflict policy")
			}
		}
	}

	return node, nil
}

// I need to make sure I can get update to work with ComputeResource interface! value vs reference issue
func (r *ReconcileNodeLabel) applyLabelsToAzureResource(namespacedName types.NamespacedName, computeResource ComputeResource, node *corev1.Node, configOptions ConfigOptions) (ComputeResource, error) {
	log := r.Log.WithValues("node-label-operator", namespacedName)
	log.V(1).Info("configOptions", "sync direction", configOptions.SyncDirection)

	if len(computeResource.Tags()) > maxNumTags {
		log.V(0).Info("can't add any more tags", "number of tags", len(computeResource.Tags()))
		return computeResource, nil
	}

	for labelName, labelVal := range node.Labels {
		if !ValidTagName(labelName, configOptions) {
			// log.Error(errors.New(fmt.Sprintf("invalid tag name")), fmt.Sprintf("label name: %s", labelName))
			log.V(0).Info("invalid tag name", "label name", labelName)
			continue
		}
		validTagName := ConvertLabelNameToValidTagName(labelName, configOptions)
		tagVal, ok := computeResource.Tags()[validTagName]
		if !ok {
			// add label as tag
			log.V(1).Info("applying labels to Azure resource", "labelName", labelName, "labelVal", labelVal)
			// is  this causing the problem in my unit tests?
			computeResource.SetTag(validTagName, &labelVal)
		} else if *tagVal != labelVal {
			switch configOptions.ConflictPolicy {
			case NodePrecedence:
				// set tag anyway
				log.V(1).Info("overriding existing ARM tag with node label", "labelName", labelName, "labelVal", labelVal)
				computeResource.SetTag(validTagName, &labelVal)
			case ARMPrecedence:
				// do nothing
				log.V(0).Info("name->value conflict found", "node label value", labelVal, "ARM tag value", *tagVal)
			case Ignore:
				// raise k8s event
				r.Recorder.Event(node, "Warning", "ConflictingTagLabelValues",
					fmt.Sprintf("node label was not applied to Azure resource because a different value for '%s' already exists (%s != %s).", labelName, labelVal, *tagVal))
				log.V(0).Info("name->value conflict found, leaving unchanged", "label value", labelVal, "tag value", *tagVal)
			default:
				return computeResource, errors.New("unrecognized conflict policy")
			}
		}
	}

	return computeResource, nil
}

func (r *ReconcileNodeLabel) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		Complete(r)
}
