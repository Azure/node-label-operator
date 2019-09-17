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
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/fake"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeComputeResource struct {
	tags map[string]*string
}

func NewFakeComputeResource() fakeComputeResource {
	return fakeComputeResource{tags: map[string]*string{}}
}

// func (c fakeComputeResource) Get(ctx context.Context, name string) (azure.Spec, error) {
// 	return fakeSpec{}, nil
// }

func (c fakeComputeResource) Update(ctx context.Context) error {
	return nil
}

func (c fakeComputeResource) Tags() map[string]*string {
	return c.tags
}

func (c fakeComputeResource) SetTag(name string, value *string) {
	c.tags[name] = value
}

func TestValidTagName(t *testing.T) {
	var tagNameTests = []struct {
		given    string
		expected bool
	}{
		{"kubernetes.io/arch", false},
		{"arch", true},
		{"tag?", false},
		{"To thine own self be true, and it must follow, as the night the day, thou canst not then be false to any man.", true},
		{`O, then, I see Queen Mab hath been with you. 
She is the fairies' midwife, and she comes 
In shape no bigger than an agate-stone 
On the fore-finger of an alderman, 
Drawn with a team of little atomies 
Athwart men's noses as they lie asleep; 
Her wagon-spokes made of long spinners' legs, 
The cover of the wings of grasshoppers, 
The traces of the smallest spider's web, 
The collars of the moonshine's watery beams, 
Her whip of cricket's bone, the lash of film, 
Her wagoner a small grey-coated gnat, 
Not so big as a round little worm 
Prick'd from the lazy finger of a maid; 
Her chariot is an empty hazel-nut 
Made by the joiner squirrel or old grub, 
Time out o' mind the fairies' coachmakers. `, false},
	}

	config := DefaultConfigOptions()

	for _, tt := range tagNameTests {
		t.Run(tt.given, func(t *testing.T) {
			valid := ValidTagName(tt.given, config)
			if valid != tt.expected {
				t.Errorf("given tag name %q, got valid=%t, want valid=%t", tt.given, valid, tt.expected)
			}
		})
	}
}

func TestValidLabelName(t *testing.T) {
	var labelNameTests = []struct {
		given    string
		expected bool
	}{
		{"os", true},
		{"_favfruit", false},
		{"favorite_vegetable!!!!", false},
	}

	for _, tt := range labelNameTests {
		t.Run(tt.given, func(t *testing.T) {
			valid := ValidLabelName(tt.given)
			if valid != tt.expected {
				t.Errorf("given label name %q, got valid=%t, want valid=%t", tt.given, valid, tt.expected)
			}
		})
	}
}

func TestConvertTagNameToValidLabelName(t *testing.T) {
	var tagNameConversionTests = []struct {
		given    string
		expected string
	}{
		{"env", fmt.Sprintf("%s/env", DefaultLabelPrefix)},
		{"dept", fmt.Sprintf("%s/dept", DefaultLabelPrefix)},
		{"Good_night_good_night._parting_is_such_sweet_sorrow._That_I_shall_say_good_night_till_it_be_morrow", fmt.Sprintf("%s/Good_night_good_night._parting_is_such_sweet_sorrow._That_I_sha", DefaultLabelPrefix)},
	}

	config := DefaultConfigOptions()

	for _, tt := range tagNameConversionTests {
		t.Run(tt.given, func(t *testing.T) {
			validLabelName := ConvertTagNameToValidLabelName(tt.given, config)
			if validLabelName != tt.expected {
				t.Errorf("given tag name %q, got label name %q, expected label name %q", tt.given, validLabelName, tt.expected)
			}
		})
	}
}

func TestConvertLabelNameToValidTagName(t *testing.T) {
	var labelNameConversionTests = []struct {
		given    string
		expected string
	}{
		{"favfruit", "favfruit"}, // have prefix?
	}

	config := DefaultConfigOptions()

	for _, tt := range labelNameConversionTests {
		t.Run(tt.given, func(t *testing.T) {
			validTagName := ConvertLabelNameToValidTagName(tt.given, config)
			if validTagName != tt.expected {
				t.Errorf("given label name %q, got tag name %q, expected tag name %q", tt.given, validTagName, tt.expected)
			}
		})
	}
}

func TestConvertTagValToValidLabelVal(t *testing.T) {
	var tagValConversionTests = []struct {
		given    string
		expected string
	}{
		{"test", "test"},
	}

	for _, tt := range tagValConversionTests {
		t.Run(tt.given, func(t *testing.T) {
			validLabelVal := ConvertTagValToValidLabelVal(tt.given)
			if validLabelVal != tt.expected {
				t.Errorf("given tag name %q, got label name %q, expected label name %q", tt.given, validLabelVal, tt.expected)
			}
		})
	}
}

func TestConvertLabelValToValidTagVal(t *testing.T) {
	var labelValConversionTests = []struct {
		given    string
		expected string
	}{
		{"test", "test"},
	}

	for _, tt := range labelValConversionTests {
		t.Run(tt.given, func(t *testing.T) {
			validTagVal := ConvertLabelValToValidTagVal(tt.given)
			if validTagVal != tt.expected {
				t.Errorf("given label name %q, got tag name %q, expected tag name %q", tt.given, validTagVal, tt.expected)
			}
		})
	}
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

	// k8sClient := newTestK8sClient()
	// nodes := k8sClient.CoreV1().Nodes()

	// _, err := nodes.Create(newTestNode(name, nil))
	// if err != nil {
	// 	t.Errorf("creating fake node failed: %q", err)
	// }
	// node := nodes.Get("testNode", metav1.GetOptions{})

	r := defaultNodeLabelReconciler()
	computeResource := NewFakeComputeResource()

	for _, tt := range armTagsTest {
		// do stuff
		t.Run(tt.name, func(t *testing.T) {
			computeResource.tags = tt.tags
			node := newTestNode(tt.name, tt.labels)

			// I think r.Update() is causing issues still since I can't actually update...
			// try/catch sort of thing in case the error is from trying to update?
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
	var vals = [2]string{"banana", "broccoli"}
	var nodeLabelsTest = []struct {
		name         string
		labels       map[string]string
		tags         map[string]*string
		expectedTags map[string]*string
	}{
		{
			"resource1",
			map[string]string{"favfruit": vals[0], "favveg": vals[1]},
			map[string]*string{},
			map[string]*string{"favfruit": &vals[0], "favveg": &vals[1]},
		},
	}

	config := DefaultConfigOptions()
	config.SyncDirection = NodeToARM
	config.ConflictPolicy = NodePrecedence
	r := defaultNodeLabelReconciler()
	computeResource := NewFakeComputeResource()

	// create a fake ComputeResource and fake Node for each test and use those I guess
	for _, tt := range nodeLabelsTest {
		t.Run(tt.name, func(t *testing.T) {
			// computeResource.tags = tt.tags // is something weird going on here?
			for k, v := range tt.tags {
				computeResource.tags[k] = v
			}
			node := newTestNode(tt.name, tt.labels)

			_, err := r.applyLabelsToAzureResource(defaultNamespacedName(tt.name), computeResource, node, config)
			if err != nil {
				t.Errorf("failed to apply labels to azure resources: %q", err)
			}

			for k, vptr := range tt.expectedTags {
				// why is it always broccoli???
				valptr, ok := computeResource.Tags()[k]
				fmt.Println(k, *vptr, *valptr)
				assert.True(t, ok)
				fmt.Println(k, *vptr, *valptr)
				v := *vptr
				val := *valptr
				assert.Equal(t, v, val)
			}

		})
	}
}

func newTestK8sClient() kubernetes.Interface {
	return fake.NewSimpleClientset()
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

func defaultNodeLabelReconciler() *ReconcileNodeLabel {
	return &ReconcileNodeLabel{
		Client: ctrlfake.NewFakeClientWithScheme(scheme.Scheme),
		Log:    ctrl.Log.WithName("test"),
		ctx:    context.Background(),
	}
}

// test authentication?
// test config stuff?
