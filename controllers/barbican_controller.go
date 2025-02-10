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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/barbican-operator/pkg/barbican"
	rabbitmqv1 "github.com/openstack-k8s-operators/infra-operator/apis/rabbitmq/v1beta1"
	keystonev1 "github.com/openstack-k8s-operators/keystone-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/lib-common/modules/common"
	"github.com/openstack-k8s-operators/lib-common/modules/common/condition"
	"github.com/openstack-k8s-operators/lib-common/modules/common/endpoint"
	"github.com/openstack-k8s-operators/lib-common/modules/common/env"
	"github.com/openstack-k8s-operators/lib-common/modules/common/helper"
	"github.com/openstack-k8s-operators/lib-common/modules/common/job"
	"github.com/openstack-k8s-operators/lib-common/modules/common/labels"
	nad "github.com/openstack-k8s-operators/lib-common/modules/common/networkattachment"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	oko_secret "github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	"golang.org/x/exp/maps"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	PKCS11PrepReadyCondition      = "PKCS11PrepReady"
	PKCS11PrepReadyInitMessage    = "PKCS11 Prep job not started"
	PKCS11PrepReadyMessage        = "PKCS11 Prep job completed"
	PKCS11PrepReadyErrorMessage   = "PKCS11 Prep job error occurred %s"
	PKCS11PrepReadyRunningMessage = "PKCS11 Prep job is still running"
	PKCS11PrepReadyNotRunMessage  = "PKCS11 Prep job not run"
)

// BarbicanReconciler reconciles a Barbican object
type BarbicanReconciler struct {
	client.Client
	Kclient kubernetes.Interface
	Scheme  *runtime.Scheme
}

// GetLogger returns a logger object with a prefix of "controller.name" and additional controller context fields
func (r *BarbicanReconciler) GetLogger(ctx context.Context) logr.Logger {
	return log.FromContext(ctx).WithName("Controllers").WithName("Barbican")
}

//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicans,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicans/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicans/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanworkers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanworkers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanworkers/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicankeystonelisteners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicankeystonelisteners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicankeystonelisteners/finalizers,verbs=update;patch
//+kubebuilder:rbac:groups=keystone.openstack.org,resources=keystoneapis,verbs=get;list;watch;
//+kubebuilder:rbac:groups=keystone.openstack.org,resources=keystoneservices,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=keystone.openstack.org,resources=keystoneendpoints,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=core,resources=secrets,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=core,resources=configmaps,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups="",resources=pods,verbs=create;delete;get;list;patch;update;watch
//+kubebuilder:rbac:groups=rabbitmq.openstack.org,resources=transporturls,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,resources=jobs,verbs=get;list;watch;create;update;patch;delete;
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbaccounts/finalizers,verbs=update
//+kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbdatabases,verbs=get;list;watch;create;update;patch;delete;
//+kubebuilder:rbac:groups=k8s.cni.cncf.io,resources=network-attachment-definitions,verbs=get;list;watch

// service account, role, rolebinding
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update;patch
//+kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid,resources=securitycontextconstraints,verbs=use

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BarbicanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	instance := &barbicanv1beta1.Barbican{}
	err := r.Client.Get(ctx, req.NamespacedName, instance)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected.
			// For additional cleanup logic use finalizers. Return and don't requeue.
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return ctrl.Result{}, err
	}

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

	// Save a copy of the conditions so that we can restore the LastTransitionTime
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

	cl := condition.CreateList(
		// Mark ReadyCondition as Unknown from the beginning, because the
		// Reconcile function is in progress. If this condition is not marked
		// as True and is still in the "Unknown" state, we `Mirror(` the actual
		// failure/in-progress operation
		condition.UnknownCondition(condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage),
		condition.UnknownCondition(condition.DBReadyCondition, condition.InitReason, condition.DBReadyInitMessage),
		condition.UnknownCondition(PKCS11PrepReadyCondition, condition.InitReason, PKCS11PrepReadyInitMessage),
		condition.UnknownCondition(condition.DBSyncReadyCondition, condition.InitReason, condition.DBSyncReadyInitMessage),
		condition.UnknownCondition(condition.InputReadyCondition, condition.InitReason, condition.InputReadyInitMessage),
		condition.UnknownCondition(condition.ServiceConfigReadyCondition, condition.InitReason, condition.ServiceConfigReadyInitMessage),
		condition.UnknownCondition(barbicanv1beta1.BarbicanAPIReadyCondition, condition.InitReason, barbicanv1beta1.BarbicanAPIReadyInitMessage),
		condition.UnknownCondition(barbicanv1beta1.BarbicanWorkerReadyCondition, condition.InitReason, barbicanv1beta1.BarbicanWorkerReadyInitMessage),
		condition.UnknownCondition(barbicanv1beta1.BarbicanKeystoneListenerReadyCondition, condition.InitReason, barbicanv1beta1.BarbicanKeystoneListenerReadyInitMessage),
		condition.UnknownCondition(condition.NetworkAttachmentsReadyCondition, condition.InitReason, condition.NetworkAttachmentsReadyInitMessage),
		// service account, role, rolebinding conditions
		condition.UnknownCondition(condition.ServiceAccountReadyCondition, condition.InitReason, condition.ServiceAccountReadyInitMessage),
		condition.UnknownCondition(condition.RoleReadyCondition, condition.InitReason, condition.RoleReadyInitMessage),
		condition.UnknownCondition(condition.RoleBindingReadyCondition, condition.InitReason, condition.RoleBindingReadyInitMessage),
	)

	instance.Status.Conditions.Init(&cl)

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) || isNewInstance {
		return ctrl.Result{}, nil
	}

	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}

	// Handle service delete
	if !instance.DeletionTimestamp.IsZero() {
		return r.reconcileDelete(ctx, instance, helper)
	}

	// Handle non-deleted clusters
	return r.reconcileNormal(ctx, instance, helper)
}

func (r *BarbicanReconciler) reconcileNormal(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Reconciling Service '%s'", instance.Name))

	serviceLabels := map[string]string{
		common.AppSelector: barbican.ServiceName,
	}

	configVars := make(map[string]env.Setter)

	//
	// create RabbitMQ transportURL CR and get the actual URL from the associated secret that is created
	//
	transportURL, op, err := r.transportURLCreateOrUpdate(ctx, instance, serviceLabels)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanRabbitMQTransportURLReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanRabbitMQTransportURLReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	if op != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("TransportURL %s successfully reconciled - operation: %s", transportURL.Name, string(op)))
	}

	instance.Status.TransportURLSecret = transportURL.Status.SecretName

	if instance.Status.TransportURLSecret == "" {
		Log.Info(fmt.Sprintf("Waiting for TransportURL %s secret to be created", transportURL.Name))
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanRabbitMQTransportURLReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			barbicanv1beta1.BarbicanRabbitMQTransportURLReadyRunningMessage))
		return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
	}

	Log.Info(fmt.Sprintf("TransportURL secret name %s", transportURL.Status.SecretName))
	instance.Status.Conditions.MarkTrue(barbicanv1beta1.BarbicanRabbitMQTransportURLReadyCondition, barbicanv1beta1.BarbicanRabbitMQTransportURLReadyMessage)

	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	ctrlResult, err := r.verifySecret(ctx, helper, instance, instance.Spec.Secret, []string{instance.Spec.PasswordSelectors.Service}, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	// check for Simple Crypto Backend secret holding the KEK
	if len(instance.Spec.EnabledSecretStores) == 0 || slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStoreSimpleCrypto) {
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.SimpleCryptoBackendSecret, []string{instance.Spec.PasswordSelectors.SimpleCryptoKEK}, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}

	// check PKCS11 secrets
	if slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStorePKCS11) && instance.Spec.PKCS11 != nil {
		// check pkcs11 login secret
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.PKCS11.LoginSecret, []string{instance.Spec.PasswordSelectors.PKCS11Pin}, &configVars)
		if err != nil {
			return ctrlResult, err
		}

		// check for PKCS11 secret holding the PKCS11 Client Data
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.PKCS11.ClientDataSecret, []string{}, &configVars)
		if err != nil {
			return ctrlResult, err
		}
	}

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// Setting this here at the top level
	instance.Spec.ServiceAccount = instance.RbacResourceName()

	//
	// create service DB instance
	//
	db, result, err := r.ensureDB(ctx, helper, instance)
	if err != nil {
		return ctrl.Result{}, err
	} else if (result != ctrl.Result{}) {
		return result, nil
	}
	// create service DB - end

	err = r.generateServiceConfig(ctx, helper, instance, &configVars, serviceLabels, db)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	// networks to attach to
	nadList := []networkv1.NetworkAttachmentDefinition{}
	for _, netAtt := range instance.Spec.BarbicanAPI.NetworkAttachments {
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

	// TODO: _ should be serviceAnnotations
	serviceAnnotations, err := nad.EnsureNetworksAnnotation(nadList)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.BarbicanAPI.NetworkAttachments, err)
	}

	// Handle service init
	ctrlResult, err = r.reconcileInit(ctx, instance, helper, serviceLabels, serviceAnnotations)
	if err != nil {
		return ctrlResult, err
	} else if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	// TODO(dmendiza): Handle service update

	// TODO(dmendiza): Handle service upgrade

	// create or update Barbican API deployment
	barbicanAPI, op, err := r.apiDeploymentCreateOrUpdate(ctx, instance, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanAPIReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanAPIReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror BarbicanAPI's condition status
	c := barbicanAPI.Status.Conditions.Mirror(barbicanv1beta1.BarbicanAPIReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	// create or update Barbican Worker deployment
	barbicanWorker, op, err := r.workerDeploymentCreateOrUpdate(ctx, instance, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanWorkerReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanWorkerReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror BarbicanWorker's condition status
	c = barbicanWorker.Status.Conditions.Mirror(barbicanv1beta1.BarbicanWorkerReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	// remove finalizers from unused MariaDBAccount records
	// this assumes all database-depedendent deployments are up and
	// running with current database account info
	err = mariadbv1.DeleteUnusedMariaDBAccountFinalizers(
		ctx, helper, barbican.DatabaseCRName,
		instance.Spec.DatabaseAccount, instance.Namespace)
	if err != nil {
		return ctrl.Result{}, err
	}

	// create or update Barbican KeystoneListener deployment
	barbicanKeystoneListener, op, err := r.keystoneListenerDeploymentCreateOrUpdate(ctx, instance, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanKeystoneListenerReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanKeystoneListenerReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if op != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(op)))
	}

	// Mirror BarbicanKeystoneListener's condition status
	c = barbicanKeystoneListener.Status.Conditions.Mirror(barbicanv1beta1.BarbicanKeystoneListenerReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	// TODO(dmendiza): Handle API endpoints

	// TODO(dmendiza): Understand what Glance is doing with the API conditions and maybe do it here too

	// Update the lastObserved generation before evaluating conditions
	instance.Status.ObservedGeneration = instance.Generation
	// We reached the end of the Reconcile, update the Ready condition based on
	// the sub conditions
	if instance.Status.Conditions.AllSubConditionIsTrue() {
		instance.Status.Conditions.MarkTrue(
			condition.ReadyCondition, condition.ReadyMessage)
	}
	return ctrl.Result{}, nil
}

func (r *BarbicanReconciler) reconcileDelete(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// remove db finalizer first
	db, err := mariadbv1.GetDatabaseByNameAndAccount(ctx, helper, barbican.DatabaseCRName, instance.Spec.DatabaseAccount, instance.Namespace)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if !k8s_errors.IsNotFound(err) {
		if err := db.DeleteFinalizer(ctx, helper); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Remove the finalizer from our KeystoneService CR
	keystoneService, err := keystonev1.GetKeystoneServiceWithName(ctx, helper, barbican.ServiceName, instance.Namespace)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if err == nil {
		if controllerutil.RemoveFinalizer(keystoneService, helper.GetFinalizer()) {
			err = r.Update(ctx, keystoneService)
			if err != nil && !k8s_errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			util.LogForObject(helper, "Removed finalizer from our KeystoneService", instance)
		}
	}

	// Remove finalizers from any existing child BarbicanAPIs
	barbicanAPI := &barbicanv1beta1.BarbicanAPI{}
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-api", instance.Name), Namespace: instance.Namespace}, barbicanAPI)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if err == nil {
		if controllerutil.RemoveFinalizer(barbicanAPI, helper.GetFinalizer()) {
			err = r.Update(ctx, barbicanAPI)
			if err != nil && !k8s_errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			util.LogForObject(helper, fmt.Sprintf("Removed finalizer from BarbicanAPI %s", barbicanAPI.Name), barbicanAPI)
		}
	}

	// Remove finalizers from any existing child barbicanWorkers
	barbicanWorker := &barbicanv1beta1.BarbicanWorker{}
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-worker", instance.Name), Namespace: instance.Namespace}, barbicanWorker)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if err == nil {
		if controllerutil.RemoveFinalizer(barbicanWorker, helper.GetFinalizer()) {
			err = r.Update(ctx, barbicanWorker)
			if err != nil && !k8s_errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			util.LogForObject(helper, fmt.Sprintf("Removed finalizer from BarbicanWorker %s", barbicanWorker.Name), barbicanWorker)
		}
	}

	// Remove finalizers from Barbican Keystone Listener
	barbicanKeystoneListener := &barbicanv1beta1.BarbicanKeystoneListener{}
	err = r.Get(ctx, types.NamespacedName{Name: fmt.Sprintf("%s-keystone-listener", instance.Name), Namespace: instance.Namespace}, barbicanKeystoneListener)
	if err != nil && !k8s_errors.IsNotFound(err) {
		return ctrl.Result{}, err
	}

	if err == nil {
		if controllerutil.RemoveFinalizer(barbicanKeystoneListener, helper.GetFinalizer()) {
			err = r.Update(ctx, barbicanKeystoneListener)
			if err != nil && !k8s_errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
			util.LogForObject(helper, fmt.Sprintf("Removed finalizer from BarbicanKeystoneListener %s", barbicanKeystoneListener.Name), barbicanKeystoneListener)
		}
	}

	// Service is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

// fields to index to reconcile when change
const (
	passwordSecretField         = ".spec.secret"
	caBundleSecretNameField     = ".spec.tls.caBundleSecretName"
	tlsAPIInternalField         = ".spec.tls.api.internal.secretName"
	tlsAPIPublicField           = ".spec.tls.api.public.secretName"
	pkcs11LoginSecretField      = ".spec.pkcs11.loginSecret"
	pkcs11ClientDataSecretField = ".spec.pkcs11.clientDataSecret"
	topologyField               = ".spec.topologyRef.Name"
)

var (
	workerWatchFields = []string{
		passwordSecretField,
		caBundleSecretNameField,
		pkcs11LoginSecretField,
		pkcs11ClientDataSecretField,
		topologyField,
	}
	apinWatchFields = []string{
		passwordSecretField,
		caBundleSecretNameField,
		tlsAPIInternalField,
		tlsAPIPublicField,
		pkcs11LoginSecretField,
		pkcs11ClientDataSecretField,
		topologyField,
	}
	listenerWatchFields = []string{
		passwordSecretField,
		caBundleSecretNameField,
		topologyField,
	}
)

// SetupWithManager sets up the controller with the Manager.
func (r *BarbicanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&barbicanv1beta1.Barbican{}).
		Owns(&barbicanv1beta1.BarbicanAPI{}).
		Owns(&barbicanv1beta1.BarbicanWorker{}).
		Owns(&barbicanv1beta1.BarbicanKeystoneListener{}).
		Owns(&rabbitmqv1.TransportURL{}).
		Owns(&mariadbv1.MariaDBDatabase{}).
		Owns(&mariadbv1.MariaDBAccount{}).
		Owns(&keystonev1.KeystoneService{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&batchv1.Job{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ServiceAccount{}).
		Owns(&rbacv1.Role{}).
		Owns(&rbacv1.RoleBinding{}).
		Complete(r)
}

func (r *BarbicanReconciler) generateServiceConfig(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.Barbican,
	envVars *map[string]env.Setter,
	serviceLabels map[string]string,
	db *mariadbv1.Database,
) error {
	Log := r.GetLogger(ctx)
	Log.Info("generateServiceConfigMaps - Barbican controller")

	// create Secret required for barbican input
	labels := labels.GetLabels(instance, labels.GetGroupLabel(barbican.ServiceName), serviceLabels)

	ospSecret, _, err := oko_secret.GetSecret(ctx, h, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		return err
	}

	transportURLSecret, _, err := oko_secret.GetSecret(ctx, h, instance.Status.TransportURLSecret, instance.Namespace)
	if err != nil {
		return err
	}

	var tlsCfg *tls.Service
	if instance.Spec.BarbicanAPI.TLS.Ca.CaBundleSecretName != "" {
		tlsCfg = &tls.Service{}
	}
	customData := map[string]string{
		barbican.CustomConfigFileName: instance.Spec.CustomServiceConfig,
		"my.cnf":                      db.GetDatabaseClientConfig(tlsCfg), //(mschuppert) for now just get the default my.cnf
	}

	for key, data := range instance.Spec.DefaultConfigOverwrite {
		customData[key] = data
	}
	keystoneAPI, err := keystonev1.GetKeystoneAPI(ctx, h, instance.Namespace, map[string]string{})
	// KeystoneAPI not available we should not aggregate the error and continue
	if err != nil {
		return err
	}
	keystoneInternalURL, err := keystoneAPI.GetEndpoint(endpoint.EndpointInternal)
	if err != nil {
		return err
	}

	databaseAccount := db.GetAccount()
	databaseSecret := db.GetSecret()

	templateParameters := map[string]interface{}{
		"DatabaseConnection": fmt.Sprintf("mysql+pymysql://%s:%s@%s/%s?read_default_file=/etc/my.cnf",
			databaseAccount.Spec.UserName,
			string(databaseSecret.Data[mariadbv1.DatabasePasswordSelector]),
			instance.Status.DatabaseHostname,
			barbican.DatabaseName,
		),
		"KeystoneAuthURL":  keystoneInternalURL,
		"ServicePassword":  string(ospSecret.Data[instance.Spec.PasswordSelectors.Service]),
		"ServiceUser":      instance.Spec.ServiceUser,
		"ServiceURL":       "TODO",
		"TransportURL":     string(transportURLSecret.Data["transport_url"]),
		"LogFile":          fmt.Sprintf("%s%s.log", barbican.BarbicanLogPath, instance.Name),
		"EnableSecureRBAC": instance.Spec.BarbicanAPI.EnableSecureRBAC,
	}

	// To avoid a json parsing error in kolla files, we always need to set PKCS11ClientDataPath
	// This gets overridden in the PKCS11 section below if needed.
	templateParameters["PKCS11ClientDataPath"] = barbicanv1beta1.DefaultPKCS11ClientDataPath

	// Set secret store parameters
	secretStoreTemplateMap, err := GenerateSecretStoreTemplateMap(
		instance.Spec.EnabledSecretStores,
		instance.Spec.GlobalDefaultSecretStore)
	if err != nil {
		return err
	}
	maps.Copy(templateParameters, secretStoreTemplateMap)

	// Set pkcs11 parameters
	if slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStorePKCS11) && instance.Spec.PKCS11 != nil {
		hsmLoginSecret, _, err := oko_secret.GetSecret(ctx, h, instance.Spec.PKCS11.LoginSecret, instance.Namespace)
		if err != nil {
			return err
		}
		templateParameters["PKCS11Login"] = string(hsmLoginSecret.Data[instance.Spec.PasswordSelectors.PKCS11Pin])
		templateParameters["PKCS11Enabled"] = true
		templateParameters["PKCS11ClientDataPath"] = instance.Spec.PKCS11.ClientDataPath
	}

	return GenerateConfigsGeneric(ctx, h, instance, envVars, templateParameters, customData, labels, true)
}

func (r *BarbicanReconciler) transportURLCreateOrUpdate(
	ctx context.Context,
	instance *barbicanv1beta1.Barbican,
	serviceLabels map[string]string,
) (*rabbitmqv1.TransportURL, controllerutil.OperationResult, error) {
	transportURL := &rabbitmqv1.TransportURL{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-barbican-transport", instance.Name),
			Namespace: instance.Namespace,
			Labels:    serviceLabels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, transportURL, func() error {
		transportURL.Spec.RabbitmqClusterName = instance.Spec.RabbitMqClusterName

		err := controllerutil.SetControllerReference(instance, transportURL, r.Scheme)
		return err
	})

	return transportURL, op, err
}

func (r *BarbicanReconciler) apiDeploymentCreateOrUpdate(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper) (*barbicanv1beta1.BarbicanAPI, controllerutil.OperationResult, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Creating barbican API spec.  transporturlsecret: '%s'", instance.Status.TransportURLSecret))
	Log.Info(fmt.Sprintf("database hostname: '%s'", instance.Status.DatabaseHostname))
	apiSpec := barbicanv1beta1.BarbicanAPISpec{
		BarbicanTemplate:    instance.Spec.BarbicanTemplate,
		BarbicanAPITemplate: instance.Spec.BarbicanAPI,
		DatabaseHostname:    instance.Status.DatabaseHostname,
		TransportURLSecret:  instance.Status.TransportURLSecret,
	}

	// If NodeSelector is not specified in BarbicanAPITemplate, the current
	// API instance inherits the value from the top-level CR.
	if apiSpec.NodeSelector == nil {
		apiSpec.NodeSelector = instance.Spec.NodeSelector
	}

	// If topology is not present in the underlying BarbicanAPITemplate,
	// inherit from the top-level CR
	if apiSpec.TopologyRef == nil {
		apiSpec.TopologyRef = instance.Spec.TopologyRef
	}

	// Note: The top-level .spec.apiTimeout ALWAYS overrides .spec.barbicanAPI.apiTimeout
	apiSpec.BarbicanAPITemplate.APITimeout = instance.Spec.APITimeout

	deployment := &barbicanv1beta1.BarbicanAPI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		Log.Info("Setting deployment spec to be apispec")
		deployment.Spec = apiSpec

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		// Add a finalizer to prevent user from manually removing child BarbicanAPI
		controllerutil.AddFinalizer(deployment, helper.GetFinalizer())

		return nil
	})

	return deployment, op, err
}

func (r *BarbicanReconciler) workerDeploymentCreateOrUpdate(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper) (*barbicanv1beta1.BarbicanWorker, controllerutil.OperationResult, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Creating barbican Worker spec.  transporturlsecret: '%s'", instance.Status.TransportURLSecret))
	Log.Info(fmt.Sprintf("database hostname: '%s'", instance.Status.DatabaseHostname))
	workerSpec := barbicanv1beta1.BarbicanWorkerSpec{
		BarbicanTemplate:       instance.Spec.BarbicanTemplate,
		BarbicanWorkerTemplate: instance.Spec.BarbicanWorker,
		DatabaseHostname:       instance.Status.DatabaseHostname,
		TransportURLSecret:     instance.Status.TransportURLSecret,
		TLS:                    instance.Spec.BarbicanAPI.TLS.Ca,
	}

	// If NodeSelector is not specified in BarbicanWorkerTemplate, the current
	// Worker instance inherits the value from the top-level CR.
	if workerSpec.NodeSelector == nil {
		workerSpec.NodeSelector = instance.Spec.NodeSelector
	}

	// If topology is not present in the underlying BarbicanWorkerTemplate,
	// inherit from the top-level CR
	if workerSpec.TopologyRef == nil {
		workerSpec.TopologyRef = instance.Spec.TopologyRef
	}

	deployment := &barbicanv1beta1.BarbicanWorker{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-worker", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		Log.Info("Setting deployment spec to be workerspec")
		deployment.Spec = workerSpec

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		// Add a finalizer to prevent user from manually removing child BarbicanWorker
		controllerutil.AddFinalizer(deployment, helper.GetFinalizer())

		return nil
	})

	return deployment, op, err
}

func (r *BarbicanReconciler) keystoneListenerDeploymentCreateOrUpdate(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper) (*barbicanv1beta1.BarbicanKeystoneListener, controllerutil.OperationResult, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Creating barbican KeystoneListener spec.  transporturlsecret: '%s'", instance.Status.TransportURLSecret))
	Log.Info(fmt.Sprintf("database hostname: '%s'", instance.Status.DatabaseHostname))
	keystoneListenerSpec := barbicanv1beta1.BarbicanKeystoneListenerSpec{
		BarbicanTemplate:                 instance.Spec.BarbicanTemplate,
		BarbicanKeystoneListenerTemplate: instance.Spec.BarbicanKeystoneListener,
		DatabaseHostname:                 instance.Status.DatabaseHostname,
		TransportURLSecret:               instance.Status.TransportURLSecret,
		TLS:                              instance.Spec.BarbicanAPI.TLS.Ca,
	}

	// If NodeSelector is not specified in BarbicanKeystoneListenerTemplate, the current
	// KeystoneListener instance inherits the value from the top-level CR.
	if keystoneListenerSpec.NodeSelector == nil {
		keystoneListenerSpec.NodeSelector = instance.Spec.NodeSelector
	}

	// If topology is not present in the underlying BarbicanKeystoneListenerTemplate,
	// inherit from the top-level CR
	if keystoneListenerSpec.TopologyRef == nil {
		keystoneListenerSpec.TopologyRef = instance.Spec.TopologyRef
	}

	deployment := &barbicanv1beta1.BarbicanKeystoneListener{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-keystone-listener", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		Log.Info("Setting deployment spec to be keystonelistenerspec")
		deployment.Spec = keystoneListenerSpec

		err := controllerutil.SetControllerReference(instance, deployment, r.Scheme)
		if err != nil {
			return err
		}

		// Add a finalizer to prevent user from manually removing child BarbicanKeystoneListener
		controllerutil.AddFinalizer(deployment, helper.GetFinalizer())

		return nil
	})

	return deployment, op, err
}

func (r *BarbicanReconciler) reconcileInit(
	ctx context.Context,
	instance *barbicanv1beta1.Barbican,
	helper *helper.Helper,
	serviceLabels map[string]string,
	serviceAnnotations map[string]string,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Reconciling Service '%s' init", instance.Name))

	// Service account, role, binding
	rbacRules := []rbacv1.PolicyRule{
		{
			APIGroups:     []string{"security.openshift.io"},
			ResourceNames: []string{"anyuid"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
		},
		{
			APIGroups: []string{""},
			Resources: []string{"pods"},
			Verbs:     []string{"create", "get", "list", "watch", "update", "patch", "delete"},
		},
	}
	rbacResult, err := common_rbac.ReconcileRbac(ctx, helper, instance, rbacRules)
	if err != nil {
		return rbacResult, err
	} else if (rbacResult != ctrl.Result{}) {
		return rbacResult, nil
	}

	//
	// create Keystone service and users
	//
	_, _, err = oko_secret.GetSecret(ctx, helper, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			Log.Info(fmt.Sprintf("OpenStack secret %s not found", instance.Spec.Secret))
			return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	ksSvcSpec := keystonev1.KeystoneServiceSpec{
		ServiceType:        barbican.ServiceType,
		ServiceName:        barbican.ServiceName,
		ServiceDescription: "Barbican Service",
		Enabled:            true,
		ServiceUser:        instance.Spec.ServiceUser,
		Secret:             instance.Spec.Secret,
		PasswordSelector:   instance.Spec.PasswordSelectors.Service,
	}

	ksSvc := keystonev1.NewKeystoneService(ksSvcSpec, instance.Namespace, serviceLabels, time.Duration(10)*time.Second)
	ctrlResult, err := ksSvc.CreateOrPatch(ctx, helper)
	if err != nil {
		return ctrlResult, err
	}

	// mirror the Status, Reason, Severity and Message of the latest keystoneservice condition
	// into a local condition with the type condition.KeystoneServiceReadyCondition
	c := ksSvc.GetConditions().Mirror(condition.KeystoneServiceReadyCondition)
	if c != nil {
		instance.Status.Conditions.Set(c)
	}

	if (ctrlResult != ctrl.Result{}) {
		return ctrlResult, nil
	}

	instance.Status.ServiceID = ksSvc.GetServiceID()

	if instance.Status.Hash == nil {
		instance.Status.Hash = map[string]string{}
	}

	//
	// run Barbican db sync
	//
	dbSyncHash := instance.Status.Hash[barbicanv1beta1.DbSyncHash]
	jobDef := barbican.DbSyncJob(instance, serviceLabels, serviceAnnotations)

	dbSyncjob := job.NewJob(
		jobDef,
		barbicanv1beta1.DbSyncHash,
		instance.Spec.PreserveJobs,
		time.Duration(5)*time.Second,
		dbSyncHash,
	)
	ctrlResult, err = dbSyncjob.DoJob(
		ctx,
		helper,
	)
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBSyncReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBSyncReadyRunningMessage))
		return ctrlResult, nil
	}
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBSyncReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBSyncReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if dbSyncjob.HasChanged() {
		instance.Status.Hash[barbicanv1beta1.DbSyncHash] = dbSyncjob.GetHash()
		Log.Info(fmt.Sprintf("Service '%s' - Job %s hash added - %s", instance.Name, jobDef.Name, instance.Status.Hash[barbicanv1beta1.DbSyncHash]))
	}
	instance.Status.Conditions.MarkTrue(condition.DBSyncReadyCondition, condition.DBSyncReadyMessage)

	//
	// run Barbican pkcs11-prep if needed
	//
	if slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStorePKCS11) && instance.Spec.PKCS11 != nil {
		pkcs11Hash := instance.Status.Hash[barbicanv1beta1.PKCS11PrepHash]
		jobDef := barbican.PKCS11PrepJob(instance, serviceLabels, serviceAnnotations)

		pkcs11job := job.NewJob(
			jobDef,
			barbicanv1beta1.PKCS11PrepHash,
			instance.Spec.PreserveJobs,
			time.Duration(5)*time.Second,
			pkcs11Hash,
		)
		ctrlResult, err = pkcs11job.DoJob(
			ctx,
			helper,
		)
		if (ctrlResult != ctrl.Result{}) {
			instance.Status.Conditions.Set(condition.FalseCondition(
				PKCS11PrepReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				PKCS11PrepReadyRunningMessage))
			return ctrlResult, nil
		}
		if err != nil {
			instance.Status.Conditions.Set(condition.FalseCondition(
				PKCS11PrepReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				PKCS11PrepReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
		if pkcs11job.HasChanged() {
			instance.Status.Hash[barbicanv1beta1.PKCS11PrepHash] = pkcs11job.GetHash()
			Log.Info(fmt.Sprintf("Service '%s' - Job %s hash added - %s", instance.Name, jobDef.Name, instance.Status.Hash[barbicanv1beta1.PKCS11PrepHash]))
		}
		instance.Status.Conditions.MarkTrue(PKCS11PrepReadyCondition, PKCS11PrepReadyMessage)
	} else {
		instance.Status.Conditions.MarkTrue(PKCS11PrepReadyCondition, PKCS11PrepReadyNotRunMessage)
	}

	// run Barbican pkcs11 prep - end

	// when job passed, mark NetworkAttachmentsReadyCondition ready
	instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)

	Log.Info(fmt.Sprintf("Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}

func (r *BarbicanReconciler) verifySecret(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.Barbican,
	secretName string,
	expectedFields []string,
	envVars *map[string]env.Setter,
) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	hash, result, err := oko_secret.VerifySecret(
		ctx,
		types.NamespacedName{Name: secretName, Namespace: instance.Namespace},
		expectedFields,
		h.GetClient(),
		time.Second*10)
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

	return ctrl.Result{}, nil
}

func (r *BarbicanReconciler) ensureDB(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.Barbican,
) (*mariadbv1.Database, ctrl.Result, error) {
	// ensure MariaDBAccount exists.  This account record may be created by
	// openstack-operator or the cloud operator up front without a specific
	// MariaDBDatabase configured yet.   Otherwise, a MariaDBAccount CR is
	// created here with a generated username as well as a secret with
	// generated password.   The MariaDBAccount is created without being
	// yet associated with any MariaDBDatabase.
	_, _, err := mariadbv1.EnsureMariaDBAccount(
		ctx, h, instance.Spec.DatabaseAccount,
		instance.Namespace, false, barbican.DatabaseUsernamePrefix,
	)

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			mariadbv1.MariaDBAccountReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			mariadbv1.MariaDBAccountNotReadyMessage,
			err.Error()))

		return nil, ctrl.Result{}, err
	}
	instance.Status.Conditions.MarkTrue(
		mariadbv1.MariaDBAccountReadyCondition,
		mariadbv1.MariaDBAccountReadyMessage)

	//
	// create barbican DB instance
	//
	db := mariadbv1.NewDatabaseForAccount(
		instance.Spec.DatabaseInstance, // mariadb/galera service to target
		barbican.DatabaseName,          // name used in CREATE DATABASE in mariadb
		barbican.DatabaseCRName,        // CR name for MariaDBDatabase
		instance.Spec.DatabaseAccount,  // CR name for MariaDBAccount
		instance.Namespace,             // namespace
	)

	// create or patch the DB
	ctrlResult, err := db.CreateOrPatchAll(ctx, h)

	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return db, ctrl.Result{}, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return db, ctrlResult, nil
	}
	// wait for the DB to be setup
	ctrlResult, err = db.WaitForDBCreated(ctx, h)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return db, ctrlResult, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return db, ctrlResult, nil
	}
	// update Status.DatabaseHostname, used to config the service
	instance.Status.DatabaseHostname = db.GetDatabaseHostname()
	instance.Status.Conditions.MarkTrue(condition.DBReadyCondition, condition.DBReadyMessage)
	return db, ctrlResult, nil
}
