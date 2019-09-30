// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package controller

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	maxTagNameLen     int    = 512
	maxTagValLen      int    = 256
	maxNumTags        int    = 50
	invalidTagChars   string = "<>%&\\?/"
	maxLabelNameLen   int    = 63
	maxLabelPrefixLen int    = 253
	maxLabelValLen    int    = 63
)

func ValidTagName(labelName string, configOptions ConfigOptions) bool {
	return validTagName(LabelWithoutPrefix(labelName, configOptions.LabelPrefix))
}

func ValidLabelName(tagName string) bool {
	return validLabelName(tagName)
}

// this shouldn't ever happen
func ValidTagVal(labelVal string) bool {
	return len(labelVal) <= maxTagValLen
}

func ValidLabelVal(tagVal string) bool {
	if len(tagVal) > maxLabelValLen {
		return false
	}
	re := regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
	return re.MatchString(tagVal)
}

func ConvertTagNameToValidLabelName(tagName string, configOptions ConfigOptions) string {
	// lstrip configOptions.TagPrefix if there
	// don't forget to get rid of '.' after 'node.labels'... are there prefixes here?
	result := tagName
	if strings.HasPrefix(tagName, configOptions.TagPrefix) {
		result = strings.TrimPrefix(tagName, configOptions.TagPrefix)
	}

	// truncate name segment to 63 characters or less
	if len(result) > maxLabelNameLen {
		result = result[:maxLabelNameLen]
	}

	return LabelWithPrefix(result, configOptions.LabelPrefix)
}

func ConvertLabelNameToValidTagName(labelName string, configOptions ConfigOptions) string {
	// get rid of '/' and other characters.
	// also detect if 'azure.tags/' or other prefix is in the name to get rid of it
	// don't add if label name is a truncated version of a tag
	result := labelName
	if strings.HasPrefix(labelName, fmt.Sprintf("%s/", configOptions.LabelPrefix)) {
		result = strings.TrimPrefix(labelName, fmt.Sprintf("%s/", configOptions.LabelPrefix))
	}

	// result = tagWithPrefix(result, configOptions.TagPrefix)
	return result
}

func ConvertTagValToValidLabelVal(tagVal string) string {
	result := tagVal
	if len(result) > maxLabelValLen {
		result = result[:maxLabelValLen]
	}
	return result
}

func ConvertLabelValToValidTagVal(labelVal string) string {
	result := labelVal
	if len(result) > maxTagValLen {
		result = result[:maxTagValLen]
	}
	return result
}

func LabelWithPrefix(labelName, prefix string) string {
	return fmt.Sprintf("%s/%s", prefix, labelName)
}

func LabelWithoutPrefix(labelName, prefix string) string {
	if strings.HasPrefix(labelName, fmt.Sprintf("%s/", prefix)) {
		return strings.TrimPrefix(labelName, fmt.Sprintf("%s/", prefix))
	}
	return labelName
}

func validTagName(labelName string) bool {
	if len(labelName) > maxTagNameLen {
		return false
	}
	if strings.ContainsAny(labelName, invalidTagChars) {
		return false
	}
	return true
}

func validLabelName(tagName string) bool {
	if len(tagName) > maxTagNameLen {
		return false
	}
	re := regexp.MustCompile("^[a-zA-Z0-9_.-]*$")
	if !re.MatchString(tagName) {
		return false
	}
	alphanumRe := regexp.MustCompile("[a-zA-Z0-9]")
	nameLen := len(tagName)
	if nameLen > 0 &&
		(!alphanumRe.MatchString(tagName[0:1]) ||
			!alphanumRe.MatchString(tagName[nameLen-1:nameLen])) {
		return false
	}
	return true
}

func HasLabelPrefix(labelName string, labelPrefix string) bool {
	return strings.HasPrefix(labelName, fmt.Sprintf("%s/", labelPrefix))
}
