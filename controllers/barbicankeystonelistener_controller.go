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
	"time"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbicankeystonelistener"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"

	// keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/deployment"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"

	//"github.com/openstack-k8s-operators/lib-common/modules/common/endpoint"
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

// BarbicanKeystoneListenerReconciler reconciles a BarbicanKeystoneListener object
type BarbicanKeystoneListenerReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *BarbicanKeystoneListenerReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("BarbicanKeystoneListener")
}

//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/finalizers,verbs=update;patch

// Reconcile BarbicanAPI
func (r *BarbicanKeystoneListenerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	instance := &barbicanv1beta1.BarbicanKeystoneListener{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Object not found
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}
	Log.Info(fmt.Sprintf("Reconciling BarbicanKeystoneListener %s", instance.Name))

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

	Log.Info(fmt.Sprintf("initilize %s", instance.Name))

	// Initialize Conditions
	instance.Status.Conditions = condition.Conditions{}
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
	//if instance.Status.APIEndpoints == nil {
	//	instance.Status.APIEndpoints = map[string]string{}
	//}
	//if instance.Status.ServiceIDs == nil {
	//	instance.Status.ServiceIDs = map[string]string{}
	//}
	if instance.Status.NetworkAttachments == nil {
		instance.Status.NetworkAttachments = map[string][]string{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	Log.Info(fmt.Sprintf("Calling reconcile normal %s", instance.Name))

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
}

func (r *BarbicanKeystoneListenerReconciler) verifySecret(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.BarbicanKeystoneListener,
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

func (r *BarbicanKeystoneListenerReconciler) createHashOfInputHashes(
	ctx context.Context,
	instance *barbicanv1beta1.BarbicanKeystoneListener,
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
	Log.Info("[KeystoneListener] ON createHashOfInputHashes")
	if hashMap, changed = util.SetHash(instance.Status.Hash, common.InputHashName, hash); changed {
		instance.Status.Hash = hashMap
		Log.Info(fmt.Sprintf("Input maps hash %s - %s", common.InputHashName, hash))
	}
	return hash, changed, nil
}

// generateServiceConfigs - create Secret which holds the service configuration
// TODO add DefaultConfigOverwrite
func (r *BarbicanKeystoneListenerReconciler) generateServiceConfigs(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.BarbicanKeystoneListener,
	envVars *map[string]env.Setter,
) error {
	Log := r.GetLogger(ctx)
	Log.Info("[KeystoneListener] generateServiceConfigs - reconciling")
	labels := labels.GetLabels(instance, labels.GetGroupLabel(barbican.ServiceName), map[string]string{})

	db, err := mariadbv1.GetDatabaseByNameAndAccount(ctx, h, barbican.DatabaseCRName, instance.Spec.DatabaseAccount, instance.Namespace)
	if err != nil {
		return err
	}
	var tlsCfg *tls.Service
	if instance.Spec.TLS.CaBundleSecretName != "" {
		tlsCfg = &tls.Service{}
	}
	// customData hold any customization for the service.
	customData := map[string]string{
		barbican.CustomServiceConfigFileName: instance.Spec.CustomServiceConfig,
		"my.cnf":                             db.GetDatabaseClientConfig(tlsCfg), //(mschuppert) for now just get the default my.cnf
	}

	// Fetch the service config snippet (CustomConfigFileName) from the top
	// level barbican controller, and add them to this service specific Secret.
	owner := barbican.GetOwningBarbicanName(instance)
	if owner != "" {
		barbicanSecretName := owner + "-config-data"
		barbicanSecret, _, err := secret.GetSecret(ctx, h, barbicanSecretName, instance.Namespace)
		if err != nil {
			return err
		}
		customData[barbican.CustomConfigFileName] = string(barbicanSecret.Data[barbican.CustomConfigFileName])
	}

	Log.Info(fmt.Sprintf("[KeystoneListener] instance type %s", instance.GetObjectKind().GroupVersionKind().Kind))

	for key, data := range instance.Spec.DefaultConfigOverwrite {
		customData[key] = data
	}

	//keystoneAPI, err := keystonev1.GetKeystoneAPI(ctx, h, instance.Namespace, map[string]string{})
	// KeystoneAPI not available we should not aggregate the error and continue
	//if err != nil {
	//	return err
	//}
	//keystoneInternalURL, err := keystoneAPI.GetEndpoint(endpoint.EndpointInternal)
	//if err != nil {
	//	return err
	//}

	transportURLSecret, _, err := secret.GetSecret(ctx, h, instance.Spec.TransportURLSecret, instance.Namespace)
	if err != nil {
		return err
	}

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)

	databaseAccount := db.GetAccount()
	databaseSecret := db.GetSecret()

	templateParameters := map[string]interface{}{
		"DatabaseConnection": fmt.Sprintf("mysql+pymysql://%s:%s@%s/%s?read_default_file=/etc/my.cnf",
			databaseAccount.Spec.UserName,
			string(databaseSecret.Data[mariadbv1.DatabasePasswordSelector]),
			instance.Spec.DatabaseHostname,
			barbican.DatabaseName,
		),
		//"KeystoneAuthURL": keystoneInternalURL,
		//"ServicePassword": string(ospSecret.Data[instance.Spec.PasswordSelectors.Service]),
		//"ServiceUser":     instance.Spec.ServiceUser,
		//"ServiceURL":      "https://barbican.openstack.svc:9311",
		"TransportURL": string(transportURLSecret.Data["transport_url"]),
		"LogFile":      fmt.Sprintf("%s%s.log", barbican.BarbicanLogPath, instance.Name),
	}

	// To avoid a json parsing error in kolla files, we always need to set PKCS11ClientDataPath
	// TODO(alee) Fix this by deduplicating the template paths
	templateParameters["PKCS11ClientDataPath"] = barbicanv1beta1.DefaultPKCS11ClientDataPath

	return GenerateConfigsGeneric(ctx, h, instance, envVars, templateParameters, customData, labels, false, map[string]string{})
}

func (r *BarbicanKeystoneListenerReconciler) reconcileInit(
	ctx context.Context,
	instance *barbicanv1beta1.BarbicanKeystoneListener,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciling Service '%s' init", instance.Name))

	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanKeystoneListenerReconciler) reconcileUpdate(ctx context.Context, instance *barbicanv1beta1.BarbicanKeystoneListener) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciling Service '%s' update", instance.Name))

	// TODO: should have minor update tasks if required
	// - delete dbsync hash from status to rerun it?

	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciled Service '%s' update successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanKeystoneListenerReconciler) reconcileUpgrade(ctx context.Context, instance *barbicanv1beta1.BarbicanKeystoneListener) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciling Service '%s' upgrade", instance.Name))

	// TODO: should have major version upgrade tasks
	// -delete dbsync hash from status to rerun it?

	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciled Service '%s' upgrade successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanKeystoneListenerReconciler) reconcileDelete(ctx context.Context, instance *barbicanv1beta1.BarbicanKeystoneListener, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// Remove the finalizer from our KeystoneEndpoint CR
	//keystoneEndpoint, err := keystonev1.GetKeystoneEndpointWithName(ctx, helper, instance.Name, instance.Namespace)
	//if err != nil && !k8s_errors.IsNotFound(err) {
	//	return ctrl.Result{}, err
	//}

	/*
		if err == nil {
			if controllerutil.RemoveFinalizer(keystoneEndpoint, helper.GetFinalizer()) {
				err = r.Update(ctx, keystoneEndpoint)
				if err != nil && !k8s_errors.IsNotFound(err) {
					return ctrl.Result{}, err
				}
				util.LogForObject(helper, "Removed finalizer from our KeystoneEndpoint", instance)
			}
		}
	*/

	// Service is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

func (r *BarbicanKeystoneListenerReconciler) reconcileNormal(ctx context.Context, instance *barbicanv1beta1.BarbicanKeystoneListener, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("[KeystoneListener] Reconciling Service '%s'", instance.Name))

	configVars := make(map[string]env.Setter)

	//
	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	//
	Log.Info(fmt.Sprintf("[KeystoneListener] Verify secret '%s'", instance.Spec.Secret))
	ctrlResult, err := r.verifySecret(ctx, helper, instance, instance.Spec.Secret, []string{instance.Spec.PasswordSelectors.Service}, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	//
	// check for required TransportURL secret holding transport URL string
	//
	Log.Info(fmt.Sprintf("[KeystoneListener] Verify secret '%s'", instance.Spec.TransportURLSecret))
	ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.TransportURLSecret, []string{TransportURL}, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	Log.Info(fmt.Sprintf("[KeystoneListener] Got secrets '%s'", instance.Name))

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
					fmt.Sprintf(condition.TLSInputReadyWaitingMessage, instance.Spec.TLS.CaBundleSecretName)))
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

	Log.Info(fmt.Sprintf("[KeystoneListener] Getting input hash '%s'", instance.Name))
	//
	// create hash over all the different input resources to identify if any those changed
	// and a restart/recreate is required.
	//
	inputHash, hashChanged, err := r.createHashOfInputHashes(ctx, instance, configVars)
	if err != nil {
		Log.Info("[KeystoneListener] ERR")
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	} else if hashChanged {
		Log.Info("[KeystoneListener] HAS CHANGED")
		// Hash changed and instance status should be updated (which will be done by main defer func),
		// so we need to return and reconcile again
		// return ctrl.Result{}, nil
	}
	Log.Info("[KeystoneListener] CONTINUE")
	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	Log.Info(fmt.Sprintf("[KeystoneListener] Getting service labels '%s'", instance.Name))
	serviceLabels := map[string]string{
		common.AppSelector:       fmt.Sprintf(barbican.ServiceName),
		common.ComponentSelector: barbican.ComponentKeystoneListener,
	}

	Log.Info(fmt.Sprintf("[KeystoneListener] Getting networks '%s'", instance.Name))
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

	Log.Info(fmt.Sprintf("[KeystoneListener] Getting service annotations '%s'", instance.Name))
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

	Log.Info(fmt.Sprintf("[KeystoneListener] Defining deployment '%s'", instance.Name))
	// Define a new Deployment object
	deplDef := barbicankeystonelistener.Deployment(instance, inputHash, serviceLabels, serviceAnnotations)
	Log.Info(fmt.Sprintf("[KeystoneListener] Getting deployment '%s'", instance.Name))
	depl := deployment.NewDeployment(
		deplDef,
		time.Duration(5)*time.Second,
	)
	Log.Info(fmt.Sprintf("[KeystoneListener] Got deployment '%s'", instance.Name))
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
	instance.Status.ReadyCount = depl.GetDeployment().Status.ReadyReplicas

	// verify if network attachment matches expectations
	networkReady, networkAttachmentStatus, err := nad.VerifyNetworkStatusFromAnnotation(ctx, helper, instance.Spec.NetworkAttachments, serviceLabels, instance.Status.ReadyCount)
	if err != nil {
		return ctrl.Result{}, err
	}

	instance.Status.NetworkAttachments = networkAttachmentStatus
	if networkReady {
		instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)
	} else {
		err := fmt.Errorf("not all pods have interfaces with ips as configured in NetworkAttachments: %s", instance.Spec.NetworkAttachments)
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.NetworkAttachmentsReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.NetworkAttachmentsReadyErrorMessage,
			err.Error()))

		return ctrl.Result{}, err
	}

	if instance.Status.ReadyCount > 0 {
		instance.Status.Conditions.MarkTrue(condition.DeploymentReadyCondition, condition.DeploymentReadyMessage)
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
func (r *BarbicanKeystoneListenerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// index passwordSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanKeystoneListener{}, passwordSecretField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanKeystoneListener)
		if cr.Spec.Secret == "" {
			return nil
		}
		return []string{cr.Spec.Secret}
	}); err != nil {
		return err
	}

	// index caBundleSecretNameField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.BarbicanKeystoneListener{}, caBundleSecretNameField, func(rawObj client.Object) []string {
		// Extract the secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.BarbicanKeystoneListener)
		if cr.Spec.TLS.CaBundleSecretName == "" {
			return nil
		}
		return []string{cr.Spec.TLS.CaBundleSecretName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&barbicanv1beta1.BarbicanKeystoneListener{}).
		// Owns(&corev1.Service{}).
		// Owns(&corev1.Secret{}).
		Owns(&appsv1.Deployment{}).
		// Owns(&routev1.Route{}).
		Watches(
			&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSrc),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *BarbicanKeystoneListenerReconciler) findObjectsForSrc(ctx context.Context, src client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	l := log.FromContext(ctx).WithName("Controllers").WithName("BarbicanKeystoneListener")

	for _, field := range listenerWatchFields {
		crList := &barbicanv1beta1.BarbicanKeystoneListenerList{}
		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, src.GetName()),
			Namespace:     src.GetNamespace(),
		}
		err := r.List(ctx, crList, listOps)
		if err != nil {
			l.Error(err, fmt.Sprintf("listing %s for field: %s - %s", crList.GroupVersionKind().Kind, field, src.GetNamespace()))
			return requests
		}

		for _, item := range crList.Items {
			l.Info(fmt.Sprintf("input source %s changed, reconcile: %s - %s", src.GetName(), item.GetName(), item.GetNamespace()))

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
