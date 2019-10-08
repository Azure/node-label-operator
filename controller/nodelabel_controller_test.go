// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/Azure/go-autorest/autorest/to"
	azrsrc "github.com/Azure/node-label-operator/azure/computeresource"
)

func TestCorrectTagsAppliedToNodes(t *testing.T) {
	var armTagsTest = []struct {
		name                string
		tags                map[string]*string
		labels              map[string]string
		expectedPatchLabels map[string]string
	}{
		{
			"node1", // starting with no labels on node
			map[string]*string{
				"env": to.StringPtr("test"),
				"v":   to.StringPtr("1"),
			},
			map[string]string{},
			map[string]string{
				LabelWithPrefix("env", DefaultLabelPrefix): "test",
				LabelWithPrefix("v", DefaultLabelPrefix):   "1",
			},
		},
		{
			"node2",
			map[string]*string{
				"env": to.StringPtr("test"),
				"v":   to.StringPtr("1"),
			},
			map[string]string{
				"favfruit": "banana",
			}, // won't be contained in patch though it shouldn't go away
			map[string]string{
				LabelWithPrefix("env", DefaultLabelPrefix): "test",
				LabelWithPrefix("v", DefaultLabelPrefix):   "1",
			},
		},
		{
			"node3", // example of deleting a tag
			map[string]*string{
				"env": to.StringPtr("test"),
			},
			map[string]string{
				LabelWithPrefix("env", DefaultLabelPrefix): "test",
				LabelWithPrefix("v", DefaultLabelPrefix):   "1",
			},
			map[string]string{
				LabelWithPrefix("env", DefaultLabelPrefix): "test",
			},
		},
		{
			"node4", // changing a preexisting tag
			map[string]*string{
				"env": to.StringPtr("test"),
				"v":   to.StringPtr("2"),
			},
			map[string]string{
				LabelWithPrefix("env", DefaultLabelPrefix): "test",
				LabelWithPrefix("v", DefaultLabelPrefix):   "1",
			},
			map[string]string{
				LabelWithPrefix("v", DefaultLabelPrefix): "2",
			},
		},
		{
			"node5", // have node with labels with different prefixes
			map[string]*string{
				"role": to.StringPtr("master"),
			},
			map[string]string{
				LabelWithPrefix("role", "k8s"): "master",
			},
			map[string]string{
				LabelWithPrefix("role", DefaultLabelPrefix): "master",
			},
		},
		{
			"node6", // invalid labels
			map[string]*string{
				"orchestrator": to.StringPtr("Kubernetes:1.18.0"),
				"agentPool":    to.StringPtr("agentpool1"),
			},
			map[string]string{},
			map[string]string{
				LabelWithPrefix("agentPool", DefaultLabelPrefix): "agentpool1",
			},
		},
	}

	config := DefaultConfigOptions() // tag-to-node only
	r := NewFakeNodeLabelReconciler()

	for _, tt := range armTagsTest {
		t.Run(tt.name, func(t *testing.T) {
			computeResource := azrsrc.NewFakeComputeResource(tt.tags)
			node := NewFakeNode(tt.name, tt.labels)

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
			for k, v := range tt.expectedPatchLabels {
				label, ok := labels[k]
				_, existed := node.Labels[k]
				assert.True(t, (ok && labels[k] != nil) || (existed && !ok && labels[k] == nil))
				if ok {
					assert.Equal(t, v, label.(string))
				}
			}
		})
	}
}

func TestCorrectLabelsAppliedToAzureResources(t *testing.T) {
	var nodeLabelsTest = []struct {
		name         string
		labels       map[string]string
		tags         map[string]*string
		expectedTags map[string]*string
	}{
		{
			"resource1",
			map[string]string{
				"favfruit": "banana",
				"favveg":   "broccoli",
			},
			map[string]*string{},
			map[string]*string{
				"favfruit": to.StringPtr("banana"),
				"favveg":   to.StringPtr("broccoli"),
			},
		},
		{
			"resource2", // only include tags that haven't been added yet
			map[string]string{
				"favfruit":  "banana",
				"favveg":    "broccoli",
				"favanimal": "gopher",
			},
			map[string]*string{
				"favanimal": to.StringPtr("gopher"),
			},
			map[string]*string{
				"favfruit": to.StringPtr("banana"),
				"favveg":   to.StringPtr("broccoli"),
			},
		},
		{
			"resource3", // invalid tags
			map[string]string{
				"fruits/favfruit": "banana",
				"favanimal":       "gopher",
			},
			map[string]*string{
				"favveg": to.StringPtr("broccoli"),
			},
			map[string]*string{
				"favanimal": to.StringPtr("gopher"),
			},
		},
	}

	config := DefaultConfigOptions()
	config.SyncDirection = NodeToARM
	config.ConflictPolicy = NodePrecedence
	r := NewFakeNodeLabelReconciler()

	for _, tt := range nodeLabelsTest {
		t.Run(tt.name, func(t *testing.T) {
			node := NewFakeNode(tt.name, tt.labels)
			computeResource := azrsrc.NewFakeComputeResource(tt.tags)

			tags, err := r.applyLabelsToAzureResource(defaultNamespacedName(tt.name), computeResource, node, &config)
			if err != nil {
				t.Errorf("failed to apply labels to azure resources: %q", err)
			}
			assert.True(t, tags != nil) // should only be nil if no changes

			fmt.Println(node.Labels)
			fmt.Println(computeResource.Tags())
			fmt.Println(tags)

			for k, expected := range tt.expectedTags {
				actual, ok := tags[k]
				assert.True(t, ok)
				assert.Equal(t, *expected, *actual)
			}
		})
	}
}

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
			node := NewFakeNode(tt.name, map[string]string{})
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
			map[string]string{"node-label-operator/last-update": "2019-09-23T20.01.43Z", "node-label-operator/min-sync-period": "1m"},
			true,
		},
		{
			"node3",
			map[string]string{"node-label-operator/last-update": strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "."), "node-label-operator/min-sync-period": "100h"},
			false,
		},
	}

	for _, tt := range timeToUpdateTest {
		t.Run(tt.name, func(t *testing.T) {
			node := NewFakeNode(tt.name, tt.labels)

			actual := timeToUpdate(node)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

func TestLabelDeletionAllowed(t *testing.T) {
	var labelDeletionAllowedTest = []struct {
		name          string
		configOptions *ConfigOptions
		expected      bool
	}{
		{
			"test1",
			&ConfigOptions{
				LabelPrefix:    DefaultLabelPrefix,
				ConflictPolicy: ARMPrecedence,
			},
			true,
		},
		{
			"test2",
			&ConfigOptions{
				LabelPrefix:    "cool-custom-label-prefix",
				ConflictPolicy: Ignore,
			},
			true,
		},
	}

	for _, tt := range labelDeletionAllowedTest {
		t.Run(tt.name, func(t *testing.T) {
			actual := labelDeletionAllowed(tt.configOptions)
			assert.Equal(t, tt.expected, actual)
		})
	}
}

// test helper functions

func NewFakeNodeLabelReconciler() *ReconcileNodeLabel {
	return &ReconcileNodeLabel{
		Client:        ctrlfake.NewFakeClientWithScheme(scheme.Scheme),
		Log:           ctrl.Log.WithName("test"),
		ctx:           context.Background(),
		MinSyncPeriod: FiveMinutes,
	}
}

func NewFakeNode(name string, labels map[string]string) *corev1.Node {
	node := &corev1.Node{}
	node.Name = name
	node.Labels = labels
	return node
}

func defaultNamespacedName(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: "default"}
}
