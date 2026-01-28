/*
Copyright 2023.

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

// Package functional implements the envTest coverage for barbican-operator
package functional

import (
	"fmt"

	"github.com/openstack-k8s-operators/barbican-operator/internal/barbican"

	"k8s.io/apimachinery/pkg/types"
)

// APIType represents the type of API endpoint (internal or public)
type APIType string

const (
	// BarbicanAPITypeInternal represents the internal API type
	BarbicanAPITypeInternal APIType = "internal"
	// BarbicanAPITypePublic represents the public API type
	BarbicanAPITypePublic APIType = "public"
	// PublicCertSecretName -
	PublicCertSecretName = "public-tls-certs" // #nosec G101
	// InternalCertSecretName -
	InternalCertSecretName = "internal-tls-certs" // #nosec G101
	// CABundleSecretName -
	CABundleSecretName = "combined-ca-bundle" // #nosec G101
	// APICustomConfigSecret1Name -
	APICustomConfigSecret1Name = "api-custom-secret-1" // #nosec G101
	// APICustomConfigSecret2Name -
	APICustomConfigSecret2Name = "api-custom-secret-2" // #nosec G101
	// PKCS11LoginSecret -
	PKCS11LoginSecret = "pkcs11-login" // #nosec G101
	// PKCS11ClientDataSecret -
	PKCS11ClientDataSecret = "pkcs11-client-data" // #nosec G101
)

// BarbicanTestData is the data structure used to provide input data to envTest
type BarbicanTestData struct {
	BarbicanPassword                     string
	BarbicanServiceUser                  string
	ContainerImage                       string
	DatabaseHostname                     string
	DatabaseInstance                     string
	RabbitmqClusterName                  string
	RabbitmqSecretName                   string
	Instance                             types.NamespacedName
	Barbican                             types.NamespacedName
	BarbicanDatabaseName                 types.NamespacedName
	BarbicanDatabaseAccount              types.NamespacedName
	BarbicanDBSync                       types.NamespacedName
	BarbicanPKCS11Prep                   types.NamespacedName
	BarbicanAPI                          types.NamespacedName
	BarbicanWorker                       types.NamespacedName
	BarbicanWorkerDeployment             types.NamespacedName
	BarbicanKeystoneListener             types.NamespacedName
	BarbicanKeystoneListenerDeployment   types.NamespacedName
	BarbicanAPIDeployment                types.NamespacedName
	BarbicanRole                         types.NamespacedName
	BarbicanRoleBinding                  types.NamespacedName
	BarbicanTransportURL                 types.NamespacedName
	BarbicanSA                           types.NamespacedName
	BarbicanKeystoneService              types.NamespacedName
	BarbicanKeystoneEndpoint             types.NamespacedName
	BarbicanServicePublic                types.NamespacedName
	BarbicanServiceInternal              types.NamespacedName
	BarbicanConfigSecret                 types.NamespacedName
	BarbicanAPIConfigSecret              types.NamespacedName
	BarbicanWorkerConfigSecret           types.NamespacedName
	BarbicanKeystoneListenerConfigSecret types.NamespacedName
	BarbicanPKCS11LoginSecret            types.NamespacedName
	BarbicanPKCS11ClientDataSecret       types.NamespacedName
	BarbicanConfigScripts                types.NamespacedName
	BarbicanConfigMapData                types.NamespacedName
	BarbicanScheduler                    types.NamespacedName
	InternalAPINAD                       types.NamespacedName
	CABundleSecret                       types.NamespacedName
	InternalCertSecret                   types.NamespacedName
	PublicCertSecret                     types.NamespacedName
	BarbicanTopologies                   []types.NamespacedName
	BaseCustomServiceConfig              string
	APICustomServiceConfig               string
	BaseDefaultConfigOverwrite           map[string]string
	APIDefaultConfigOverwrite            map[string]string
	APICustomServiceConfigSecrets        []string
	APICustomConfigSecret1               types.NamespacedName
	APICustomConfigSecret2               types.NamespacedName
	APICustomConfigSecret1Contents       map[string][]byte
	APICustomConfigSecret2Contents       map[string][]byte
}

// GetBarbicanTestData is a function that initialize the BarbicanTestData
// used in the test
func GetBarbicanTestData(barbicanName types.NamespacedName) BarbicanTestData {
	m := barbicanName
	return BarbicanTestData{
		Instance: m,

		Barbican: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      barbicanName.Name,
		},
		BarbicanDatabaseName: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      barbican.DatabaseCRName,
		},
		BarbicanDatabaseAccount: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      "barbican",
		},
		BarbicanDBSync: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-db-sync", barbicanName.Name),
		},
		BarbicanPKCS11Prep: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-pkcs11-prep", barbicanName.Name),
		},
		BarbicanAPIDeployment: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-api", barbicanName.Name),
		},
		BarbicanAPI: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-api", barbicanName.Name),
		},
		BarbicanKeystoneListener: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-keystone-listener", barbicanName.Name),
		},
		BarbicanKeystoneListenerDeployment: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-keystone-listener", barbicanName.Name),
		},
		BarbicanWorker: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-worker", barbicanName.Name),
		},
		BarbicanWorkerDeployment: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-worker", barbicanName.Name),
		},
		BarbicanRole: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("barbican-%s-role", barbicanName.Name),
		},
		BarbicanRoleBinding: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("barbican-%s-rolebinding", barbicanName.Name),
		},
		BarbicanTransportURL: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("barbican-%s-transport", barbicanName.Name),
		},
		BarbicanSA: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("barbican-%s", barbicanName.Name),
		},
		BarbicanKeystoneService: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      barbicanName.Name,
		},
		BarbicanKeystoneEndpoint: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-api", barbicanName.Name),
		},
		// Also used to identify BarbicanRoutePublic
		BarbicanServicePublic: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-public", barbicanName.Name),
		},
		BarbicanServiceInternal: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-internal", barbicanName.Name),
		},
		BarbicanConfigSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-%s", barbicanName.Name, "config-data"),
		},
		BarbicanAPIConfigSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-%s", barbicanName.Name, "api-config-data"),
		},
		BarbicanWorkerConfigSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-%s", barbicanName.Name, "worker-config-data"),
		},
		BarbicanKeystoneListenerConfigSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-%s", barbicanName.Name, "keystone-listener-config-data"),
		},
		BarbicanPKCS11LoginSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      PKCS11LoginSecret,
		},
		BarbicanPKCS11ClientDataSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      PKCS11ClientDataSecret,
		},
		BarbicanConfigScripts: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-%s", barbicanName.Name, "scripts"),
		},
		BarbicanConfigMapData: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-%s", barbicanName.Name, "config-data"),
		},
		BarbicanScheduler: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-scheduler", barbicanName.Name),
		},
		InternalAPINAD: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      "internalapi",
		},
		CABundleSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      CABundleSecretName,
		},
		InternalCertSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      InternalCertSecretName,
		},
		PublicCertSecret: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      PublicCertSecretName,
		},
		RabbitmqClusterName: "rabbitmq",
		RabbitmqSecretName:  "rabbitmq-secret",
		DatabaseInstance:    "openstack",
		// Password used for service
		BarbicanPassword:    "12345678",
		BarbicanServiceUser: "barbican",
		ContainerImage:      "test://barbican",
		DatabaseHostname:    "database-hostname",
		// A set of topologies to Test how the reference is propagated to the
		// resulting StatefulSets and if a potential override produces the
		// expected values
		BarbicanTopologies: []types.NamespacedName{
			{
				Namespace: barbicanName.Namespace,
				Name:      fmt.Sprintf("%s-global-topology", barbicanName.Name),
			},
			{
				Namespace: barbicanName.Namespace,
				Name:      fmt.Sprintf("%s-api-topology", barbicanName.Name),
			},
			{
				Namespace: barbicanName.Namespace,
				Name:      fmt.Sprintf("%s-klistener-topology", barbicanName.Name),
			},
			{
				Namespace: barbicanName.Namespace,
				Name:      fmt.Sprintf("%s-worker-topology", barbicanName.Name),
			},
		},
		BaseCustomServiceConfig: "[DEFAULT]\nbase_custom_service_config=true",
		APICustomServiceConfig:  "[DEFAULT]\napi_custom_service_config=true",
		BaseDefaultConfigOverwrite: map[string]string{
			"policy.json":      "random base policy json stuff",
			"base-custom.conf": "[DEFAULT]\nrandom_api_custom_config_override=true",
		},
		APIDefaultConfigOverwrite: map[string]string{
			"policy.json":     "random api policy json stuff",
			"api-custom.conf": "[DEFAULT]\nrandom_api_custom_config_override=true",
		},
		APICustomServiceConfigSecrets: []string{
			APICustomConfigSecret1Name,
			APICustomConfigSecret2Name,
		},
		APICustomConfigSecret1: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      APICustomConfigSecret1Name,
		},
		APICustomConfigSecret2: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      APICustomConfigSecret2Name,
		},
		APICustomConfigSecret1Contents: map[string][]byte{
			"secret1": []byte("[custom_header1]\ncustom_attribute1=true\ncustom_attribute2=false"),
		},
		APICustomConfigSecret2Contents: map[string][]byte{
			"secret2": []byte("[custom_header2]\ncustom_attribute3=true\ncustom_attribute4=false"),
		},
	}
}
