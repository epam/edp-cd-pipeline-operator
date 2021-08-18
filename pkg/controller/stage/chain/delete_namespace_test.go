package chain

import (
	"context"
	"fmt"
	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

func TestDeleteNamespace_NSDoestExists(t *testing.T) {
	ch := DeleteNamespace{
		client: fake.NewClientBuilder().Build(),
		log:    logr.DiscardLogger{},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
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
		ObjectMeta: metav1.ObjectMeta{
			Name: n,
		},
	}

	ch := DeleteNamespace{
		client: fake.NewClientBuilder().WithRuntimeObjects(ns).Build(),
		log:    logr.DiscardLogger{},
	}

	s := &cdPipeApi.Stage{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
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
