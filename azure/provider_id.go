// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

// example: azure:///subscriptions/8643025a-c059-4a48-85d0-d76f51d63a74/resourceGroups/shoshanargwestus2/providers/Microsoft.Compute/virtualMachineScaleSets/default-9c6ddaaa-f207fc77/virtualMachines/3

package azure

import (
	"fmt"
	"regexp"
	"strings"
)

type Resource struct {
	SubscriptionID string
	ResourceGroup  string
	Provider       string
	ResourceType   string
	ResourceName   string
}

func ParseProviderID(providerID string) (Resource, error) {
	return parseResourceID(providerID)
}

// ParseResourceID parses a resource ID into a ResourceDetails struct.
// See https://docs.microsoft.com/en-us/azure/azure-resource-manager/resource-group-template-functions-resource#return-value-4.
func parseResourceID(resourceID string) (Resource, error) {

	const resourceIDPatternText = `(?i)subscriptions/(.+)/resourceGroups/(.+)/providers/(.+?)/(.+?)/(.+)`
	resourceIDPattern := regexp.MustCompile(resourceIDPatternText)
	match := resourceIDPattern.FindStringSubmatch(resourceID)

	if len(match) < 6 {
		return Resource{}, fmt.Errorf("parsing failed for %s. Invalid resource Id format", resourceID)
	}

	v := strings.Split(match[5], "/")

	result := Resource{
		SubscriptionID: match[1],
		ResourceGroup:  match[2],
		Provider:       match[3],
		ResourceType:   match[4],
		ResourceName:   v[0],
	}

	return result, nil
}
