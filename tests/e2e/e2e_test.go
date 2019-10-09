// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package tests

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/node-label-operator/azure"
	azrsrc "github.com/Azure/node-label-operator/azure/computeresource"
	"github.com/Azure/node-label-operator/labelsync"
	"github.com/Azure/node-label-operator/labelsync/naming"
	"github.com/Azure/node-label-operator/labelsync/options"
)

func Test(t *testing.T) {
	c := &Cluster{}
	c.KubeConfig = os.Getenv("KUBECONFIG_OUT")
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(c.KubeConfig), &config)
	require.NoError(t, err)
	suite.Run(t, &TestSuite{Cluster: c})
}

func (s *TestSuite) TestARMTagToNodeLabel_DefaultSettings() {
	g := gomega.NewGomegaWithT(s.T())

	tags := map[string]*string{
		"fruit1": to.StringPtr("watermelon"),
		"fruit2": to.StringPtr("dragonfruit"),
		"fruit3": to.StringPtr("banana"),
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	s.CleanupAzComputeResource(computeResource, tags, numStartingTags)
	g.Eventually(func() bool {
		err := s.CheckTagLabelsDeletedFromNodes(tags, configOptions, numStartingLabels)
		return err == nil
	}, time.Minute, 5*time.Second)
}

func (s *TestSuite) TestNodeLabelToARMTag() {
	g := gomega.NewGomegaWithT(s.T())
	assert := assert.New(s.T())
	require := require.New(s.T())

	labels := map[string]string{
		"veg1": "zucchini",
		"veg2": "swiss-chard",
		"veg3": "jalapeno",
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.NodeToARM
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	g.Eventually(func() bool {
		err := s.CheckAzComputeResourceTagsForLabels(computeResource, labels, numStartingTags)
		return err == nil
	}, time.Minute, 5*time.Second)

	// delete node labels first b/c if tags are deleted first, they will just come back
	s.CleanupNodes(computeResourceNodes, labels)
	for _, node := range computeResourceNodes {
		assert.Equal(numStartingLabels[node.Name], len(node.Labels)) // might not be true yet?
	}
	s.T().Logf("Deleted test labels on nodes: %s", computeResource.Name())

	// clean up compute resource by deleting tags, because currently operator doesn't do that for label->tag sync
	for key := range labels {
		delete(computeResource.Tags(), key)
	}
	err := computeResource.Update(context.Background())
	require.NoError(err)
	assert.Equal(numStartingTags, len(computeResource.Tags()))
}

func (s *TestSuite) TestTwoWaySync() {
	g := gomega.NewGomegaWithT(s.T())
	assert := assert.New(s.T())
	require := require.New(s.T())

	tags := map[string]*string{
		"favveg":    to.StringPtr("broccoli"),
		"favanimal": to.StringPtr("gopher"),
	}

	labels := map[string]string{
		"favfruit": "banana",
		"favfungi": "shiitake_mushroom",
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.TwoWay
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	g.Eventually(func() bool {
		err1 := s.CheckAzComputeResourceTagsForLabels(computeResource, labels, numStartingTags)
		err2 := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels, configOptions)
		return err1 == nil && err2 == nil
	}, time.Minute, 5*time.Second)

	// reset configmap first so that tags and labels won't automatically come back?

	// clean up vmss by deleting tags, which should also delete off of nodes
	computeResource = s.CleanupAzComputeResource(computeResource, tags, numStartingTags)

	// clean up nodes by deleting labels
	s.CleanupNodes(computeResourceNodes, labels)
	g.Eventually(func() bool {
		for _, node := range computeResourceNodes {
			assert.Equal(numStartingLabels[node.Name], len(node.Labels)) // might not be true yet?
			if numStartingLabels[node.Name] != len(node.Labels) {
				return false
			}
		}
		return true
	}, 30*time.Second, 5*time.Second)
	s.T().Logf("Deleted test labels on nodes: %s", computeResource.Name())

	// still need to remove labels from azure resource
	for key := range labels {
		delete(computeResource.Tags(), key)
	}
	err := computeResource.Update(context.Background())
	require.NoError(err)
	assert.Equal(numStartingTags, len(computeResource.Tags()))

	// check that tags and labels got deleted off each other
	for key := range computeResource.Tags() {
		// assert not in tags
		_, ok := labels[key]
		assert.False(ok)
	}
	for _, node := range computeResourceNodes {
		// needs to be key without prefix
		for key := range node.Labels {
			_, ok := tags[naming.LabelWithoutPrefix(key, options.DefaultLabelPrefix)]
			assert.False(ok)
		}
	}
}

func (s *TestSuite) TestARMTagToNodeLabel_InvalidLabels() {
	g := gomega.NewGomegaWithT(s.T())

	tags := map[string]*string{
		"veg4":      to.StringPtr("broccoli"),
		"veg5":      to.StringPtr("brussels sprouts"),   // invalid label value
		"orchstrtr": to.StringPtr("Kubernetes:1.13.10"), // invalid label value
	}
	// tags that are valid labels
	validTags := map[string]*string{
		"veg4": to.StringPtr("broccoli"),
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, validTags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	s.CleanupAzComputeResource(computeResource, tags, numStartingTags)
	g.Eventually(func() bool {
		err := s.CheckTagLabelsDeletedFromNodes(validTags, configOptions, numStartingLabels)
		return err == nil
	}, time.Minute, 5*time.Second)
}

func (s *TestSuite) TestNodeLabelToARMTagInvalidTags() {
	g := gomega.NewGomegaWithT(s.T())
	assert := assert.New(s.T())
	require := require.New(s.T())

	labels := map[string]string{
		"last-update": "200_BCE",
		"k8s/role":    "master", // invalid tag name
	}
	// labels that are valid tags
	validLabels := map[string]string{
		"last-update": "200_BCE",
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.NodeToARM
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	g.Eventually(func() bool {
		err := s.CheckAzComputeResourceTagsForLabels(computeResource, validLabels, numStartingTags)
		return err == nil
	}, time.Minute, 5*time.Second)

	// delete node labels first b/c if tags are deleted first, they will just come back
	s.CleanupNodes(computeResourceNodes, labels)
	for _, node := range computeResourceNodes {
		assert.Equal(numStartingLabels[node.Name], len(node.Labels)) // might not be true yet?
	}
	s.T().Logf("Deleted test labels on nodes: %s", computeResource.Name())

	// clean up compute resource by deleting tags, because currently operator doesn't do that for label->tag sync
	for key := range validLabels {
		delete(computeResource.Tags(), key)
	}
	err := computeResource.Update(context.Background())
	require.NoError(err)
	assert.Equal(numStartingTags, len(computeResource.Tags()))

}

func (s *TestSuite) TestARMTagToNodeLabel_ConflictPolicyARMPrecedence() {
	g := gomega.NewGomegaWithT(s.T())

	startingTags := map[string]*string{
		"a":          to.StringPtr("b"),
		"best-coast": to.StringPtr("west"),
	}

	tags := map[string]*string{
		"best-coast": to.StringPtr("east"),
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	configOptions.ConflictPolicy = options.ARMPrecedence
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	// update Azure compute resource first with original values
	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, startingTags)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, startingTags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	numUntouchedLabels := map[string]int{}
	for _, node := range computeResourceNodes {
		numUntouchedLabels[node.Name] = numStartingLabels[node.Name] + 1 // include whatever other tags were added
	}
	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	// wait for labels to update to new values, with arm tag value overriding
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numUntouchedLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	s.CleanupAzComputeResource(computeResource, startingTags, numStartingTags)
	g.Eventually(func() bool {
		err := s.CheckTagLabelsDeletedFromNodes(startingTags, configOptions, numStartingLabels)
		return err == nil
	}, time.Minute, 5*time.Second)
}

// Assumption is that node precedence is being used with NodeToARM, otherwise
// might as well use Ignore (same effect for NodeToARM but Ignore creates event)
func (s *TestSuite) TestConflictPolicyNodePrecedence() {
	g := gomega.NewGomegaWithT(s.T())

	assert := assert.New(s.T())
	require := require.New(s.T())

	startingLabels := map[string]string{
		"a":          "b",
		"best-coast": "west",
	}

	labels := map[string]string{
		"best-coast": "east",
	}

	expectedLabels := map[string]*string{
		"a":          to.StringPtr("b"),
		"best-coast": to.StringPtr("east"),
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.NodeToARM
	configOptions.ConflictPolicy = options.NodePrecedence
	s.UpdateConfigOptions(configOptions)

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	s.UpdateLabelsOnNodes(computeResourceNodes, startingLabels)
	g.Eventually(func() bool {
		// check that compute resource has accurate labels
		err := s.CheckAzComputeResourceTagsForLabels(computeResource, startingLabels, numStartingTags)
		return err == nil
	}, time.Minute, 5*time.Second)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	g.Eventually(func() bool {
		err1 := s.CheckNodeLabelsForTags(computeResourceNodes, expectedLabels, numStartingLabels, configOptions)
		err2 := s.CheckAzComputeResourceTagsForLabels(computeResource, labels, numStartingTags+1)
		return err1 == nil && err2 == nil
	}, time.Minute, 5*time.Second)

	// delete node labels first b/c if tags are deleted first, they will just come back
	s.CleanupNodes(computeResourceNodes, labels)
	for _, node := range computeResourceNodes {
		assert.Equal(numStartingLabels[node.Name], len(node.Labels)) // might not be true yet?
	}
	s.T().Logf("Deleted test labels on nodes: %s", computeResource.Name())

	// clean up compute resource by deleting tags, because currently operator doesn't do that for label->tag sync
	for key := range labels {
		delete(computeResource.Tags(), key)
	}
	err := computeResource.Update(context.Background())
	require.NoError(err)
	assert.Equal(numStartingTags, len(computeResource.Tags()))

	configOptions = s.GetConfigOptions()
	configOptions.ConflictPolicy = options.ARMPrecedence
	s.UpdateConfigOptions(configOptions)
}

func (s *TestSuite) TestARMTagToNodeLabel_ConflictPolicyIgnore() {
	g := gomega.NewGomegaWithT(s.T())

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode // should be similar results either way
	configOptions.ConflictPolicy = options.Ignore
	s.UpdateConfigOptions(configOptions)

	startingTags := map[string]*string{
		"a":          to.StringPtr("b"),
		"best-coast": to.StringPtr("west"),
	}

	tags := map[string]*string{
		"best-coast": to.StringPtr("east"),
	}

	// labels expected to become tags
	expectedLabels := map[string]string{
		"a":          "b",
		"best-coast": "east",
	}

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, startingTags)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, startingTags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	numCurrentLabels := s.GetNumLabelsPerNode(nodeList)
	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	g.Eventually(func() bool {
		err1 := s.CheckNodeLabelsForTags(computeResourceNodes, startingTags, numCurrentLabels, configOptions) // node labels shouldn't have changed
		err2 := s.CheckAzComputeResourceTagsForLabels(computeResource, expectedLabels, numStartingTags)       // should be the new tags
		return err1 == nil && err2 == nil
	}, time.Minute, 5*time.Second)

	s.CleanupAzComputeResource(computeResource, startingTags, numStartingTags)
	g.Eventually(func() bool {
		err := s.CheckTagLabelsDeletedFromNodes(startingTags, configOptions, numStartingLabels)
		return err == nil
	}, time.Minute, 5*time.Second)

	configOptions = s.GetConfigOptions()
	configOptions.ConflictPolicy = options.ARMPrecedence
	s.UpdateConfigOptions(configOptions)
}

func (s *TestSuite) TestARMTagToNodeLabel_ResourceGroupFilter() {
	g := gomega.NewGomegaWithT(s.T())

	tags := map[string]*string{
		"month": to.StringPtr("october"),
	}

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	// update resource group filter to specify resource group that nodes aren't in
	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	configOptions.ResourceGroupFilter = "non-existent-rg"
	s.UpdateConfigOptions(configOptions)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	g.Eventually(func() bool {
		// check that nodes don't have labels, technically not deleted
		err := s.CheckTagLabelsDeletedFromNodes(tags, configOptions, numStartingLabels)
		return err == nil
	}, time.Minute, 5*time.Second)

	configOptions = s.GetConfigOptions()
	configOptions.ResourceGroupFilter = s.ResourceGroup
	s.UpdateConfigOptions(configOptions)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	g.Eventually(func() bool {
		err1 := s.CleanupAzComputeResource(computeResource, tags, numStartingTags)
		// check that corresponding labels were deleted
		err2 := s.CheckTagLabelsDeletedFromNodes(tags, configOptions, numStartingLabels)
		return err1 == nil && err2 == nil
	}, time.Minute, 5*time.Second)

	configOptions = s.GetConfigOptions()
	configOptions.ResourceGroupFilter = options.DefaultResourceGroupFilter
	s.UpdateConfigOptions(configOptions)
}

// if label prefix is changed, there will still be all of the old labels. should this be dealt with in the operator?
// normally someone won't be changing the prefix so it shouldn't be much of an issue
func (s *TestSuite) TestARMTagToNodeLabel_CustomLabelPrefix() {
	g := gomega.NewGomegaWithT(s.T())

	tags := map[string]*string{
		"tree1": to.StringPtr("birch"),
		"tree2": to.StringPtr("maple"),
		"tree3": to.StringPtr("fir"),
	}

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	customPrefix := "cloudprovider.tags"
	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	configOptions.LabelPrefix = customPrefix
	s.UpdateConfigOptions(configOptions) // more labels are going to be added

	// delete labels with "azure.tags" prefix
	s.DeleteLabelsWithPrefix(options.DefaultLabelPrefix)
	g.Eventually(func() bool {
		nodeList = s.GetNodes()
		for _, node := range nodeList.Items { // checking there are the same amount of tags as started with
			if numStartingLabels[node.Name] != len(node.Labels) {
				return false
			}
		}
		return true
	}, time.Minute, 5*time.Second)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	s.CleanupAzComputeResource(computeResource, tags, numStartingTags)
	g.Eventually(func() bool {
		// check that corresponding labels were deleted
		err := s.client.List(context.Background(), nodeList)
		if err != nil {
			return false
		}
		for key := range tags {
			validLabelName := naming.ConvertTagNameToValidLabelName(key, configOptions.LabelPrefix)
			for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
				_, ok := node.Labels[validLabelName]
				if ok { // check that tag was deleted
					return false
				}
			}
		}
		return true
	}, time.Minute, 5*time.Second)

	configOptions = s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	configOptions.LabelPrefix = options.DefaultLabelPrefix
	s.UpdateConfigOptions(configOptions) // tags with 'azure.tags' prefix should come back

	s.DeleteLabelsWithPrefix(customPrefix)
	g.Eventually(func() bool {
		nodeList = s.GetNodes()
		for _, node := range nodeList.Items { // checking to see if original labels are there
			if numStartingLabels[node.Name] != len(node.Labels) {
				return false
			}
		}
		return true
	}, time.Minute, 5*time.Second)
}

// will be named TestARMTagToNodeLabel_EmptyLabelPrefix
func (s *TestSuite) TestEmptyLabelPrefix() {
	g := gomega.NewGomegaWithT(s.T())
	assert := assert.New(s.T())
	require := require.New(s.T())

	tags := map[string]*string{
		"flower1": to.StringPtr("daisy"),
		"flower2": to.StringPtr("sunflower"),
		"flower3": to.StringPtr("orchid"),
	}

	// get current tags so I can make sure to delete them later
	computeResource := s.NewAzComputeResourceClient()
	startingTags := computeResource.Tags()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	configOptions.LabelPrefix = ""
	s.UpdateConfigOptions(configOptions)

	// delete labels with "azure.tags" prefix
	s.DeleteLabelsWithPrefix(options.DefaultLabelPrefix)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	g.Eventually(func() bool {
		err := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	s.CleanupAzComputeResource(computeResource, tags, numStartingTags)
	g.Eventually(func() bool {
		// check that corresponding labels were not deleted b/c no label prefix
		err := s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels, configOptions)
		return err == nil
	}, time.Minute, 5*time.Second)

	// delete corresponding tag labels
	err := s.client.List(context.Background(), nodeList)
	require.NoError(err)
	computeResourceNodes = s.GetNodesOnAzComputeResource(computeResource, nodeList)
	for _, node := range computeResourceNodes {
		newLabels := map[string]*string{}
		for key := range node.Labels {
			if _, ok := tags[key]; ok {
				delete(node.Labels, key)
				newLabels[key] = nil
			}
		}
		patch, err := labelsync.LabelPatchWithDelete(newLabels)
		require.NoError(err)
		err = s.client.Patch(context.Background(), &node, client.ConstantPatch(types.MergePatchType, patch))
		require.NoError(err)
	}

	// check that corresponding labels were deleted
	err = s.client.List(context.Background(), nodeList)
	require.NoError(err)
	for key := range tags {
		// validLabelName should be same as key
		validLabelName := naming.ConvertTagNameToValidLabelName(key, configOptions.LabelPrefix)
		for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
			_, ok := node.Labels[validLabelName]
			assert.False(ok) // check that tag was deleted
		}
	}

	configOptions = s.GetConfigOptions()
	configOptions.SyncDirection = options.ARMToNode
	configOptions.LabelPrefix = options.DefaultLabelPrefix
	s.UpdateConfigOptions(configOptions)

	// delete labels from pre-existing tags
	err = s.client.List(context.Background(), nodeList)
	require.NoError(err)
	for _, node := range nodeList.Items {
		newLabels := map[string]*string{}
		for key := range startingTags {
			if _, ok := node.Labels[key]; ok {
				delete(node.Labels, key)
				newLabels[key] = nil
			}
		}
		patch, err := labelsync.LabelPatchWithDelete(newLabels)
		require.NoError(err)
		err = s.client.Patch(context.Background(), &node, client.ConstantPatch(types.MergePatchType, patch))
		require.NoError(err)
		// checking to see if original labels are there
		err = s.client.Get(context.Background(), types.NamespacedName{Name: node.Name, Namespace: node.Namespace}, &node)
		require.NoError(err)
		assert.Equal(numStartingLabels[node.Name], len(node.Labels))
	}

}

// will be named TestNodeLabelToARMTag_TooManyTags
func (s *TestSuite) TestTooManyTags() {
	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = options.NodeToARM
	s.UpdateConfigOptions(configOptions) // make sure tags have time to update

	// > maxNumTags labels
}

// Helper functions

func (s *TestSuite) NewAzComputeResourceClient() azrsrc.ComputeResource {
	if s.ResourceType == azrsrc.VMSS {
		return s.NewVMSS()
	}
	return s.NewVM()
}

func (s *TestSuite) NewVMSS() azrsrc.VirtualMachineScaleSet {
	assert := assert.New(s.T())
	require := require.New(s.T())

	vmssClient, err := azure.NewScaleSetClient(s.SubscriptionID) // I should check resource type here
	require.NoError(err)
	vmssList, err := vmssClient.List(context.Background(), s.ResourceGroup)
	if err != nil {
		s.T().Logf("Failed listing vmss in resource group %s: %q", s.ResourceGroup, err)
	}
	require.NoError(err)
	assert.NotEqual(0, len(vmssList.Values()))
	vmss := vmssList.Values()[0]
	vmss = azrsrc.VMSSUserAssignedIdentity(vmss)
	s.T().Logf("Successfully found %d vmss: using %s", len(vmssList.Values()), *vmss.Name)
	return *azrsrc.NewVMSSInitialized(context.Background(), s.ResourceGroup, &vmssClient, &vmss)
}

func (s *TestSuite) NewVM() azrsrc.VirtualMachine {
	assert := assert.New(s.T())
	require := require.New(s.T())

	assert.True(s.ResourceType == azrsrc.VM)
	vmClient, err := azure.NewVMClient(s.SubscriptionID)
	require.NoError(err)
	vmList, err := vmClient.List(context.Background(), s.ResourceGroup)
	if err != nil {
		s.T().Logf("Failed listing vms in resource group %s: %q", s.ResourceGroup, err)
	}
	require.NoError(err)
	assert.NotEqual(0, len(vmList.Values()))
	vm := vmList.Values()[0]
	vm = azrsrc.VMUserAssignedIdentity(vm)
	s.T().Logf("Successfully found %d vms: using %s", len(vmList.Values()), *vm.Name)
	return *azrsrc.NewVMInitialized(context.Background(), s.ResourceGroup, &vmClient, &vm)
}

func (s *TestSuite) GetConfigOptions() *options.ConfigOptions {
	var configMap corev1.ConfigMap
	optionsNamespacedName := options.OptionsConfigMapNamespacedName()
	err := s.client.Get(context.Background(), optionsNamespacedName, &configMap)
	require.NoError(s.T(), err)
	configOptions, err := options.NewConfigOptions(configMap)
	require.NoError(s.T(), err)

	return configOptions
}

func (s *TestSuite) UpdateConfigOptions(configOptions *options.ConfigOptions) {
	configMap, err := options.GetConfigMapFromConfigOptions(configOptions)
	require.NoError(s.T(), err)
	err = s.client.Update(context.Background(), &configMap)
	require.NoError(s.T(), err)

	updatedConfigOptions := s.GetConfigOptions()
	require.Equal(s.T(), configOptions.SyncDirection, updatedConfigOptions.SyncDirection)
	require.Equal(s.T(), configOptions.LabelPrefix, updatedConfigOptions.LabelPrefix)
	require.Equal(s.T(), configOptions.ConflictPolicy, updatedConfigOptions.ConflictPolicy)
	require.Equal(s.T(), configOptions.ResourceGroupFilter, updatedConfigOptions.ResourceGroupFilter)
	s.T().Logf("Config options - syncDirection: %s, conflictPolicy: %s, minSyncPeriod: %s",
		configOptions.SyncDirection, configOptions.ConflictPolicy, configOptions.MinSyncPeriod)
}

func (s *TestSuite) GetNodes() *corev1.NodeList {
	nodeList := &corev1.NodeList{}
	err := s.client.List(context.Background(), nodeList)
	if err != nil {
		s.T().Logf("Failed listing nodes: %s", err)
	}
	require.NoError(s.T(), err)
	// pass the expected number of nodes and check it here?
	assert.NotEqual(s.T(), 0, len(nodeList.Items))
	s.T().Logf("Successfully found %d nodes", len(nodeList.Items))

	return nodeList
}

func (s *TestSuite) GetNumLabelsPerNode(nodeList *corev1.NodeList) map[string]int {
	numLabels := map[string]int{}
	for _, node := range nodeList.Items {
		numLabels[node.Name] = len(node.Labels)
	}
	return numLabels
}

func (s *TestSuite) GetNodesOnAzComputeResource(computeResource azrsrc.ComputeResource, nodeList *corev1.NodeList) []corev1.Node {
	computeResourceNodes := []corev1.Node{}
	for _, node := range nodeList.Items {
		provider, err := azure.ParseProviderID(node.Spec.ProviderID)
		require.NoError(s.T(), err)
		resource, err := azure.ParseProviderID(computeResource.ID())
		require.NoError(s.T(), err)
		if provider.ResourceType == resource.ResourceType && provider.ResourceName == resource.ResourceName {
			computeResourceNodes = append(computeResourceNodes, node)
		}
	}
	assert.NotEqual(s.T(), 0, len(computeResourceNodes))
	s.T().Logf("Found %d nodes on Azure compute resource %s", len(computeResourceNodes), computeResource.Name())

	return computeResourceNodes
}

func (s *TestSuite) UpdateTagsOnAzComputeResource(computeResource azrsrc.ComputeResource, tags map[string]*string) azrsrc.ComputeResource {
	for tag, val := range tags {
		computeResource.Tags()[tag] = val
	}
	err := computeResource.Update(context.Background())
	require.NoError(s.T(), err)
	// check that computeResource tags have been updated
	for key, val := range tags {
		result, ok := computeResource.Tags()[key]
		assert.True(s.T(), ok)
		assert.Equal(s.T(), *val, *result)
	}
	s.T().Logf("Updated tags on Azure compute resource %s", computeResource.Name())

	return computeResource
}

func (s *TestSuite) UpdateLabelsOnNodes(nodes []corev1.Node, labels map[string]string) []corev1.Node {
	for _, node := range nodes {
		for key, val := range labels {
			node.Labels[key] = val
		}
		// should this be a patch instead?
		err := s.client.Update(context.Background(), &node)
		require.NoError(s.T(), err)
	}
	// check that labels have been updated
	updatedNodes := []corev1.Node{}
	for _, node := range nodes {
		updatedNode := &corev1.Node{}
		err := s.client.Get(context.Background(), types.NamespacedName{Name: node.Name, Namespace: node.Namespace}, updatedNode)
		require.NoError(s.T(), err)
		for key, val := range labels {
			v, ok := node.Labels[key]
			assert.True(s.T(), ok)
			assert.Equal(s.T(), val, v)
		}
		updatedNodes = append(updatedNodes, *updatedNode)
	}
	s.T().Logf("Updated node labels")

	return updatedNodes
}

func (s *TestSuite) CheckNodeLabelsForTags(nodes []corev1.Node, tags map[string]*string, numStartingLabels map[string]int, configOptions *options.ConfigOptions) error {
	s.T().Logf("Checking nodes for accurate labels")
	numErrs := 0
	for _, node := range nodes {
		updatedNode := &corev1.Node{}
		if err := s.client.Get(context.Background(), types.NamespacedName{Name: node.Name, Namespace: node.Namespace}, updatedNode); err != nil {
			return err
		}
		if len(tags) != len(updatedNode.Labels)-numStartingLabels[updatedNode.Name] {
			return fmt.Errorf("len(tags) != len(updatedNode.Labels)-numStartingLabels[updatedNode.Name] (%d != %d)", len(tags), len(updatedNode.Labels)-numStartingLabels[updatedNode.Name])
		}
		for key, val := range tags {
			validLabelName := naming.ConvertTagNameToValidLabelName(key, configOptions.LabelPrefix) // make sure this is config options I use
			result, ok := updatedNode.Labels[validLabelName]
			// assert.True(s.T(), ok)
			if !ok {
				s.T().Logf("expected node %s to have label %s", updatedNode.Name, validLabelName)
				numErrs += 1
				continue
			}
			// assert.Equal(s.T(), *val, result)
			if *val != result {
				s.T().Logf("expected node %s to have key/value pair %s=%s", updatedNode.Name, validLabelName, *val)
				numErrs += 1
			}
		}
	}
	if numErrs > 0 {
		return fmt.Errorf("labels did not match tags: %d mismatches", numErrs)
	}
	return nil
}

func (s *TestSuite) CheckAzComputeResourceTagsForLabels(computeResource azrsrc.ComputeResource, labels map[string]string, numStartingTags int) error {
	s.T().Logf("Checking Azure compute resource for accurate labels")
	if len(labels) != len(computeResource.Tags())-numStartingTags {
		return fmt.Errorf("len(labels) != len(computeResource.Tags())-numStartingTags (%d != %d)", len(labels), len(computeResource.Tags())-numStartingTags)
	}
	numErrs := 0
	for key, val := range labels {
		v, ok := computeResource.Tags()[key]
		if !ok {
			s.T().Logf("expected Azure compute resource %s to have label %s", computeResource.Name(), key)
			numErrs += 1
			continue
		}
		if val != *v {
			s.T().Logf("expected Azure compute resource %s to have key/value pair %s=%s", computeResource.Name(), key, *v)
			numErrs += 1
		}
	}
	if numErrs > 0 {
		return fmt.Errorf("tags did not match labels: %d mismatches", numErrs)
	}
	return nil
}

func (s *TestSuite) CleanupAzComputeResource(computeResource azrsrc.ComputeResource, tags map[string]*string, numStartingTags int) azrsrc.ComputeResource {
	for key := range tags {
		delete(computeResource.Tags(), key)
	}
	err := computeResource.Update(context.Background())
	require.NoError(s.T(), err)
	assert.Equal(s.T(), numStartingTags, len(computeResource.Tags())) // is this always true? two-way sync?
	s.T().Logf("Deleted test tags on Azure compute resource %s", computeResource.Name())

	return computeResource
}

func (s *TestSuite) CleanupNodes(nodes []corev1.Node, labels map[string]string) {
	for _, node := range nodes {
		for key := range labels {
			_, ok := node.Labels[key]
			assert.True(s.T(), ok)
			delete(node.Labels, key)
		}
		err := s.client.Update(context.Background(), &node)
		require.NoError(s.T(), err)
	}
}

func (s *TestSuite) CheckTagLabelsDeletedFromNodes(tags map[string]*string, configOptions *options.ConfigOptions, numStartingLabels map[string]int) error {
	nodeList := &corev1.NodeList{}
	if err := s.client.List(context.Background(), nodeList); err != nil {
		return err
	}
	numErrs := 0
	for key := range tags {
		validLabelName := naming.ConvertTagNameToValidLabelName(key, configOptions.LabelPrefix)
		for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
			_, ok := node.Labels[validLabelName]
			if ok {
				s.T().Logf("label %s was not deleted from node %s", validLabelName, node.Name)
				numErrs += 1
			}
		}
	}
	for _, node := range nodeList.Items {
		// checking to see if original labels are there.
		if numStartingLabels[node.Name] != len(node.Labels) {
			s.T().Logf("numStartingLabels[node.Name] != len(node.Labels) (%d != %d)", numStartingLabels[node.Name], len(node.Labels))
			numErrs += 1
		}
	}
	if numErrs > 0 {
		return fmt.Errorf("labels corresponding to tags were not properly deleted from nodes")
	}
	return nil
}

func (s *TestSuite) DeleteLabelsWithPrefix(labelPrefix string) {
	nodeList := &corev1.NodeList{}
	err := s.client.List(context.Background(), nodeList)
	require.NoError(s.T(), err)
	for _, node := range nodeList.Items {
		newLabels := map[string]*string{}
		for key := range node.Labels {
			if naming.HasLabelPrefix(key, labelPrefix) {
				delete(node.Labels, key)
				newLabels[key] = nil
			}
		}
		patch, err := labelsync.LabelPatchWithDelete(newLabels)
		require.NoError(s.T(), err)
		err = s.client.Patch(context.Background(), &node, client.ConstantPatch(types.MergePatchType, patch))
		require.NoError(s.T(), err)
	}
}
