/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"testing"

	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// The OpenStackControlplane webhook validates a BarbicanSpecCore, never a
// BarbicanSpec, so the secret store validations have to fire on that path too.
// Otherwise a bad spec is admitted on the control plane and only fails later,
// when the Barbican CR itself is created.
func TestBarbicanSpecCoreValidatesSecretStores(t *testing.T) {
	basePath := field.NewPath("spec")

	tests := []struct {
		name        string
		mutate      func(spec *BarbicanSpecCore)
		expectedErr string
	}{
		{
			name: "pkcs11 enabled without a pkcs11 spec",
			mutate: func(spec *BarbicanSpecCore) {
				spec.EnabledSecretStores = []SecretStore{SecretStorePKCS11}
			},
			expectedErr: "PKCS11 is required when pkcs11 is an enabled SecretStore",
		},
		{
			name: "pkcs11 enabled without a clientDataSecret",
			mutate: func(spec *BarbicanSpecCore) {
				spec.EnabledSecretStores = []SecretStore{SecretStorePKCS11}
				spec.PKCS11 = &BarbicanPKCS11Template{LoginSecret: "hsm-login"}
			},
			expectedErr: "clientDataSecret is required when pkcs11 is an enabled SecretStore",
		},
		{
			name: "kmip enabled without a kmip spec",
			mutate: func(spec *BarbicanSpecCore) {
				spec.EnabledSecretStores = []SecretStore{SecretStoreKMIP}
			},
			expectedErr: "KMIP is required when kmip is an enabled SecretStore",
		},
		{
			name: "kmip enabled without a clientDataSecret",
			mutate: func(spec *BarbicanSpecCore) {
				spec.EnabledSecretStores = []SecretStore{SecretStoreKMIP}
				spec.KMIP = &BarbicanKMIPTemplate{}
			},
			expectedErr: "clientDataSecret is required when kmip is an enabled SecretStore",
		},
		{
			name: "pkcs11 and kmip sharing a clientDataPath",
			mutate: func(spec *BarbicanSpecCore) {
				spec.EnabledSecretStores = []SecretStore{SecretStorePKCS11, SecretStoreKMIP}
				spec.PKCS11 = &BarbicanPKCS11Template{
					LoginSecret: "hsm-login",
					ClientDataSpec: ClientDataSpec{
						ClientDataSecret: "hsm-client",
						ClientDataPath:   "/etc/shared-client",
					},
				}
				spec.KMIP = &BarbicanKMIPTemplate{
					ClientDataSpec: ClientDataSpec{
						ClientDataSecret: "kmip-client",
						ClientDataPath:   "/etc/shared-client",
					},
				}
			},
			expectedErr: "clientDataPath must differ from spec.pkcs11.clientDataPath",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			g := NewWithT(t)

			spec := &BarbicanSpecCore{}
			tt.mutate(spec)

			_, errs := spec.ValidateCreate(basePath, "openstack")
			g.Expect(errs.ToAggregate()).To(MatchError(ContainSubstring(tt.expectedErr)))

			_, errs = spec.ValidateUpdate(BarbicanSpecCore{}, basePath, "openstack")
			g.Expect(errs.ToAggregate()).To(MatchError(ContainSubstring(tt.expectedErr)))
		})
	}
}

// A BarbicanSpecCore that fully configures the secret stores it enables has to
// stay admissible, so the validations above cannot over-reject.
func TestBarbicanSpecCoreValidSecretStores(t *testing.T) {
	g := NewWithT(t)
	basePath := field.NewPath("spec")

	spec := &BarbicanSpecCore{}
	spec.EnabledSecretStores = []SecretStore{SecretStorePKCS11, SecretStoreKMIP}
	spec.PKCS11 = &BarbicanPKCS11Template{
		LoginSecret:    "hsm-login",
		ClientDataSpec: ClientDataSpec{ClientDataSecret: "hsm-client"},
	}
	spec.KMIP = &BarbicanKMIPTemplate{
		ClientDataSpec: ClientDataSpec{ClientDataSecret: "kmip-client"},
	}
	spec.Default()

	_, errs := spec.ValidateCreate(basePath, "openstack")
	g.Expect(errs).To(BeEmpty())

	_, errs = spec.ValidateUpdate(*spec, basePath, "openstack")
	g.Expect(errs).To(BeEmpty())
}
