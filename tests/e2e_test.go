// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package tests

import (
	"context"

	"github.com/Azure/node-label-operator/azure"
	"github.com/Azure/node-label-operator/controller"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// should I add tags in the test? and then remove them?
func (s *TestSuite) TestARMTagToNodeLabel() {
	keys := [3]string{"fruit1", "fruit2", "fruit3"}
	values := [3]string{"watermelon", "dragonfruit", "banana"}
	tags := map[string]*string{}
	for i, key := range keys {
		tags[key] = &values[i]
	}

	// get tags
	vmssClient, err := azure.NewScaleSetClient(s.SubscriptionID)
	require.NoError(s.T(), err)
	vmssList, err := vmssClient.List(context.Background(), s.ResourceGroup)
	require.NoError(s.T(), err)
	s.T().Log(vmssList.Values())
	assert.NotEqual(s.T(), len(vmssList.Values()), 0)

	// get labels
	nodeList := &corev1.NodeList{}
	err = s.client.List(context.Background(), nodeList, client.InNamespace("node-label-operator"))
	require.NoError(s.T(), err)
	assert.NotEqual(s.T(), len(nodeList.Items), 0)

	// check that every tag is a a label (if it's convertible to a valid label name)

	for _, vmss := range vmssList.Values() {
		vmss.Tags = tags
		// update
		f, err := vmssClient.CreateOrUpdate(context.Background(), s.ResourceGroup, *vmss.Name, vmss)
		require.NoError(s.T(), err)
		err = f.WaitForCompletionRef(context.Background(), vmssClient.Client)
		require.NoError(s.T(), err)
		updatedVmss, err := f.Result(vmssClient)
		require.NoError(s.T(), err)
		// check that vmss tags have been updated
		for key, val := range tags {
			result, ok := updatedVmss.Tags[key]
			assert.True(s.T(), ok)
			assert.Equal(s.T(), result, val)
		}
	}

	// wait
	// how long? like 5 minutes? how long is reconcile expected to take?

	// check that nodes now have accurate labels
	for _, node := range nodeList.Items {
		assert.Equal(s.T(), len(tags), len(node.Labels))
		for key, val := range tags {
			validLabelName := controller.ConvertTagNameToValidLabelName(key, controller.DefaultConfigOptions())
			result, ok := node.Labels[validLabelName]
			assert.True(s.T(), ok)
			assert.Equal(s.T(), result, val)
		}
	}

	// clean up by deleting tags and labels

	// clean up vmss
	for _, vmss := range vmssList.Values() {
		vmss.Tags = map[string]*string{}
		// update
		f, err := vmssClient.CreateOrUpdate(context.Background(), s.ResourceGroup, *vmss.Name, vmss)
		require.NoError(s.T(), err)
		err = f.WaitForCompletionRef(context.Background(), vmssClient.Client)
		require.NoError(s.T(), err)
		updatedVmss, err := f.Result(vmssClient)
		require.NoError(s.T(), err)
		assert.Equal(s.T(), len(updatedVmss.Tags), 0)
	}

	// clean up nodes
	for key, _ := range tags {
		// delete from every node
		validLabelName := controller.ConvertTagNameToValidLabelName(key, controller.DefaultConfigOptions())
		for _, node := range nodeList.Items {
			_, ok := node.Labels[validLabelName]
			assert.True(s.T(), ok)
			delete(node.Labels, validLabelName)
		}
	}

	for _, _ = range nodeList.Items {
		// update somehow??
	}
}

func (s *TestSuite) TestNodeLabelToARMTag() {

}

func (s *TestSuite) TestTwoWaySync() {
}

// test:
// invalid label names
// too many tags or labels
