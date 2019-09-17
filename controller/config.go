package controller

import (
	"encoding/json"
	"time"

	corev1 "k8s.io/api/core/v1"
)

const (
	DefaultLabelPrefix         string = "azure.tags"
	DefaultTagPrefix           string = "node.labels"
	DefaultResourceGroupFilter string = "none"
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
	SyncDirection       SyncDirection  `json:"syncDirection"`       // how do I validate this?
	Interval            string         `type:"int" json:"interval"` // how can I use a different type instead?
	LabelPrefix         string         `json:"labelPrefix"`
	TagPrefix           string         `json:"tagPrefix"`
	ConflictPolicy      ConflictPolicy `json:"conflictPolicy"`
	ResourceGroupFilter string         `json:"resourceGroupFilter"` // actually resource group filter
}

func NewConfigOptions(configMap corev1.ConfigMap) (ConfigOptions, error) {
	configOptions, err := loadConfigOptionsFromConfigMap(configMap)
	if err != nil {
		return ConfigOptions{}, err
	}

	if configOptions.SyncDirection != TwoWay &&
		configOptions.SyncDirection != ARMToNode &&
		configOptions.SyncDirection != NodeToARM {
		configOptions.SyncDirection = ARMToNode
	}

	if _, err := time.ParseDuration(configOptions.Interval); err != nil {
		// error?
		configOptions.Interval = "10h"
	}

	// I need a different way to check if not set b/c I need to allow for empty prefixes
	if configOptions.LabelPrefix == "" {
		configOptions.LabelPrefix = DefaultLabelPrefix
	}

	// also validate prefix?
	if configOptions.TagPrefix == "" {
		configOptions.TagPrefix = DefaultTagPrefix
	}

	if configOptions.ConflictPolicy != Ignore &&
		configOptions.ConflictPolicy != ARMPrecedence &&
		configOptions.ConflictPolicy != NodePrecedence {
		configOptions.ConflictPolicy = ARMPrecedence
	}

	if configOptions.ResourceGroupFilter == "" {
		configOptions.ResourceGroupFilter = DefaultResourceGroupFilter
	}

	return configOptions, nil
}

func DefaultConfigOptions() ConfigOptions {
	return ConfigOptions{
		SyncDirection:       ARMToNode,
		Interval:            "1", // todo
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

	configOptions := ConfigOptions{}
	if err := json.Unmarshal(data, &configOptions); err != nil {
		return ConfigOptions{}, err
	}

	return configOptions, nil
}
