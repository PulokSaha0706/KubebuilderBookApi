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

package controller

import (
	"context"
	"fmt"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/utils/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	corev1alpha1 "github.com/PulokSaha0706/kubebuilder-bookapi/api/v1alpha1"
)

// KlusterReconciler reconciles a Kluster object
type KlusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=core.pulok.dev,resources=klusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core.pulok.dev,resources=klusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=core.pulok.dev,resources=klusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Kluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *KlusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := log.FromContext(ctx)

	log.Info("Reconciling Kluster", "namespace", req.Namespace, "name", req.Name)

	var kluster corev1alpha1.Kluster
	if err := r.Get(ctx, req.NamespacedName, &kluster); err != nil {
		if errors.IsNotFound(err) {
			log.Info("Kluster resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Kluster")
		return ctrl.Result{}, err
	}

	log.Info("Fetched Kluster", "spec", kluster.Spec)

	// Use fields from Kluster Spec to customize Deployment
	replicas := kluster.Spec.Replicas
	if replicas == nil {
		// Provide default replica count
		replicas = pointer.Int32Ptr(1)
	}

	image := kluster.Spec.Image
	if image == "" {
		image = "puloksaha/bookapi:latest"
	}

	port := kluster.Spec.Port
	if port == 0 {
		port = 9090
	}

	// Define Deployment object
	deployment := appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bookapi-deployment",
			Namespace: req.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": "bookapi"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{"app": "bookapi"},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{{
						Name:  "bookapi",
						Image: image,
						Args:  []string{"start", "-p", fmt.Sprint(port)},
						Ports: []corev1.ContainerPort{{
							ContainerPort: port,
						}},
					}},
				},
			},
		},
	}

	// Set owner reference so deployment is deleted with Kluster
	if err := ctrl.SetControllerReference(&kluster, &deployment, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference on Deployment")
		return ctrl.Result{}, err
	}

	var existingDeployment appsv1.Deployment
	err := r.Get(ctx, types.NamespacedName{Name: deployment.Name, Namespace: deployment.Namespace}, &existingDeployment)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Deployment", "name", deployment.Name)
		if err := r.Create(ctx, &deployment); err != nil {
			log.Error(err, "Failed to create Deployment")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	} else {
		// Optionally: you can update Deployment if spec differs here
		log.Info("Deployment already exists", "name", deployment.Name)
	}

	// Define Service object
	service := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "bookapi-service",
			Namespace: req.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeNodePort,
			Selector: map[string]string{
				"app": "bookapi",
			},
			Ports: []corev1.ServicePort{{
				Port:       port,
				TargetPort: intstr.FromInt(int(port)),
				NodePort:   30090,
				Protocol:   corev1.ProtocolTCP,
			}},
		},
	}

	if err := ctrl.SetControllerReference(&kluster, service, r.Scheme); err != nil {
		log.Error(err, "Failed to set owner reference on Service")
		return ctrl.Result{}, err
	}

	var existingService corev1.Service
	err = r.Get(ctx, types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, &existingService)
	if err != nil && errors.IsNotFound(err) {
		log.Info("Creating Service", "name", service.Name)
		if err := r.Create(ctx, service); err != nil {
			log.Error(err, "Failed to create Service")
			return ctrl.Result{}, err
		}
	} else if err != nil {
		log.Error(err, "Failed to get Service")
		return ctrl.Result{}, err
	} else {
		log.Info("Service already exists", "name", service.Name)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *KlusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&corev1alpha1.Kluster{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
