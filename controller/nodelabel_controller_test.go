// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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
	var mapVals1 = map[string]string{"env": vals[0], "dept": vals[1]}
	var mapVals2 = map[string]string{"env": vals[0]}
	var armTagsTest = []struct {
		name                string
		tags                map[string]*string
		labels              map[string]string
		expectedPatchLabels map[string]string
	}{
		{
			"node1", // starting with no labels on node
			labelMapToTagMap(mapVals1),
			map[string]string{},
			map[string]string{labelWithPrefix("env", DefaultLabelPrefix): vals[0], labelWithPrefix("dept", DefaultLabelPrefix): vals[1]},
		},
		{
			"node2",
			labelMapToTagMap(mapVals1),
			map[string]string{"favfruit": "banana"}, // won't be contained in patch though it shouldn't go away
			map[string]string{labelWithPrefix("env", DefaultLabelPrefix): vals[0], labelWithPrefix("dept", DefaultLabelPrefix): vals[1]},
		},
		{
			"node3", // example of deleting a tag
			labelMapToTagMap(mapVals2),
			map[string]string{labelWithPrefix("env", DefaultLabelPrefix): vals[0], labelWithPrefix("dept", DefaultLabelPrefix): vals[1]},
			map[string]string{labelWithPrefix("env", DefaultLabelPrefix): vals[0]},
		},
		// should also have test of changing value of tag that exists
		// have node with labels with different prefixes maybe
	}

	config := DefaultConfigOptions() // tag-to-node only

	r := NewFakeNodeLabelReconciler()
	computeResource := NewFakeComputeResource()

	for _, tt := range armTagsTest {
		t.Run(tt.name, func(t *testing.T) {
			computeResource.tags = tt.tags
			node := newTestNode(tt.name, tt.labels)

			// I should probably check the return value of patch :/
			patch, err := r.applyTagsToNodes(defaultNamespacedName(tt.name), computeResource, node, &config)
			if err != nil {
				t.Errorf("failed to apply tags to nodes: %q", err)
			}

			spec := map[string]interface{}{}
			if err := json.Unmarshal(patch, &spec); err != nil {
				t.Errorf("failed to unmarshal patch data into map")
			}
			metadata, ok := spec["metadata"].(map[string]interface{})
			assert.True(t, ok)
			labels, ok := metadata["labels"].(map[string]interface{})
			assert.True(t, ok)
			assert.Equal(t, len(tt.expectedPatchLabels), len(labels))
			for k := range tt.expectedPatchLabels {
				_, ok := labels[k]
				_, existed := node.Labels[k]
				assert.True(t, (!existed && ok && labels[k] != nil) || (existed && !ok && labels[k] == nil))
				// ideally would check value here
				// if !existed && ok {
				// 	_, ok := vptr.(*string)
				// 	assert.True(t, !ok) // this should be ok though
				// 	assert.Equal(t, v, *vptr)
				// }
			}
		})
	}
}

func TestCorrectLabelsAppliedToAzureResources(t *testing.T) {
	labels1 := map[string]string{"favfruit": "banana", "favveg": "broccoli"}
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

			tags, err := r.applyLabelsToAzureResource(defaultNamespacedName(tt.name), computeResource, node, &config)
			if err != nil {
				t.Errorf("failed to apply labels to azure resources: %q", err)
			}

			for k, expected := range tt.expectedTags {
				actual, ok := tags[k]
				assert.True(t, ok)
				assert.Equal(t, expected, actual)
			}

		})
	}
}

// test helper functions
func TestLastUpdateLabel(t *testing.T) {
	var lastUpdateLabelTest = []struct {
		name          string
		minSyncPeriod time.Duration
		expected      string
	}{
		{
			"node1",
			FiveMinutes,
			FiveMinutes.String(),
		},
		{
			"node2",
			time.Minute,
			time.Minute.String(),
		},
	}

	for _, tt := range lastUpdateLabelTest {
		t.Run(tt.name, func(t *testing.T) {
			reconciler := NewFakeNodeLabelReconciler()
			reconciler.MinSyncPeriod = tt.minSyncPeriod
			node := newTestNode(tt.name, map[string]string{})
			reconciler.lastUpdateLabel(node)
			label, ok := node.Labels[minSyncPeriodLabel]
			assert.True(t, ok)
			assert.Equal(t, label, tt.expected)

		})
	}
}

func TestTimeToUpdate(t *testing.T) {
	var timeToUpdateTest = []struct {
		name     string
		labels   map[string]string
		expected bool
	}{
		{
			"node1",
			map[string]string{},
			true,
		},
		{
			"node2",
			map[string]string{"last-update": "2019-09-23T20.01.43Z", "min-sync-period": "1m"},
			true,
		},
	}

	for _, tt := range timeToUpdateTest {
		t.Run(tt.name, func(t *testing.T) {
			node := newTestNode(tt.name, tt.labels)

			actual := timeToUpdate(node)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// test helper functions	// test helper functions

func NewFakeNodeLabelReconciler() *ReconcileNodeLabel {
	return &ReconcileNodeLabel{
		Client:        ctrlfake.NewFakeClientWithScheme(scheme.Scheme),
		Log:           ctrl.Log.WithName("test"),
		ctx:           context.Background(),
		MinSyncPeriod: FiveMinutes,
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
