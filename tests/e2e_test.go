package tests

import (
	"context"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"node-label-operator.io/azure"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (s *TestSuite) TestARMTagToNodeLabel() {
	vmssClient, err := azure.NewScaleSetClient(s.SubscriptionID)
	require.NoError(s.T(), err)
	// vmssList, err := vmssClient.List(context.Background(), s.ResourceGroup)
	_, err := vmssClient.List(context.Background(), s.ResourceGroup)
	require.NoError(s.T(), err)
	// assert.NotEqual(len(vmssList.Values()), 0)

	nodeList := &corev1.NodeList{}
	err = s.client.List(context.Background(), nodeList, client.InNamespace("node-label-operator"))
	require.NoError(s.T(), err)
	// assert.NotEqual(len(nodeList.Items), 0)

	// get tags
	// get labels
	// check that every tag is a a label (if it's convertible to a valid label name)
}

func (s *TestSuite) TestNodeLabelToARMTag() {

}

func (s *TestSuite) TestTwoWaySync() {
}

// test:
// invalid label names
// too many tags or labels
