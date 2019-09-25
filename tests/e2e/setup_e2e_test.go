// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package tests

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var Scheme = runtime.NewScheme()

func AddToScheme(scheme *runtime.Scheme) {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = corev1.AddToScheme(scheme)
}

// necessary?
type Cluster struct {
	KubeConfigPath string
	SubscriptionID string
	ResourceGroup  string
}

type TestSuite struct {
	suite.Suite
	*Cluster
	client client.Client
}

// necessary?
func initialize(c *Cluster) error {
	if c.KubeConfigPath == "" {
		return errors.New("missing parameters: KubeConfigPath must be set")
	}
	// get these as environment variables??
	// do I want to create the test cluster(s) here somehow?
	return nil
}

func loadConfigOrFail(t *testing.T, kubeconfig string) *rest.Config {
	c, err := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfig},
		&clientcmd.ConfigOverrides{}).ClientConfig()
	require.NoError(t, err)
	return c
}

func (s *TestSuite) SetupSuite() {
	s.T().Logf("\nSetupSuite")
	err := initialize(s.Cluster)
	require.Nil(s.T(), err)
	AddToScheme(Scheme)
	cl, err := client.New(loadConfigOrFail(s.T(), s.KubeConfigPath), client.Options{Scheme: Scheme})
	require.NoError(s.T(), err)
	s.client = cl
}
