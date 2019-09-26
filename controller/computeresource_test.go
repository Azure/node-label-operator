package controller

import (
	"context"

	"github.com/Azure/go-autorest/autorest/to"
)

type FakeComputeResource struct {
	tags map[string]*string
}

func NewFakeComputeResourceEmptyTags() *FakeComputeResource {
	return &FakeComputeResource{tags: map[string]*string{}}
}

func NewFakeComputeResource(labels map[string]string) *FakeComputeResource {
	tagMap := map[string]*string{}
	for key, val := range labels {
		tagMap[key] = to.StringPtr(val)
	}
	return &FakeComputeResource{tags: tagMap}
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
