package chain

import (
	"context"
	"fmt"
	"testing"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilRuntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/consts"
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
	}
	err := ch.ServeRequest(s)
	assert.NoError(t, err)

	ns := &v1.Namespace{}
	n := fmt.Sprintf("%v-%v", namespace, name)
	err = ch.client.Get(context.TODO(), types.NamespacedName{
		Name: n,
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
	}
	err := ch.ServeRequest(s)
	assert.NoError(t, err)
}

func TestSetFailedStatus(t *testing.T) {
	scheme := runtime.NewScheme()

	utilRuntime.Must(cdPipeApi.AddToScheme(scheme))

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}

	ch := PutNamespace{
		client: fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(s).Build(),
		log:    logr.Discard(),
	}

	err := ch.setFailedStatus(context.TODO(), s, errors.New("stub_error"))
	assert.NoError(t, err)

	rs := &cdPipeApi.Stage{}
	err = ch.client.Get(context.TODO(), types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}, rs)
	assert.NoError(t, err)

	assert.Equal(t, consts.FailedStatus, rs.Status.Status)
}
