package kubernetes

import (
	"crypto/rand"
	"fmt"
	"github.com/pkg/errors"
	rbacV1 "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/rest"
	"math/big"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
)

var log = logf.Log.WithName("kubernetes_service")

// K8SService struct for K8S platform service
type K8SService struct {
	coreClient *coreV1Client.CoreV1Client
	rbacClient *rbacV1Client.RbacV1Client
}

// New initializes K8SService
func New(config *rest.Config) (*K8SService, error) {
	coreClient, err := coreV1Client.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to init core client for K8S")
	}

	rbacClient, err := rbacV1Client.NewForConfig(config)
	if err != nil {
		return nil, errors.Wrap(err, "Failed to init RBAC client for K8S")
	}

	return &K8SService{
		coreClient,
		rbacClient}, nil
}

// CreateRoleBinding creates RoleBinding
func (service K8SService) CreateRoleBinding(edpName string, namespace string, roleRef rbacV1.RoleRef, subjects []rbacV1.Subject) error {
	log.Info("Start creating role binding", "edp name", edpName, "namespace", namespace, "role name", roleRef)
	randPostfix, err := rand.Int(rand.Reader, big.NewInt(10000))
	_, err = service.rbacClient.RoleBindings(namespace).Create(
		&rbacV1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s-%d", edpName, roleRef.Name, randPostfix),
			},
			RoleRef:  roleRef,
			Subjects: subjects,
		},
	)

	return err
}

// GetSecret return data field of Secret
func (service K8SService) GetSecretData(namespace string, name string) (map[string][]byte, error) {
	reqLog := log.WithValues("secret name", name, "namespace", namespace)
	reqLog.Info("Start retrieving secret data...")

	secret, err := service.coreClient.Secrets(namespace).Get(name, metav1.GetOptions{})
	if err != nil && k8sErrors.IsNotFound(err) {
		reqLog.Info("Secret not found")
		return nil, nil
	} else if err != nil {
		return nil, err
	}
	return secret.Data, nil
}
