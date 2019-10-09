package controller

import (
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
)

func updateFunc(e event.UpdateEvent) bool {
	node, ok := e.ObjectNew.(*corev1.Node)
	if !ok {
		return false
	}
	return timeToUpdate(node)
}

// somehow there's a ton of create events
func createFunc(e event.CreateEvent) bool {
	node, ok := e.Object.(*corev1.Node)
	if !ok {
		return false
	}
	return timeToUpdate(node)
}

// return true because vmss might need to be updated?
func deleteFunc(e event.DeleteEvent) bool {
	return true
}

func genericFunc(e event.GenericEvent) bool {
	return false
}

func timeToUpdate(node *corev1.Node) bool {
	label, ok := node.Labels[lastUpdateLabel]
	if !ok {
		return true // let things through the first time
	}
	var period time.Duration
	// if lastUpdate formatted incorrectly, do I let stuff through?
	lastUpdate, err := time.Parse(time.RFC3339, strings.ReplaceAll(label, ".", ":"))
	if err != nil {
		return true // letting everything through if label formatted incorrectly
	}
	minSyncPeriod, ok := node.Labels[minSyncPeriodLabel]
	if ok {
		period, err = time.ParseDuration(minSyncPeriod)
		if err != nil {
			period = FiveMinutes
		}
	} else {
		period = FiveMinutes
	}
	syncPeriodStart := time.Now().Add(-period)
	return lastUpdate.Before(syncPeriodStart)
}
