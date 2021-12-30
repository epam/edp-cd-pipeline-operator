package chain

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	loftKioskApi "github.com/loft-sh/kiosk/pkg/apis/tenancy/v1alpha1"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/kiosk"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
)

const (
	account = "account"
)

func kioskSpaceScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &loftKioskApi.Space{}, &cdPipeApi.Stage{})
	return scheme
}

func TestSpaceExist_NotFound(t *testing.T) {
	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).Build()

	space := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  space,
		client: client,
		log:    logr.DiscardLogger{},
	}

	exists, err := putKioskSpace.spaceExists(name)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestSpaceExist_Success(t *testing.T) {
	space := &loftKioskApi.Space{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).WithObjects(space).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.DiscardLogger{},
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
		log:    logr.DiscardLogger{},
	}

	err := putKioskSpace.createSpace(name, account)
	assert.NoError(t, err)

	space, err := spaceManager.Get(name)
	assert.NoError(t, err)
	assert.Equal(t, name, space.Name)
}

func TestCreateSpace_AlreadyExists(t *testing.T) {
	clientSpace := &loftKioskApi.Space{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}

	client := fake.NewClientBuilder().WithScheme(kioskSpaceScheme(t)).WithObjects(clientSpace).Build()

	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.DiscardLogger{},
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
		log:    logr.DiscardLogger{},
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
		log:    logr.DiscardLogger{},
	}

	stage := emptyStageInit(t)

	err := putKioskSpace.ServeRequest(stage)
	assert.NoError(t, err)

	name := fmt.Sprintf("%s-%s", stage.Namespace, stage.Name)

	space, err := spaceManager.Get(name)
	assert.NoError(t, err)
	assert.Equal(t, name, space.Name)
}

func TestPutKioskSpace_ServeRequest_Error(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.Stage{})

	emptyStage := emptyStageInit(t)

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(emptyStage).Build()
	spaceManager := kiosk.InitSpace(client)

	putKioskSpace := PutKioskSpace{
		space:  spaceManager,
		client: client,
		log:    logr.DiscardLogger{},
	}

	err := putKioskSpace.ServeRequest(emptyStage)
	assert.True(t, strings.Contains(err.Error(), "unable to create"))

	stage := &cdPipeApi.Stage{}
	err = putKioskSpace.client.Get(context.Background(), types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}, stage)
	assert.NoError(t, err)
	assert.Equal(t, consts.FailedStatus, stage.Status.Status)
}
