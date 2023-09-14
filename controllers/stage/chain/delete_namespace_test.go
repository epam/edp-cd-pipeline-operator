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

func TestDeleteNamespace_NSDoestExists(t *testing.T) {
	ch := DeleteNamespace{
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
	assert.Error(t, err, "ns doesn't exist")
}

func TestDeleteNamespace_DeleteNS(t *testing.T) {
	n := fmt.Sprintf("%v-%v", namespace, name)
	ns := &v1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: n,
		},
	}

	ch := DeleteNamespace{
		client: fake.NewClientBuilder().WithRuntimeObjects(ns).Build(),
		log:    logr.Discard(),
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Namespace: n,
		},
	}
	err := ch.ServeRequest(s)
	assert.NoError(t, err)

	ns = &v1.Namespace{}
	err = ch.client.Get(context.TODO(), types.NamespacedName{
		Name: n,
	}, ns)
	assert.Error(t, err, "ns doesn't exist")
}
