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

package controllers

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	oko_secret "github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// GenerateConfigsGeneric - generates config files
func GenerateConfigsGeneric(
	ctx context.Context, h *helper.Helper,
	instance client.Object,
	envVars *map[string]env.Setter,
	templateParameters map[string]interface{},
	customData map[string]string,
	cmLabels map[string]string,
	scripts bool,
) error {

	cms := []util.Template{
		// Templates where the BarbicanAPI config is stored
		{
			Name:          fmt.Sprintf("%s-config-data", instance.GetName()),
			Namespace:     instance.GetNamespace(),
			Type:          util.TemplateTypeConfig,
			InstanceType:  instance.GetObjectKind().GroupVersionKind().Kind,
			ConfigOptions: templateParameters,
			CustomData:    customData,
			Labels:        cmLabels,
		},
	}
	if scripts {
		cms = append(cms, util.Template{
			Name:          fmt.Sprintf("%s-scripts", instance.GetName()),
			Namespace:     instance.GetNamespace(),
			Type:          util.TemplateTypeScripts,
			InstanceType:  instance.GetObjectKind().GroupVersionKind().Kind,
			ConfigOptions: templateParameters,
			Labels:        cmLabels,
		})
	}
	return secret.EnsureSecrets(ctx, h, instance, cms, envVars)
}

func GenerateSecretStoreTemplateMap(
	enabledSecretStores []barbicanv1beta1.SecretStore,
	globalDefaultSecretStore barbicanv1beta1.SecretStore,
) (map[string]interface{}, error) {
	// Log := r.GetLogger(ctx)
	stores := []string{}
	if len(enabledSecretStores) == 0 {
		stores = []string{"simple_crypto"}
	} else {
		for _, value := range enabledSecretStores {
			stores = append(stores, string(value))
		}
	}

	if len(globalDefaultSecretStore) == 0 {
		globalDefaultSecretStore = "simple_crypto"
	}

	tempMap := map[string]interface{}{
		"EnabledSecretStores":      strings.Join(stores, ","),
		"GlobalDefaultSecretStore": globalDefaultSecretStore,
		"SimpleCryptoEnabled":      slices.Contains(stores, "simple_crypto"),
		"PKCS11CryptoEnabled":      slices.Contains(stores, "pkcs11"),
	}
	return tempMap, nil
}

func GeneratePKCS11TemplateMap(
	ctx context.Context,
	h *helper.Helper,
	pkcs11 barbicanv1beta1.BarbicanPKCS11Template,
	namespace string,
) (map[string]interface{}, error) {
	tempMap := map[string]interface{}{}
	hsmLoginSecret, _, err := oko_secret.GetSecret(ctx, h, pkcs11.LoginSecret, namespace)
	if err != nil {
		return nil, err
	}

	if len(pkcs11.TokenSerialNumber) > 0 {
		tempMap["P11TokenSerialNumber"] = pkcs11.TokenSerialNumber
	}
	if len(pkcs11.TokenLabels) > 0 {
		tempMap["P11TokenLabels"] = pkcs11.TokenLabels
	}
	if len(pkcs11.SlotId) > 0 {
		tempMap["P11SlotId"] = pkcs11.SlotId
	}

	// Checking if a supported HSM type has been provided.
	if !slices.Contains(barbicanv1beta1.HSMTypes, strings.ToLower(pkcs11.Type)) {
		return nil, errors.New("No valid HSM type provided!")
	}

	tempMap["P11Enabled"] = true
	tempMap["P11LibraryPath"] = pkcs11.LibraryPath
	tempMap["P11CertificatesMountPoint"] = pkcs11.CertificatesMountPoint
	tempMap["P11Login"] = string(hsmLoginSecret.Data["hsmLogin"])
	tempMap["P11MKEKLabel"] = pkcs11.MKEKLabel
	tempMap["P11MKEKLength"] = pkcs11.MKEKLength
	tempMap["P11HMACLabel"] = pkcs11.HMACLabel
	tempMap["P11HMACKeyType"] = pkcs11.HMACKeyType
	tempMap["P11HMACKeygenMechanism"] = pkcs11.HMACKeygenMechanism
	tempMap["P11HMACMechanism"] = pkcs11.HMACMechanism
	tempMap["P11LoggingLevel"] = pkcs11.LoggingLevel
	tempMap["P11ServerAddress"] = pkcs11.ServerAddress
	tempMap["P11ClientAddress"] = pkcs11.ClientAddress
	tempMap["P11Type"] = strings.ToLower(pkcs11.Type)
	tempMap["P11EncryptionMechanism"] = pkcs11.EncryptionMechanism
	tempMap["P11KeyWrapMechanism"] = pkcs11.KeyWrapMechanism
	tempMap["P11AESGCMGenerateIV"] = pkcs11.AESGCMGenerateIV
	tempMap["P11KeyWrapGenerateIV"] = pkcs11.KeyWrapGenerateIV
	tempMap["P11AlwaysSetCKASensitive"] = pkcs11.AlwaysSetCKASensitive
	tempMap["P11OSLockingOK"] = pkcs11.OSLockingOK

	return tempMap, nil
}
