// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package naming

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	MaxTagNameLen     int    = 512
	MaxTagValLen      int    = 256
	MaxNumTags        int    = 50
	InvalidTagChars   string = "<>%&\\?/"
	MaxLabelNameLen   int    = 63
	MaxLabelPrefixLen int    = 253
	MaxLabelValLen    int    = 63
)

func ValidTagName(labelName, labelPrefix string) bool {
	return validTagName(LabelWithoutPrefix(labelName, labelPrefix))
}

func ValidLabelName(tagName string) bool {
	return validLabelName(tagName)
}

// this shouldn't ever happen
func ValidTagVal(labelVal string) bool {
	return len(labelVal) <= MaxTagValLen
}

func ValidLabelVal(tagVal string) bool {
	if len(tagVal) > MaxLabelValLen {
		return false
	}
	re := regexp.MustCompile("^(([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9])?$")
	return re.MatchString(tagVal)
}

func ConvertTagNameToValidLabelName(tagName, labelPrefix string) string {
	// lstrip configOptions.TagPrefix if there
	// don't forget to get rid of '.' after 'node.labels'... are there prefixes here?
	result := tagName
	// if strings.HasPrefix(tagName, configOptions.TagPrefix) {
	// 	result = strings.TrimPrefix(tagName, configOptions.TagPrefix)
	// }

	// truncate name segment to 63 characters or less
	if len(result) > MaxLabelNameLen {
		result = result[:MaxLabelNameLen]
	}

	return LabelWithPrefix(result, labelPrefix)
}

func ConvertLabelNameToValidTagName(labelName, labelPrefix string) string {
	// get rid of '/' and other characters.
	// also detect if 'azure.tags/' or other prefix is in the name to get rid of it
	// don't add if label name is a truncated version of a tag
	result := labelName
	if strings.HasPrefix(labelName, fmt.Sprintf("%s/", labelPrefix)) {
		result = strings.TrimPrefix(labelName, fmt.Sprintf("%s/", labelPrefix))
	}

	// result = tagWithPrefix(result, configOptions.TagPrefix)
	return result
}

func ConvertTagValToValidLabelVal(tagVal string) string {
	result := tagVal
	if len(result) > MaxLabelValLen {
		result = result[:MaxLabelValLen]
	}
	return result
}

func ConvertLabelValToValidTagVal(labelVal string) string {
	result := labelVal
	if len(result) > MaxTagValLen {
		result = result[:MaxTagValLen]
	}
	return result
}

func LabelWithPrefix(labelName, prefix string) string {
	if len(prefix) == 0 {
		return labelName
	}
	return fmt.Sprintf("%s/%s", prefix, labelName)
}

func LabelWithoutPrefix(labelName, prefix string) string {
	if strings.HasPrefix(labelName, fmt.Sprintf("%s/", prefix)) {
		return strings.TrimPrefix(labelName, fmt.Sprintf("%s/", prefix))
	}
	return labelName
}

func validTagName(labelName string) bool {
	if len(labelName) > MaxTagNameLen {
		return false
	}
	if strings.ContainsAny(labelName, InvalidTagChars) {
		return false
	}
	return true
}

func validLabelName(tagName string) bool {
	if len(tagName) > MaxTagNameLen {
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
