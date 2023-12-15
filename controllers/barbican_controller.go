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

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/go-logr/logr"
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
	"github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	oko_secret "github.com/openstack-k8s-operators/lib-common/modules/common/secret"
	"github.com/openstack-k8s-operators/lib-common/modules/common/util"
	mariadbv1 "github.com/openstack-k8s-operators/mariadb-operator/api/v1beta1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8s_errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicans/finalizers,verbs=update
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanapis/finalizers,verbs=update
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanworkers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanworkers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicanworkers/finalizers,verbs=update
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicankeystonelisteners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicankeystonelisteners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=barbican.openstack.org,resources=barbicankeystonelisteners/finalizers,verbs=update
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
//+kubebuilder:rbac:groups=mariadb.openstack.org,resources=mariadbdatabases,verbs=get;list;watch;create;update;patch;delete;
// service account, role, rolebinding
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=roles,verbs=get;list;watch;create;update
//+kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=rolebindings,verbs=get;list;watch;create;update
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

	// Always patch the instance status when exiting this function so we can persist any changes.
	defer func() {
		// update the Ready condition based on the sub conditions
		if instance.Status.Conditions.AllSubConditionIsTrue() {
			instance.Status.Conditions.MarkTrue(
				condition.ReadyCondition, condition.ReadyMessage)
		} else {
			// something is not ready so reset the Ready condition
			instance.Status.Conditions.MarkUnknown(
				condition.ReadyCondition, condition.InitReason, condition.ReadyInitMessage)
			// and recalculate it based on the state of the rest of the conditions
			instance.Status.Conditions.Set(
				instance.Status.Conditions.Mirror(condition.ReadyCondition))
		}
		err := helper.PatchInstance(ctx, instance)
		if err != nil {
			_err = err
			return
		}
	}()

	// If we're not deleting this and the service object doesn't have our finalizer, add it.
	if instance.DeletionTimestamp.IsZero() && controllerutil.AddFinalizer(instance, helper.GetFinalizer()) {
		return ctrl.Result{}, nil
	}

	//
	// initialize status
	//
	if instance.Status.Conditions == nil {
		instance.Status.Conditions = condition.Conditions{}
		// initialize conditions used later as Status=Unknown
		cl := condition.CreateList(
			condition.UnknownCondition(condition.DBReadyCondition, condition.InitReason, condition.DBReadyInitMessage),
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

		// Register overall status immediately to have an early feedback e.g. in the cli
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

	//
	// check for required OpenStack secret holding passwords for service/admin user and add hash to the vars map
	//
	ospSecret, hash, err := secret.GetSecret(ctx, helper, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.InputReadyCondition,
				condition.RequestedReason,
				condition.SeverityInfo,
				condition.InputReadyWaitingMessage))
			return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, fmt.Errorf("OpenStack secret %s not found", instance.Spec.Secret)
		}
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.InputReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.InputReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	// Add a prefix to the var name to avoid accidental collision with other non-secret vars.
	configVars["secret-"+ospSecret.Name] = env.SetValue(hash)

	instance.Status.Conditions.MarkTrue(condition.InputReadyCondition, condition.InputReadyMessage)
	// Setting this here at the top level
	instance.Spec.ServiceAccount = instance.RbacResourceName()

	err = r.generateServiceConfig(ctx, helper, instance, &configVars, serviceLabels)
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
	for _, netAtt := range instance.Spec.BarbicanAPI.NetworkAttachments {
		_, err := nad.GetNADWithName(ctx, helper, netAtt, instance.Namespace)
		if err != nil {
			if k8s_errors.IsNotFound(err) {
				instance.Status.Conditions.Set(condition.FalseCondition(
					condition.NetworkAttachmentsReadyCondition,
					condition.RequestedReason,
					condition.SeverityInfo,
					condition.NetworkAttachmentsReadyWaitingMessage,
					netAtt))
				return ctrl.Result{RequeueAfter: time.Second * 10}, fmt.Errorf("network-attachment-definition %s not found", netAtt)
			}
			instance.Status.Conditions.Set(condition.FalseCondition(
				condition.NetworkAttachmentsReadyCondition,
				condition.ErrorReason,
				condition.SeverityWarning,
				condition.NetworkAttachmentsReadyErrorMessage,
				err.Error()))
			return ctrl.Result{}, err
		}
	}

	// TODO: _ should be serviceAnnotations
	serviceAnnotations, err := nad.CreateNetworksAnnotation(instance.Namespace, instance.Spec.BarbicanAPI.NetworkAttachments)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed create network annotation from %s: %w",
			instance.Spec.BarbicanAPI.NetworkAttachments, err)
	}

	// Handle service init
	ctrlResult, err := r.reconcileInit(ctx, instance, helper, serviceLabels, serviceAnnotations)
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

	return ctrl.Result{}, nil
}

func (r *BarbicanReconciler) reconcileDelete(ctx context.Context, instance *barbicanv1beta1.Barbican, helper *helper.Helper) (ctrl.Result, error) {
	Log := r.GetLogger(ctx)
	Log.Info(fmt.Sprintf("Reconciling Service '%s' delete", instance.Name))

	// remove db finalizer first
	db, err := mariadbv1.GetDatabaseByName(ctx, helper, instance.Name)
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

// SetupWithManager sets up the controller with the Manager.
func (r *BarbicanReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&barbicanv1beta1.Barbican{}).
		Owns(&barbicanv1beta1.BarbicanAPI{}).
		Owns(&barbicanv1beta1.BarbicanWorker{}).
		Owns(&barbicanv1beta1.BarbicanKeystoneListener{}).
		Owns(&mariadbv1.MariaDBDatabase{}).
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
) error {
	Log := r.GetLogger(ctx)
	Log.Info("generateServiceConfigMaps - Barbican controller")

	// create Secret required for barbican input
	labels := labels.GetLabels(instance, labels.GetGroupLabel(barbican.ServiceName), serviceLabels)

	ospSecret, _, err := secret.GetSecret(ctx, h, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		return err
	}

	transportURLSecret, _, err := secret.GetSecret(ctx, h, instance.Status.TransportURLSecret, instance.Namespace)
	if err != nil {
		return err
	}

	customData := map[string]string{barbican.CustomConfigFileName: instance.Spec.CustomServiceConfig}

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

	templateParameters := map[string]interface{}{
		"DatabaseConnection": fmt.Sprintf("mysql+pymysql://%s:%s@%s/%s",
			instance.Spec.DatabaseUser,
			string(ospSecret.Data[instance.Spec.PasswordSelectors.Database]),
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

	return GenerateConfigsGeneric(ctx, h, instance, envVars, templateParameters, customData, labels, false)
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
	// create service DB instance
	//
	db := mariadbv1.NewDatabase(
		instance.Name,
		instance.Spec.DatabaseUser,
		instance.Spec.Secret,
		map[string]string{
			"dbName": instance.Spec.DatabaseInstance,
		},
	)
	// create or patch the DB
	ctrlResult, err := db.CreateOrPatchDB(
		ctx,
		helper,
	)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return ctrl.Result{}, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return ctrlResult, nil
	}
	// wait for the DB to be setup
	ctrlResult, err = db.WaitForDBCreated(ctx, helper)
	if err != nil {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.ErrorReason,
			condition.SeverityWarning,
			condition.DBReadyErrorMessage,
			err.Error()))
		return ctrlResult, err
	}
	if (ctrlResult != ctrl.Result{}) {
		instance.Status.Conditions.Set(condition.FalseCondition(
			condition.DBReadyCondition,
			condition.RequestedReason,
			condition.SeverityInfo,
			condition.DBReadyRunningMessage))
		return ctrlResult, nil
	}
	// update Status.DatabaseHostname, used to config the service
	instance.Status.DatabaseHostname = db.GetDatabaseHostname()
	instance.Status.Conditions.MarkTrue(condition.DBReadyCondition, condition.DBReadyMessage)
	// create service DB - end

	//
	// create Keystone service and users
	//
	_, _, err = oko_secret.GetSecret(ctx, helper, instance.Spec.Secret, instance.Namespace)
	if err != nil {
		if k8s_errors.IsNotFound(err) {
			return ctrl.Result{RequeueAfter: time.Duration(10) * time.Second}, fmt.Errorf("OpenStack secret %s not found", instance.Spec.Secret)
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
	ctrlResult, err = ksSvc.CreateOrPatch(ctx, helper)
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

	// when job passed, mark NetworkAttachmentsReadyCondition ready
	instance.Status.Conditions.MarkTrue(condition.NetworkAttachmentsReadyCondition, condition.NetworkAttachmentsReadyMessage)

	// run Barbican db sync - end

	Log.Info(fmt.Sprintf("Reconciled Service '%s' init successfully", instance.Name))
	return ctrl.Result{}, nil
}
