// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/go-logr/logr"
	"github.com/prometheus/common/log"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/Azure/node-label-operator/azure"
	azrsrc "github.com/Azure/node-label-operator/azure/computeresource"
	"github.com/Azure/node-label-operator/conversion"
	"github.com/Azure/node-label-operator/conversion/options"
)

const (
	lastUpdateLabel    string        = "node-label-operator/last-update"
	minSyncPeriodLabel string        = "node-label-operator/min-sync-period"
	FiveMinutes        time.Duration = time.Minute * 5
)

type ReconcileNodeLabel struct {
	client.Client
	Log           logr.Logger
	Scheme        *runtime.Scheme
	Recorder      record.EventRecorder
	MinSyncPeriod time.Duration
	ctx           context.Context
	lock          sync.Mutex
}

// +kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=nodes/status,verbs=get
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch

func (r *ReconcileNodeLabel) Reconcile(req reconcile.Request) (reconcile.Result, error) {
	r.ctx = context.Background()
	log := r.Log.WithValues("node-label-operator", req.NamespacedName)

	var configMap corev1.ConfigMap
	optionsNamespacedName := options.OptionsConfigMapNamespacedName() // assuming "node-label-operator" and "node-label-operator-system", is this okay
	if err := r.Get(r.ctx, optionsNamespacedName, &configMap); err != nil {
		log.V(1).Info("unable to fetch ConfigMap, instead using default configuration settings")
		// create default options config map and requeue
		configMap, err := options.NewDefaultConfigOptions()
		if err != nil {
			log.Error(err, "failed to get new default options configmap")
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
		}
		if err := r.Create(r.ctx, configMap); err != nil {
			log.Error(err, "failed to create new default options configmap")
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
		}
		return ctrl.Result{RequeueAfter: time.Minute}, nil
	}
	configOptions, err := options.NewConfigOptions(configMap)
	if err != nil {
		log.Error(err, "failed to load options from config file")
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	configMinSyncPeriod, err := time.ParseDuration(configOptions.MinSyncPeriod)
	if err != nil {
		log.Error(err, "failed to parse minSyncPeriod")
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	if configMinSyncPeriod.Milliseconds() != r.MinSyncPeriod.Milliseconds() {
		r.setMinSyncPeriod(configMinSyncPeriod)
	}

	var node corev1.Node
	if err := r.Get(r.ctx, req.NamespacedName, &node); err != nil {
		log.Error(err, "unable to fetch Node")
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	log.V(1).Info("provider info", "provider ID", node.Spec.ProviderID)
	provider, err := azure.ParseProviderID(node.Spec.ProviderID)
	if err != nil {
		log.Error(err, "invalid provider ID", "node", node.Name)
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}
	if configOptions.ResourceGroupFilter != options.DefaultResourceGroupFilter &&
		provider.ResourceGroup != configOptions.ResourceGroupFilter {
		log.V(1).Info("found node not in resource group filter", "resource group filter", configOptions.ResourceGroupFilter, "node", node.Name)
		return ctrl.Result{}, nil
	}

	switch provider.ResourceType {
	case azrsrc.VMSS:
		// Add VMSS tags to node
		if err := r.reconcileVMSS(req.NamespacedName, &provider, &node, configOptions); err != nil {
			log.Error(err, "failed to apply tags to nodes")
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
		}
	case azrsrc.VM:
		// Add VM tags to node
		if err := r.reconcileVMs(req.NamespacedName, &provider, &node, configOptions); err != nil {
			log.Error(err, "failed to apply tags to nodes")
			return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
		}
	default:
		log.V(1).Info("unrecognized resource type", "resource type", provider.ResourceType)
	}

	// update lastUpdate label on node
	if err = r.updateMinSyncPeriodLabels(&node); err != nil {
		return ctrl.Result{RequeueAfter: 5 * time.Minute}, nil
	}

	return ctrl.Result{}, nil
}

// pass VMSS -> tags info and assign to nodes on VMs (unless node already has label)
func (r *ReconcileNodeLabel) reconcileVMSS(namespacedName types.NamespacedName, provider *azure.Resource,
	node *corev1.Node, configOptions *options.ConfigOptions) error {
	vmss, err := azrsrc.NewVMSS(r.ctx, provider.SubscriptionID, provider.ResourceGroup, provider.ResourceName)
	if err != nil {
		return err
	}

	if configOptions.SyncDirection == options.TwoWay || configOptions.SyncDirection == options.ARMToNode {
		// only update if there are changes to labels
		patch, err := r.applyTagsToNodes(namespacedName, *vmss, node, configOptions)
		if err != nil {
			return err
		}
		if patch != nil { // gross looking
			if err = r.Patch(r.ctx, node, client.ConstantPatch(types.MergePatchType, patch)); err != nil {
				return err
			}
		}
	}

	// assign all labels on Node to VMSS, if not already there
	if configOptions.SyncDirection == options.TwoWay || configOptions.SyncDirection == options.NodeToARM {
		// only update if there are changes to labels
		tags, err := r.applyLabelsToAzureResource(namespacedName, *vmss, node, configOptions)
		if err != nil {
			return err
		}
		if tags != nil {
			for key, val := range tags {
				vmss.SetTag(key, val)
			}
			if err = vmss.Update(r.ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ReconcileNodeLabel) reconcileVMs(namespacedName types.NamespacedName, provider *azure.Resource,
	node *corev1.Node, configOptions *options.ConfigOptions) error {
	vm, err := azrsrc.NewVM(r.ctx, provider.SubscriptionID, provider.ResourceGroup, provider.ResourceName)
	if err != nil {
		return err
	}

	if configOptions.SyncDirection == options.TwoWay || configOptions.SyncDirection == options.ARMToNode {
		patch, err := r.applyTagsToNodes(namespacedName, *vm, node, configOptions)
		if err != nil {
			return err
		}
		if patch != nil {
			if err = r.Patch(r.ctx, node, client.ConstantPatch(types.MergePatchType, patch)); err != nil {
				return err
			}
		}
	}

	if configOptions.SyncDirection == options.TwoWay || configOptions.SyncDirection == options.NodeToARM {
		tags, err := r.applyLabelsToAzureResource(namespacedName, *vm, node, configOptions)
		if err != nil {
			return err
		}
		if tags != nil {
			for key, val := range tags {
				vm.SetTag(key, val)
			}
			if err = vm.Update(r.ctx); err != nil {
				return err
			}
		}
	}

	return nil
}

// return patch with new labels, if any, otherwise return nil for no new labels or an error
func (r *ReconcileNodeLabel) applyTagsToNodes(namespacedName types.NamespacedName, computeResource azrsrc.ComputeResource, node *corev1.Node, configOptions *options.ConfigOptions) ([]byte, error) {
	log := r.Log.WithValues("node-label-operator", namespacedName)

	newLabels := map[string]*string{} // should allow for null JSON values
	for tagName, tagVal := range computeResource.Tags() {
		if !conversion.ValidLabelName(tagName) {
			log.V(0).Info("invalid label name", "tag name", tagName)
			continue
		}
		if !conversion.ValidLabelVal(*tagVal) {
			log.V(0).Info("invalid label value", "tag value", *tagVal)
			continue
		}
		validLabelName := conversion.ConvertTagNameToValidLabelName(tagName, configOptions.LabelPrefix)
		labelVal, ok := node.Labels[validLabelName]
		if !ok {
			// add tag as label
			log.V(1).Info("applying tags to nodes", "tag name", tagName, "tag value", *tagVal)
			newLabels[validLabelName] = tagVal
		} else if labelVal != *tagVal {
			switch configOptions.ConflictPolicy {
			case options.ARMPrecedence:
				// set label anyway
				log.V(1).Info("overriding existing node label with ARM tag", "tag name", tagName, "tag value", tagVal)
				newLabels[validLabelName] = tagVal
			case options.NodePrecedence:
				// do nothing
				log.V(0).Info("name->value conflict found", "node label value", labelVal, "ARM tag value", *tagVal)
			case options.Ignore:
				// raise k8s event
				r.Recorder.Event(node, "Warning", "ConflictingTagLabelValues",
					fmt.Sprintf("ARM tag was not applied to node because a different value for '%s' already exists (%s != %s).", tagName, *tagVal, labelVal))
				log.V(0).Info("name->value conflict found, leaving unchanged", "label value", labelVal, "tag value", *tagVal)
			default:
				return nil, errors.New("unrecognized conflict policy")
			}
		}
	}

	// delete labels if tag has been deleted
	// if conflict policy is node precedence (which it will most likely not be), then don't delete tags if they exist on node
	if labelDeletionAllowed(configOptions) {
		for labelFullName, labelVal := range node.Labels {
			if conversion.HasLabelPrefix(labelFullName, configOptions.LabelPrefix) {
				// check if exists on vm/vmss
				labelName := conversion.LabelWithoutPrefix(labelFullName, configOptions.LabelPrefix)
				_, ok := computeResource.Tags()[labelName]
				if !ok { // if label doesn't exist on ARM resource, delete
					log.V(1).Info("deleting label from node", "label name", labelFullName, "label value", labelVal)
					delete(node.Labels, labelFullName) // for some reason this is needed
					newLabels[labelFullName] = nil     // this should becomes 'null' in JSON, necessary for merge patch
				}
			}
		}
	}

	if len(newLabels) == 0 { // to avoid unnecessary patching
		return nil, nil
	}

	patch, err := LabelPatchWithDelete(newLabels)
	if err != nil {
		return nil, err
	}

	return patch, nil
}

func (r *ReconcileNodeLabel) applyLabelsToAzureResource(namespacedName types.NamespacedName, computeResource azrsrc.ComputeResource, node *corev1.Node, configOptions *options.ConfigOptions) (map[string]*string, error) {
	log := r.Log.WithValues("node-label-operator", namespacedName)

	if len(computeResource.Tags()) >= conversion.MaxNumTags {
		log.V(0).Info("can't add any more tags", "number of tags", len(computeResource.Tags()))
		return computeResource.Tags(), nil
	}

	newTags := map[string]*string{}
	for labelName, labelVal := range node.Labels {
		if !conversion.ValidTagName(labelName, configOptions.LabelPrefix) {
			log.V(2).Info("invalid tag name", "label name", labelName)
			continue
		}
		if !conversion.ValidTagVal(labelVal) {
			log.V(2).Info("invalid tag name", "label name", labelName)
			continue
		}
		validTagName := conversion.ConvertLabelNameToValidTagName(labelName, configOptions.LabelPrefix)
		tagVal, ok := computeResource.Tags()[validTagName]
		if !ok {
			// add label as tag
			log.V(1).Info("applying labels to Azure resource", "label name", labelName, "label value", labelVal)
			newTags[validTagName] = to.StringPtr(labelVal)
		} else if *tagVal != labelVal {
			switch configOptions.ConflictPolicy {
			case options.NodePrecedence:
				// set tag anyway
				log.V(1).Info("overriding existing ARM tag with node label", "label name", labelName, "label value", labelVal)
				newTags[validTagName] = to.StringPtr(labelVal)
			case options.ARMPrecedence:
				// do nothing
				log.V(0).Info("name->value conflict found", "node label value", labelVal, "ARM tag value", *tagVal)
			case options.Ignore:
				// raise k8s event
				r.Recorder.Event(node, "Warning", "ConflictingTagLabelValues",
					fmt.Sprintf("node label was not applied to Azure resource because a different value for '%s' already exists (%s != %s).", labelName, labelVal, *tagVal))
				log.V(0).Info("name->value conflict found, leaving unchanged", "label value", labelVal, "tag value", *tagVal)
			default:
				return nil, errors.New("unrecognized conflict policy")
			}
		}
	}

	if len(newTags) == 0 { // if unchanged
		return nil, nil
	}

	return newTags, nil
}

func (r *ReconcileNodeLabel) updateMinSyncPeriodLabels(node *corev1.Node) error {
	r.lastUpdateLabel(node)
	patch, err := LabelPatch(node.Labels)
	if err != nil {
		return err
	}
	if err = r.Patch(r.ctx, node, client.ConstantPatch(types.MergePatchType, patch)); err != nil {
		log.Error(err, "failed to patch lastUpdate label")
		return err
	}
	return nil
}

// update the lastUpdate label on node, or create if not there
func (r *ReconcileNodeLabel) lastUpdateLabel(node *corev1.Node) {
	node.Labels[lastUpdateLabel] = strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", ".")
	node.Labels[minSyncPeriodLabel] = r.MinSyncPeriod.String()
}

func (r *ReconcileNodeLabel) setMinSyncPeriod(duration time.Duration) {
	r.lock.Lock()
	defer r.lock.Unlock()
	r.MinSyncPeriod = duration
}

func (r *ReconcileNodeLabel) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Node{}).
		WithEventFilter(predicate.Funcs{
			UpdateFunc:  updateFunc,
			CreateFunc:  createFunc,
			DeleteFunc:  deleteFunc,
			GenericFunc: genericFunc,
		}).
		Complete(r)
}
