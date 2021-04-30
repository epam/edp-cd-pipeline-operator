package rbac

import (
	"context"
	"github.com/go-logr/logr"
	k8sApi "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type RbacManager interface {
	GetRoleBinding(name, namespace string) (*k8sApi.RoleBinding, error)
	CreateRoleBinding(name, namespace string, subjects []k8sApi.Subject, roleRef k8sApi.RoleRef) error
}

type KubernetesRbac struct {
	client client.Client
	log    logr.Logger
}

func InitRbacManager(client client.Client) RbacManager {
	return KubernetesRbac{
		client: client,
		log:    ctrl.Log.WithName("rbac-manager"),
	}
}

func (s KubernetesRbac) GetRoleBinding(name, namespace string) (*k8sApi.RoleBinding, error) {
	log := s.log.WithValues("name", name, "namespace", namespace)
	log.Info("getting role binding")
	rb := &k8sApi.RoleBinding{}
	if err := s.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, rb); err != nil {
		return nil, err
	}
	return rb, nil
}

func (s KubernetesRbac) CreateRoleBinding(name, namespace string, subjects []k8sApi.Subject, roleRef k8sApi.RoleRef) error {
	log := s.log.WithValues("name", name)
	log.Info("creating rolebinding")
	rb := &k8sApi.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Subjects: subjects,
		RoleRef:  roleRef,
	}
	if err := s.client.Create(context.Background(), rb); err != nil {
		return err
	}
	log.Info("rolebinding has been created")
	return nil
}
