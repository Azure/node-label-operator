package labelsync

import (
	"errors"
	"fmt"

	"github.com/Azure/go-autorest/autorest/to"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"

	azrsrc "github.com/Azure/node-label-operator/azure/computeresource"
	"github.com/Azure/node-label-operator/labelsync/naming"
	"github.com/Azure/node-label-operator/labelsync/options"
)

// return patch with new labels, if any, otherwise return nil for no new labels or an error
func TagsToNodes(namespacedName types.NamespacedName, computeResource azrsrc.ComputeResource,
	node *corev1.Node, configOptions *options.ConfigOptions, log logr.Logger, recorder record.EventRecorder) ([]byte, error) {

	newLabels := map[string]*string{} // should allow for null JSON values
	for tagName, tagVal := range computeResource.Tags() {
		if !naming.ValidLabelName(tagName) {
			log.V(0).Info("invalid label name", "tag name", tagName)
			continue
		}
		if !naming.ValidLabelVal(*tagVal) {
			log.V(0).Info("invalid label value", "tag value", *tagVal)
			continue
		}
		validLabelName := naming.ConvertTagNameToValidLabelName(tagName, configOptions.LabelPrefix)
		labelVal, ok := node.Labels[validLabelName]
		if !ok {
			// add tag as label
			log.V(1).Info("applying tags to nodes", "tag name", tagName, "tag value", *tagVal)
			newLabels[validLabelName] = tagVal
		} else if labelVal != *tagVal {
			switch configOptions.ConflictPolicy {
			case options.ARMPrecedence:
				// set label anyway
				log.V(1).Info("overriding existing node label with ARM tag", "tag name", tagName, "tag value", tagVal)
				newLabels[validLabelName] = tagVal
			case options.NodePrecedence:
				// do nothing
				log.V(0).Info("name->value conflict found", "node label value", labelVal, "ARM tag value", *tagVal)
			case options.Ignore:
				// raise k8s event
				recorder.Event(node, "Warning", "ConflictingTagLabelValues",
					fmt.Sprintf("ARM tag was not applied to node because a different value for '%s' already exists (%s != %s).", tagName, *tagVal, labelVal))
				log.V(0).Info("name->value conflict found, leaving unchanged", "label value", labelVal, "tag value", *tagVal)
			default:
				return nil, errors.New("unrecognized conflict policy")
			}
		}
	}

	// delete labels if tag has been deleted
	// if conflict policy is node precedence (which it will most likely not be), then don't delete tags if they exist on node
	if labelDeletionAllowed(configOptions) {
		for labelFullName, labelVal := range node.Labels {
			if naming.HasLabelPrefix(labelFullName, configOptions.LabelPrefix) {
				// check if exists on vm/vmss
				labelName := naming.LabelWithoutPrefix(labelFullName, configOptions.LabelPrefix)
				_, ok := computeResource.Tags()[labelName]
				if !ok { // if label doesn't exist on ARM resource, delete
					log.V(1).Info("deleting label from node", "label name", labelFullName, "label value", labelVal)
					delete(node.Labels, labelFullName) // for some reason this is needed
					newLabels[labelFullName] = nil     // this should becomes 'null' in JSON, necessary for merge patch
				}
			}
		}
	}

	if len(newLabels) == 0 { // to avoid unnecessary patching
		return nil, nil
	}

	patch, err := LabelPatchWithDelete(newLabels)
	if err != nil {
		return nil, err
	}

	return patch, nil
}

func LabelsToAzureResource(namespacedName types.NamespacedName, computeResource azrsrc.ComputeResource,
	node *corev1.Node, configOptions *options.ConfigOptions, log logr.Logger, recorder record.EventRecorder) (map[string]*string, error) {

	if len(computeResource.Tags()) >= naming.MaxNumTags {
		log.V(0).Info("can't add any more tags", "number of tags", len(computeResource.Tags()))
		return computeResource.Tags(), nil
	}

	newTags := map[string]*string{}
	for labelName, labelVal := range node.Labels {
		if !naming.ValidTagName(labelName, configOptions.LabelPrefix) {
			log.V(2).Info("invalid tag name", "label name", labelName)
			continue
		}
		if !naming.ValidTagVal(labelVal) {
			log.V(2).Info("invalid tag name", "label name", labelName)
			continue
		}
		validTagName := naming.ConvertLabelNameToValidTagName(labelName, configOptions.LabelPrefix)
		tagVal, ok := computeResource.Tags()[validTagName]
		if !ok {
			// add label as tag
			log.V(1).Info("applying labels to Azure resource", "label name", labelName, "label value", labelVal)
			newTags[validTagName] = to.StringPtr(labelVal)
		} else if *tagVal != labelVal {
			switch configOptions.ConflictPolicy {
			case options.NodePrecedence:
				// set tag anyway
				log.V(1).Info("overriding existing ARM tag with node label", "label name", labelName, "label value", labelVal)
				newTags[validTagName] = to.StringPtr(labelVal)
			case options.ARMPrecedence:
				// do nothing
				log.V(0).Info("name->value conflict found", "node label value", labelVal, "ARM tag value", *tagVal)
			case options.Ignore:
				// raise k8s event
				recorder.Event(node, "Warning", "ConflictingTagLabelValues",
					fmt.Sprintf("node label was not applied to Azure resource because a different value for '%s' already exists (%s != %s).", labelName, labelVal, *tagVal))
				log.V(0).Info("name->value conflict found, leaving unchanged", "label value", labelVal, "tag value", *tagVal)
			default:
				return nil, errors.New("unrecognized conflict policy")
			}
		}
	}

	if len(newTags) == 0 { // if unchanged
		return nil, nil
	}

	return newTags, nil
}
