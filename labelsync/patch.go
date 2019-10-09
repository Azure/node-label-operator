package labelsync

import (
	"encoding/json"

	"github.com/Azure/node-label-operator/labelsync/options"
)

func LabelPatch(labels map[string]string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": labels,
		},
	})
}

func LabelPatchWithDelete(labels map[string]*string) ([]byte, error) {
	return json.Marshal(map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels": labels,
		},
	})
}

func LabelDeletionAllowed(configOptions *options.ConfigOptions) bool {
	return configOptions.LabelPrefix != "" && (configOptions.ConflictPolicy == options.ARMPrecedence || configOptions.ConflictPolicy == options.Ignore)
}
