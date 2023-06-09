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

	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	"webhook-operator/api/v1alpha1"
	ldapv1alpha1 "webhook-operator/api/v1alpha1"
)

// LdapconfigReconciler reconciles a Ldapconfig object
type LdapconfigReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=ldap.tokenservice.com,resources=ldapconfigs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=ldap.tokenservice.com,resources=ldapconfigs/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=ldap.tokenservice.com,resources=ldapconfigs/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Ldapconfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *LdapconfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	lg := log.FromContext(ctx)

	ldapinst := &ldapv1alpha1.Ldapconfig{}
	err := r.Get(ctx, req.NamespacedName, ldapinst)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			lg.Info("ldapconfig not found so creating one")
			cfg := &ldapv1alpha1.Ldapconfig{
				ObjectMeta: metav1.ObjectMeta{Name: "ldap", Namespace: "webhook-service"},
				Spec:       ldapv1alpha1.LdapconfigSpec{}}
			r.Create(ctx, cfg)
			return ctrl.Result{}, nil
		} else {
			lg.Error(err, "An error occured while getting ldap config ")
			return ctrl.Result{}, err
		}
	}
	cfg := &ldapv1alpha1.Ldapconfig{}
	err = r.Get(ctx, req.NamespacedName, cfg)
	if (v1alpha1.LdapconfigSpec{}) == cfg.Spec {
		lg.Info("Found empty ldap config, nothing to do")
		return ctrl.Result{}, nil
	}

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *LdapconfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ldapv1alpha1.Ldapconfig{}).
		Complete(r)
}
