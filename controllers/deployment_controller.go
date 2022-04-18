/*
Copyright 2022.

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
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	labels "k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/selection"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	client "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

const dynamicConfigurationLabelKey = "app.lebiller.dev/dynamic-configuration"
const dynamicConfigurationLabelValueWatch = "watch"
const configurationHashAnnotationKey = "app.lebiller.dev/configuration-hash"

var reconcilerLogger = log.Log.WithName("predicate").WithName("eventFilters")

// DeploymentReconciler reconciles a Deployment object
type DeploymentReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=deployments/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps,resources=deployments/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.11.0/pkg/reconcile
func (r *DeploymentReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.V(10).Info("Start reconciliation")

	var deployment appsv1.Deployment
	if err := r.Get(ctx, req.NamespacedName, &deployment); err != nil {
		logger.Error(err, "Unable to fetch Deployment")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var dynamicResourceVersions bytes.Buffer
	for _, volume := range deployment.Spec.Template.Spec.Volumes {
		if volume.ConfigMap != nil {
			namespacedName := types.NamespacedName{Namespace: deployment.Namespace, Name: volume.ConfigMap.Name}
			var configMap corev1.ConfigMap
			if err := r.Get(ctx, namespacedName, &configMap); err != nil {
				logger.Error(err, "Unable to fetch ConfigMap volume")
				return ctrl.Result{}, err
			}
			if val, ok := configMap.GetLabels()[dynamicConfigurationLabelKey]; ok && val == dynamicConfigurationLabelValueWatch {
				logger.Info("Found dynamic ConfigMap volume", "volume", volume.Name, "version", configMap.ResourceVersion)
				dynamicResourceVersions.WriteString(volume.Name)
				dynamicResourceVersions.WriteByte('=')
				dynamicResourceVersions.WriteString(configMap.ResourceVersion)
				dynamicResourceVersions.WriteByte(';')
			} else {
				logger.V(10).Info("Ignoring ConfigMap volume", "volume", volume.Name)
			}
		}
	}

	logger.Info("test", "test", dynamicResourceVersions.String())
	newHashValue := calculateHashValue(dynamicResourceVersions)
	if val, ok := deployment.Spec.Template.GetAnnotations()[configurationHashAnnotationKey]; !ok || val != newHashValue {
		updatedDeployment := deployment.DeepCopy()
		if updatedDeployment.Spec.Template.Annotations == nil {
			updatedDeployment.Spec.Template.Annotations = map[string]string{}
		}
		updatedDeployment.Spec.Template.Annotations[configurationHashAnnotationKey] = newHashValue
		if err := r.Patch(ctx, updatedDeployment, client.StrategicMergeFrom(&deployment)); err != nil {
			logger.Error(err, "Unable to patch Deployment")
			return ctrl.Result{}, err
		}
		logger.Info("Updated configuration hash", "hash", newHashValue, "version", deployment.GetResourceVersion())
	} else {
		logger.Info("Configuration hash is already up-to-date", "version", deployment.GetResourceVersion())
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DeploymentReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(
			&appsv1.Deployment{},
			builder.WithPredicates(
				predicate.And(predicate.GenerationChangedPredicate{}, LabeledForDynamicConfigurationPredicate{}),
			),
		).
		Watches(
			&source.Kind{Type: &corev1.ConfigMap{}},
			handler.EnqueueRequestsFromMapFunc(r.findObjectsForConfigMap),
			builder.WithPredicates(LabeledForDynamicConfigurationPredicate{}),
		).
		Complete(r)
}

func (r *DeploymentReconciler) findObjectsForConfigMap(configMap client.Object) []reconcile.Request {
	labelRequirement, err := labels.NewRequirement(dynamicConfigurationLabelKey, selection.Equals,
		[]string{dynamicConfigurationLabelValueWatch})
	if err != nil {
		return []reconcile.Request{}
	}

	watchedDeployments := &appsv1.DeploymentList{}
	listOps := &client.ListOptions{
		LabelSelector: labels.NewSelector().Add(*labelRequirement),
		Namespace:     configMap.GetNamespace(),
	}
	err = r.List(context.TODO(), watchedDeployments, listOps)
	if err != nil {
		reconcilerLogger.Error(err, "Unable to list watched Deployments")
		return []reconcile.Request{}
	}

	var requests []reconcile.Request
	for _, deployment := range watchedDeployments.Items {
		for _, volume := range deployment.Spec.Template.Spec.Volumes {
			if volume.ConfigMap != nil && volume.ConfigMap.Name == configMap.GetName() {
				requests = append(requests, reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      deployment.GetName(),
						Namespace: deployment.GetNamespace(),
					},
				})
			}
		}
	}
	return requests
}

func calculateHashValue(dynamicResourceVersions bytes.Buffer) string {
	if dynamicResourceVersions.Len() == 0 {
		return ""
	} else {
		return fmt.Sprintf("%x", sha256.Sum256(dynamicResourceVersions.Bytes()))
	}
}
