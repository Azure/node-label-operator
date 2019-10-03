// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package tests

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/Azure/node-label-operator/azure"
	"github.com/Azure/node-label-operator/controller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func Test(t *testing.T) {
	c := &Cluster{}
	c.KubeConfig = os.Getenv("KUBECONFIG_OUT")
	var config map[string]interface{}
	err := yaml.Unmarshal([]byte(c.KubeConfig), &config)
	require.NoError(t, err)
	suite.Run(t, &TestSuite{Cluster: c})
}

// aks uses vms and aks-engine uses vm for master and vmss for workers

func (s *TestSuite) TestARMTagToNodeLabel() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	tags := map[string]*string{
		"fruit1": to.StringPtr("watermelon"),
		"fruit2": to.StringPtr("dragonfruit"),
		"fruit3": to.StringPtr("banana"),
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = controller.ARMToNode
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	WaitForReconcile() // wait for labels to update

	s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels)

	s.CleanupAzComputeResource(computeResource, tags, numStartingTags)
	WaitForReconcile() // wait for labels to be removed, assuming minSyncPeriod=1m

	// check that corresponding labels were deleted
	err := s.client.List(context.Background(), nodeList)
	require.NoError(err)
	for key := range tags {
		validLabelName := controller.ConvertTagNameToValidLabelName(key, *configOptions)
		for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
			// check that tag was deleted
			_, ok := node.Labels[validLabelName]
			assert.False(ok)
		}
	}
	for _, node := range nodeList.Items {
		// checking to see if original labels are there.
		assert.Equal(numStartingLabels[node.Name], len(node.Labels))
	}
}

func (s *TestSuite) TestNodeLabelToARMTag() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	labels := map[string]string{
		"veg1": "zucchini",
		"veg2": "swiss-chard",
		"veg3": "jalapeno",
	}

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = controller.NodeToARM
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)

	WaitForReconcile() // wait for tags to update

	// check that compute resource has accurate labels
	s.CheckAzComputeResourceTagsForLabels(computeResource, labels, numStartingTags)

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
	configOptions.SyncDirection = controller.TwoWay
	s.UpdateConfigOptions(configOptions)

	WaitForReconcile()

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	WaitForReconcile()

	s.CheckAzComputeResourceTagsForLabels(computeResource, labels, numStartingTags)

	s.CheckNodeLabelsForTags(computeResourceNodes, tags, numStartingLabels)

	// reset configmap first so that tags and labels won't automatically come back?

	// clean up vmss by deleting tags, which should also delete off of nodes
	computeResource = s.CleanupAzComputeResource(computeResource, tags, numStartingTags)

	// clean up nodes by deleting labels
	s.CleanupNodes(computeResourceNodes, labels)
	WaitForReconcile()

	for _, node := range computeResourceNodes {
		assert.Equal(numStartingLabels[node.Name], len(node.Labels)) // might not be true yet? sleep for 1m?
	}
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
			_, ok := tags[controller.LabelWithoutPrefix(key, controller.DefaultLabelPrefix)]
			assert.False(ok)
		}
	}
}

func (s *TestSuite) TestARMTagToNodeLabelInvalidLabels() {
	assert := assert.New(s.T())
	require := require.New(s.T())

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
	configOptions.SyncDirection = controller.ARMToNode
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	WaitForReconcile() // wait for labels to update

	s.CheckNodeLabelsForTags(computeResourceNodes, validTags, numStartingLabels)

	s.CleanupAzComputeResource(computeResource, tags, numStartingTags)

	WaitForReconcile() // wait for labels to be removed, assuming minSyncPeriod=1m

	// check that corresponding labels were deleted
	err := s.client.List(context.Background(), nodeList)
	require.NoError(err)
	for key := range validTags {
		validLabelName := controller.ConvertTagNameToValidLabelName(key, *configOptions)
		for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
			// check that tag was deleted
			_, ok := node.Labels[validLabelName]
			assert.False(ok) // failing
		}
	}
	for _, node := range nodeList.Items {
		// Checking to see if original labels are there
		assert.Equal(numStartingLabels[node.Name], len(node.Labels))
	}
}

func (s *TestSuite) TestNodeLabelToARMTagInvalidTags() {
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
	configOptions.SyncDirection = controller.NodeToARM
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	WaitForReconcile() // wait for tags to update

	// check that compute resource has accurate labels
	s.CheckAzComputeResourceTagsForLabels(computeResource, validLabels, numStartingTags)

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
	assert := assert.New(s.T())
	require := require.New(s.T())

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = controller.ARMToNode
	configOptions.ConflictPolicy = controller.ARMPrecedence
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	startingTags := map[string]*string{
		"a":          to.StringPtr("b"),
		"best-coast": to.StringPtr("west"),
	}

	tags := map[string]*string{
		"best-coast": to.StringPtr("east"),
	}

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	// update Azure compute resource first with original values
	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, startingTags)
	WaitForReconcile() // wait to update node labels
	s.CheckNodeLabelsForTags(computeResourceNodes, startingTags, numStartingLabels)

	numUntouchedLabels := map[string]int{}
	for _, node := range computeResourceNodes {
		numUntouchedLabels[node.Name] = numStartingLabels[node.Name] + 1 // include whatever other tags were added
	}
	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	// wait for labels to update to new values, with arm tag value overriding
	WaitForReconcile()
	s.CheckNodeLabelsForTags(computeResourceNodes, tags, numUntouchedLabels)

	s.CleanupAzComputeResource(computeResource, startingTags, numStartingTags)
	WaitForReconcile() // wait for labels to be removed, assuming minSyncPeriod=1m
	// check that corresponding labels were deleted
	err := s.client.List(context.Background(), nodeList)
	require.NoError(err)
	for key := range startingTags {
		validLabelName := controller.ConvertTagNameToValidLabelName(key, *configOptions)
		for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
			_, ok := node.Labels[validLabelName]
			assert.False(ok) // check that tag was deleted
		}
	}
	for _, node := range nodeList.Items {
		// checking to see if original labels are there
		assert.Equal(numStartingLabels[node.Name], len(node.Labels))
	}
}

// Assumption is that node precedence is being used with NodeToARM, otherwise
// might as well use Ignore (same effect for NodeToARM but Ignore creates event)
func (s *TestSuite) TestConflictPolicyNodePrecedence() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	configOptions := s.GetConfigOptions()
	// configOptions.SyncDirection = controller.ARMToNode
	configOptions.SyncDirection = controller.NodeToARM
	configOptions.ConflictPolicy = controller.NodePrecedence
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

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

	computeResource := s.NewAzComputeResourceClient()
	nodeList := s.GetNodes()
	numStartingTags := len(computeResource.Tags())
	numStartingLabels := s.GetNumLabelsPerNode(nodeList)
	computeResourceNodes := s.GetNodesOnAzComputeResource(computeResource, nodeList)

	s.UpdateLabelsOnNodes(computeResourceNodes, startingLabels)
	WaitForReconcile() // wait for tags to update

	// check that compute resource has accurate labels
	s.CheckAzComputeResourceTagsForLabels(computeResource, startingLabels, numStartingTags)

	s.UpdateLabelsOnNodes(computeResourceNodes, labels)
	WaitForReconcile() // wait for tags to update

	s.CheckNodeLabelsForTags(computeResourceNodes, expectedLabels, numStartingLabels)
	s.CheckAzComputeResourceTagsForLabels(computeResource, labels, numStartingTags+1)

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

func (s *TestSuite) TestARMTagToNodeLabel_ConflictPolicyIgnore() {
	assert := assert.New(s.T())
	require := require.New(s.T())

	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = controller.ARMToNode // should be similar results either way
	configOptions.ConflictPolicy = controller.Ignore
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

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
	WaitForReconcile()
	s.CheckNodeLabelsForTags(computeResourceNodes, startingTags, numStartingLabels)

	numCurrentLabels := s.GetNumLabelsPerNode(nodeList)
	computeResource = s.UpdateTagsOnAzComputeResource(computeResource, tags)
	WaitForReconcile()
	s.CheckNodeLabelsForTags(computeResourceNodes, startingTags, numCurrentLabels)          // node labels shouldn't have changed
	s.CheckAzComputeResourceTagsForLabels(computeResource, expectedLabels, numStartingTags) // should be the new tags

	// clean up compute resource by deleting tags
	s.CleanupAzComputeResource(computeResource, startingTags, numStartingTags)
	WaitForReconcile() // wait for labels to be removed, assuming minSyncPeriod=1m
	// check that corresponding labels were deleted
	err := s.client.List(context.Background(), nodeList)
	require.NoError(err)
	for key := range tags {
		validLabelName := controller.ConvertTagNameToValidLabelName(key, *configOptions)
		for _, node := range nodeList.Items { // also checking none of nodes on other compute resource were affected
			// check that tag was deleted
			_, ok := node.Labels[validLabelName]
			assert.False(ok)
		}
	}
	for _, node := range nodeList.Items {
		// Checking to see if original labels are there.
		assert.Equal(numStartingLabels[node.Name], len(node.Labels))
	}
}

// will be named TestARMTagToNodeLabelResourceGroupFilter
// how do I get multiple resource groups? aks cluster with multiple node pools?
func (s *TestSuite) TestResourceGroupFilter() {
	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = controller.NodeToARM
	configOptions.ResourceGroupFilter = s.ResourceGroup // reset at end?
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	// I will need to try tagging all VMs, or at least one in each nodepool,
	// and checking that only the one in the chosen resource group was updated
	// but how should I get the other resource groups?
}

// will be named TestNodeLabelToARMTag_TooManyTags
func (s *TestSuite) TestTooManyTags() {
	configOptions := s.GetConfigOptions()
	configOptions.SyncDirection = controller.NodeToARM
	s.UpdateConfigOptions(configOptions)
	WaitForReconcile()

	// > maxNumTags labels
}

// Helper functions

func (s *TestSuite) NewAzComputeResourceClient() controller.ComputeResource {
	if s.ResourceType == controller.VMSS {
		return s.NewVMSS()
	}
	return s.NewVM()
}

func (s *TestSuite) NewVMSS() controller.VirtualMachineScaleSet {
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
	vmss = controller.VMSSUserAssignedIdentity(vmss)
	s.T().Logf("Successfully found %d vmss: using %s", len(vmssList.Values()), *vmss.Name)
	return *controller.NewVMSSInitialized(context.Background(), s.ResourceGroup, &vmssClient, &vmss)
}

func (s *TestSuite) NewVM() controller.VirtualMachine {
	assert := assert.New(s.T())
	require := require.New(s.T())

	assert.True(s.ResourceType == controller.VM)
	vmClient, err := azure.NewVMClient(s.SubscriptionID)
	require.NoError(err)
	vmList, err := vmClient.List(context.Background(), s.ResourceGroup)
	if err != nil {
		s.T().Logf("Failed listing vms in resource group %s: %q", s.ResourceGroup, err)
	}
	require.NoError(err)
	assert.NotEqual(0, len(vmList.Values()))
	vm := vmList.Values()[0]
	vm = controller.VMUserAssignedIdentity(vm)
	s.T().Logf("Successfully found %d vms: using %s", len(vmList.Values()), *vm.Name)
	return *controller.NewVMInitialized(context.Background(), s.ResourceGroup, &vmClient, &vm)
}

func (s *TestSuite) GetConfigOptions() *controller.ConfigOptions {
	var configMap corev1.ConfigMap
	optionsNamespacedName := controller.OptionsConfigMapNamespacedName()
	err := s.client.Get(context.Background(), optionsNamespacedName, &configMap)
	require.NoError(s.T(), err)
	configOptions, err := controller.NewConfigOptions(configMap)
	require.NoError(s.T(), err)

	return configOptions
}

func (s *TestSuite) UpdateConfigOptions(configOptions *controller.ConfigOptions) {
	configMap, err := controller.GetConfigMapFromConfigOptions(configOptions)
	require.NoError(s.T(), err)
	err = s.client.Update(context.Background(), &configMap)
	require.NoError(s.T(), err)

	updatedConfigOptions := s.GetConfigOptions()
	require.Equal(s.T(), configOptions.SyncDirection, updatedConfigOptions.SyncDirection)
	require.Equal(s.T(), configOptions.ResourceGroupFilter, updatedConfigOptions.ResourceGroupFilter)
	s.T().Logf("Config options - syncDirection: %s, conflictPolicy: %s, minSyncPeriod: %s",
		configOptions.SyncDirection, configOptions.ConflictPolicy, configOptions.MinSyncPeriod)
}

func (s *TestSuite) GetNodes() *corev1.NodeList {
	assert := assert.New(s.T())
	require := require.New(s.T())

	nodeList := &corev1.NodeList{}
	err := s.client.List(context.Background(), nodeList)
	if err != nil {
		s.T().Logf("Failed listing nodes: %s", err)
	}
	require.NoError(err)
	// pass the expected number of nodes and check it here?
	assert.NotEqual(0, len(nodeList.Items))
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

func (s *TestSuite) GetNodesOnAzComputeResource(computeResource controller.ComputeResource, nodeList *corev1.NodeList) []corev1.Node {
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

func (s *TestSuite) UpdateTagsOnAzComputeResource(computeResource controller.ComputeResource, tags map[string]*string) controller.ComputeResource {
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

func (s *TestSuite) CheckNodeLabelsForTags(nodes []corev1.Node, tags map[string]*string, numStartingLabels map[string]int) {
	s.T().Logf("Checking nodes for accurate labels")
	for _, node := range nodes {
		updatedNode := &corev1.Node{}
		err := s.client.Get(context.Background(), types.NamespacedName{Name: node.Name, Namespace: node.Namespace}, updatedNode)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), len(tags), len(updatedNode.Labels)-numStartingLabels[updatedNode.Name])
		for key, val := range tags {
			validLabelName := controller.ConvertTagNameToValidLabelName(key, controller.DefaultConfigOptions()) // make sure this is config options I use
			result, ok := updatedNode.Labels[validLabelName]
			assert.True(s.T(), ok)
			assert.Equal(s.T(), *val, result)
		}
	}
}

func (s *TestSuite) CheckAzComputeResourceTagsForLabels(computeResource controller.ComputeResource, labels map[string]string, numStartingTags int) {
	s.T().Logf("Checking Azure compute resource for accurate labels")
	assert.Equal(s.T(), len(labels), len(computeResource.Tags())-numStartingTags)
	for key, val := range labels {
		v, ok := computeResource.Tags()[key]
		assert.True(s.T(), ok)
		assert.Equal(s.T(), val, *v)
	}
}

func (s *TestSuite) CleanupAzComputeResource(computeResource controller.ComputeResource, tags map[string]*string, numStartingTags int) controller.ComputeResource {
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

func WaitForReconcile() {
	time.Sleep(20 * time.Second)
}
