package clustersecret

import (
	"context"
	"encoding/json"
	"fmt"

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

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/multiclusterclient"
)

const (
	// nolint:gosec // Cluster secret label.
	integrationSecretTypeLabel    = "app.edp.epam.com/secret-type"
	integrationSecretTypeLabelVal = "cluster"
)

type ReconcileClusterSecret struct {
	client client.Client
}

func NewReconcileClusterSecret(k8sClient client.Client) *ReconcileClusterSecret {
	return &ReconcileClusterSecret{client: k8sClient}
}

func (r *ReconcileClusterSecret) SetupWithManager(mgr ctrl.Manager) error {
	p := predicate.Funcs{
		CreateFunc: func(event event.CreateEvent) bool {
			return hasClusterSecretLabelLabel(event.Object)
		},
		DeleteFunc: func(deleteEvent event.DeleteEvent) bool {
			return false
		},
		UpdateFunc: func(updateEvent event.UpdateEvent) bool {
			return hasClusterSecretLabelLabel(updateEvent.ObjectNew)
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return hasClusterSecretLabelLabel(genericEvent.Object)
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

// Reconcile process secrets with app.edp.epam.com/secret-type: cluster label.
func (r *ReconcileClusterSecret) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	secret := &corev1.Secret{}
	if err := r.client.Get(ctx, request.NamespacedName, secret); err != nil {
		if k8sErrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}

		return reconcile.Result{}, fmt.Errorf("failed to get Secret: %w", err)
	}

	if err := r.createArgoCDClusterSecret(ctx, secret); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

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
		argoClusterConf := &ClusterConfig{}
		argoClusterConf.BearerToken = restConf.BearerToken
		argoClusterConf.CAData = restConf.TLSClientConfig.CAData
		argoClusterConf.Insecure = restConf.TLSClientConfig.Insecure

		var rawConf json.RawMessage

		if rawConf, err = json.Marshal(argoClusterConf); err != nil {
			return fmt.Errorf("failed to marshal cluster config: %w", err)
		}

		addClusterLabel(argoClusterSecret)

		argoClusterSecret.Data = map[string][]byte{
			"name":   []byte(secret.Name),
			"server": []byte(restConf.Host),
			"config": rawConf,
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

func hasClusterSecretLabelLabel(object client.Object) bool {
	return object.GetLabels()[integrationSecretTypeLabel] == integrationSecretTypeLabelVal
}

func addClusterLabel(argoClusterSecret *corev1.Secret) {
	labels := argoClusterSecret.GetLabels()
	if labels == nil {
		labels = make(map[string]string, 1)
	}

	labels[argoCDClusterLabel] = argoCDClusterLabelVal

	argoClusterSecret.SetLabels(labels)
}
