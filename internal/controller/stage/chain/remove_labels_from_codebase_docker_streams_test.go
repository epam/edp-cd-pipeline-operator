package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

// cleanupStage builds a first (order 0) auto-deploy Stage with lowercase names so
// the resulting "<pipeline>/<stage>" label is a valid Kubernetes label key.
func cleanupStage(pipeName, stageName string) *cdPipeApi.Stage {
	return &cdPipeApi.Stage{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      pipeName + "-" + stageName,
			Namespace: namespace,
		},
		Spec: cdPipeApi.StageSpec{
			Name:        stageName,
			Order:       0,
			CdPipeline:  pipeName,
			TriggerType: cdPipeApi.TriggerTypeAutoDeploy,
		},
	}
}

func cleanupPipeline(pipeName string, inputStreams []string) *cdPipeApi.CDPipeline {
	return &cdPipeApi.CDPipeline{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      pipeName,
			Namespace: namespace,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			Name:               pipeName,
			InputDockerStreams: inputStreams,
		},
	}
}

// cisWithEnvLabel builds a CodebaseImageStream resolvable by its branch label and
// carrying the given env label (an empty envLabel adds none).
func cisWithEnvLabel(cisName, branch, codebaseName, envLabel string) *codebaseApi.CodebaseImageStream {
	labels := map[string]string{cluster.CodebaseBranchLabel: branch}
	if envLabel != "" {
		labels[envLabel] = ""
	}

	return &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      cisName,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase: codebaseName,
		},
	}
}

// cisInputStream models a promoted application's input stream: resolvable by its branch
// label but unlabelled, because PutEnvironmentLabel labels the verified stream instead.
func cisInputStream(cisName, branch, codebaseName string) *codebaseApi.CodebaseImageStream {
	return cisWithEnvLabel(cisName, branch, codebaseName, "")
}

func hasEnvLabel(t *testing.T, c client.Client, cisName, envLabel string) bool {
	t.Helper()

	cis := &codebaseApi.CodebaseImageStream{}
	require.NoError(t, c.Get(context.Background(), client.ObjectKey{Name: cisName, Namespace: namespace}, cis))

	_, ok := cis.GetLabels()[envLabel]

	return ok
}

// The core regression test: an application removed from spec.inputDockerStreams must
// have the "<pipeline>/<stage>" label stripped from its CodebaseImageStream, while a
// stream that is still part of the pipeline keeps it.
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_PurgesOrphanKeepsMember(t *testing.T) {
	pipeName, stageName := "demo", "dev"
	envLabel := createLabelName(pipeName, stageName)

	stage := cleanupStage(pipeName, stageName)
	pipe := cleanupPipeline(pipeName, []string{"app-member-main"})
	member := cisWithEnvLabel("app-member-main", "app-member-main", "app-member", envLabel)
	orphan := cisWithEnvLabel("app-removed-main", "app-removed-main", "app-removed", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(stage, pipe, member, orphan).Build(),
	}

	require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))

	assert.True(t, hasEnvLabel(t, h.client, "app-member-main", envLabel),
		"stream still referenced by the pipeline must keep the env label")
	assert.False(t, hasEnvLabel(t, h.client, "app-removed-main", envLabel),
		"stream removed from the pipeline must lose the env label")
}

// Covers the !IsFirst() && promoted branch of desiredLabeledStreams: a promoted app on a
// non-first stage is labelled on its upstream "<pipe>-<prevStage>-<codebase>-verified"
// stream, so removing it must strip that verified stream while a still-promoted app keeps it.
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_PurgesOrphanVerifiedKeepsPromotedMember(t *testing.T) {
	pipeName, prevStageName, stageName := "demo", "dev", "qa"
	envLabel := createLabelName(pipeName, stageName)

	// FindPreviousStageName resolves this order-0 stage by its pipeline label to build
	// the "<pipe>-<prevStage>-<codebase>-verified" stream name.
	prevStage := cleanupStage(pipeName, prevStageName)
	prevStage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: pipeName}

	stage := cleanupStage(pipeName, stageName)
	stage.Spec.Order = 1
	stage.Labels = map[string]string{cdPipeApi.StageCdPipelineLabelName: pipeName}

	pipe := cleanupPipeline(pipeName, []string{"app-member-main"})
	pipe.Spec.ApplicationsToPromote = []string{"app-member", "app-removed"}

	// Surviving promoted app: input stream resolves the codebase, verified stream holds
	// the label and must be preserved.
	input := cisInputStream("app-member-main", "app-member-main", "app-member")
	memberVerified := cisWithEnvLabel(
		createCisName(pipeName, prevStageName, "app-member"), "app-member-verified", "app-member", envLabel)
	orphanVerified := cisWithEnvLabel(
		createCisName(pipeName, prevStageName, "app-removed"), "app-removed-verified", "app-removed", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(prevStage, stage, pipe, input, memberVerified, orphanVerified).Build(),
	}

	require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))

	assert.True(t, hasEnvLabel(t, h.client, memberVerified.Name, envLabel),
		"verified stream of a still-promoted application must keep the env label")
	assert.False(t, hasEnvLabel(t, h.client, orphanVerified.Name, envLabel),
		"verified stream of a removed promoted application must lose the env label")
}

// Covers the !IsFirst() && !promoted branch of desiredLabeledStreams, which neither the
// first-stage tests nor PurgesOrphanVerifiedKeepsPromotedMember reach: a non-promoted app
// on a non-first stage is labelled on its input stream, not a verified one.
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_PurgesOrphanKeepsNonPromotedInputOnNonFirstStage(t *testing.T) {
	pipeName, stageName := "demo", "qa"
	envLabel := createLabelName(pipeName, stageName)

	// Non-first, but app-keep isn't promoted, so its input stream is labelled directly
	// and FindPreviousStageName is never needed.
	stage := cleanupStage(pipeName, stageName)
	stage.Spec.Order = 1

	pipe := cleanupPipeline(pipeName, []string{"app-keep-main"})

	member := cisWithEnvLabel("app-keep-main", "app-keep-main", "app-keep", envLabel)
	orphan := cisWithEnvLabel(
		createCisName(pipeName, "dev", "app-removed"), "app-removed-verified", "app-removed", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(stage, pipe, member, orphan).Build(),
	}

	require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))

	assert.True(t, hasEnvLabel(t, h.client, "app-keep-main", envLabel),
		"input stream of a non-promoted app on a non-first stage must keep the env label")
	assert.False(t, hasEnvLabel(t, h.client, orphan.Name, envLabel),
		"stream no longer referenced by the pipeline must lose the env label")
}

// A manual-trigger stage never owns env labels, so the handler must leave everything untouched.
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_SkipManual(t *testing.T) {
	pipeName, stageName := "demo", "dev"
	envLabel := createLabelName(pipeName, stageName)

	stage := cleanupStage(pipeName, stageName)
	stage.Spec.TriggerType = cdPipeApi.TriggerTypeManual
	pipe := cleanupPipeline(pipeName, []string{"app-member-main"})
	orphan := cisWithEnvLabel("app-removed-main", "app-removed-main", "app-removed", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(stage, pipe, orphan).Build(),
	}

	require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))

	assert.True(t, hasEnvLabel(t, h.client, "app-removed-main", envLabel),
		"manual-trigger stage must not modify any env labels")
}

// Re-running the handler when nothing was removed must be a no-op (idempotency).
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_NoOpWhenAllReferenced(t *testing.T) {
	pipeName, stageName := "demo", "dev"
	envLabel := createLabelName(pipeName, stageName)

	stage := cleanupStage(pipeName, stageName)
	pipe := cleanupPipeline(pipeName, []string{"app-a-main", "app-b-main"})
	a := cisWithEnvLabel("app-a-main", "app-a-main", "app-a", envLabel)
	b := cisWithEnvLabel("app-b-main", "app-b-main", "app-b", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(stage, pipe, a, b).Build(),
	}

	require.NoError(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))

	assert.True(t, hasEnvLabel(t, h.client, "app-a-main", envLabel))
	assert.True(t, hasEnvLabel(t, h.client, "app-b-main", envLabel))
}

func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_CantGetCdPipeline(t *testing.T) {
	stage := cleanupStage("demo", "dev")

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).WithObjects(stage).Build(),
	}

	err := h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage)
	require.Error(t, err)
	assert.True(t, k8sErrors.IsNotFound(err))
}

// A failure to resolve a referenced input stream must NOT strip labels: the handler
// errors (and requeues) instead, leaving every existing label intact.
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_ResolveErrorKeepsLabels(t *testing.T) {
	pipeName, stageName := "demo", "dev"
	envLabel := createLabelName(pipeName, stageName)

	stage := cleanupStage(pipeName, stageName)
	// References "app-member-main", but no CodebaseImageStream with that branch label
	// exists, so resolution fails.
	pipe := cleanupPipeline(pipeName, []string{"app-member-main"})
	orphan := cisWithEnvLabel("app-removed-main", "app-removed-main", "app-removed", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(stage, pipe, orphan).Build(),
	}

	require.Error(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))
	assert.True(t, hasEnvLabel(t, h.client, "app-removed-main", envLabel),
		"a resolve error must not strip any label")
}

// An empty inputDockerStreams must not purge every labelled stream; the handler errors.
func TestRemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate_ServeRequest_EmptyInputKeepsLabels(t *testing.T) {
	pipeName, stageName := "demo", "dev"
	envLabel := createLabelName(pipeName, stageName)

	stage := cleanupStage(pipeName, stageName)
	pipe := cleanupPipeline(pipeName, nil)
	orphan := cisWithEnvLabel("app-removed-main", "app-removed-main", "app-removed", envLabel)

	h := RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate{
		client: fake.NewClientBuilder().WithScheme(schemeInit(t)).
			WithObjects(stage, pipe, orphan).Build(),
	}

	require.Error(t, h.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), stage))
	assert.True(t, hasEnvLabel(t, h.client, "app-removed-main", envLabel),
		"an empty inputDockerStreams must not purge labels")
}
