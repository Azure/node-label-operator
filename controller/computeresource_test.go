package controller

import (
	"context"
)

type FakeComputeResource struct {
	tags map[string]*string
}

func NewFakeComputeResource(labelMap map[string]*string) *FakeComputeResource {
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
