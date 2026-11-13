package security

import (
	"context"
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	kindReplicaSet = "ReplicaSet"
	kindDeployment = "Deployment"
	kindJob        = "Job"
	kindCronJob    = "CronJob"
	// revisionAnnotation is set by the Deployment controller on a Deployment and on its current ReplicaSet.
	revisionAnnotation = "deployment.kubernetes.io/revision"
)

// WorkloadState tells whether the workload a report belongs to is running now.
type WorkloadState int

const (
	// WorkloadGone means the workload does not exist or is scaled down: its report is history.
	WorkloadGone WorkloadState = iota
	// WorkloadSuperseded means the workload exists but a newer revision is current:
	// the report must not overwrite what the current revision reported.
	WorkloadSuperseded
	// WorkloadCurrent means the report describes what runs now.
	WorkloadCurrent
)

// Workload is the stable identity of a workload: a Deployment for its ReplicaSets, so a rollout
// keeps writing to the same DefectDojo test.
type Workload struct {
	Kind string
	Name string
}

func (w Workload) String() string { return w.Kind + "/" + w.Name }

// Tag is the workload in the form allowed in a DefectDojo tag value.
func (w Workload) Tag() string { return w.Kind + "." + w.Name }

// WorkloadResolver maps the resource a report was produced for to a Workload.
// It reads the ReplicaSets, Deployments and Jobs of the namespace once each and is meant to live for one pass.
type WorkloadResolver struct {
	reader    client.Reader
	namespace string

	loaded      bool
	replicaSets map[string]*appsv1.ReplicaSet
	deployments map[string]*appsv1.Deployment

	jobsLoaded bool
	jobs       map[string]*batchv1.Job
	// latestJob holds, per CronJob, the newest Job: the only one whose report is current.
	latestJob map[string]*batchv1.Job
}

// NewWorkloadResolver returns a resolver that reads the workloads of namespace.
func NewWorkloadResolver(reader client.Reader, namespace string) *WorkloadResolver {
	return &WorkloadResolver{reader: reader, namespace: namespace}
}

// Resolve returns the workload of a report's resource kind and name and whether it is current.
// ReplicaSets resolve to their Deployment and Jobs to their CronJob; every other kind is its own workload.
func (r *WorkloadResolver) Resolve(ctx context.Context, kind, name string) (Workload, WorkloadState, error) {
	switch kind {
	case kindReplicaSet:
		return r.resolveReplicaSet(ctx, name)
	case kindJob:
		return r.resolveJob(ctx, name)
	default:
		return Workload{Kind: strings.ToLower(kind), Name: name}, WorkloadCurrent, nil
	}
}

func (r *WorkloadResolver) resolveReplicaSet(ctx context.Context, name string) (Workload, WorkloadState, error) {
	if err := r.load(ctx); err != nil {
		return Workload{}, WorkloadGone, err
	}

	rs, ok := r.replicaSets[name]
	if !ok {
		return Workload{}, WorkloadGone, nil
	}

	owner := deploymentOwner(rs)
	if owner == "" {
		w := Workload{Kind: "replicaset", Name: name}
		if ptr.Deref(rs.Spec.Replicas, 1) == 0 {
			return w, WorkloadGone, nil
		}

		return w, WorkloadCurrent, nil
	}

	w := Workload{Kind: "deployment", Name: owner}

	deployment, ok := r.deployments[owner]
	if !ok {
		return w, WorkloadGone, nil
	}

	if rs.Annotations[revisionAnnotation] != deployment.Annotations[revisionAnnotation] {
		return w, WorkloadSuperseded, nil
	}

	return w, WorkloadCurrent, nil
}

// resolveJob maps a Job owned by a CronJob to the CronJob; only the newest Job of a CronJob is current.
// A Job without a CronJob is its own workload.
func (r *WorkloadResolver) resolveJob(ctx context.Context, name string) (Workload, WorkloadState, error) {
	if err := r.loadJobs(ctx); err != nil {
		return Workload{}, WorkloadGone, err
	}

	job, ok := r.jobs[name]
	if !ok {
		return Workload{Kind: "job", Name: name}, WorkloadCurrent, nil
	}

	owner := cronJobOwner(job)
	if owner == "" {
		return Workload{Kind: "job", Name: name}, WorkloadCurrent, nil
	}

	w := Workload{Kind: "cronjob", Name: owner}
	if r.latestJob[owner].Name != name {
		return w, WorkloadSuperseded, nil
	}

	return w, WorkloadCurrent, nil
}

func (r *WorkloadResolver) loadJobs(ctx context.Context) error {
	if r.jobsLoaded {
		return nil
	}

	list := &batchv1.JobList{}
	if err := r.reader.List(ctx, list, client.InNamespace(r.namespace)); err != nil {
		return fmt.Errorf("failed to list Jobs: %w", err)
	}

	r.jobs = make(map[string]*batchv1.Job, len(list.Items))
	r.latestJob = map[string]*batchv1.Job{}

	for i := range list.Items {
		job := &list.Items[i]
		r.jobs[job.Name] = job

		owner := cronJobOwner(job)
		if owner == "" {
			continue
		}

		if latest, ok := r.latestJob[owner]; !ok || newerJob(job, latest) {
			r.latestJob[owner] = job
		}
	}

	r.jobsLoaded = true

	return nil
}

func newerJob(a, b *batchv1.Job) bool {
	if !a.CreationTimestamp.Equal(&b.CreationTimestamp) {
		return b.CreationTimestamp.Before(&a.CreationTimestamp)
	}

	return a.Name > b.Name
}

func cronJobOwner(job *batchv1.Job) string {
	for _, ref := range job.OwnerReferences {
		if ref.Kind == kindCronJob && ptr.Deref(ref.Controller, false) {
			return ref.Name
		}
	}

	return ""
}

func (r *WorkloadResolver) load(ctx context.Context) error {
	if r.loaded {
		return nil
	}

	rsList := &appsv1.ReplicaSetList{}
	if err := r.reader.List(ctx, rsList, client.InNamespace(r.namespace)); err != nil {
		return fmt.Errorf("failed to list ReplicaSets: %w", err)
	}

	deploymentList := &appsv1.DeploymentList{}
	if err := r.reader.List(ctx, deploymentList, client.InNamespace(r.namespace)); err != nil {
		return fmt.Errorf("failed to list Deployments: %w", err)
	}

	r.replicaSets = make(map[string]*appsv1.ReplicaSet, len(rsList.Items))
	for i := range rsList.Items {
		r.replicaSets[rsList.Items[i].Name] = &rsList.Items[i]
	}

	r.deployments = make(map[string]*appsv1.Deployment, len(deploymentList.Items))
	for i := range deploymentList.Items {
		r.deployments[deploymentList.Items[i].Name] = &deploymentList.Items[i]
	}

	r.loaded = true

	return nil
}

func deploymentOwner(rs *appsv1.ReplicaSet) string {
	for _, ref := range rs.OwnerReferences {
		if ref.Kind == kindDeployment && ptr.Deref(ref.Controller, false) {
			return ref.Name
		}
	}

	return ""
}
