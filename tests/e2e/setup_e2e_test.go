// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package tests

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/Azure/node-label-operator/azure"
	"github.com/Azure/node-label-operator/controller"
)

var Scheme = runtime.NewScheme()

func AddToScheme(scheme *runtime.Scheme) {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
}

type Cluster struct {
	KubeConfig     string
	SubscriptionID string
	ResourceGroup  string
	ResourceType   string
}

type TestSuite struct {
	suite.Suite
	suite.TearDownAllSuite
	*Cluster
	client client.Client
}

func initialize(c *Cluster) error {
	if c.KubeConfig == "" {
		return errors.New("missing parameters: KubeConfig must be set")
	}
	return nil
}

func loadConfigFromBytes(t *testing.T, kubeconfig_out string) *rest.Config {
	c, err := clientcmd.RESTConfigFromKubeConfig([]byte(kubeconfig_out))
	require.NoError(t, err)
	return c
}

func (s *TestSuite) setConfigOptions(minSyncPeriod string) {
	var configMap corev1.ConfigMap
	optionsNamespacedName := controller.OptionsConfigMapNamespacedName()
	err := s.client.Get(context.Background(), optionsNamespacedName, &configMap)
	require.NoError(s.T(), err)
	configOptions, err := controller.NewConfigOptions(configMap)
	require.NoError(s.T(), err)
	configOptions.SyncDirection = controller.ARMToNode
	configOptions.ConflictPolicy = controller.ARMPrecedence
	configOptions.MinSyncPeriod = minSyncPeriod
	configMap, err = controller.GetConfigMapFromConfigOptions(configOptions)
	require.NoError(s.T(), err)
	err = s.client.Update(context.Background(), &configMap)
	require.NoError(s.T(), err)
}

func (s *TestSuite) SetupSuite() {
	s.T().Logf("\nSetupSuite")
	err := initialize(s.Cluster)
	require.Nil(s.T(), err)
	AddToScheme(Scheme)
	cl, err := client.New(loadConfigFromBytes(s.T(), s.KubeConfig), client.Options{Scheme: Scheme})
	require.NoError(s.T(), err)
	s.client = cl

	// better to get metadata endpoint? would that be an issue w/ aad-pod-identity running?
	nodeList := &corev1.NodeList{}
	err = cl.List(context.Background(), nodeList, client.InNamespace("default"))
	require.NoError(s.T(), err)
	require.NotEmpty(s.T(), nodeList.Items)
	// for testing resource group filter, need to instead find every resource group
	resource, err := azure.ParseProviderID(nodeList.Items[0].Spec.ProviderID)
	require.NoError(s.T(), err)

	s.SubscriptionID = resource.SubscriptionID
	s.ResourceGroup = resource.ResourceGroup
	s.ResourceType = resource.ResourceType // ends up depending on which node is chosen first for aks-engine

	s.T().Logf("Resetting configmap")
	s.setConfigOptions("10s")
	time.Sleep(90 * time.Second)
}

// is this doing anything?
func (s *TestSuite) TearDownSuite() {
	s.T().Logf("\nTearDownSuite")

	s.T().Logf("Resetting configmap")
	s.setConfigOptions("1m")

	// make sure necessary tags/labels deleted? I would maybe save current tags and current labels?

	s.T().Logf("Finished tearing down suite")
}
