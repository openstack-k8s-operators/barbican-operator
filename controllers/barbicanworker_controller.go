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
	"time"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbicanworker"
	topologyv1 "github.com/openstack-k8s-operators/infra-operator/apis/topology/v1beta1"

	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/deployment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"

	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
)

// BarbicanWorkerReconciler reconciles a BarbicanWorker object
type BarbicanWorkerReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *BarbicanWorkerReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("BarbicanWorker")
}

// +kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=topology.openstack.org,resources=topologies,verbs=get;list;watch;update

// Reconcile BarbicanWorker
func (r *BarbicanWorkerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	instance := &barbicanv1beta1.BarbicanWorker{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Object not found
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	Log.Info(fmt.Sprintf("Reconciling BarbicanWorker %s", instance.Name))

	helper, err := helper.NewHelper(
		instance,
		r.Client,
		r.Kclient,
		r.Scheme,
		Log,
	)
	if err != nil {
		return ctrl.Result{}, err
	}

	// initialize status if Conditions is nil, but do not reset if it already
	// exists
	isNewInstance := instance.Status.Conditions == nil
	if isNewInstance {
		instance.Status.Conditions = condition.Conditions{}
	}

	// Save a copy of the condtions so that we can restore the LastTransitionTime
	// when a condition's state doesn't change.
	savedConditions := instance.Status.Conditions.DeepCopy()

	// Always patch the instance status when exiting this function so we can
	// persist any changes.
	defer func() {
		// Don't update the status, if reconciler Panics
		if r := recover(); r != nil {
			Log.Info(fmt.Sprintf("panic during reconcile %v\n", r))
			panic(r)
		}
		condition.RestoreLastTransitionTimes(
			&instance.Status.Conditions, savedConditions)
		if instance.Status.Conditions.IsUnknown(condition.ReadyCondition) {
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// Initialize Conditions
	cl := condition.CreateList(
		condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
		condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
		condition.UnknownCondition(condition.DeploymentReadyCondition, condition.InitReason, condition.DeploymentReadyInitMessage),
		condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		condition.UnknownCondition(condition.TLSInputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
	)
	instance.Status.Conditions.Init(&cl)

	Log.Info(fmt.Sprintf("Add finalizer %s", instance.Name))
	// Add Finalizer
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) || isNewInstance {
		return ctrl.Result{}, nil
	}

	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}

	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
	}

	// Init Topology condition if there's a reference
	if instance.Spec.TopologyRef != nil {
		c := condition.UnknownCondition(condition.TopologyReadyCondition, condition.InitReason, condition.TopologyReadyInitMessage)
		cl.Set(c)
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	Log.Info(fmt.Sprintf("Calling reconcile normal %s", instance.Name))

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
}

func (r *BarbicanWorkerReconciler) verifySecret(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.BarbicanWorker,
	secretName string,
	expectedFields []string,
	envVars *map[string]env.Setter,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	hash, result, err := secret.VerifySecret(ctx, types.NamespacedName{Name: secretName, Namespace: instance.Namespace}, expectedFields, h.GetClient(), time.Second*10)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	} else if (result != ctrl.Result{}) {
		Log.Info(fmt.Sprintf("OpenStack secret %s not found", secretName))
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.InputReadyWaitingMessage))
		return result, nil
	}

	// Add a prefix to the var name to avoid accidental collision with other non-secret
	// vars. The secret names themselves will be unique.
	(*envVars)["secret-"+secretName] = env.SetValue(hash)
	// env[secret-osp-secret] = hash?

	return ctrl.Result{}, nil
}

func (r *BarbicanWorkerReconciler) createHashOfInputHashes(
	ctx context.Context,
	instance *barbicanv1beta1.BarbicanWorker,
	envVars map[string]env.Setter,
) (string, bool, error) {
	Log := r.GetLogger(ctx)
	var hashMap map[string]string
	changed := false
	mergedMapVars := env.MergeEnvs([]corev1.EnvVar{}, envVars)
	hash, err := util.ObjectHash(mergedMapVars)
	if err != nil {
		return hash, changed, err
	}
	Log.Info("[Worker] ON createHashOfInputHashes")
	if hashMap, changed = util.SetHash(instance.Status.Hash, common.InputHashName, hash); changed {
		instance.Status.Hash = hashMap
		Log.Info(fmt.Sprintf("Input maps hash %s - %s", common.InputHashName, hash))
	}
	return hash, changed, nil
}

// generateServiceConfigs - create Secret which holds the service configuration
// TODO add DefaultConfigOverwrite
func (r *BarbicanWorkerReconciler) generateServiceConfigs(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.BarbicanWorker,
	envVars *map[string]env.Setter,
) error {
	Log := r.GetLogger(ctx)
	Log.Info("[Worker] generateServiceConfigs - reconciling")
	labels := labels.GetLabels(instance, labels.GetGroupLabel(barbican.ServiceName), map[string]string{})

	// customData hold any customization for the service.
	customData := map[string]string{
		barbican.CustomServiceConfigFileName: instance.Spec.CustomServiceConfig,
	}

	// Fetch the two service config snippets (DefaultsConfigFileName and
	// CustomConfigFileName) from the Secret generated by the top level
	// barbican controller, and add them to this service specific Secret.
	owner := barbican.GetOwningBarbicanName(instance)
	if owner != "" {
		barbicanSecretName := owner + "-config-data"
		barbicanSecret, _, err := secret.GetSecret(ctx, h, barbicanSecretName, instance.Namespace)
		if err != nil {
			return err
		}
		customData[barbican.DefaultsConfigFileName] = string(barbicanSecret.Data[barbican.DefaultsConfigFileName])
		customData[barbican.CustomConfigFileName] = string(barbicanSecret.Data[barbican.CustomConfigFileName])

		// TODO(alee) Get custom config overwrites from the parent barbican
	}

	for key, data := range instance.Spec.DefaultConfigOverwrite {
		customData[key] = data
	}

	customSecrets := ""
	for _, secretName := range instance.Spec.CustomServiceConfigSecrets {
		secret, _, err := secret.GetSecret(ctx, h, secretName, instance.Namespace)
		if err != nil {
			return err
		}
		for _, data := range secret.Data {
			customSecrets += string(data) + "\n"
		}
	}
	customData[barbican.CustomServiceConfigSecretsFileName] = customSecrets

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	templateParameters := map[string]interface{}{
		"LogFile": fmt.Sprintf("%s%s.log", barbican.BarbicanLogPath, instance.Name),
	}

	// To avoid a json parsing error in kolla files, we always need to set PKCS11ClientDataPath
	// This gets overridden in the PKCS11 section below if needed.
	templateParameters["PKCS11ClientDataPath"] = barbicanv1beta1.DefaultPKCS11ClientDataPath

	return GenerateConfigsGeneric(ctx, h, instance, envVars, templateParameters, customData, labels, false)
}

func (r *BarbicanWorkerReconciler) reconcileInit(
	ctx context.Context,
	instance *barbicanv1beta1.BarbicanWorker,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[Worker] Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanWorkerReconciler) reconcileUpdate(ctx context.Context, instance *barbicanv1beta1.BarbicanWorker) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[Worker] Reconciling Service '%s' update", instance.Name))

	// TODO: should have minor update tasks if required
	// - delete dbsync hash from status to rerun it?

	Log.Info(fmt.Sprintf("[Worker] Reconciled Service '%s' update successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanWorkerReconciler) reconcileUpgrade(ctx context.Context, instance *barbicanv1beta1.BarbicanWorker) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[Worker] Reconciling Service '%s' upgrade", instance.Name))

	// TODO: should have major version upgrade tasks
	// -delete dbsync hash from status to rerun it?

	Log.Info(fmt.Sprintf("[Worker] Reconciled Service '%s' upgrade successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanWorkerReconciler) reconcileDelete(ctx context.Context, instance *barbicanv1beta1.BarbicanWorker, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// Remove finalizer on the Topology CR
	if ctrlResult, err := topologyv1.EnsureDeletedTopologyRef(
		ctx,
		helper,
		instance.Status.LastAppliedTopology,
		instance.Name,
	); err != nil {
		return ctrlResult, err
	}

	// Service is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *BarbicanWorkerReconciler) reconcileNormal(ctx context.Context, instance *barbicanv1beta1.BarbicanWorker, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[Worker] Reconciling Service '%s'", instance.Name))

	configVars := make(map[string]env.Setter)

	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	Log.Info(fmt.Sprintf("[API] Verify secret '%s'", instance.Spec.Secret))
	ctrlResult, err := r.verifySecret(ctx, helper, instance, instance.Spec.Secret, []string{instance.Spec.PasswordSelectors.Service}, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	// check for required TransportURL secret holding transport URL string
	Log.Info(fmt.Sprintf("[Worker] Verify secret '%s'", instance.Spec.TransportURLSecret))
	ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.TransportURLSecret, []string{TransportURL}, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	// check for Simple Crypto Backend secret holding the KEK
	if len(instance.Spec.EnabledSecretStores) == 0 || slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStoreSimpleCrypto) {
		Log.Info(fmt.Sprintf("[Worker] Verify secret '%s'", instance.Spec.SimpleCryptoBackendSecret))
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.SimpleCryptoBackendSecret, []string{instance.Spec.PasswordSelectors.SimpleCryptoKEK}, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}

	// check PKCS11 secrets
	if slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStorePKCS11) && instance.Spec.PKCS11 != nil {
		// check pkcs11 login secret
		Log.Info(fmt.Sprintf("[Worker] Verify secret '%s'", instance.Spec.PKCS11.LoginSecret))
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.PKCS11.LoginSecret, []string{instance.Spec.PasswordSelectors.PKCS11Pin}, &configVars)
		if err != nil {
			return ctrlResult, err
		}

		// check for PKCS11 secret holding the PKCS11 Client Data
		Log.Info(fmt.Sprintf("[Worker] Verify secret '%s'", instance.Spec.PKCS11.ClientDataSecret))
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.PKCS11.ClientDataSecret, []string{}, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}

	//check CustomServiceConfigSecrets
	for _, v := range instance.Spec.CustomServiceConfigSecrets {
		Log.Info(fmt.Sprintf("[Worker] Verify secret '%s' from CustomServiceConfigSecrets", v))
		ctrlResult, err = r.verifySecret(ctx, helper, instance, v, []string{}, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}

	Log.Info(fmt.Sprintf("[Worker] Got secrets '%s'", instance.Name))

	//
	// TLS input validation
	//
	// Validate the CA cert secret if provided
	if instance.Spec.TLS.CaBundleSecretName != "" {
		hash, err := tls.ValidateCACertSecret(
			ctx,
			helper.GetClient(),
			types.NamespacedName{
				Name:      instance.Spec.TLS.CaBundleSecretName,
				Namespace: instance.Namespace,
			},
		)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.TLSInputReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.TLSInputReadyWaitingMessage,
					instance.Spec.TLS.CaBundleSecretName))
				return ctrl.Result{}, nil
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.TLSInputReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.TLSInputErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}

		if hash != "" {
			configVars[tls.CABundleKey] = env.SetValue(hash)
		}
	}

	// all cert input checks out so report InputReady
	instance.Status.Conditions.MarkTrue(condition.TLSInputReadyCondition, condition.InputReadyMessage)

	//
	// create custom config for this barbican service
	//
	err = r.generateServiceConfigs(ctx, helper, instance, &configVars)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	Log.Info(fmt.Sprintf("[Worker] Getting input hash '%s'", instance.Name))
	//
	// create hash over all the different input resources to identify if any those changed
	// and a restart/recreate is required.
	//
	inputHash, hashChanged, err := r.createHashOfInputHashes(ctx, instance, configVars)
	if err != nil {
		Log.Info("[Worker] ERR")
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	} else if hashChanged {
		Log.Info("[Worker] HAS CHANGED")
		// Hash changed and instance status should be updated (which will be done by main defer func),
		// so we need to return and reconcile again
		// return ctrl.Result{}, nil
	}
	Log.Info("[Worker] CONTINUE")
	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	Log.Info(fmt.Sprintf("[Worker] Getting service labels '%s'", instance.Name))
	serviceLabels := map[string]string{
		common.AppSelector:       fmt.Sprintf(barbican.ServiceName),
		common.ComponentSelector: barbican.ComponentWorker,
	}

	Log.Info(fmt.Sprintf("[Worker] Getting networks '%s'", instance.Name))
	// networks to attach to
	nadList := []networkv1.NetworkAttachmentDefinition{}
	for _, netAtt := range instance.Spec.NetworkAttachments {
		nad, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				Log.Info(fmt.Sprintf("network-attachment-definition %s not found", netAtt))
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, nil
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}

		if nad != nil {
			nadList = append(nadList, *nad)
		}
	}

	Log.Info(fmt.Sprintf("[Worker] Getting service annotations '%s'", instance.Name))
	serviceAnnotations, err := nad.EnsureNetworksAnnotation(nadList)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.NetworkAttachments, err)
	}
	Log.Info(fmt.Sprintf("[DELETE] %s", serviceAnnotations))

	// Handle service init
	ctrlResult, err = r.reconcileInit(ctx, instance)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service update
	ctrlResult, err = r.reconcileUpdate(ctx, instance)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// Handle service upgrade
	ctrlResult, err = r.reconcileUpgrade(ctx, instance)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	//
	// Handle Topology
	//
	topology, err := ensureTopology(
		ctx,
		helper,
		instance,      // topologyHandler
		instance.Name, // finalizer
		&instance.Status.Conditions,
		labels.GetLabelSelector(serviceLabels),
	)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.TopologyReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.TopologyReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, fmt.Errorf("waiting for Topology requirements: %w", err)
	}

	Log.Info(fmt.Sprintf("[Worker] Defining deployment '%s'", instance.Name))
	// Define a new Deployment object
	deplDef := barbicanworker.Deployment(instance, inputHash, serviceLabels, serviceAnnotations, topology)
	Log.Info(fmt.Sprintf("[Worker] Getting deployment '%s'", instance.Name))
	depl := deployment.NewDeployment(
		deplDef,
		time.Duration(5)*time.Second,
	)
	Log.Info(fmt.Sprintf("[Worker] Got deployment '%s'", instance.Name))
	ctrlResult, err = depl.CreateOrPatch(ctx, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DeploymentReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
		return ctrlResult, nil
	}

	deploy := depl.GetDeployment()
	if deploy.Generation == deploy.Status.ObservedGeneration {
		instance.Status.ReadyCount = deploy.Status.ReadyReplicas
	}

	// verify if network attachment matches expectations
	networkReady, networkAttachmentStatus, err := nad.VerifyNetworkStatusFromAnnotation(ctx, helper, instance.Spec.NetworkAttachments, serviceLabels, instance.Status.ReadyCount)
	if err != nil {
		return ctrl.Result{}, err
	}

	instance.Status.NetworkAttachments = networkAttachmentStatus
	if networkReady {
		instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)
	} else {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.NetworkAttachmentsReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanNetworkAttachmentsReadyErrorMessage,
			instance.Spec.NetworkAttachments))

		return ctrl.Result{}, err
	}

	// Mark the Deployment as Ready only if the number of Replicas is equals
	// to the Deployed instances (ReadyCount), and the the Status.Replicas
	// match Status.ReadyReplicas. If a deployment update is in progress,
	// Replicas > ReadyReplicas.
	// In addition, make sure the controller sees the last Generation
	// by comparing it with the ObservedGeneration.
	if deployment.IsReady(deploy) {
		oldDepName := fmt.Sprintf("%s-worker", instance.Name)
		if err := cleanupOldDeployment(ctx, r.Client, instance, oldDepName); err != nil {
			return ctrl.Result{}, err
		}
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
	} else {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DeploymentReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
	}
	// create Deployment - end

	// We reached the end of the Reconcile, update the Ready condition based on
	// the sub conditions
	if instance.Status.Conditions.AllSubConditionIsTrue() {
		instance.Status.Conditions.MarkTrue(
			condition.ReadyCondition, condition.ReadyMessage)
	}
	Log.Info(fmt.Sprintf("Reconciled Service '%s' in barbicanAPI successfully", instance.Name))
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *BarbicanWorkerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// index customServiceConfigSecrets
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, customServiceConfigSecretsField, func(rawObj client.Object) []string {
		// Extract the customServiceConfigSecrets names from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)

		return cr.Spec.CustomServiceConfigSecrets
	}); err != nil {
		return err
	}

	// index passwordSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, passwordSecretField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)
		if cr.Spec.Secret == "" {
			return nil
		}
		return []string{cr.Spec.Secret}
	}); err != nil {
		return err
	}

	// index caBundleSecretNameField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, caBundleSecretNameField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)
		if cr.Spec.TLS.CaBundleSecretName == "" {
			return nil
		}
		return []string{cr.Spec.TLS.CaBundleSecretName}
	}); err != nil {
		return err
	}

	// index pkcs11LoginSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, pkcs11LoginSecretField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)
		if cr.Spec.PKCS11 == nil || cr.Spec.PKCS11.LoginSecret == "" {
			return nil
		}
		return []string{cr.Spec.PKCS11.LoginSecret}
	}); err != nil {
		return err
	}

	// index pkcs11ClientDataSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, pkcs11ClientDataSecretField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)
		if cr.Spec.PKCS11 == nil || cr.Spec.PKCS11.ClientDataSecret == "" {
			return nil
		}
		return []string{cr.Spec.PKCS11.ClientDataSecret}
	}); err != nil {
		return err
	}

	// index topologyField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, topologyField, func(rawObj client.Object) []string {
		// Extract the topology name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)
		if cr.Spec.TopologyRef == nil {
			return nil
		}
		return []string{cr.Spec.TopologyRef.Name}
	}); err != nil {
		return err
	}

	// index parentBarbicanConfigDataSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanWorker{}, parentBarbicanConfigDataSecretField, func(rawObj client.Object) []string {
		// Extract the parent barbican config-data secret name
		cr := rawObj.(*barbicanv1beta1.BarbicanWorker)
		owner := barbican.GetOwningBarbicanName(cr)
		if owner == "" {
			return nil
		}
		return []string{owner + "-config-data"}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&barbicanv1beta1.BarbicanWorker{}).
		// Owns(&corev1.Service{}).
		// Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		// Owns(&routev1.Route{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSrc),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Watches(&topologyv1.Topology{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSrc),
			builder.WithPredicates(predicate.GenerationChangedPredicate{})).
		Complete(r)
}

func (r *BarbicanWorkerReconciler) findObjectsForSrc(ctx context.Context, src client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	Log := r.GetLogger(ctx)

	for _, field := range workerWatchFields {
		crList := &barbicanv1beta1.BarbicanWorkerList{}
		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, src.GetName()),
			Namespace:     src.GetNamespace(),
		}
		err := r.List(ctx, crList, listOps)
		if err != nil {
			Log.Error(err, fmt.Sprintf("listing %s for field: %s - %s", crList.GroupVersionKind().Kind, field, src.GetNamespace()))
			return requests
		}

		for _, item := range crList.Items {
			Log.Info(fmt.Sprintf("input source %s changed, reconcile: %s - %s", src.GetName(), item.GetName(), item.GetNamespace()))

			requests = append(requests,
				reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      item.GetName(),
						Namespace: item.GetNamespace(),
					},
				},
			)
		}
	}

	return requests
}
