// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"encoding/json"
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	DefaultLabelPrefix         string = "azure.tags"
	DefaultTagPrefix           string = "node.labels"
	DefaultResourceGroupFilter string = "none"
	UNSET                      string = "unset"
)

type SyncDirection string

const (
	TwoWay    SyncDirection = "two-way"
	ARMToNode SyncDirection = "arm-to-node"
	NodeToARM SyncDirection = "node-to-arm"
)

type ConflictPolicy string

const (
	Ignore         ConflictPolicy = "ignore"
	ARMPrecedence  ConflictPolicy = "arm-precedence"
	NodePrecedence ConflictPolicy = "node-precedence"
)

type ConfigOptions struct {
	SyncDirection       SyncDirection  `json:"syncDirection"` // how do I validate this?
	LabelPrefix         string         `json:"labelPrefix"`
	TagPrefix           string         `json:"tagPrefix"`
	ConflictPolicy      ConflictPolicy `json:"conflictPolicy"`
	ResourceGroupFilter string         `json:"resourceGroupFilter"`
}

func NewConfigOptions(configMap corev1.ConfigMap) (ConfigOptions, error) {
	configOptions, err := loadConfigOptionsFromConfigMap(configMap)
	if err != nil {
		return ConfigOptions{}, err
	}

	if configOptions.SyncDirection == "" {
		configOptions.SyncDirection = ARMToNode
	} else if configOptions.SyncDirection != TwoWay &&
		configOptions.SyncDirection != ARMToNode &&
		configOptions.SyncDirection != NodeToARM {
		return ConfigOptions{}, errors.New("invalid sync direction")
	}

	if configOptions.LabelPrefix == UNSET {
		configOptions.LabelPrefix = DefaultLabelPrefix
	}

	// also validate prefix?
	if configOptions.TagPrefix == UNSET {
		configOptions.TagPrefix = DefaultTagPrefix
	}

	if configOptions.ConflictPolicy == "" {
		configOptions.ConflictPolicy = ARMPrecedence
	} else if configOptions.ConflictPolicy != Ignore &&
		configOptions.ConflictPolicy != ARMPrecedence &&
		configOptions.ConflictPolicy != NodePrecedence {
		return ConfigOptions{}, errors.New("invalid tag-to-label conflict policy")
	}

	if configOptions.ResourceGroupFilter == "" {
		configOptions.ResourceGroupFilter = DefaultResourceGroupFilter
	}

	return configOptions, nil
}

func DefaultConfigOptions() ConfigOptions {
	return ConfigOptions{
		SyncDirection:       ARMToNode,
		LabelPrefix:         DefaultLabelPrefix,
		TagPrefix:           DefaultTagPrefix,
		ConflictPolicy:      ARMPrecedence,
		ResourceGroupFilter: DefaultResourceGroupFilter,
	}
}

func loadConfigOptionsFromConfigMap(configMap corev1.ConfigMap) (ConfigOptions, error) {
	data, err := json.Marshal(configMap.Data)
	if err != nil {
		return ConfigOptions{}, err
	}

	configOptions := ConfigOptions{LabelPrefix: UNSET, TagPrefix: UNSET}
	if err := json.Unmarshal(data, &configOptions); err != nil {
		return ConfigOptions{}, err
	}

	return configOptions, nil
}

func OptionsConfigMapNamespacedName() types.NamespacedName {
	return types.NamespacedName{Name: "node-label-operator", Namespace: "node-label-operator-system"}
}
