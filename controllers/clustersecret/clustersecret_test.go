package clustersecret

import (
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/argocd"
)

var _ = Describe("secret with kube config", func() {
	When("secret contains valid kube config", func() {
		var clusterSecretName = "cluster-secret-with-valid-kube-config"

		BeforeEach(func() {
			By("creating cluster secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
					Labels: map[string]string{
						clusterTypeLabel:           clusterTypeBearer,
						integrationSecretTypeLabel: integrationSecretTypeCluster,
					},
				},
				Data: map[string][]byte{
					"config": rawKubeConfig,
				},
			}
			Expect(k8sClient.Create(ctx, secret)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
				},
			}
			err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, secret))
			Expect(err).ToNot(HaveOccurred())
		})

		When("argocd secret doesn't exist", func() {
			It("should update connection annotation in cluster secret", func() {
				Eventually(func(g Gomega) {
					secret := &corev1.Secret{}
					err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
						Name:      clusterSecretName,
						Namespace: "default",
					}, secret)

					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(secret.GetAnnotations()).ShouldNot(BeEmpty())
					g.Expect(secret.GetAnnotations()[clusterSecretConnectionAnnotation]).Should(Equal("true"))
					g.Expect(secret.GetAnnotations()[clusterSecretErrorAnnotation]).Should(BeEmpty())
				}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
			})
			It("should create argocd secret based on cluster secret", func() {
				Eventually(func(g Gomega) {
					argoSecret := &corev1.Secret{}
					err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
						Name:      clusterSecretNameToArgocdSecretName(clusterSecretName),
						Namespace: "default",
					}, argoSecret)

					g.Expect(err).ShouldNot(HaveOccurred())
					g.Expect(argoSecret.Data["config"]).ShouldNot(BeEmpty())
					g.Expect(argoSecret.Data["name"]).ShouldNot(BeEmpty())
					g.Expect(argoSecret.Data["server"]).ShouldNot(BeEmpty())
					g.Expect(argoSecret.GetLabels()).ShouldNot(BeEmpty())
					g.Expect(argoSecret.GetLabels()).Should(HaveKeyWithValue(argocd.ClusterLabel, argocd.ClusterLabelVal))
				}).Should(Succeed())
			})
			AfterEach(func() {
				argoSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterSecretNameToArgocdSecretName(clusterSecretName),
						Namespace: "default",
					},
				}
				err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, argoSecret))
				Expect(err).ToNot(HaveOccurred())
			})
		})
		When("argocd secret exists", func() {
			BeforeEach(func() {
				By("creating argocd secret")
				argoSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterSecretNameToArgocdSecretName(clusterSecretName),
						Namespace: "default",
					},
				}
				Expect(k8sClient.Create(ctx, argoSecret)).ToNot(HaveOccurred())
			})
			AfterEach(func() {
				argoSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      clusterSecretNameToArgocdSecretName(clusterSecretName),
						Namespace: "default",
					},
				}
				err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, argoSecret))
				Expect(err).ToNot(HaveOccurred())
			})
			It("should update argocd secret", func() {
				Eventually(func(g Gomega) {
					argoSecret := &corev1.Secret{}
					err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
						Name:      clusterSecretNameToArgocdSecretName(clusterSecretName),
						Namespace: "default",
					}, argoSecret)

					g.Expect(err).ShouldNot(HaveOccurred())
					g.Expect(argoSecret.Data["config"]).ShouldNot(BeEmpty())
					g.Expect(argoSecret.Data["name"]).ShouldNot(BeEmpty())
					g.Expect(argoSecret.Data["server"]).ShouldNot(BeEmpty())
					g.Expect(argoSecret.GetLabels()).ShouldNot(BeEmpty())
					g.Expect(argoSecret.GetLabels()).Should(HaveKeyWithValue(argocd.ClusterLabel, argocd.ClusterLabelVal))
				}).Should(Succeed())
			})
		})
	})
	When("secret contains invalid kube config", func() {
		var clusterSecretName = "cluster-secret-with-invalid-kube-config"

		BeforeEach(func() {
			By("creating cluster secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
					Labels: map[string]string{
						clusterTypeLabel:           clusterTypeBearer,
						integrationSecretTypeLabel: integrationSecretTypeCluster,
					},
				},
				Data: map[string][]byte{
					"config": []byte("invalid kube config"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
				},
			}
			err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, secret))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should not create argocd secret", func() {
			Consistently(func(g Gomega) {
				argoSecret := &corev1.Secret{}
				err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
					Name:      clusterSecretNameToArgocdSecretName(clusterSecretName),
					Namespace: "default",
				}, argoSecret)

				g.Expect(err).Should(HaveOccurred())
				g.Expect(k8sErrors.IsNotFound(err)).To(Equal(true), "argocd secret should not be created")
			}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
		})
	})
	When("connection to cluster can't be established", func() {
		var clusterSecretName = "cluster-secret-with-invalid-kube-host"

		BeforeEach(func() {
			By("creating config with invalid host")
			cfgInvalid := rest.CopyConfig(cfg)
			cfgInvalid.Host = "https://invalid-host"
			raw, err := ConvertRestConfigToKubeConfig(cfgInvalid)
			Expect(err).ToNot(HaveOccurred())

			By("creating cluster secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
					Labels: map[string]string{
						clusterTypeLabel:           clusterTypeBearer,
						integrationSecretTypeLabel: integrationSecretTypeCluster,
					},
				},
				Data: map[string][]byte{
					"config": raw,
				},
			}
			Expect(k8sClient.Create(ctx, secret)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
				},
			}
			err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, secret))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should update connection annotation in cluster secret", func() {
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
					Name:      clusterSecretName,
					Namespace: "default",
				}, secret)

				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(secret.GetAnnotations()).ShouldNot(BeEmpty())
				g.Expect(secret.GetAnnotations()[clusterSecretConnectionAnnotation]).Should(Equal("false"))
				g.Expect(secret.GetAnnotations()[clusterSecretErrorAnnotation]).ShouldNot(BeEmpty())
			}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
		})
	})
})

var _ = Describe("secret with irsa config", func() {
	When("secret contains valid irsa config", func() {
		var clusterSecretName = "cluster-secret-with-valid-irsa-config-cluster"

		BeforeEach(func() {
			By("creating irsa secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
					Labels: map[string]string{
						clusterTypeLabel:           clusterTypeIRSA,
						integrationSecretTypeLabel: integrationSecretTypeCluster,
					},
				},
				Data: map[string][]byte{
					"config": []byte("{}"),
					"server": []byte("https://fake-cluster-success"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
				},
			}
			err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, secret))
			Expect(err).ToNot(HaveOccurred())
		})
		When("cluster secret with kube config doesn't exist", func() {
			It("should update connection annotation in irsa secret", func() {
				Eventually(func(g Gomega) {
					secret := &corev1.Secret{}
					err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
						Name:      clusterSecretName,
						Namespace: "default",
					}, secret)

					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(secret.GetAnnotations()).ShouldNot(BeEmpty())
					g.Expect(secret.GetAnnotations()[clusterSecretConnectionAnnotation]).Should(Equal("true"))
					g.Expect(secret.GetAnnotations()[clusterSecretErrorAnnotation]).Should(BeEmpty())
				}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
			})
			It("should create cluster secret with kube config based on irsa secret", func() {
				Eventually(func(g Gomega) {
					kubeConfigSecret := &corev1.Secret{}
					err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
						Name:      "cluster-secret-with-valid-irsa-config",
						Namespace: "default",
					}, kubeConfigSecret)

					g.Expect(err).ToNot(HaveOccurred())
					g.Expect(kubeConfigSecret.Data["config"]).ShouldNot(BeEmpty())
				}).Should(Succeed())
			})
			AfterEach(func() {
				kubeConfigSecret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name:      "cluster-secret-with-valid-irsa-config",
						Namespace: "default",
					},
				}
				err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, kubeConfigSecret))
				Expect(err).ToNot(HaveOccurred())
			})
		})
	})
	When("secret contains valid irsa config", func() {
		var clusterSecretName = "cluster-secret-with-invalid-irsa-config-cluster"

		BeforeEach(func() {
			By("creating irsa secret")
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
					Labels: map[string]string{
						clusterTypeLabel:           clusterTypeIRSA,
						integrationSecretTypeLabel: integrationSecretTypeCluster,
					},
				},
				Data: map[string][]byte{
					"config": []byte("{}"),
					"server": []byte("https://fake-cluster-error"),
				},
			}
			Expect(k8sClient.Create(ctx, secret)).ToNot(HaveOccurred())
		})
		AfterEach(func() {
			secret := &corev1.Secret{
				ObjectMeta: metav1.ObjectMeta{
					Name:      clusterSecretName,
					Namespace: "default",
				},
			}
			err := ctrlclient.IgnoreNotFound(k8sClient.Delete(ctx, secret))
			Expect(err).ToNot(HaveOccurred())
		})
		It("should update connection annotation in irsa secret", func() {
			Eventually(func(g Gomega) {
				secret := &corev1.Secret{}
				err := k8sClient.Get(ctx, ctrlclient.ObjectKey{
					Name:      clusterSecretName,
					Namespace: "default",
				}, secret)

				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(secret.GetAnnotations()).ShouldNot(BeEmpty())
				g.Expect(secret.GetAnnotations()[clusterSecretConnectionAnnotation]).Should(Equal("false"))
				g.Expect(secret.GetAnnotations()[clusterSecretErrorAnnotation]).ShouldNot(BeEmpty())
			}).WithTimeout(time.Second * 5).WithPolling(time.Second).Should(Succeed())
		})
	})
})
