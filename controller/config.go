package controller

import (
	"encoding/json"

	corev1 "k8s.io/api/core/v1"
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

	if configOptions.SyncDirection != TwoWay &&
		configOptions.SyncDirection != ARMToNode &&
		configOptions.SyncDirection != NodeToARM {
		configOptions.SyncDirection = ARMToNode
	}

	if configOptions.LabelPrefix == UNSET {
		configOptions.LabelPrefix = DefaultLabelPrefix
	}

	// also validate prefix?
	if configOptions.TagPrefix == UNSET {
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
