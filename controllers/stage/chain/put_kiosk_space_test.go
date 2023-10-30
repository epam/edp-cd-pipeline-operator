package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

const (
	account = "account"
)

func kioskSpaceScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.Stage{})

	return scheme
}

func TestSpaceExist_NotFound(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).Build()

	space := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  space,
		client: client,
		log:    logr.Discard(),
	}

	exists, err := putKioskSpace.spaceExists(name)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestSpaceExist_Success(t *testing.T) {
	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}

	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).WithObjects(space).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.Discard(),
	}

	exists, err := putKioskSpace.spaceExists(name)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCreateSpace_Success(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.Discard(),
	}

	err := putKioskSpace.createSpace(name, account)
	assert.NoError(t, err)

	space, err := spaceManager.Get(name)
	assert.NoError(t, err)
	assert.Equal(t, name, space.GetName())
}

func TestCreateSpace_AlreadyExists(t *testing.T) {
	clientSpace := &unstructured.Unstructured{}
	clientSpace.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "tenancy.kiosk.sh/v1alpha1",
		"metadata": map[string]interface{}{
			"name": name,
		},
	}

	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).WithObjects(clientSpace).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.Discard(),
	}

	_, err := spaceManager.Get(name)
	assert.NoError(t, err)

	err = putKioskSpace.createSpace(name, account)
	assert.NoError(t, err)
}

func TestSetFailedStatus_Success(t *testing.T) {
	stage := emptyStageInit(t)

	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).WithObjects(stage).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.Discard(),
	}

	err := putKioskSpace.setFailedStatus(context.Background(), stage, errors.New(""))
	assert.NoError(t, err)
	assert.Equal(t, consts.FailedStatus, stage.Status.Status)
}

func TestPutKioskSpace_ServeRequest_Success(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.Discard(),
	}

	stage := emptyStageInit(t)

	err := putKioskSpace.ServeRequest(stage)
	assert.NoError(t, err)

	space, err := spaceManager.Get(stage.Spec.Namespace)
	assert.NoError(t, err)
	assert.Equal(t, stage.Spec.Namespace, space.GetName())
}
