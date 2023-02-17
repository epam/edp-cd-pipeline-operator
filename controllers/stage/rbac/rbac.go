package rbac

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	rbacApi "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	crNameLogKey = "name"
)

type Manager interface {
	GetRoleBinding(name, namespace string) (*rbacApi.RoleBinding, error)
	CreateRoleBinding(name, namespace string, subjects []rbacApi.Subject, roleRef rbacApi.RoleRef) error
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
