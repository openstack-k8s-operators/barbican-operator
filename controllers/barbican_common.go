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
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
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

// ensureBarbicanTopology - when a Topology CR is referenced, remove the
// finalizer from a previous referenced Topology (if any), and retrieve the
// newly referenced topology object
func ensureBarbicanTopology(
	ctx context.Context,
	helper *helper.Helper,
	tpRef *topologyv1.TopoRef,
	lastAppliedTopology *topologyv1.TopoRef,
	finalizer string,
	selector string,
) (*topologyv1.Topology, error) {

	var podTopology *topologyv1.Topology
	var err error

	// Remove (if present) the finalizer from a previously referenced topology
	//
	// 1. a topology reference is removed (tpRef == nil) from the Barbican Component
	//    subCR and the finalizer should be deleted from the last applied topology
	//    (lastAppliedTopology != "")
	// 2. a topology reference is updated in the Barbican Component CR (tpRef != nil)
	//    and the finalizer should be removed from the previously
	//    referenced topology (tpRef.Name != lastAppliedTopology.Name)
	if (tpRef == nil && lastAppliedTopology.Name != "") ||
		(tpRef != nil && tpRef.Name != lastAppliedTopology.Name) {
		_, err = topologyv1.EnsureDeletedTopologyRef(
			ctx,
			helper,
			lastAppliedTopology,
			finalizer,
		)
		if err != nil {
			return nil, err
		}
	}
	// TopologyRef is passed as input, get the Topology object
	if tpRef != nil {
		// no Namespace is provided, default to instance.Namespace
		if tpRef.Namespace == "" {
			tpRef.Namespace = helper.GetBeforeObject().GetNamespace()
		}
		// Build a defaultLabelSelector (component=barbican-[api|keystonelistener|worker])
		defaultLabelSelector := labels.GetSingleLabelSelector(
			common.ComponentSelector,
			selector,
		)
		// Retrieve the referenced Topology
		podTopology, _, err = topologyv1.EnsureTopologyRef(
			ctx,
			helper,
			tpRef,
			finalizer,
			&defaultLabelSelector,
		)
		if err != nil {
			return nil, err
		}
	}
	return podTopology, nil
}
