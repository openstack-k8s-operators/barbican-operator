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
	"fmt"
	"slices"
	"strings"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"gopkg.in/yaml.v3"
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
	customTemplates map[string]string,
) error {

	// Marshal the templateParameters map to YAML
	yamlData, err := yaml.Marshal(templateParameters)
	if err != nil {
		return fmt.Errorf("Error marshalling to YAML: %w", err)
	}
	customData[common.TemplateParameters] = string(yamlData)

	cms := []util.Template{
		// Templates where the BarbicanAPI config is stored
		{
			Name:           fmt.Sprintf("%s-config-data", instance.GetName()),
			Namespace:      instance.GetNamespace(),
			Type:           util.TemplateTypeConfig,
			InstanceType:   instance.GetObjectKind().GroupVersionKind().Kind,
			ConfigOptions:  templateParameters,
			CustomData:     customData,
			Labels:         cmLabels,
			StringTemplate: customTemplates,
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
