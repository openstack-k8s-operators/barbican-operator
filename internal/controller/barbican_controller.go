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

package controller

import (
	"context"
	"fmt"
	maps0 "maps"
	"slices"
	"time"

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

	"github.com/go-logr/logr"
	networkv1 "github.com/k8snetworkplumbingwg/network-attachment-definition-client/pkg/apis/k8s.cni.cncf.io/v1"
	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
	"github.com/openstack-k8s-operators/barbican-operator/internal/barbican"
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
	"github.com/openstack-k8s-operators/lib-common/modules/common/object"
	common_rbac "github.com/openstack-k8s-operators/lib-common/modules/common/rbac"
	oko_secret "github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/service"
	"github.com/openstack-k8s-operators/lib-common/modules/common/tls"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	"golang.org/x/exp/maps"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
)

const (
	// PKCS11PrepReadyCondition indicates whether PKCS11 preparation is ready
	PKCS11PrepReadyCondition = "PKCS11PrepReady"
	// PKCS11PrepReadyInitMessage is the initial message for PKCS11 prep status
	PKCS11PrepReadyInitMessage = "PKCS11 Prep job not started"
	// PKCS11PrepReadyMessage is the message when PKCS11 prep job is completed
	PKCS11PrepReadyMessage = "PKCS11 Prep job completed"
	// PKCS11PrepReadyErrorMessage is the error message template for PKCS11 prep job failures
	PKCS11PrepReadyErrorMessage = "PKCS11 Prep job error occurred %s"
	// PKCS11PrepReadyRunningMessage is the message when PKCS11 prep job is still running
	PKCS11PrepReadyRunningMessage = "PKCS11 Prep job is still running"
	// PKCS11PrepReadyNotRunMessage is the message when PKCS11 prep job has not been run
	PKCS11PrepReadyNotRunMessage = "PKCS11 Prep job not run"
)

// BarbicanReconciler reconciles a Barbican object
type BarbicanReconciler struct {
	client.Client
	Kclient   kubernetes.Interface
	Scheme    *runtime.Scheme
	APIReader client.Reader
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
//+kubebuilder:rbac:groups=core,resources=pods,verbs=get;list
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
//+kubebuilder:rbac:groups="security.openshift.io",resourceNames=anyuid;nonroot-v2,resources=securitycontextconstraints,verbs=use

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *BarbicanReconciler) Reconcile(ctx context.Context, req ctrl.Request) (result ctrl.Result, _err error) {
	Log := r.GetLogger(ctx)

	instance := &barbicanv1beta1.Barbican{}
	err := r.Get(ctx, req.NamespacedName, instance)
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

	if instance.Spec.NotificationsBus != nil {
		c := condition.UnknownCondition(
			condition.NotificationBusInstanceReadyCondition,
			condition.InitReason,
			condition.NotificationBusInstanceReadyInitMessage)
		cl.Set(c)
	}

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
	transportURL, op, err := r.transportURLCreateOrUpdate(ctx, instance, serviceLabels, nil)
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

	if transportURL.Status.SecretName == "" {
		Log.Info(fmt.Sprintf("Waiting for TransportURL %s secret to be created", transportURL.Name))
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanRabbitMQTransportURLReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			barbicanv1beta1.BarbicanRabbitMQTransportURLReadyRunningMessage))
		return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
	}

	// Set status early for first-time setup so PatchInstance persists it
	// even on early returns. During rotation (old != current), the status
	// is only updated by FinalizeSecretRotation at end of reconcile.
	if instance.Status.TransportURLSecret == "" ||
		instance.Status.TransportURLSecret == transportURL.Status.SecretName {
		instance.Status.TransportURLSecret = transportURL.Status.SecretName
	}

	if err := object.ManageSecretConsumerFinalizer(
		ctx, helper, instance.Namespace,
		transportURL.Status.SecretName,
		barbican.TransportConsumerFinalizer,
	); err != nil {
		return ctrl.Result{}, err
	}

	Log.Info(fmt.Sprintf("TransportURL secret name %s", transportURL.Status.SecretName))
	instance.Status.Conditions.MarkTrue(barbicanv1beta1.BarbicanRabbitMQTransportURLReadyCondition, barbicanv1beta1.BarbicanRabbitMQTransportURLReadyMessage)

	// end main transportURL

	//
	// create NotificationsBus transportURL CR and get the actual URL from the
	// associated secret that is created
	//

	var notificationBusInstanceURL *rabbitmqv1.TransportURL

	if instance.Spec.NotificationsBus != nil {
		// Always pass the NotificationsBus config to ensure a separate TransportURL is created,
		// even when using the same cluster as messaging (to allow different vhost/user)
		var notifOp controllerutil.OperationResult
		notificationBusInstanceURL, notifOp, err = r.transportURLCreateOrUpdate(ctx, instance, serviceLabels, instance.Spec.NotificationsBus)
		if err != nil {
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NotificationBusInstanceReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NotificationBusInstanceReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}

		if notifOp != controllerutil.OperationResultNone {
			Log.Info(fmt.Sprintf("NotificationBusInstanceURL %s successfully reconciled - operation: %s", notificationBusInstanceURL.Name, string(notifOp)))
		}

		if notificationBusInstanceURL.Status.SecretName == "" {
			Log.Info(fmt.Sprintf("Waiting for NotificationBusInstanceURL %s secret to be created", notificationBusInstanceURL.Name))
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NotificationBusInstanceReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.NotificationBusInstanceReadyRunningMessage))
			return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, nil
		}

		// Set status early for first-time setup so PatchInstance persists it
		// even on early returns. During rotation (old != current), the status
		// is only updated by FinalizeSecretRotation at end of reconcile.
		if instance.Status.NotificationsURLSecret == nil ||
			*instance.Status.NotificationsURLSecret == notificationBusInstanceURL.Status.SecretName {
			instance.Status.NotificationsURLSecret = &notificationBusInstanceURL.Status.SecretName
		}

		if err := object.ManageSecretConsumerFinalizer(
			ctx, helper, instance.Namespace,
			notificationBusInstanceURL.Status.SecretName,
			barbican.TransportConsumerFinalizer,
		); err != nil {
			return ctrl.Result{}, err
		}

		instance.Status.Conditions.MarkTrue(condition.NotificationBusInstanceReadyCondition, condition.NotificationBusInstanceReadyMessage)
	} else {
		// Notifications bus disabled. Config regenerated below no longer
		// references the notifications transport URL, so its input hash
		// changes and the Deployment rolls. Defer teardown of the
		// TransportURL and its consumer finalizer until that rollout is
		// complete (guardReady at end of reconcile), otherwise the RabbitMQ
		// user backing the secret would be revoked while pods still use it.
		instance.Status.Conditions.Remove(condition.NotificationBusInstanceReadyCondition)
	}

	// end notificationBusInstanceURL

	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	ctrlResult, err := r.verifySecret(ctx, helper, instance, instance.Spec.Secret, []string{instance.Spec.PasswordSelectors.Service}, &configVars)
	if err != nil {
		return ctrlResult, err
	}

	// check for Simple Crypto Backend secret holding the KEK
	if len(instance.Spec.EnabledSecretStores) == 0 || slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStoreSimpleCrypto) {
		fields := []string{instance.Spec.PasswordSelectors.SimpleCryptoKEK}
		fields = append(fields, instance.Spec.PasswordSelectors.SimpleCryptoAdditionalKEKs...)
		ctrlResult, err = r.verifySecret(ctx, helper, instance, instance.Spec.SimpleCryptoBackendSecret, fields, &configVars)
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

	notifBusSecretNameForConfig := ""
	if notificationBusInstanceURL != nil {
		notifBusSecretNameForConfig = notificationBusInstanceURL.Status.SecretName
	}
	err = r.generateServiceConfig(ctx, helper, instance, &configVars, serviceLabels, db, transportURL.Status.SecretName, notifBusSecretNameForConfig)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.ServiceConfigReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.ServiceConfigReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}

	// Add consumer finalizer to the new AC secret early, before deployment.
	// The old secret's finalizer is removed later (after all services deploy)
	// so that rapid rotations don't revoke a credential still in use by pods.
	if instance.Spec.Auth.ApplicationCredentialSecret != "" {
		if err := object.ManageSecretConsumerFinalizer(ctx, helper, instance.Namespace,
			instance.Spec.Auth.ApplicationCredentialSecret,
			barbican.ACConsumerFinalizer); err != nil {
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.ServiceConfigReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.ServiceConfigReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
	}
	instance.Status.Conditions.MarkTrue(condition.ServiceConfigReadyCondition, condition.ServiceConfigReadyMessage)

	// networks to attach to
	nadList := []networkv1.NetworkAttachmentDefinition{}
	for _, netAtt := range instance.Spec.BarbicanAPI.NetworkAttachments {
		nad, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				// Since the net-attach-def CR should have been manually created by the user and referenced in the spec,
				// we treat this as a warning because it means that the service will not be able to start.
				Log.Info(fmt.Sprintf("network-attachment-definition %s not found", netAtt))
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.ErrorReason,
					condition.SeverityWarning,
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
	notificationBusSecretName := ""
	if notificationBusInstanceURL != nil {
		notificationBusSecretName = notificationBusInstanceURL.Status.SecretName
	}

	expectedInputHash, err := util.ObjectHash([]string{transportURL.Status.SecretName, notificationBusSecretName})
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to compute expected input hash: %w", err)
	}
	allServicesReady := true

	barbicanAPI, opAPI, err := r.apiDeploymentCreateOrUpdate(ctx, instance, helper, transportURL.Status.SecretName, notificationBusSecretName)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanAPIReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanAPIReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if opAPI != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(opAPI)))
	}
	if barbicanAPI.Generation == barbicanAPI.Status.ObservedGeneration &&
		barbicanAPI.Status.AppliedInputSecretHash == expectedInputHash {
		c := barbicanAPI.Status.Conditions.Mirror(barbicanv1beta1.BarbicanAPIReadyCondition)
		if c != nil {
			instance.Status.Conditions.Set(c)
		}
	} else {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanAPIReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
	}
	allServicesReady = allServicesReady &&
		barbicanAPI.Generation == barbicanAPI.Status.ObservedGeneration &&
		barbicanAPI.Status.AppliedInputSecretHash == expectedInputHash &&
		barbicanAPI.Status.Conditions.IsTrue(condition.ReadyCondition)

	// create or update Barbican Worker deployment
	barbicanWorker, opWorker, err := r.workerDeploymentCreateOrUpdate(ctx, instance, helper, transportURL.Status.SecretName, notificationBusSecretName)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanWorkerReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanWorkerReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if opWorker != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(opWorker)))
	}
	if barbicanWorker.Generation == barbicanWorker.Status.ObservedGeneration &&
		barbicanWorker.Status.AppliedInputSecretHash == expectedInputHash {
		c := barbicanWorker.Status.Conditions.Mirror(barbicanv1beta1.BarbicanWorkerReadyCondition)
		if c != nil {
			instance.Status.Conditions.Set(c)
		}
	} else {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanWorkerReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
	}
	allServicesReady = allServicesReady &&
		barbicanWorker.Generation == barbicanWorker.Status.ObservedGeneration &&
		barbicanWorker.Status.AppliedInputSecretHash == expectedInputHash &&
		barbicanWorker.Status.Conditions.IsTrue(condition.ReadyCondition)

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
	barbicanKeystoneListener, opKeystoneListener, err := r.keystoneListenerDeploymentCreateOrUpdate(ctx, instance, helper, transportURL.Status.SecretName, notificationBusSecretName)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanKeystoneListenerReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			barbicanv1beta1.BarbicanKeystoneListenerReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if opKeystoneListener != controllerutil.OperationResultNone {
		Log.Info(fmt.Sprintf("Deployment %s successfully reconciled - operation: %s", instance.Name, string(opKeystoneListener)))
	}
	if barbicanKeystoneListener.Generation == barbicanKeystoneListener.Status.ObservedGeneration &&
		barbicanKeystoneListener.Status.AppliedInputSecretHash == expectedInputHash {
		c := barbicanKeystoneListener.Status.Conditions.Mirror(barbicanv1beta1.BarbicanKeystoneListenerReadyCondition)
		if c != nil {
			instance.Status.Conditions.Set(c)
		}
	} else {
		instance.Status.Conditions.Set(condition.FalseCondition(
			barbicanv1beta1.BarbicanKeystoneListenerReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DeploymentReadyRunningMessage))
	}
	allServicesReady = allServicesReady &&
		barbicanKeystoneListener.Generation == barbicanKeystoneListener.Status.ObservedGeneration &&
		barbicanKeystoneListener.Status.AppliedInputSecretHash == expectedInputHash &&
		barbicanKeystoneListener.Status.Conditions.IsTrue(condition.ReadyCondition)

	// TODO(dmendiza): Handle API endpoints

	// Update the lastObserved generation before evaluating conditions
	instance.Status.ObservedGeneration = instance.Generation
	// We reached the end of the Reconcile, update the Ready condition based on
	// the sub conditions
	if instance.Status.Conditions.AllSubConditionIsTrue() {
		instance.Status.Conditions.MarkTrue(
			condition.ReadyCondition, condition.ReadyMessage)
	}

	guardReady := allServicesReady

	// Finalize transport URL rotation
	transportSecretName, err := object.FinalizeSecretRotation(
		ctx, helper, instance.Namespace,
		instance.Status.TransportURLSecret,
		transportURL.Status.SecretName,
		barbican.TransportConsumerFinalizer,
		guardReady,
	)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.Status.TransportURLSecret = transportSecretName

	// Finalize notifications URL rotation
	if notificationBusInstanceURL != nil {
		notifStatusSecret := ""
		if instance.Status.NotificationsURLSecret != nil {
			notifStatusSecret = *instance.Status.NotificationsURLSecret
		}
		notifSecretName, err := object.FinalizeSecretRotation(
			ctx, helper, instance.Namespace,
			notifStatusSecret,
			notificationBusInstanceURL.Status.SecretName,
			barbican.TransportConsumerFinalizer,
			guardReady,
		)
		if err != nil {
			return ctrl.Result{}, err
		}
		instance.Status.NotificationsURLSecret = &notifSecretName
	} else if instance.Status.NotificationsURLSecret != nil &&
		*instance.Status.NotificationsURLSecret != "" && guardReady {
		// Notifications bus disabled and the Deployment has rolled out a
		// config that no longer references it: now it is safe to release the
		// consumer finalizer and delete the notifications TransportURL.
		if err := object.RemoveSecretConsumerFinalizer(ctx, helper, instance.Namespace,
			*instance.Status.NotificationsURLSecret, barbican.TransportConsumerFinalizer); err != nil {
			return ctrl.Result{}, err
		}
		notificationTransportURLName := fmt.Sprintf("%s-barbican-notifications-transport", instance.Name)
		if err := r.transportURLDeleted(ctx, instance, notificationTransportURLName); err != nil {
			Log.Error(err, fmt.Sprintf("Could not delete notification TransportURL %s", notificationTransportURLName))
			return ctrl.Result{}, err
		}
		instance.Status.NotificationsURLSecret = nil
	}

	// Finalize AC secret rotation
	acSecretName, err := object.FinalizeSecretRotation(
		ctx, helper, instance.Namespace,
		instance.Status.ApplicationCredentialSecret,
		instance.Spec.Auth.ApplicationCredentialSecret,
		barbican.ACConsumerFinalizer,
		guardReady,
	)
	if err != nil {
		return ctrl.Result{}, err
	}
	instance.Status.ApplicationCredentialSecret = acSecretName

	// Self-heal consumer finalizers stranded on secrets superseded during
	// rapid rotation (A -> B -> C before the workload became ready):
	// FinalizeSecretRotation only ever releases the single tracked "old"
	// secret, so any intermediate secret's finalizer would otherwise leak.
	// keep enumerates every secret that legitimately still holds the
	// finalizer; all others in the namespace are pruned.
	notifKeep := ""
	if instance.Status.NotificationsURLSecret != nil {
		notifKeep = *instance.Status.NotificationsURLSecret
	}
	currentNotifKeep := ""
	if notificationBusInstanceURL != nil {
		currentNotifKeep = notificationBusInstanceURL.Status.SecretName
	}
	if err := object.PruneSecretConsumerFinalizers(
		ctx, helper, instance.Namespace, barbican.TransportConsumerFinalizer,
		instance.Status.TransportURLSecret, transportURL.Status.SecretName,
		notifKeep, currentNotifKeep,
	); err != nil {
		return ctrl.Result{}, err
	}
	if err := object.PruneSecretConsumerFinalizers(
		ctx, helper, instance.Namespace, barbican.ACConsumerFinalizer,
		instance.Status.ApplicationCredentialSecret,
		instance.Spec.Auth.ApplicationCredentialSecret,
	); err != nil {
		return ctrl.Result{}, err
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

	// Remove consumer finalizer from AC secrets barbican was consuming.
	// Check both status and spec to handle the edge case where the reconciler
	// crashed after adding the finalizer but before updating the status.
	for _, secretName := range []string{
		instance.Status.ApplicationCredentialSecret,
		instance.Spec.Auth.ApplicationCredentialSecret,
	} {
		if err := object.RemoveSecretConsumerFinalizer(ctx, helper, instance.Namespace,
			secretName, barbican.ACConsumerFinalizer); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Remove consumer finalizer from transport secrets Barbican was consuming.
	// Check both status and the TransportURL CR to handle the case where
	// status was reverted to empty by the defer-based rotation guard.
	transportSecrets := []string{instance.Status.TransportURLSecret}
	for _, tuName := range []string{
		fmt.Sprintf("%s-barbican-transport", instance.Name),
		fmt.Sprintf("%s-barbican-notifications-transport", instance.Name),
	} {
		tu := &rabbitmqv1.TransportURL{}
		if err := r.Get(ctx, types.NamespacedName{Name: tuName, Namespace: instance.Namespace}, tu); err != nil {
			if !k8s_errors.IsNotFound(err) {
				return ctrl.Result{}, err
			}
		} else {
			transportSecrets = append(transportSecrets, tu.Status.SecretName)
		}
	}
	if instance.Status.NotificationsURLSecret != nil {
		transportSecrets = append(transportSecrets, *instance.Status.NotificationsURLSecret)
	}
	for _, secretName := range transportSecrets {
		if err := object.RemoveSecretConsumerFinalizer(ctx, helper, instance.Namespace,
			secretName, barbican.TransportConsumerFinalizer); err != nil {
			return ctrl.Result{}, err
		}
	}

	// Release any finalizer stranded on secrets superseded during rotation.
	if err := object.PruneSecretConsumerFinalizers(
		ctx, helper, instance.Namespace, barbican.TransportConsumerFinalizer,
	); err != nil {
		return ctrl.Result{}, err
	}
	if err := object.PruneSecretConsumerFinalizers(
		ctx, helper, instance.Namespace, barbican.ACConsumerFinalizer,
	); err != nil {
		return ctrl.Result{}, err
	}

	// Service is deleted so remove the finalizer.
	controllerutil.RemoveFinalizer(instance, helper.GetFinalizer())
	Log.Info(fmt.Sprintf("Reconciled Service '%s' delete successfully", instance.Name))

	return ctrl.Result{}, nil
}

// fields to index to reconcile when change
const (
	passwordSecretField                 = ".spec.secret"
	simpleCryptoBackendSecretField      = ".spec.simpleCryptoBackendSecret" // #nosec G101
	caBundleSecretNameField             = ".spec.tls.caBundleSecretName"    // #nosec G101
	tlsAPIInternalField                 = ".spec.tls.api.internal.secretName"
	tlsAPIPublicField                   = ".spec.tls.api.public.secretName"
	pkcs11LoginSecretField              = ".spec.pkcs11.loginSecret"      // #nosec G101
	pkcs11ClientDataSecretField         = ".spec.pkcs11.clientDataSecret" // #nosec G101
	topologyField                       = ".spec.topologyRef.Name"
	customServiceConfigSecretsField     = ".spec.customServiceConfigSecrets" // #nosec G101
	parentBarbicanConfigDataSecretField = ".status.parentBarbicanConfigDataSecret"
	authAppCredSecretField              = ".spec.auth.applicationCredentialSecret" // #nosec G101
)

var (
	workerWatchFields = []string{
		passwordSecretField,
		simpleCryptoBackendSecretField,
		caBundleSecretNameField,
		pkcs11LoginSecretField,
		pkcs11ClientDataSecretField,
		topologyField,
		customServiceConfigSecretsField,
		parentBarbicanConfigDataSecretField,
	}
	apiWatchFields = []string{
		passwordSecretField,
		simpleCryptoBackendSecretField,
		caBundleSecretNameField,
		tlsAPIInternalField,
		tlsAPIPublicField,
		pkcs11LoginSecretField,
		pkcs11ClientDataSecretField,
		topologyField,
		customServiceConfigSecretsField,
		parentBarbicanConfigDataSecretField,
	}
	listenerWatchFields = []string{
		passwordSecretField,
		simpleCryptoBackendSecretField,
		caBundleSecretNameField,
		topologyField,
		customServiceConfigSecretsField,
		parentBarbicanConfigDataSecretField,
	}
)

// SetupWithManager sets up the controller with the Manager.
func (r *BarbicanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	// Index authAppCredSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.Barbican{}, authAppCredSecretField, func(rawObj client.Object) []string {
		// Extract the application credential secret name from the spec, if one is provided
		cr := rawObj.(*barbicanv1beta1.Barbican)
		if cr.Spec.Auth.ApplicationCredentialSecret == "" {
			return nil
		}
		return []string{cr.Spec.Auth.ApplicationCredentialSecret}
	}); err != nil {
		return err
	}

	// Index passwordSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.Barbican{}, passwordSecretField, func(rawObj client.Object) []string {
		cr := rawObj.(*barbicanv1beta1.Barbican)
		if cr.Spec.Secret == "" {
			return nil
		}
		return []string{cr.Spec.Secret}
	}); err != nil {
		return err
	}

	// Index simpleCryptoBackendSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.Barbican{}, simpleCryptoBackendSecretField, func(rawObj client.Object) []string {
		cr := rawObj.(*barbicanv1beta1.Barbican)
		if cr.Spec.SimpleCryptoBackendSecret == "" {
			return nil
		}
		return []string{cr.Spec.SimpleCryptoBackendSecret}
	}); err != nil {
		return err
	}

	// Index pkcs11LoginSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.Barbican{}, pkcs11LoginSecretField, func(rawObj client.Object) []string {
		cr := rawObj.(*barbicanv1beta1.Barbican)
		if cr.Spec.PKCS11 == nil || cr.Spec.PKCS11.LoginSecret == "" {
			return nil
		}
		return []string{cr.Spec.PKCS11.LoginSecret}
	}); err != nil {
		return err
	}

	// Index pkcs11ClientDataSecretField
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &barbicanv1beta1.Barbican{}, pkcs11ClientDataSecretField, func(rawObj client.Object) []string {
		cr := rawObj.(*barbicanv1beta1.Barbican)
		if cr.Spec.PKCS11 == nil || cr.Spec.PKCS11.ClientDataSecret == "" {
			return nil
		}
		return []string{cr.Spec.PKCS11.ClientDataSecret}
	}); err != nil {
		return err
	}

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
		Watches(&keystonev1.KeystoneAPI{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectForSrc),
			builder.WithPredicates(keystonev1.KeystoneAPIStatusChangedPredicate)).
		Watches(&corev1.Secret{},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForSrc),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}

func (r *BarbicanReconciler) findObjectForSrc(ctx context.Context, src client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	Log := r.GetLogger(ctx)

	crList := &barbicanv1beta1.BarbicanList{}
	listOps := &client.ListOptions{
		Namespace: src.GetNamespace(),
	}
	err := r.List(ctx, crList, listOps)
	if err != nil {
		Log.Error(err, fmt.Sprintf("listing %s for namespace: %s", crList.GroupVersionKind().Kind, src.GetNamespace()))
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

	return requests
}

func (r *BarbicanReconciler) findObjectsForSrc(ctx context.Context, src client.Object) []reconcile.Request {
	requests := []reconcile.Request{}

	Log := r.GetLogger(ctx)

	for _, field := range []string{
		passwordSecretField,
		simpleCryptoBackendSecretField,
		pkcs11LoginSecretField,
		pkcs11ClientDataSecretField,
		authAppCredSecretField,
	} {
		crList := &barbicanv1beta1.BarbicanList{}
		listOps := &client.ListOptions{
			FieldSelector: fields.OneTermEqualSelector(field, src.GetName()),
			Namespace:     src.GetNamespace(),
		}
		err := r.List(ctx, crList, listOps)
		if err != nil {
			Log.Error(err, fmt.Sprintf("listing for field: %s - %s", field, src.GetName()))
			continue
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

func (r *BarbicanReconciler) generateServiceConfig(
	ctx context.Context,
	h *helper.Helper,
	instance *barbicanv1beta1.Barbican,
	envVars *map[string]env.Setter,
	serviceLabels map[string]string,
	db *mariadbv1.Database,
	transportURLSecretName string,
	notificationBusSecretName string,
) error {
	Log := r.GetLogger(ctx)
	Log.Info("generateServiceConfigMaps - Barbican controller")

	// create Secret required for barbican input
	labels := labels.GetLabels(instance, labels.GetGroupLabel(barbican.ServiceName), serviceLabels)

	ospSecret, _, err := oko_secret.GetSecret(ctx, h, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		return err
	}

	transportURLSecret, _, err := oko_secret.GetSecret(ctx, h, transportURLSecretName, instance.Namespace)
	if err != nil {
		return err
	}
	transportURLSecretData := string(transportURLSecret.Data["transport_url"])

	var tlsCfg *tls.Service
	if instance.Spec.BarbicanAPI.TLS.CaBundleSecretName != "" {
		tlsCfg = &tls.Service{}
	}
	customData := map[string]string{
		barbican.CustomConfigFileName: instance.Spec.CustomServiceConfig,
		"my.cnf":                      db.GetDatabaseClientConfig(tlsCfg), //(mschuppert) for now just get the default my.cnf
	}

	maps0.Copy(customData, instance.Spec.DefaultConfigOverwrite)
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

	templateParameters := map[string]any{
		"DatabaseConnection": fmt.Sprintf("mysql+pymysql://%s:%s@%s/%s?read_default_file=/etc/my.cnf",
			databaseAccount.Spec.UserName,
			string(databaseSecret.Data[mariadbv1.DatabasePasswordSelector]),
			instance.Status.DatabaseHostname,
			barbican.DatabaseName,
		),
		"KeystoneAuthURL":  keystoneInternalURL,
		"ServicePassword":  string(ospSecret.Data[instance.Spec.PasswordSelectors.Service]),
		"ServiceUser":      instance.Spec.ServiceUser,
		"TransportURL":     transportURLSecretData,
		"LogFile":          fmt.Sprintf("%s%s.log", barbican.BarbicanLogPath, instance.Name),
		"EnableSecureRBAC": instance.Spec.BarbicanAPI.EnableSecureRBAC,
		"Region":           keystoneAPI.GetRegion(),
	}

	templateParameters["UseApplicationCredentials"] = false
	// Retrieve Application Credential data if configured
	// This AC data will be available to all Barbican components via the shared secret
	if instance.Spec.Auth.ApplicationCredentialSecret != "" {
		acSecretObj, _, err := oko_secret.GetSecret(ctx, h, instance.Spec.Auth.ApplicationCredentialSecret, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				Log.Info("ApplicationCredential secret not found, waiting", "secret", instance.Spec.Auth.ApplicationCredentialSecret)
				return fmt.Errorf("%w: %s", ErrACSecretNotFound, instance.Spec.Auth.ApplicationCredentialSecret)
			}
			Log.Error(err, "Failed to get ApplicationCredential secret", "secret", instance.Spec.Auth.ApplicationCredentialSecret)
			return err
		}
		acID, okID := acSecretObj.Data[keystonev1.ACIDSecretKey]
		acSecretData, okSecret := acSecretObj.Data[keystonev1.ACSecretSecretKey]
		if !okID || len(acID) == 0 || !okSecret || len(acSecretData) == 0 {
			Log.Info("ApplicationCredential secret missing required keys", "secret", instance.Spec.Auth.ApplicationCredentialSecret)
			return fmt.Errorf("%w: %s", ErrACSecretMissingKeys, instance.Spec.Auth.ApplicationCredentialSecret)
		}
		templateParameters["UseApplicationCredentials"] = true
		templateParameters["ACID"] = string(acID)
		templateParameters["ACSecret"] = string(acSecretData)
		// Also add to customData so child controllers can access it
		customData["ACID"] = string(acID)
		customData["ACSecret"] = string(acSecretData)
		Log.Info("Using ApplicationCredentials auth (centralized from parent Barbican CR)", "secret", instance.Spec.Auth.ApplicationCredentialSecret)
	}

	// Set transportURL quorum queues
	templateParameters["QuorumQueues"] = string(transportURLSecret.Data["quorumqueues"]) == "true"

	// Add NotificationsURL if configured
	// Always get the separate notification secret since we always create separate TransportURLs
	var notificationInstanceURLSecret *corev1.Secret
	if notificationBusSecretName != "" {
		notificationInstanceURLSecret, _, err = oko_secret.GetSecret(ctx, h, notificationBusSecretName, instance.Namespace)
		if err != nil {
			return err
		}
		templateParameters["NotificationsURL"] = string(notificationInstanceURLSecret.Data["transport_url"])
	}

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
	}

	// Set simpleCrypto parameters
	if len(instance.Spec.EnabledSecretStores) == 0 || slices.Contains(instance.Spec.EnabledSecretStores, barbicanv1beta1.SecretStoreSimpleCrypto) {
		simpleCryptoSecret, _, err := oko_secret.GetSecret(ctx, h, instance.Spec.SimpleCryptoBackendSecret, instance.Namespace)
		if err != nil {
			return err
		}
		keks := []string{string(simpleCryptoSecret.Data[instance.Spec.PasswordSelectors.SimpleCryptoKEK])}
		for _, secretField := range instance.Spec.PasswordSelectors.SimpleCryptoAdditionalKEKs {
			keks = append(keks, string(simpleCryptoSecret.Data[secretField]))
		}
		templateParameters["SimpleCryptoKEKs"] = keks
	}

	// create httpd  vhost template parameters
	httpdVhostConfig := map[string]any{}
	for _, endpt := range []service.Endpoint{service.EndpointInternal, service.EndpointPublic} {
		endptConfig := map[string]any{}
		endptConfig["ServerName"] = fmt.Sprintf("%s-%s.%s.svc", barbican.ServiceName, endpt.String(), instance.Namespace)
		endptConfig["TLS"] = false // default TLS to false, and set it bellow to true if enabled
		if instance.Spec.BarbicanAPI.TLS.API.Enabled(endpt) {
			endptConfig["TLS"] = true
			endptConfig["SSLCertificateFile"] = fmt.Sprintf("/etc/pki/tls/certs/%s.crt", endpt.String())
			endptConfig["SSLCertificateKeyFile"] = fmt.Sprintf("/etc/pki/tls/private/%s.key", endpt.String())
		}
		httpdVhostConfig[endpt.String()] = endptConfig
	}
	templateParameters["VHosts"] = httpdVhostConfig
	templateParameters["TimeOut"] = instance.Spec.APITimeout

	return GenerateConfigsGeneric(ctx, h, instance, envVars, templateParameters, customData, labels, true, []string{"ssl.conf"})
}

func (r *BarbicanReconciler) transportURLCreateOrUpdate(
	ctx context.Context,
	instance *barbicanv1beta1.Barbican,
	serviceLabels map[string]string,
	rabbitmqConfig *rabbitmqv1.RabbitMqConfig,
) (*rabbitmqv1.TransportURL, controllerutil.OperationResult, error) {
	// Default values for regular messagingBus transportURL
	rmqName := fmt.Sprintf("%s-barbican-transport", instance.Name)
	config := &instance.Spec.MessagingBus

	// When rabbitmqConfig is passed (notificationsBus use case)
	// update the default rmqName and use the provided config
	if rabbitmqConfig != nil {
		rmqName = fmt.Sprintf("%s-barbican-notifications-transport", instance.Name)
		config = rabbitmqConfig
	}

	transportURL := &rabbitmqv1.TransportURL{
		ObjectMeta: metav1.ObjectMeta{
			Name:      rmqName,
			Namespace: instance.Namespace,
			Labels:    serviceLabels,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, transportURL, func() error {
		transportURL.Spec.RabbitmqClusterName = config.Cluster
		// Always set Username and Vhost to allow clearing/resetting them
		// The infra-operator TransportURL controller handles empty values:
		// - Empty Username: uses default cluster admin credentials
		// - Empty Vhost: defaults to "/" vhost
		transportURL.Spec.Username = config.User
		transportURL.Spec.Vhost = config.Vhost
		return controllerutil.SetControllerReference(instance, transportURL, r.Scheme)
	})

	return transportURL, op, err
}

func (r *BarbicanReconciler) transportURLDeleted(
	ctx context.Context,
	instance *barbicanv1beta1.Barbican,
	transportURLName string,
) error {
	Log := r.GetLogger(ctx)
	transportURL := &rabbitmqv1.TransportURL{
		ObjectMeta: metav1.ObjectMeta{
			Name:      transportURLName,
			Namespace: instance.Namespace,
		},
	}

	err := r.Delete(ctx, transportURL)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return nil
		}
		Log.Info(fmt.Sprintf("Could not delete TransportURL %s err: %s", transportURLName, err))
		return err
	}

	Log.Info("Deleted transportURL", ":", transportURLName)

	return nil
}

func (r *BarbicanReconciler) apiDeploymentCreateOrUpdate(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper, transportURLSecret string, notificationBusSecretName string) (*barbicanv1beta1.BarbicanAPI, controllerutil.OperationResult, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Creating barbican API spec.  transporturlsecret: '%s'", transportURLSecret))
	Log.Info(fmt.Sprintf("database hostname: '%s'", instance.Status.DatabaseHostname))
	apiSpec := barbicanv1beta1.BarbicanAPISpec{
		BarbicanTemplate:    instance.Spec.BarbicanTemplate,
		BarbicanAPITemplate: instance.Spec.BarbicanAPI,
		DatabaseHostname:    instance.Status.DatabaseHostname,
		TransportURLSecret:  transportURLSecret,
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
	apiSpec.APITimeout = instance.Spec.APITimeout

	deployment := &barbicanv1beta1.BarbicanAPI{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-api", instance.Name),
			Namespace: instance.Namespace,
		},
	}

	op, err := controllerutil.CreateOrUpdate(ctx, r.Client, deployment, func() error {
		Log.Info("Setting deployment spec to be apispec")
		deployment.Spec = apiSpec

		if instance.Spec.NotificationsBus != nil {
			deployment.Spec.NotificationsURLSecret = notificationBusSecretName
		}

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

func (r *BarbicanReconciler) workerDeploymentCreateOrUpdate(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper, transportURLSecret string, notificationBusSecretName string) (*barbicanv1beta1.BarbicanWorker, controllerutil.OperationResult, error) {
	Log := r.GetLogger(ctx)

	Log.Info(fmt.Sprintf("Creating barbican Worker spec.  transporturlsecret: '%s'", transportURLSecret))
	Log.Info(fmt.Sprintf("database hostname: '%s'", instance.Status.DatabaseHostname))
	workerSpec := barbicanv1beta1.BarbicanWorkerSpec{
		BarbicanTemplate:       instance.Spec.BarbicanTemplate,
		BarbicanWorkerTemplate: instance.Spec.BarbicanWorker,
		DatabaseHostname:       instance.Status.DatabaseHostname,
		TransportURLSecret:     transportURLSecret,
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

		if instance.Spec.NotificationsBus != nil {
			deployment.Spec.NotificationsURLSecret = notificationBusSecretName
		}

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

func (r *BarbicanReconciler) keystoneListenerDeploymentCreateOrUpdate(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper, transportURLSecret string, notificationBusSecretName string) (*barbicanv1beta1.BarbicanKeystoneListener, controllerutil.OperationResult, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Creating barbican KeystoneListener spec.  transporturlsecret: '%s'", transportURLSecret))
	Log.Info(fmt.Sprintf("database hostname: '%s'", instance.Status.DatabaseHostname))
	keystoneListenerSpec := barbicanv1beta1.BarbicanKeystoneListenerSpec{
		BarbicanTemplate:                 instance.Spec.BarbicanTemplate,
		BarbicanKeystoneListenerTemplate: instance.Spec.BarbicanKeystoneListener,
		DatabaseHostname:                 instance.Status.DatabaseHostname,
		TransportURLSecret:               transportURLSecret,
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

		if instance.Spec.NotificationsBus != nil {
			deployment.Spec.NotificationsURLSecret = notificationBusSecretName
		}

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
			ResourceNames: []string{"anyuid", "nonroot-v2"},
			Resources:     []string{"securitycontextconstraints"},
			Verbs:         []string{"use"},
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

	// All expectedFields are password fields, so we associate each with a
	// password validator to ensure invalid detected patterns are rejected.
	validateFields := map[string]oko_secret.Validator{}
	for _, f := range expectedFields {
		validateFields[f] = oko_secret.PasswordValidator{}
	}
	hash, result, err := oko_secret.VerifySecretFields(
		ctx,
		types.NamespacedName{Name: secretName, Namespace: instance.Namespace},
		validateFields,
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
		// We treat this as a warning because it means that the service will not be able to start
		// while we are waiting for the secret to be created manually by the user.
		Log.Info(fmt.Sprintf("OpenStack secret %s not found", secretName))
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
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

func cleanupOldDeployment(
	ctx context.Context,
	c client.Client,
	owner client.Object,
	oldName string,
) error {
	// Retrieve the old deployment to check ownership
	oldDep := &appsv1.Deployment{}
	key := types.NamespacedName{
		Name:      oldName,
		Namespace: owner.GetNamespace(),
	}
	if err := c.Get(ctx, key, oldDep); err != nil {
		return client.IgnoreNotFound(err)
	}

	if !object.CheckOwnerRefExist(owner.GetUID(), oldDep.OwnerReferences) {
		return nil
	}

	// If it’s owned, delete the old deployment
	if err := c.Delete(ctx, oldDep); err != nil && !k8s_errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete old deployment '%s': %w", oldName, err)
	}

	return nil
}
