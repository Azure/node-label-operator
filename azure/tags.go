package azure

import "strings"

// Convert string such as a node label to a valid tag
// I would like to figure out regex stuff
// Tag names can't contain these characters: <, >, %, &, \, ?, /
// prepend an optional prefix here? in case a vmss has multiple nodes on it?
func ConvertToValidTagName(label string) string {
	return strings.Replace(label, "/", "-", -1)
}
