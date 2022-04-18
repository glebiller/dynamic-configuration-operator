package controllers

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"time"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	defaultNamespace = "default"

	timeout  = time.Second * 10
	duration = time.Second * 2
	interval = time.Millisecond * 250
)

var _ = Describe("Deployment controller", func() {
	Context("With Deployment without label", func() {
		It("Should not have configuration-hash annotation", func() {
			deploymentName := "deployment-having-no-label"
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{}, true)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Consistently(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Annotations
			}, duration, interval).Should(Not(HaveKey(configurationHashAnnotationKey)))
		})
	})

	Context("With labeled Deployment having no volume", func() {
		It("Should not have configuration-hash annotation", func() {
			deploymentName := "deployment-having-no-volume"
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{}, true)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Consistently(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Annotations
			}, duration, interval).Should(Not(HaveKey(configurationHashAnnotationKey)))
		})
	})
})

func deploymentWithVolumes(deploymentName string, volumes []corev1.Volume, dynamic bool) *appsv1.Deployment {
	labels := map[string]string{}
	if dynamic {
		labels[dynamicConfigurationLabelKey] = dynamicConfigurationLabelValueWatch
	}
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      deploymentName,
			Namespace: defaultNamespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					"app": deploymentName,
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": deploymentName,
					},
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "application",
							Image: "nginx",
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}
