package util

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	k8sApi "k8s.io/api/rbac/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

const (
	name      = "stub-name"
	namespace = "stub-namespace"
	ownerKind = "CDPipeline"
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
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.StageList{}, &cdPipeApi.Stage{})

	prevStage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage1",
			Namespace: namespace,
			Labels:    map[string]string{cdPipeApi.CodebaseTypeLabelName: name},
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      0,
		},
	}

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage2",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      1,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(prevStage).Build()

	stageName, err := FindPreviousStageName(context.Background(), client, stage)
	if assert.NoError(t, err) {
		assert.Equal(t, prevStage.Spec.Name, stageName)
	}
}

func TestFindPreviousStageName_PrevStageWithoutLabel(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.StageList{}, &cdPipeApi.Stage{})

	prevStage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage1",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      0,
		},
	}

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage2",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      1,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(prevStage).Build()

	_, err := FindPreviousStageName(context.Background(), client, stage)
	assert.Error(t, err)
}

func TestFindPreviousStageName_ForFirstStage(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.StageList{}, &cdPipeApi.Stage{})

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage2",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      0,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := FindPreviousStageName(context.Background(), client, stage)
	assert.Error(t, err)
}

func TestFindPreviousStageName_EmptyStages(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.StageList{}, &cdPipeApi.Stage{})

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage2",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      1,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	_, err := FindPreviousStageName(context.Background(), client, stage)
	assert.Error(t, err)
}

func TestFindPreviousStageName_PreviousStageNotFound(t *testing.T) {
	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(k8sApi.SchemeGroupVersion, &cdPipeApi.StageList{}, &cdPipeApi.Stage{})

	prevStage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage1",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: "another-pipeline",
			Order:      0,
		},
	}

	stage := &cdPipeApi.Stage{
		TypeMeta: metaV1.TypeMeta{},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stage2",
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			CdPipeline: name,
			Order:      1,
		},
	}

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(prevStage).Build()

	_, err := FindPreviousStageName(context.Background(), client, stage)
	assert.Error(t, err)
}
