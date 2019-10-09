package labelsync

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"

	azrsrc "github.com/Azure/node-label-operator/azure/computeresource"
	"github.com/Azure/node-label-operator/labelsync/naming"
	"github.com/Azure/node-label-operator/labelsync/options"
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
				naming.LabelWithPrefix("env", options.DefaultLabelPrefix): "test",
				naming.LabelWithPrefix("v", options.DefaultLabelPrefix):   "1",
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
				naming.LabelWithPrefix("env", options.DefaultLabelPrefix): "test",
				naming.LabelWithPrefix("v", options.DefaultLabelPrefix):   "1",
			},
		},
		{
			"node3", // example of deleting a tag
			map[string]*string{
				"env": to.StringPtr("test"),
			},
			map[string]string{
				naming.LabelWithPrefix("env", options.DefaultLabelPrefix): "test",
				naming.LabelWithPrefix("v", options.DefaultLabelPrefix):   "1",
			},
			map[string]string{
				naming.LabelWithPrefix("env", options.DefaultLabelPrefix): "test",
			},
		},
		{
			"node4", // changing a preexisting tag
			map[string]*string{
				"env": to.StringPtr("test"),
				"v":   to.StringPtr("2"),
			},
			map[string]string{
				naming.LabelWithPrefix("env", options.DefaultLabelPrefix): "test",
				naming.LabelWithPrefix("v", options.DefaultLabelPrefix):   "1",
			},
			map[string]string{
				naming.LabelWithPrefix("v", options.DefaultLabelPrefix): "2",
			},
		},
		{
			"node5", // have node with labels with different prefixes
			map[string]*string{
				"role": to.StringPtr("master"),
			},
			map[string]string{
				naming.LabelWithPrefix("role", "k8s"): "master",
			},
			map[string]string{
				naming.LabelWithPrefix("role", options.DefaultLabelPrefix): "master",
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
				naming.LabelWithPrefix("agentPool", options.DefaultLabelPrefix): "agentpool1",
			},
		},
	}

	config := options.DefaultConfigOptions() // tag-to-node only

	for _, tt := range armTagsTest {
		t.Run(tt.name, func(t *testing.T) {
			computeResource := azrsrc.NewFakeComputeResource(tt.tags)
			node := NewFakeNode(tt.name, tt.labels)

			log := ctrl.Log.WithName("node-label-operator-test")
			patch, err := TagsToNodes(defaultNamespacedName(tt.name), computeResource, node, &config, log, record.NewFakeRecorder(0))
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

	config := options.DefaultConfigOptions()
	config.SyncDirection = options.NodeToARM
	config.ConflictPolicy = options.NodePrecedence

	for _, tt := range nodeLabelsTest {
		t.Run(tt.name, func(t *testing.T) {
			node := NewFakeNode(tt.name, tt.labels)
			computeResource := azrsrc.NewFakeComputeResource(tt.tags)

			log := ctrl.Log.WithName("node-label-operator-test")
			tags, err := LabelsToAzureResource(defaultNamespacedName(tt.name), computeResource, node, &config, log, record.NewFakeRecorder(0))
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

func TestLabelDeletionAllowed(t *testing.T) {
	var labelDeletionAllowedTest = []struct {
		name          string
		configOptions *options.ConfigOptions
		expected      bool
	}{
		{
			"test1",
			&options.ConfigOptions{
				LabelPrefix:    options.DefaultLabelPrefix,
				ConflictPolicy: options.ARMPrecedence,
			},
			true,
		},
		{
			"test2",
			&options.ConfigOptions{
				LabelPrefix:    "cool-custom-label-prefix",
				ConflictPolicy: options.Ignore,
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

func NewFakeNode(name string, labels map[string]string) *corev1.Node {
	node := &corev1.Node{}
	node.Name = name
	node.Labels = labels
	return node
}

func defaultNamespacedName(name string) types.NamespacedName {
	return types.NamespacedName{Name: name, Namespace: "default"}
}
