package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

var (
	name      = "stub_name"
	namespace = "stub_ns"
)

func TestPutNamespace_CreateNs(t *testing.T) {
	ch := PutNamespace{
		client: fake.NewClientBuilder().Build(),
		log:    logr.Discard(),
	}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Namespace: "default-stage-1",
		},
	}
	err := ch.ServeRequest(s)
	assert.NoError(t, err)

	ns := &v1.Namespace{}
	err = ch.client.Get(context.Background(), types.NamespacedName{
		Name: s.Spec.Namespace,
	}, ns)
	assert.NoError(t, err)
}

func TestPutNamespace_NSExists(t *testing.T) {
	ns := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: fmt.Sprintf("%v-%v", namespace, name),
		},
	}

	ch := PutNamespace{
		client: fake.NewClientBuilder().WithRuntimeObjects(ns).Build(),
		log:    logr.Discard(),
	}
	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Namespace: ns.Name,
		},
	}
	err := ch.ServeRequest(s)
	assert.NoError(t, err)
}
