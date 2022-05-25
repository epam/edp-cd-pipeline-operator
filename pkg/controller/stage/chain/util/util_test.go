package util

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
)

const (
	name      = "stub-name"
	namespace = "stub-namespace"
	ownerKind = "CDPipeline"
	stageName = "stub-stage-name"
)

func TestGetCdPipeline_WithObjectReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.CDPipeline{}, &cdPipeApi.Stage{})

	ownerReferences := []metaV1.OwnerReference{{
		Kind: ownerKind,
		Name: name,
	}}

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
		},
	}

	cdPipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:            name,
			Namespace:       namespace,
			OwnerReferences: ownerReferences,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage, cdPipeline).Build()

	resultCdPipeline, err := GetCdPipeline(client, stage)
	assert.NoError(t, err)
	assert.Equal(t, resultCdPipeline.Name, name)
}

func TestGetCdPipeline_WithoutObjectReferences(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.CDPipeline{}, &cdPipeApi.Stage{})

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
		},
	}

	cdPipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stage, cdPipeline).Build()

	resultCdPipeline, err := GetCdPipeline(client, stage)
	assert.NoError(t, err)
	assert.Equal(t, resultCdPipeline.Name, name)
}

func TestFindPreviousStageName_Success(t *testing.T) {
	annotations := make(map[string]string)
	annotations[previousStageNameAnnotationKey] = stageName

	previousStageName, err := FindPreviousStageName(annotations)
	assert.NoError(t, err)
	assert.Equal(t, stageName, previousStageName)
}

func TestFindPreviousStageName_EmptyAnnotations(t *testing.T) {
	annotations := make(map[string]string)

	previousStageName, err := FindPreviousStageName(annotations)
	assert.Equal(t, err, fmt.Errorf("stage doesnt contain %v annotation", previousStageNameAnnotationKey))
	assert.Equal(t, "", previousStageName)
}

func TestFindPreviousStageName_NilMap(t *testing.T) {
	var annotations map[string]string

	previousStageName, err := FindPreviousStageName(annotations)
	assert.Equal(t, err, fmt.Errorf("there're no any annotation"))
	assert.Equal(t, "", previousStageName)
}
