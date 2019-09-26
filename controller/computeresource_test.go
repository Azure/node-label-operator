package controller

import (
	"context"
)

type FakeComputeResource struct {
	tags map[string]*string
}

func NewFakeComputeResourceEmptyTags() *FakeComputeResource {
	return &FakeComputeResource{tags: map[string]*string{}}
}

func NewFakeComputeResource(labelMap map[string]*string) *FakeComputeResource {
	// tagMap := map[string]*string{}
	// for key, val := range labels {
	// 	tagMap[key] = to.StringPtr(val)
	// }
	// return &FakeComputeResource{tags: tagMap}

	// computeResource := NewFakeComputeResourceEmptyTags()
	// for key, val := range labels {
	// 	computeResource.SetTag(key, to.StringPtr(val))
	// }
	// return computeResource

	return &FakeComputeResource{tags: labelMap}
}

func (c FakeComputeResource) Update(ctx context.Context) error {
	return nil
}

func (c FakeComputeResource) Tags() map[string]*string {
	return c.tags
}

func (c FakeComputeResource) SetTag(name string, value *string) {
	c.tags[name] = value
}
