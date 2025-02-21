package clustersecret

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/argocd"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/aws"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/multiclusterclient"
)

const (
	// nolint:gosec // Cluster secret label.
	integrationSecretTypeLabel   = "app.edp.epam.com/secret-type"
	integrationSecretTypeCluster = "cluster"
	clusterTypeLabel             = "app.edp.epam.com/cluster-type"
	clusterTypeBearer            = "bearer"
	clusterTypeIRSA              = "irsa"

	// Generated kubeconfig secret will be updated after 10 minutes.
	// Generated token is valid for 15 minutes.
	// So we need to update kubeconfig secret before token expiration and add some time for processing.
	// https://aws.github.io/aws-eks-best-practices/security/docs/iam/#controlling-access-to-eks-clusters
	irsaSecretProcessAfter = time.Minute * 10
)

type ReconcileClusterSecret struct {
	client                client.Client
	aimAuthTokenGenerator aws.AIMAuthTokenGenerator
}

func NewReconcileClusterSecret(k8sClient client.Client, irsaToneGenerator aws.AIMAuthTokenGenerator) *ReconcileClusterSecret {
	return &ReconcileClusterSecret{client: k8sClient, aimAuthTokenGenerator: irsaToneGenerator}
}

func (r *ReconcileClusterSecret) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return needToReconcile(event.Object)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return needToReconcile(updateEvent.ObjectNew)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return needToReconcile(genericEvent.Object)
		},
	}

	err := ctrl.NewControllerManagedBy(mgr).
		For(&corev1.Secret{}, builder.WithPredicates(p)).
		Complete(r)
	if err != nil {
		return fmt.Errorf("failed to build ClusterSecret controller: %w", err)
	}

	return nil
}

//+kubebuilder:rbac:groups="",namespace=placeholder,resources=secrets,verbs=get;list;watch;update;patch;create

// Reconcile process secrets with labelapp.edp.epam.com/secret-type=cluster.
// Based on the second label app.edp.epam.com/cluster-type the secret will be processed in different ways:
// - app.edp.epam.com/cluster-type=bearer - secret contains kubeconfig and should be converted to ArgoCD cluster secret.
// - app.edp.epam.com/cluster-type=irsa - secret contains AWS IRSA configuration and should be converted to kubeconfig.
// - if not specified - secret will be treated as bearer secret.
func (r *ReconcileClusterSecret) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	l := ctrl.LoggerFrom(ctx)

	secret := &corev1.Secret{}
	if err := r.client.Get(ctx, request.NamespacedName, secret); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get Secret: %w", err)
	}

	switch secret.GetLabels()[clusterTypeLabel] {
	case clusterTypeIRSA:
		l.Info("Start processing IRSA cluster secret")

		if err := r.irsaToKubeConfigSecret(ctx, secret); err != nil {
			return reconcile.Result{}, err
		}

		l.Info("IRSA cluster secret has been processed")

		return reconcile.Result{
			RequeueAfter: irsaSecretProcessAfter,
		}, nil
	default:
		l.Info("Start processing kubeconfig cluster secret")

		if err := r.createArgoCDClusterSecret(ctx, secret); err != nil {
			return reconcile.Result{}, err
		}

		l.Info("Kubeconfig cluster secret has been processed")

		return reconcile.Result{}, nil
	}
}

// createArgoCDClusterSecret creates ArgoCD cluster secret from secret that contains kubeconfig.
func (r *ReconcileClusterSecret) createArgoCDClusterSecret(ctx context.Context, secret *corev1.Secret) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start creating ArgoCD cluster secret")

	restConf, err := multiclusterclient.ClusterSecretToRestConfig(secret)
	if err != nil {
		return fmt.Errorf("failed to convert cluster secret to rest config: %w", err)
	}

	argoClusterSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-argocd-cluster", secret.Name),
			Namespace: secret.Namespace,
		},
	}

	var res controllerutil.OperationResult

	if res, err = controllerutil.CreateOrUpdate(ctx, r.client, argoClusterSecret, func() error {
		argoClusterConf := &argocd.ClusterConfig{}
		argoClusterConf.BearerToken = restConf.BearerToken
		argoClusterConf.CAData = restConf.TLSClientConfig.CAData
		argoClusterConf.Insecure = restConf.TLSClientConfig.Insecure

		var rawConf json.RawMessage

		if rawConf, err = json.Marshal(argoClusterConf); err != nil {
			return fmt.Errorf("failed to marshal cluster config: %w", err)
		}

		argocd.AddClusterLabel(argoClusterSecret)

		argoClusterSecret.Data = map[string][]byte{
			"name":   []byte(secret.Name),
			"server": []byte(restConf.Host),
			"config": rawConf,
		}

		if metav1.GetControllerOfNoCopy(argoClusterSecret) != nil {
			return nil
		}

		if err = controllerutil.SetControllerReference(secret, argoClusterSecret, r.client.Scheme()); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		return nil
	}); err != nil {
		return fmt.Errorf("failed to create/update or update ArgoCD cluster secret: %w", err)
	}

	log.Info(fmt.Sprintf("ArgoCD cluster secret has been %s", res))

	return nil
}

// irsaToKubeConfigSecret creates secret with kube config from ArgoCD cluster secret.
// Kube config is generated from AWS IRSA configuration.
// https://argo-cd.readthedocs.io/en/stable/operator-manual/declarative-setup/#eks
// The secret will be updated after 10 minutes as the token is valid only short period of time.
func (r *ReconcileClusterSecret) irsaToKubeConfigSecret(ctx context.Context, secret *corev1.Secret) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Start converting IRSA to kubeconfig format")

	kubeConfSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.TrimSuffix(secret.Name, "-cluster"),
			Namespace: secret.Namespace,
		},
	}

	res, err := controllerutil.CreateOrUpdate(ctx, r.client, kubeConfSecret, func() error {
		kubeConf, err := argocd.ArgoIRSAClusterSecretToKubeconfig(secret, r.aimAuthTokenGenerator)
		if err != nil {
			return fmt.Errorf("failed to generate kubeconfig: %w", err)
		}

		kubeConfSecret.Data = map[string][]byte{
			"config": kubeConf,
		}

		if metav1.GetControllerOfNoCopy(kubeConfSecret) != nil {
			return nil
		}

		if err = controllerutil.SetControllerReference(secret, kubeConfSecret, r.client.Scheme()); err != nil {
			return fmt.Errorf("failed to set controller reference: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to create/update or update cluster secret: %w", err)
	}

	log.Info(fmt.Sprintf("Cluster secret has been %s", res))

	return nil
}

func needToReconcile(object client.Object) bool {
	labels := object.GetLabels()
	if labels == nil {
		return false
	}

	return labels[integrationSecretTypeLabel] == integrationSecretTypeCluster
}
