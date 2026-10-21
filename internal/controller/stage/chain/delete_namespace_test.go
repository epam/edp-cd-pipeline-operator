package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestDeleteNamespace_NSDoestExists(t *testing.T) {
	ch := DeleteNamespace{
		multiClusterClient: fake.NewClientBuilder().Build(),
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
	err := ch.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	assert.NoError(t, err)

	ns := &v1.Namespace{}
	n := fmt.Sprintf("%v-%v", namespace, name)
	err = ch.multiClusterClient.Get(context.TODO(), types.NamespacedName{
		Name: n,
	}, ns)
	assert.Error(t, err, "ns doesn't exist")
}

func TestDeleteNamespace_DeleteNS(t *testing.T) {
	n := fmt.Sprintf("%v-%v", namespace, name)
	ns := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name:   n,
			Labels: map[string]string{util.TenantLabelName: namespace},
		},
	}

	ch := DeleteNamespace{
		multiClusterClient: fake.NewClientBuilder().WithRuntimeObjects(ns).Build(),
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Namespace: n,
		},
	}
	err := ch.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	assert.NoError(t, err)

	ns = &v1.Namespace{}
	err = ch.multiClusterClient.Get(context.Background(), types.NamespacedName{
		Name: n,
	}, ns)
	assert.Error(t, err, "ns doesn't exist")
}

func deleteExistingNamespace(t *testing.T, labels map[string]string) error {
	t.Helper()

	ns := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{Name: "existing-ns", Labels: labels},
	}
	c := fake.NewClientBuilder().WithRuntimeObjects(ns).Build()
	ch := DeleteNamespace{multiClusterClient: c}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{Name: name, Namespace: namespace},
		Spec:       cdPipeApi.StageSpec{Namespace: ns.Name},
	}

	err := ch.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), s)
	if err != nil {
		return err
	}

	return c.Get(context.Background(), types.NamespacedName{Name: ns.Name}, &v1.Namespace{})
}

func TestDeleteNamespace_ForeignNamespaceKept(t *testing.T) {
	err := deleteExistingNamespace(t, map[string]string{util.TenantLabelName: "other"})
	assert.NoError(t, err, "namespace must still exist")
}

func TestDeleteNamespace_UnlabeledKept(t *testing.T) {
	err := deleteExistingNamespace(t, nil)
	assert.NoError(t, err, "namespace must still exist")
}

func TestDeleteNamespace_CapsuleInCluster(t *testing.T) {
	t.Setenv(platform.TenancyEngineEnv, platform.TenancyEngineCapsule)

	err := deleteExistingNamespace(t, nil)
	assert.True(t, apierrors.IsNotFound(err), "namespace must be deleted")
}
