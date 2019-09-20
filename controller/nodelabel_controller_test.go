// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type FakeComputeResource struct {
	tags map[string]*string
}

func NewFakeComputeResource() FakeComputeResource {
	return FakeComputeResource{tags: map[string]*string{}}
}

// func (c FakeComputeResource) Get(ctx context.Context, name string) (FakeComputeResource, error) {
// 	return c, nil
// }

func (c FakeComputeResource) Update(ctx context.Context) error {
	return nil
}

func (c FakeComputeResource) Tags() map[string]*string {
	return c.tags
}

func (c FakeComputeResource) SetTag(name string, value *string) {
	c.tags[name] = value
}

// I need a way of creating configurations of vms and nodes that have tags and checking that they are assigned correctly
// ideally without having to be e2e... can I fake all of this somehow? current issue is reconciler object
func TestCorrectTagsAppliedToNodes(t *testing.T) {
	var vals = [2]string{"test", "hr"}
	var armTagsTest = []struct {
		name           string
		tags           map[string]*string
		labels         map[string]string
		expectedLabels map[string]string
	}{
		{
			"node1",
			map[string]*string{"env": &vals[0], "dept": &vals[1]},
			map[string]string{},
			map[string]string{fmt.Sprintf("%s/env", DefaultLabelPrefix): vals[0], fmt.Sprintf("%s/dept", DefaultLabelPrefix): vals[1]},
		},
	}

	config := DefaultConfigOptions() // tag-to-node only

	r := NewFakeNodeLabelReconciler()
	computeResource := NewFakeComputeResource()

	for _, tt := range armTagsTest {
		// do stuff
		t.Run(tt.name, func(t *testing.T) {
			computeResource.tags = tt.tags
			node := newTestNode(tt.name, tt.labels)

			// I should probably check the return value of patch :/
			_, err := r.applyTagsToNodes(defaultNamespacedName(tt.name), computeResource, node, config)
			if err != nil {
				t.Errorf("failed to apply tags to nodes: %q", err)
			}

			for k, v := range tt.expectedLabels {
				val, ok := node.Labels[k]
				assert.True(t, ok)
				assert.Equal(t, v, val)
			}
		})
	}
}

func TestCorrectLabelsAppliedToAzureResources(t *testing.T) {
	labels1 := map[string]string{"favfruit": "banana", "favveg": "broccoli"}
	tags1 := map[string]*string{}
	for key, val := range labels1 {
		tags1[key] = &val
	}
	var nodeLabelsTest = []struct {
		name         string
		labels       map[string]string
		tags         map[string]*string
		expectedTags map[string]*string
	}{
		{
			"resource1",
			labels1,
			map[string]*string{},
			labelMapToTagMap(labels1),
		},
	}

	config := DefaultConfigOptions()
	config.SyncDirection = NodeToARM
	config.ConflictPolicy = NodePrecedence
	r := NewFakeNodeLabelReconciler()
	computeResource := NewFakeComputeResource()

	// create a fake ComputeResource and fake Node for each test and use those I guess
	for _, tt := range nodeLabelsTest {
		t.Run(tt.name, func(t *testing.T) {
			computeResource.tags = tt.tags
			node := newTestNode(tt.name, tt.labels)

			tags, err := r.applyLabelsToAzureResource(defaultNamespacedName(tt.name), computeResource, node, config)
			if err != nil {
				t.Errorf("failed to apply labels to azure resources: %q", err)
			}

			for k, expectedPtr := range tt.expectedTags {
				// why is it always broccoli???
				actualPtr, ok := tags[k]
				assert.True(t, ok)
				fmt.Println(k, *expectedPtr, *actualPtr)
				expected := *expectedPtr
				actual := *actualPtr
				assert.Equal(t, expected, actual)
			}

		})
	}
}

// test helper functions

func NewFakeNodeLabelReconciler() *ReconcileNodeLabel {
	return &ReconcileNodeLabel{
		Client: ctrlfake.NewFakeClientWithScheme(scheme.Scheme),
		Log:    ctrl.Log.WithName("test"),
		ctx:    context.Background(),
	}
}

func NewFakeConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		Data: map[string]string{"syncDirection": "two-way", "labelPrefix": ""},
	}
}

func newTestNode(name string, labels map[string]string) *corev1.Node {
	node := &corev1.Node{}
	node.Name = name
	if labels != nil {
		node.Labels = labels
	}
	return node
}

func defaultNamespacedName(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: "default"}
}

func labelMapToTagMap(labels map[string]string) map[string]*string {
	tags := map[string]*string{}
	for key, val := range labels {
		tags[key] = &val
	}
	return tags
}

// test authentication?
// test config stuff?
