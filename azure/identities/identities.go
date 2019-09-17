// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT license.

package identities

import (
	"github.com/Azure/azure-sdk-for-go/services/msi/mgmt/2018-11-30/msi"
	uuid "github.com/satori/go.uuid"
)

type SpecOption func(*Spec) *Spec

type Spec struct {
	identity *msi.Identity
}

func Name(name string) SpecOption {
	return func(o *Spec) *Spec {
		o.identity.Name = &name
		return o
	}
}

func Location(location string) SpecOption {
	return func(o *Spec) *Spec {
		o.identity.Location = &location
		return o
	}
}

func (s *Spec) Set(options ...SpecOption) {
	for _, option := range options {
		s = option(s)
	}
}

func (s *Spec) ID() string {
	if s == nil || s.identity == nil || s.identity.ID == nil {
		return ""
	}
	return *s.identity.ID
}

func (s *Spec) PrincipalID() string {
	if s == nil || s.identity == nil || s.identity.IdentityProperties == nil || s.identity.PrincipalID == nil {
		return ""
	}
	return s.identity.PrincipalID.String()
}

func (s *Spec) TenantID() *uuid.UUID {
	if s == nil || s.identity == nil || s.identity.IdentityProperties == nil || s.identity.TenantID == nil {
		return &uuid.UUID{}
	}
	return s.identity.TenantID
}
