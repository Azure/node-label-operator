// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	ctrlfake "sigs.k8s.io/controller-runtime/pkg/client/fake"
)

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
