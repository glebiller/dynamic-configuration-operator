package controllers

import (
	"context"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// +kubebuilder:docs-gen:collapse=Imports

const (
	configMapNameStaticPrefix  = "configmap-static-"
	configMapNameDynamicPrefix = "configmap-dynamic-"
)

var _ = Describe("Deployment controller with ConfigMap", func() {
	var (
		configMapStaticName  string
		configMapDynamicName string
	)

	BeforeEach(func() {
		ctx := context.Background()

		configMapStaticName = configMapNameStaticPrefix + RandomSuffix()
		configMapStatic := configMapWithData(configMapStaticName, map[string]string{"key-static": "value-static"}, false)
		Expect(k8sClient.Create(ctx, configMapStatic)).Should(Succeed())

		configMapDynamicName = configMapNameDynamicPrefix + RandomSuffix()
		configMapDynamic := configMapWithData(configMapDynamicName, map[string]string{"key-dynamic": "value-dynamic"}, true)
		Expect(k8sClient.Create(ctx, configMapDynamic)).Should(Succeed())
	})

	Context("With labeled Deployment having one ConfigMap volume without dynamic label", func() {
		It("Should have an empty value configuration-hash annotation", func() {
			deploymentName := "deployment-having-one-configmap-static"
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{
				{
					Name: "configmap-static",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapStaticName,
							},
						},
					},
				},
			}, true)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Spec.Template.Annotations
			}, timeout, interval).Should(HaveKeyWithValue(configurationHashAnnotationKey, ""))
		})
	})

	Context("With labeled Deployment having one ConfigMap volume with dynamic label", func() {
		It("Should have configuration-hash annotation", func() {
			deploymentName := "deployment-having-one-configmap-dynamic"
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{
				{
					Name: "configmap-dynamic",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapDynamicName,
							},
						},
					},
				},
			}, true)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())

			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Spec.Template.Annotations
			}, timeout, interval).Should(HaveKey(configurationHashAnnotationKey))
		})
	})

	Context("With labeled Deployment having one ConfigMap volume with dynamic label and one without", func() {
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "deployment-having-one-configmap-dynamic-and-one-static-" + RandomSuffix()
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{
				{
					Name: "configmap-static",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapStaticName,
							},
						},
					},
				},
				{
					Name: "configmap-dynamic",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &corev1.ConfigMapVolumeSource{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: configMapDynamicName,
							},
						},
					},
				},
			}, true)
			Expect(k8sClient.Create(ctx, deployment)).Should(Succeed())
		})

		It("Should have configuration-hash annotation", func() {
			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Spec.Template.Annotations
			}, timeout, interval).Should(HaveKey(configurationHashAnnotationKey))
		})

		It("Should update configuration-hash annotation when dynamic ConfigMap is updated", func() {
			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Spec.Template.Annotations
			}, timeout, interval).Should(HaveKey(configurationHashAnnotationKey))

			originalHash := createdDeployment.Spec.Template.Annotations[configurationHashAnnotationKey]

			configMapNamespaceName := types.NamespacedName{Name: configMapDynamicName, Namespace: defaultNamespace}
			existingConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, configMapNamespaceName, existingConfigMap)).To(Succeed())

			existingConfigMap.Data = map[string]string{"dynamic-new-key": "dynamic-new-value"}
			Expect(k8sClient.Update(ctx, existingConfigMap)).To(Succeed())

			Eventually(func() string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return originalHash
				}
				return createdDeployment.Spec.Template.Annotations[configurationHashAnnotationKey]
			}, timeout, interval).Should(Not(Equal(originalHash)))
		})

		It("Should not change configuration-hash annotation when static ConfigMap is updated", func() {
			deploymentNamespaceName := types.NamespacedName{Name: deploymentName, Namespace: defaultNamespace}
			createdDeployment := &appsv1.Deployment{}
			Eventually(func() map[string]string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return nil
				}
				return createdDeployment.Spec.Template.Annotations
			}, timeout, interval).Should(HaveKey(configurationHashAnnotationKey))

			originalHash := createdDeployment.Spec.Template.Annotations[configurationHashAnnotationKey]

			configMapNamespaceName := types.NamespacedName{Name: configMapStaticName, Namespace: defaultNamespace}
			existingConfigMap := &corev1.ConfigMap{}
			Expect(k8sClient.Get(ctx, configMapNamespaceName, existingConfigMap)).To(Succeed())

			existingConfigMap.Data = map[string]string{"static-new-key": "static-new-value"}
			Expect(k8sClient.Update(ctx, existingConfigMap)).To(Succeed())

			Consistently(func() string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return originalHash
				}
				return createdDeployment.Spec.Template.Annotations[configurationHashAnnotationKey]
			}, duration, interval).Should(Equal(originalHash))
		})
	})
})

func configMapWithData(configMapName string, data map[string]string, dynamic bool) *corev1.ConfigMap {
	labels := map[string]string{}
	if dynamic {
		labels[dynamicConfigurationLabelKey] = dynamicConfigurationLabelValueWatch
	}
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "v1",
			APIVersion: "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      configMapName,
			Namespace: defaultNamespace,
			Labels:    labels,
		},
		Data: data,
	}
}
