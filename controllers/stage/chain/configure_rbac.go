package chain

import (
	"fmt"

	"github.com/go-logr/logr"
	k8sApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/handler"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

const (
	clusterRoleKind    = "ClusterRole"
	groupKind          = "Group"
	jenkinsAdminRbName = "jenkins-admin"
	crNameLogKey       = "name"
	namespaceLogKey    = "namespace"
)

type ConfigureRbac struct {
	next   handler.CdStageHandler
	client client.Client
	log    logr.Logger
	rbac   rbac.Manager
}

type options struct {
	subjects []k8sApi.Subject
	rf       k8sApi.RoleRef
}

func (h ConfigureRbac) ServeRequest(stage *cdPipeApi.Stage) error {
	targetNamespace := util.GenerateNamespaceName(stage)
	logger := h.log.WithValues(namespaceLogKey, targetNamespace)
	logger.Info("Configuring RBAC for newly created namespace.")

	jenkinsAdminOpts := buildJenkinsAdminRoleOptions(stage.Namespace)
	if err := h.createRoleBinding(jenkinsAdminRbName, targetNamespace, jenkinsAdminOpts); err != nil {
		return err
	}

	if platform.IsOpenshift() {
		viewGroupRbName := generateViewGroupRoleBindingName(stage.Namespace)
		viewGroupOpts := buildViewGroupRoleOptions(stage.Namespace)

		if err := h.createRoleBinding(viewGroupRbName, targetNamespace, viewGroupOpts); err != nil {
			return err
		}
	}

	logger.Info("RBAC has been configured.")

	return nextServeOrNil(h.next, stage)
}

func (h ConfigureRbac) roleBindingExists(name, namespace string) (bool, error) {
	logger := h.log.WithValues(crNameLogKey, name, namespaceLogKey, namespace)
	logger.Info("Check existence of RoleBinding.")

	if _, err := h.rbac.GetRoleBinding(name, namespace); err != nil {
		if k8sErrors.IsNotFound(err) {
			logger.Info("RoleBinding doesn't exist.")
			return false, nil
		}

		return false, fmt.Errorf("failed to get role binding: %w", err)
	}

	logger.Info("RoleBinding exists.")

	return true, nil
}

func (h ConfigureRbac) createRoleBinding(rbName, namespace string, opts options) error {
	exists, err := h.roleBindingExists(rbName, namespace)
	if err != nil {
		return fmt.Errorf("failed to to check existence of %s rolebinding: %w", rbName, err)
	}

	if exists {
		log.Info("Skip creating the RoleBinding, it already exists.", crNameLogKey, rbName, namespaceLogKey, namespace)
		return nil
	}

	if err = h.rbac.CreateRoleBinding(rbName, namespace, opts.subjects, opts.rf); err != nil {
		return fmt.Errorf("failed to create %s rolebinding: %w", rbName, err)
	}

	return nil
}

func buildJenkinsAdminRoleOptions(sourceNamespace string) options {
	const adminClusterRoleName = "admin"

	return options{
		subjects: getJenkinsAdminRoleSubjects(sourceNamespace),
		rf: k8sApi.RoleRef{
			Name:     adminClusterRoleName,
			APIGroup: k8sApi.GroupName,
			Kind:     clusterRoleKind,
		},
	}
}

func getJenkinsAdminRoleSubjects(sourceNamespace string) []k8sApi.Subject {
	const (
		serviceAccountKind        = "ServiceAccount"
		jenkinsServiceAccountName = "jenkins"
	)

	if !platform.IsOpenshift() {
		return []k8sApi.Subject{
			{
				Kind:      serviceAccountKind,
				Name:      jenkinsServiceAccountName,
				Namespace: sourceNamespace,
			},
		}
	}

	return []k8sApi.Subject{
		{
			Kind: groupKind,
			Name: fmt.Sprintf("%v-edp-super-admin", sourceNamespace),
		},
		{
			Kind: groupKind,
			Name: fmt.Sprintf("%v-edp-admin", sourceNamespace),
		},
		{
			Kind:      serviceAccountKind,
			Name:      jenkinsServiceAccountName,
			Namespace: sourceNamespace,
		},
	}
}

func buildViewGroupRoleOptions(sourceNamespace string) options {
	const viewClusterRoleName = "view"

	return options{
		subjects: []k8sApi.Subject{
			{
				Kind: groupKind,
				Name: fmt.Sprintf("%v-edp-view", sourceNamespace),
			},
		},
		rf: k8sApi.RoleRef{
			Name:     viewClusterRoleName,
			APIGroup: k8sApi.GroupName,
			Kind:     clusterRoleKind,
		},
	}
}

func generateViewGroupRoleBindingName(namespace string) string {
	return fmt.Sprintf("%s-view", namespace)
}
