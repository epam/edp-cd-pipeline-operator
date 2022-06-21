package chain

import (
	"context"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/apps/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/pkg/apis/edp/v1"
	k8sMockClient "github.com/epam/edp-common/pkg/mock/controller-runtime/client"
	componentApi "github.com/epam/edp-component-operator/pkg/apis/v1/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1"
)

func TestPutCodebaseImageStream_ShouldCreateCis(t *testing.T) {

	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	ec := &componentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerRegistryName,
			Namespace: "stub-namespace",
		},
		Spec: componentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, cdp, s, ec)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdp, s, ec, cis).Build()

	cisChain := PutCodebaseImageStream{
		client: client,
		log:    logr.DiscardLogger{},
	}

	err := cisChain.ServeRequest(s)
	assert.NoError(t, err)

	cisResp := &codebaseApi.CodebaseImageStream{}
	err = client.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
}

func TestPutCodebaseImageStream_ShouldNotFindCDPipeline(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name:       "stage-name",
			CdPipeline: "non-existing-pipeline",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, s, cdp)
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdp, s).Build()

	cisChain := PutCodebaseImageStream{
		client: client,
		log:    logr.DiscardLogger{},
	}

	err := cisChain.ServeRequest(s)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "couldn't get non-existing-pipeline cd pipeline") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCodebaseImageStream_ShouldNotFindEDPComponent(t *testing.T) {
	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name:       "stage-name",
			CdPipeline: "cdp-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, s, cdp)
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdp, s).Build()

	cisChain := PutCodebaseImageStream{
		client: client,
		log:    logr.DiscardLogger{},
	}

	err := cisChain.ServeRequest(s)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "couldn't get docker-registry EDP component") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCodebaseImageStream_ShouldNotFindCbis(t *testing.T) {

	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	ec := &componentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerRegistryName,
			Namespace: "stub-namespace",
		},
		Spec: componentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, cdp, s, ec)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdp, s, ec).Build()

	cisChain := PutCodebaseImageStream{
		client: client,
		log:    logr.DiscardLogger{},
	}

	err := cisChain.ServeRequest(s)
	assert.Error(t, err)

	if !strings.Contains(err.Error(), "unable to get cbis-name codebase image stream") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}

func TestPutCodebaseImageStream_ShouldNotFailWithExistingCbis(t *testing.T) {

	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	ec := &componentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerRegistryName,
			Namespace: "stub-namespace",
		},
		Spec: componentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	exsitingCis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, cdp, s, ec)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis, exsitingCis)
	client := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdp, s, ec, cis, exsitingCis).Build()

	cisChain := PutCodebaseImageStream{
		client: client,
		log:    logr.DiscardLogger{},
	}

	err := cisChain.ServeRequest(s)
	assert.NoError(t, err)

	cisResp := &codebaseApi.CodebaseImageStream{}
	err = client.Get(context.TODO(),
		types.NamespacedName{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		cisResp)
	assert.NoError(t, err)
}

func TestPutCodebaseImageStream_ShouldFailCreatingCbis(t *testing.T) {
	mc := k8sMockClient.Client{}

	cdp := &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.CDPipelineSpec{
			InputDockerStreams: []string{
				"cbis-name",
			},
		},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			OwnerReferences: []metaV1.OwnerReference{{
				Kind: "CDPipeline",
				Name: "cdp-name",
			}},
			Name:      "stub-stage-name",
			Namespace: "stub-namespace",
		},
		Spec: cdPipeApi.StageSpec{
			Name: "stage-name",
		},
	}

	ec := &componentApi.EDPComponent{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      dockerRegistryName,
			Namespace: "stub-namespace",
		},
		Spec: componentApi.EDPComponentSpec{
			Url: "stub-url",
		},
	}

	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cbis-name",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: "cb-name",
		},
	}

	exsitingCis := &codebaseApi.CodebaseImageStream{
		TypeMeta: metaV1.TypeMeta{
			APIVersion: "v2.edp.epam.com/v1",
			Kind:       "CodebaseImageStream",
		},
		ObjectMeta: metaV1.ObjectMeta{
			Name:      "cdp-name-stage-name-cb-name-verified",
			Namespace: "stub-namespace",
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  "cb-name",
			ImageName: "stub-url/stub-namespace",
		},
	}

	scheme := runtime.NewScheme()
	scheme.AddKnownTypes(v1.SchemeGroupVersion, cdp, s, ec)
	scheme.AddKnownTypes(schema.GroupVersion{Group: "v2.edp.epam.com", Version: "v1"}, cis)
	fakeCl := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(cdp, s, ec, cis).Build()

	mockErr := errors.New("fatal")

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      "cdp-name",
	}, &cdPipeApi.CDPipeline{}).Return(fakeCl)

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      dockerRegistryName,
	}, &componentApi.EDPComponent{}).Return(fakeCl)

	mc.On("Get", types.NamespacedName{
		Namespace: "stub-namespace",
		Name:      "cbis-name",
	}, &codebaseApi.CodebaseImageStream{}).Return(fakeCl)

	var createOpts []client.CreateOption
	mc.On("Create", exsitingCis, createOpts).Return(mockErr)

	cisChain := PutCodebaseImageStream{
		client: &mc,
		log:    logr.DiscardLogger{},
	}

	err := cisChain.ServeRequest(s)
	assert.Error(t, err)

	if errors.Cause(err) != mockErr {
		t.Fatal("wrong error returned")
	}
	if !strings.Contains(err.Error(), "couldn't create cdp-name-stage-name-cb-name-verified") {
		t.Fatalf("wrong error returned: %s", err.Error())
	}
}
