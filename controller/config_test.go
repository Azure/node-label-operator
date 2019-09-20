package controller

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLoadConfigOptionsFromConfigMap(t *testing.T) {
	configMap := NewFakeConfigMap()
	configOptions, err := loadConfigOptionsFromConfigMap(*configMap)
	if err != nil {
		t.Errorf("failed to load config options from config map: %q", err)
	}
	assert.Equal(t, TwoWay, configOptions.SyncDirection)
	assert.Equal(t, UNSET, configOptions.TagPrefix)
	assert.Equal(t, "", configOptions.LabelPrefix)
}

func TestDefaultConfigOptions(t *testing.T) {
	configOptions := DefaultConfigOptions()
	assert.Equal(t, DefaultTagPrefix, configOptions.TagPrefix)
	assert.Equal(t, DefaultResourceGroupFilter, configOptions.ResourceGroupFilter)

}

func TestNewConfigOptions(t *testing.T) {
	configMap := NewFakeConfigMap()
	configOptions, err := NewConfigOptions(*configMap)
	if err != nil {
		t.Errorf("failed to load new config options from map: %q", err)
	}
	assert.Equal(t, TwoWay, configOptions.SyncDirection)
	assert.Equal(t, DefaultTagPrefix, configOptions.TagPrefix)
	assert.Equal(t, "", configOptions.LabelPrefix)

}

func TestGetConfigMapFromConfigOptions(t *testing.T) {
}

func NewDefNewDefaultConfigOptions(t *testing.T) {
}
