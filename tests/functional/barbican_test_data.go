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

	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"

	"k8s.io/apimachinery/pkg/types"
)

type APIType string

const (
	// BarbicanAPIInternal -
	BarbicanAPITypeInternal APIType = "internal"
	// BarbicanAPITypeExternal
	BarbicanAPITypePublic APIType = "public"
	// PublicCertSecretName -
	PublicCertSecretName = "public-tls-certs"
	// InternalCertSecretName -
	InternalCertSecretName = "internal-tls-certs"
	// CABundleSecretName -
	CABundleSecretName = "combined-ca-bundle"
)

// BarbicanTestData is the data structure used to provide input data to envTest
type BarbicanTestData struct {
	BarbicanPassword               string
	BarbicanServiceUser            string
	ContainerImage                 string
	DatabaseHostname               string
	DatabaseInstance               string
	RabbitmqClusterName            string
	RabbitmqSecretName             string
	Instance                       types.NamespacedName
	Barbican                       types.NamespacedName
	BarbicanDatabaseName           types.NamespacedName
	BarbicanDatabaseAccount        types.NamespacedName
	BarbicanDBSync                 types.NamespacedName
	BarbicanPKCS11Prep             types.NamespacedName
	BarbicanAPI                    types.NamespacedName
	BarbicanRole                   types.NamespacedName
	BarbicanRoleBinding            types.NamespacedName
	BarbicanTransportURL           types.NamespacedName
	BarbicanSA                     types.NamespacedName
	BarbicanKeystoneService        types.NamespacedName
	BarbicanKeystoneEndpoint       types.NamespacedName
	BarbicanServicePublic          types.NamespacedName
	BarbicanServiceInternal        types.NamespacedName
	BarbicanConfigSecret           types.NamespacedName
	BarbicanAPIConfigSecret        types.NamespacedName
	BarbicanPKCS11LoginSecret      types.NamespacedName
	BarbicanPKCS11ClientDataSecret types.NamespacedName
	BarbicanConfigScripts          types.NamespacedName
	BarbicanConfigMapData          types.NamespacedName
	BarbicanScheduler              types.NamespacedName
	InternalAPINAD                 types.NamespacedName
	CABundleSecret                 types.NamespacedName
	InternalCertSecret             types.NamespacedName
	PublicCertSecret               types.NamespacedName
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
		BarbicanAPI: types.NamespacedName{
			Namespace: barbicanName.Namespace,
			Name:      fmt.Sprintf("%s-api-api", barbicanName.Name),
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
	}
}
