package security

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/client/interceptor"
)

const testNamespace = "krci-demo-dev"

func newReplicaSet(name, deployment, revision string, replicas int32) *appsv1.ReplicaSet {
	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   testNamespace,
			Annotations: map[string]string{revisionAnnotation: revision},
		},
		Spec: appsv1.ReplicaSetSpec{Replicas: ptr.To(replicas)},
	}

	if deployment != "" {
		rs.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: "apps/v1", Kind: "Deployment", Name: deployment, Controller: ptr.To(true),
		}}
	}

	return rs
}

func newDeployment(name, revision string) *appsv1.Deployment {
	return &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{
		Name:        name,
		Namespace:   testNamespace,
		Annotations: map[string]string{revisionAnnotation: revision},
	}}
}

func TestWorkloadResolver_Resolve(t *testing.T) {
	objects := []client.Object{
		newDeployment("app", "2"),
		newReplicaSet("app-old", "app", "1", 0),
		newReplicaSet("app-new", "app", "2", 3),
		newReplicaSet("owner-gone", "removed", "1", 1),
		newReplicaSet("standalone", "", "", 1),
		newReplicaSet("standalone-scaled-down", "", "", 0),
	}

	tests := []struct {
		name      string
		kind      string
		resource  string
		want      Workload
		wantState WorkloadState
	}{
		{name: "current revision", kind: "ReplicaSet", resource: "app-new", want: Workload{"deployment", "app"}, wantState: WorkloadCurrent},
		{name: "old revision", kind: "ReplicaSet", resource: "app-old", want: Workload{"deployment", "app"}, wantState: WorkloadSuperseded},
		{name: "replicaset deleted", kind: "ReplicaSet", resource: "missing", wantState: WorkloadGone},
		{name: "deployment deleted", kind: "ReplicaSet", resource: "owner-gone", want: Workload{"deployment", "removed"}, wantState: WorkloadGone},
		{name: "replicaset without deployment", kind: "ReplicaSet", resource: "standalone", want: Workload{"replicaset", "standalone"}, wantState: WorkloadCurrent},
		{name: "replicaset without deployment scaled down", kind: "ReplicaSet", resource: "standalone-scaled-down", want: Workload{"replicaset", "standalone-scaled-down"}, wantState: WorkloadGone},
		{name: "statefulset", kind: "StatefulSet", resource: "db", want: Workload{"statefulset", "db"}, wantState: WorkloadCurrent},
		{name: "cronjob", kind: "CronJob", resource: "nightly", want: Workload{"cronjob", "nightly"}, wantState: WorkloadCurrent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objects...).Build()

			got, state, err := NewWorkloadResolver(c, testNamespace).Resolve(context.Background(), tt.kind, tt.resource)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantState, state)
		})
	}
}

func TestWorkloadResolver_Resolve_ListsOnce(t *testing.T) {
	lists := 0
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).
		WithObjects(newDeployment("app", "1"), newReplicaSet("app-1", "app", "1", 1)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(ctx context.Context, cl client.WithWatch, list client.ObjectList, opts ...client.ListOption) error {
				lists++

				return cl.List(ctx, list, opts...)
			},
		}).Build()

	r := NewWorkloadResolver(c, testNamespace)

	for range 3 {
		_, _, err := r.Resolve(context.Background(), "ReplicaSet", "app-1")
		require.NoError(t, err)
	}

	assert.Equal(t, 2, lists)
}

func TestWorkloadResolver_Resolve_ListError(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return assert.AnError
			},
		}).Build()

	_, _, err := NewWorkloadResolver(c, testNamespace).Resolve(context.Background(), "ReplicaSet", "app-1")

	require.ErrorIs(t, err, assert.AnError)
}

func testScheme(t *testing.T) *runtime.Scheme {
	t.Helper()

	s := runtime.NewScheme()
	require.NoError(t, appsv1.AddToScheme(s))
	require.NoError(t, batchv1.AddToScheme(s))

	return s
}

func newJob(name, cronJob string, age time.Duration) *batchv1.Job {
	job := &batchv1.Job{ObjectMeta: metav1.ObjectMeta{
		Name:              name,
		Namespace:         testNamespace,
		CreationTimestamp: metav1.NewTime(time.Date(2026, 11, 10, 12, 0, 0, 0, time.UTC).Add(-age)),
	}}

	if cronJob != "" {
		job.OwnerReferences = []metav1.OwnerReference{{
			APIVersion: "batch/v1", Kind: "CronJob", Name: cronJob, Controller: ptr.To(true),
		}}
	}

	return job
}

func TestWorkloadResolver_Resolve_Jobs(t *testing.T) {
	objects := []client.Object{
		newJob("nightly-3", "nightly", 0),
		newJob("nightly-2", "nightly", time.Hour),
		newJob("nightly-1", "nightly", 2*time.Hour),
		newJob("migrate", "", 0),
	}

	tests := []struct {
		name      string
		resource  string
		want      Workload
		wantState WorkloadState
	}{
		{name: "newest run is current", resource: "nightly-3", want: Workload{"cronjob", "nightly"}, wantState: WorkloadCurrent},
		{name: "older run is superseded", resource: "nightly-2", want: Workload{"cronjob", "nightly"}, wantState: WorkloadSuperseded},
		{name: "oldest run is superseded", resource: "nightly-1", want: Workload{"cronjob", "nightly"}, wantState: WorkloadSuperseded},
		{name: "job without cronjob", resource: "migrate", want: Workload{"job", "migrate"}, wantState: WorkloadCurrent},
		{name: "deleted job", resource: "missing", want: Workload{"job", "missing"}, wantState: WorkloadCurrent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := fake.NewClientBuilder().WithScheme(testScheme(t)).WithObjects(objects...).Build()

			got, state, err := NewWorkloadResolver(c, testNamespace).Resolve(context.Background(), "Job", tt.resource)

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
			assert.Equal(t, tt.wantState, state)
		})
	}
}

func TestWorkloadResolver_Resolve_JobListError(t *testing.T) {
	c := fake.NewClientBuilder().WithScheme(testScheme(t)).
		WithInterceptorFuncs(interceptor.Funcs{
			List: func(context.Context, client.WithWatch, client.ObjectList, ...client.ListOption) error {
				return assert.AnError
			},
		}).Build()

	_, _, err := NewWorkloadResolver(c, testNamespace).Resolve(context.Background(), "Job", "nightly-1")

	require.ErrorIs(t, err, assert.AnError)
}
