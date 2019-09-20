package azure

import (
	"testing"
)

func TestParseProviderID(t *testing.T) {
	var parseProviderIDTest = []struct {
		providerID string
		expected   string
	}{
		{
			"",
			"",
		},
	}

	for _, tt := range parseProviderIDTest {
		// do stuff
		t.Run(tt.providerID, func(t *testing.T) {
		})
	}
}
