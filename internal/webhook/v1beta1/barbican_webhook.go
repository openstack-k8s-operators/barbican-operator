/*
Copyright 2025.

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

// Package v1beta1 contains webhook implementations for the Barbican v1beta1 API.
package v1beta1

import (
	"context"
	"errors"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	barbicanv1beta1 "github.com/openstack-k8s-operators/barbican-operator/api/v1beta1"
)

var (
	// ErrUnexpectedObject is returned when an unexpected object type is received
	ErrUnexpectedObject = errors.New("unexpected object type")
)

// nolint:unused
// log is for logging in this package.
var barbicanlog = logf.Log.WithName("barbican-resource")

// SetupBarbicanWebhookWithManager registers the webhook for Barbican in the manager.
func SetupBarbicanWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).For(&barbicanv1beta1.Barbican{}).
		WithValidator(&BarbicanCustomValidator{}).
		WithDefaulter(&BarbicanCustomDefaulter{}).
		Complete()
}

// TODO(user): EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-barbican-openstack-org-v1beta1-barbican,mutating=true,failurePolicy=fail,sideEffects=None,groups=barbican.openstack.org,resources=barbicans,verbs=create;update,versions=v1beta1,name=mbarbican-v1beta1.kb.io,admissionReviewVersions=v1

// BarbicanCustomDefaulter struct is responsible for setting default values on the custom resource of the
// Kind Barbican when those are created or updated.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as it is used only for temporary operations and does not need to be deeply copied.
type BarbicanCustomDefaulter struct {
	// TODO(user): Add more fields as needed for defaulting
}

var _ webhook.CustomDefaulter = &BarbicanCustomDefaulter{}

// Default implements webhook.CustomDefaulter so a webhook will be registered for the Kind Barbican.
func (d *BarbicanCustomDefaulter) Default(_ context.Context, obj runtime.Object) error {
	barbican, ok := obj.(*barbicanv1beta1.Barbican)

	if !ok {
		return fmt.Errorf("expected an Barbican object but got %T: %w", obj, ErrUnexpectedObject)
	}
	barbicanlog.Info("Defaulting for Barbican", "name", barbican.GetName())

	barbican.Default()

	return nil
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// NOTE: The 'path' attribute must follow a specific pattern and should not be modified directly here.
// Modifying the path for an invalid path can cause API server errors; failing to locate the webhook.
// +kubebuilder:webhook:path=/validate-barbican-openstack-org-v1beta1-barbican,mutating=false,failurePolicy=fail,sideEffects=None,groups=barbican.openstack.org,resources=barbicans,verbs=create;update,versions=v1beta1,name=vbarbican-v1beta1.kb.io,admissionReviewVersions=v1

// BarbicanCustomValidator struct is responsible for validating the Barbican resource
// when it is created, updated, or deleted.
//
// NOTE: The +kubebuilder:object:generate=false marker prevents controller-gen from generating DeepCopy methods,
// as this struct is used only for temporary operations and does not need to be deeply copied.
type BarbicanCustomValidator struct {
	// TODO(user): Add more fields as needed for validation
}

var _ webhook.CustomValidator = &BarbicanCustomValidator{}

// ValidateCreate implements webhook.CustomValidator so a webhook will be registered for the type Barbican.
func (v *BarbicanCustomValidator) ValidateCreate(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	barbican, ok := obj.(*barbicanv1beta1.Barbican)
	if !ok {
		return nil, fmt.Errorf("expected a Barbican object but got %T: %w", obj, ErrUnexpectedObject)
	}
	barbicanlog.Info("Validation for Barbican upon creation", "name", barbican.GetName())

	return barbican.ValidateCreate()
}

// ValidateUpdate implements webhook.CustomValidator so a webhook will be registered for the type Barbican.
func (v *BarbicanCustomValidator) ValidateUpdate(_ context.Context, oldObj, newObj runtime.Object) (admission.Warnings, error) {
	barbican, ok := newObj.(*barbicanv1beta1.Barbican)
	if !ok {
		return nil, fmt.Errorf("expected a Barbican object for the newObj but got %T: %w", newObj, ErrUnexpectedObject)
	}
	barbicanlog.Info("Validation for Barbican upon update", "name", barbican.GetName())

	return barbican.ValidateUpdate(oldObj)
}

// ValidateDelete implements webhook.CustomValidator so a webhook will be registered for the type Barbican.
func (v *BarbicanCustomValidator) ValidateDelete(_ context.Context, obj runtime.Object) (admission.Warnings, error) {
	barbican, ok := obj.(*barbicanv1beta1.Barbican)
	if !ok {
		return nil, fmt.Errorf("expected a Barbican object but got %T: %w", obj, ErrUnexpectedObject)
	}
	barbicanlog.Info("Validation for Barbican upon deletion", "name", barbican.GetName())

	return barbican.ValidateDelete()
}
