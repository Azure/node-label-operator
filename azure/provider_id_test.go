package azure

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseProviderId(t *testing.T) {
	var providerIdTests = []struct {
		resourceId    string
		expectSuccess bool
		expected      Resource
	}{
		{
			"azure:///subscriptions/<sub-id>/resourceGroups/<resource-group>/providers/Microsoft.Compute/virtualMachineScaleSets/<vmss>/virtualMachines/0",
			true,
			Resource{
				SubscriptionID: "<sub-id>",
				ResourceGroup:  "<resource-group>",
				Provider:       "Microsoft.Compute",
				ResourceType:   "virtualMachineScaleSets",
				ResourceName:   "<vmss>",
			},
		},
		{
			"",
			false,
			Resource{},
		},
		{
			"https://management.azure.com/subscriptions/{subscriptionId}/resourceGroups/{resourceGroupName}/providers/Microsoft.Compute/virtualMachines/{vmName}",
			true,
			Resource{
				SubscriptionID: "{subscriptionId}",
				ResourceGroup:  "{resourceGroupName}",
				Provider:       "Microsoft.Compute",
				ResourceType:   "virtualMachines",
				ResourceName:   "{vmName}",
			},
		},
	}

	for _, tt := range providerIdTests {
		t.Run(tt.resourceId, func(t *testing.T) {
			resource, err := ParseProviderID(tt.resourceId)
			if err != nil {
				if tt.expectSuccess {
					t.Errorf("failed to parse resource ID: %q", err)
				}
				return
			}
			if !tt.expectSuccess {
				t.Errorf("expected to fail to parse resource ID")
			}
			assert.Equal(t, resource.SubscriptionID, tt.expected.SubscriptionID)
			assert.Equal(t, resource.ResourceGroup, tt.expected.ResourceGroup)
			assert.Equal(t, resource.Provider, tt.expected.Provider)
			assert.Equal(t, resource.ResourceType, tt.expected.ResourceType)
			assert.Equal(t, resource.ResourceName, tt.expected.ResourceName)
		})
	}
}
