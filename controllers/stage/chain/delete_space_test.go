package chain

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/kiosk"
	commonmock "github.com/epam/edp-common/pkg/mock"
)

func emptyStageInit(t *testing.T) *cdPipeApi.Stage {
	t.Helper()

	return &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Namespace: "stage-1-ns",
		},
	}
}

func TestDeleteSpace_DeleteSpaceSuccess(t *testing.T) {
	logger := commonmock.NewLogr()
	stage := emptyStageInit(t)
	spaceName := stage.Spec.Namespace
	space := &unstructured.Unstructured{}
	space.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "v1",
		"spec": map[string]interface{}{
			"name": spaceName,
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.Stage{})

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

	emptySpace := &unstructured.Unstructured{}
	emptySpace.Object = map[string]interface{}{
		"kind":       "Space",
		"apiVersion": "v1",
	}
	err = testSpace.Client.Get(context.Background(), types.NamespacedName{
		Name: spaceName,
	}, emptySpace)
	assert.True(t, k8sErrors.IsNotFound(err))
}

func TestDeleteSpace_SpaceDoesntExist(t *testing.T) {
	stage := emptyStageInit(t)

	log := commonmock.NewLogr()

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, &cdPipeApi.Stage{})

	testSpace := kiosk.Space{
		Client: fake.NewClientBuilder().WithScheme(scheme).Build(),
		Log:    log,
	}

	deleteSpaceInstance := DeleteSpace{
		log:   log,
		space: testSpace,
	}

	err := deleteSpaceInstance.ServeRequest(stage)

	loggerSink, ok := log.GetSink().(*commonmock.Logger)
	assert.True(t, ok)

	_, isDeleted := loggerSink.InfoMessages()["loft kiosk space resource is already deleted"]
	assert.True(t, isDeleted)
	assert.NoError(t, err)
}
