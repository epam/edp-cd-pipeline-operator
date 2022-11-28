package chain

import (
	"context"
	"fmt"
	"testing"

	loftKioskApi "github.com/loft-sh/kiosk/pkg/apis/tenancy/v1alpha1"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/controller/stage/kiosk"
	edpLog "github.com/epam/edp-common/pkg/mock"
)

func emptyStageInit(t *testing.T) *cdPipeApi.Stage {
	t.Helper()

	return &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestDeleteSpace_DeleteSpaceSuccess(t *testing.T) {
	logger := &edpLog.Logger{}
	stage := emptyStageInit(t)
	spaceName := fmt.Sprintf("%s-%s", stage.Namespace, stage.Name)
	space := &loftKioskApi.Space{
		ObjectMeta: metaV1.ObjectMeta{
			Name: spaceName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &loftKioskApi.Space{}, &cdPipeApi.Stage{})

	testSpace := kiosk.Space{
		Client: fake.NewClientBuilder().WithScheme(scheme).WithObjects(space).Build(),
		Log:    logger,
	}

	deleteSpaceInstance := DeleteSpace{
		log:   logger,
		space: testSpace,
	}

	err := deleteSpaceInstance.ServeRequest(stage)
	assert.NoError(t, err)

	emptySpace := &loftKioskApi.Space{}
	err = testSpace.Client.Get(context.Background(), types.NamespacedName{
		Name: spaceName,
	}, emptySpace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestDeleteSpace_SpaceDoesntExist(t *testing.T) {
	stage := emptyStageInit(t)

	log := &edpLog.Logger{}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &loftKioskApi.Space{}, &cdPipeApi.Stage{})

	testSpace := kiosk.Space{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Log:    log,
	}

	deleteSpaceInstance := DeleteSpace{
		log:   log,
		space: testSpace,
	}

	err := deleteSpaceInstance.ServeRequest(stage)
	_, isDeleted := log.InfoMessages["loft kiosk space resource is already deleted"]
	assert.True(t, isDeleted)
	assert.NoError(t, err)
}
