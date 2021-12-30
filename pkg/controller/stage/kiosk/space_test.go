package kiosk

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	loftKioskApi "github.com/loft-sh/kiosk/pkg/apis/tenancy/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/chain/util"
)

const (
	name            = "stub-name"
	account         = "stub-account"
	resourceVersion = "1"
)

func expectedSpaceInit(t *testing.T) *loftKioskApi.Space {
	t.Helper()
	return &loftKioskApi.Space{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Space",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
			Labels: map[string]string{
				util.TenantLabelName: account,
			},
			ResourceVersion: resourceVersion,
		},
		Spec: loftKioskApi.SpaceSpec{
			Account: account,
		},
	}
}

func emptySpaceInit(t *testing.T) Space {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &loftKioskApi.Space{})
	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	return Space{
		Client: client,
		Log:    logr.DiscardLogger{},
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

	emptySpace := &loftKioskApi.Space{}

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
