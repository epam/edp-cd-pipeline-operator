package rbac

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crNameLogKey    = "name"
	ClusterRoleKind = "ClusterRole"
)

type Manager interface {
	GetRoleBinding(name, namespace string) (*rbacApi.RoleBinding, error)
	RoleBindingExists(ctx context.Context, name, namespace string) (bool, error)
	CreateRoleBinding(name, namespace string, subjects []rbacApi.Subject, roleRef rbacApi.RoleRef) error
	CreateRoleBindingIfNotExists(ctx context.Context, name, namespace string, subjects []rbacApi.Subject, roleRef rbacApi.RoleRef) error
	GetRole(name, namespace string) (*rbacApi.Role, error)
	CreateRole(name, namespace string, rules []rbacApi.PolicyRule) error
}

type KubernetesRbac struct {
	client client.Client
	log    logr.Logger
}

func NewRbacManager(c client.Client, log logr.Logger) Manager {
	return KubernetesRbac{
		client: c,
		log:    log,
	}
}

func (s KubernetesRbac) GetRoleBinding(name, namespace string) (*rbacApi.RoleBinding, error) {
	log := s.log.WithValues(crNameLogKey, name, "namespace", namespace)
	log.Info("getting role binding")

	rb := &rbacApi.RoleBinding{}
	if err := s.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, rb); err != nil {
		return nil, fmt.Errorf("failed to get role binding: %w", err)
	}

	return rb, nil
}

// RoleBindingExists checks if a RoleBinding exists in the given namespace.
func (s KubernetesRbac) RoleBindingExists(ctx context.Context, name, namespace string) (bool, error) {
	log := s.log.WithValues(crNameLogKey, name)
	log.Info("Checking if RoleBinding exists")

	if err := s.client.Get(ctx, types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, &rbacApi.RoleBinding{}); err != nil {
		if k8sErrors.IsNotFound(err) {
			log.Info("RoleBinding does not exist")

			return false, nil
		}

		return false, fmt.Errorf("failed to get role binding: %w", err)
	}

	log.Info("RoleBinding exists")

	return true, nil
}

func (s KubernetesRbac) CreateRoleBinding(name, namespace string, subjects []rbacApi.Subject, roleRef rbacApi.RoleRef) error {
	log := s.log.WithValues(crNameLogKey, name)
	log.Info("creating rolebinding")

	rb := &rbacApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subjects: subjects,
		RoleRef:  roleRef,
	}
	if err := s.client.Create(context.Background(), rb); err != nil {
		return fmt.Errorf("failed to create role binding: %w", err)
	}

	log.Info("rolebinding has been created")

	return nil
}

// CreateRoleBindingIfNotExists creates a RoleBinding if it does not exist in the given namespace.
func (s KubernetesRbac) CreateRoleBindingIfNotExists(
	ctx context.Context,
	name,
	namespace string,
	subjects []rbacApi.Subject,
	roleRef rbacApi.RoleRef,
) error {
	exists, err := s.RoleBindingExists(ctx, name, namespace)
	if err != nil {
		return fmt.Errorf("failed to check if role binding exists: %w", err)
	}

	if exists {
		return nil
	}

	return s.CreateRoleBinding(name, namespace, subjects, roleRef)
}

func (s KubernetesRbac) GetRole(name, namespace string) (*rbacApi.Role, error) {
	log := s.log.WithValues(crNameLogKey, name, "namespace", namespace)
	log.Info("getting role binding")

	r := &rbacApi.Role{}
	if err := s.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, r); err != nil {
		return nil, fmt.Errorf("failed to get role: %w", err)
	}

	return r, nil
}

func (s KubernetesRbac) CreateRole(name, namespace string, rules []rbacApi.PolicyRule) error {
	log := s.log.WithValues(crNameLogKey, name)
	log.Info("creating role")

	r := &rbacApi.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Rules: rules,
	}
	if err := s.client.Create(context.Background(), r); err != nil {
		return fmt.Errorf("failed to create role: %w", err)
	}

	log.Info("role has been created")

	return nil
}
