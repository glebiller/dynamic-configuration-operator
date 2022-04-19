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
	secretNameStaticPrefix  = "secret-static-"
	secretNameDynamicPrefix = "secret-dynamic-"
)

var _ = Describe("Deployment controller with Secret", func() {
	var (
		secretStaticName  string
		secretDynamicName string
	)

	BeforeEach(func() {
		ctx := context.Background()

		secretStaticName = secretNameStaticPrefix + RandomSuffix()
		secretStatic := secretWithData(secretStaticName, map[string]string{"key-static": "value-static"}, false)
		Expect(k8sClient.Create(ctx, secretStatic)).Should(Succeed())

		secretDynamicName = secretNameDynamicPrefix + RandomSuffix()
		secretDynamic := secretWithData(secretDynamicName, map[string]string{"key-dynamic": "value-dynamic"}, true)
		Expect(k8sClient.Create(ctx, secretDynamic)).Should(Succeed())
	})

	Context("With labeled Deployment having one Secret volume without dynamic label", func() {
		It("Should have an empty value configuration-hash annotation", func() {
			deploymentName := "deployment-having-one-secret-static"
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{
				{
					Name: "secret-static",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretStaticName,
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

	Context("With labeled Deployment having one Secret volume with dynamic label", func() {
		It("Should have configuration-hash annotation", func() {
			deploymentName := "deployment-having-one-secret-dynamic"
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{
				{
					Name: "secret-dynamic",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretDynamicName,
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

	Context("With labeled Deployment having one Secret volume with dynamic label and one without", func() {
		var deploymentName string

		BeforeEach(func() {
			deploymentName = "deployment-having-one-secret-dynamic-and-one-static-" + RandomSuffix()
			deployment := deploymentWithVolumes(deploymentName, []corev1.Volume{
				{
					Name: "secret-static",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretStaticName,
						},
					},
				},
				{
					Name: "secret-dynamic",
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName: secretDynamicName,
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

		It("Should update configuration-hash annotation when dynamic Secret is updated", func() {
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

			secretNamespaceName := types.NamespacedName{Name: secretDynamicName, Namespace: defaultNamespace}
			existingSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, secretNamespaceName, existingSecret)).To(Succeed())

			existingSecret.StringData = map[string]string{"dynamic-new-key": "dynamic-new-value"}
			Expect(k8sClient.Update(ctx, existingSecret)).To(Succeed())

			Eventually(func() string {
				err := k8sClient.Get(ctx, deploymentNamespaceName, createdDeployment)
				if err != nil {
					return originalHash
				}
				return createdDeployment.Spec.Template.Annotations[configurationHashAnnotationKey]
			}, timeout, interval).Should(Not(Equal(originalHash)))
		})

		It("Should not change configuration-hash annotation when static Secret is updated", func() {
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

			secretNamespaceName := types.NamespacedName{Name: secretStaticName, Namespace: defaultNamespace}
			existingSecret := &corev1.Secret{}
			Expect(k8sClient.Get(ctx, secretNamespaceName, existingSecret)).To(Succeed())

			existingSecret.StringData = map[string]string{"static-new-key": "static-new-value"}
			Expect(k8sClient.Update(ctx, existingSecret)).To(Succeed())

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

func secretWithData(secretName string, data map[string]string, dynamic bool) *corev1.Secret {
	labels := map[string]string{}
	if dynamic {
		labels[dynamicConfigurationLabelKey] = dynamicConfigurationLabelValueWatch
	}
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "v1",
			APIVersion: "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: defaultNamespace,
			Labels:    labels,
		},
		StringData: data,
	}
}
