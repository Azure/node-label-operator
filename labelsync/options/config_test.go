package options

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestLoadConfigOptionsFromConfigMap(t *testing.T) {
	configMap := NewFakeConfigMap()
	configOptions, err := LoadConfigOptionsFromConfigMap(*configMap)
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

func TestNewConfig(t *testing.T) {
	configMap := NewFakeConfigMap()
	configOptions, err := NewConfig(*configMap)
	if err != nil {
		t.Errorf("failed to load new config options from map: %q", err)
	}
	assert.Equal(t, TwoWay, configOptions.SyncDirection)
	assert.Equal(t, DefaultTagPrefix, configOptions.TagPrefix)
	assert.Equal(t, "", configOptions.LabelPrefix)

}

func TestGetConfigMapFromConfigOptions(t *testing.T) {
	namespacecedName := ConfigMapNamespacedName()
	var configMapTests = []struct {
		configName string
		given      ConfigOptions
		expected   corev1.ConfigMap
	}{
		{
			"config1",
			ConfigOptions{
				SyncDirection: "node-to-arm",
			},
			corev1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      namespacecedName.Name,
					Namespace: namespacecedName.Namespace,
				},
				Data: map[string]string{
					"syncDirection": "node-to-arm",
				},
			},
		},
	}

	for _, tt := range configMapTests {
		t.Run(tt.configName, func(t *testing.T) {
			configMap, err := GetConfigMapFromConfigOptions(&tt.given)
			if err != nil {
				t.Errorf("failed to load config map from config options: %q", err)
			}
			syncDirection, ok := configMap.Data["syncDirection"]
			assert.True(t, ok)
			assert.Equal(t, tt.given.SyncDirection, SyncDirection(syncDirection))
			labelPrefix, ok := configMap.Data["labelPrefix"]
			assert.True(t, ok)
			assert.Equal(t, tt.given.LabelPrefix, labelPrefix)
			minSyncPeriod, ok := configMap.Data["minSyncPeriod"]
			assert.True(t, ok)
			assert.Equal(t, tt.given.MinSyncPeriod, minSyncPeriod)

		})
	}
}

func TestNewDefaultConfig(t *testing.T) {
	configOptions := DefaultConfigOptions()
	namespacecedName := ConfigMapNamespacedName()
	configMap, err := NewDefaultConfig()
	if err != nil {
		t.Errorf("failed to load config map from config options: %q", err)
	}
	syncDirection, ok := configMap.Data["syncDirection"]
	assert.True(t, ok)
	assert.Equal(t, configOptions.SyncDirection, SyncDirection(syncDirection))
	labelPrefix, ok := configMap.Data["labelPrefix"]
	assert.True(t, ok)
	assert.Equal(t, configOptions.LabelPrefix, labelPrefix)
	minSyncPeriod, ok := configMap.Data["minSyncPeriod"]
	assert.True(t, ok)
	assert.Equal(t, configOptions.MinSyncPeriod, minSyncPeriod)
	assert.Equal(t, configMap.Name, namespacecedName.Name)
	assert.Equal(t, configMap.Namespace, namespacecedName.Namespace)

}

// helpers

func NewFakeConfigMap() *corev1.ConfigMap {
	return &corev1.ConfigMap{
		Data: map[string]string{"syncDirection": "two-way", "labelPrefix": "", "minSyncPeriod": "1m"},
	}
}
