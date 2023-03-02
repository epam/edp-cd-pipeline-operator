package kiosk

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
)

const (
	name            = "stub-name"
	account         = "stub-account"
	resourceVersion = "1"
)

func expectedSpaceInit(t *testing.T) *unstructured.Unstructured {
	t.Helper()

	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
		"metadata": map[string]interface{}{
			"name": name,
			"labels": map[string]interface{}{
				util.TenantLabelName: account,
			},
			"resourceVersion": resourceVersion,
		},
		"spec": map[string]interface{}{
			"account": account,
		},
	}

	return space
}

func emptySpaceInit(t *testing.T) Space {
	t.Helper()

	scheme := runtime.NewScheme()
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	return Space{
		Client: client,
		Log:    logr.Discard(),
	}
}

func TestSpace_InitSpaceSuccess(t *testing.T) {
	client := fake.NewClientBuilder().Build()
	log := ctrl.Log.WithName("space-manager")

	space := Space{
		Client: client,
		Log:    log,
	}

	initializedSpace := InitSpace(client)
	assert.Equal(t, space, initializedSpace)
}

func TestSpace_CreateSuccess(t *testing.T) {
	space := emptySpaceInit(t)

	expectedSpace := expectedSpaceInit(t)

	err := space.Create(name, account)
	assert.NoError(t, err)

	emptySpace := &unstructured.Unstructured{}
	emptySpace.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
	}

	err = space.Client.Get(context.Background(), types.NamespacedName{
		Name: name,
	}, emptySpace)

	assert.NoError(t, err)
	assert.Equal(t, expectedSpace, emptySpace)
}

func TestSpace_GetSuccess(t *testing.T) {
	space := emptySpaceInit(t)

	expectedSpace := expectedSpaceInit(t)

	err := space.Create(name, account)
	assert.NoError(t, err)

	createdSpace, err := space.Get(name)
	assert.NoError(t, err)

	assert.Equal(t, expectedSpace, createdSpace)
}

func TestSpace_DeleteSuccess(t *testing.T) {
	space := emptySpaceInit(t)

	err := space.Create(name, account)
	assert.NoError(t, err)

	_, err = space.Get(name)
	assert.NoError(t, err)

	err = space.Delete(name)
	assert.NoError(t, err)

	_, err = space.Get(name)
	assert.True(t, k8sErrors.IsNotFound(err))
}
