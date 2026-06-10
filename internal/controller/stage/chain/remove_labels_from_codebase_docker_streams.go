package chain

import (
	"context"
	"fmt"

	"slices"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/internal/controller/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
)

// RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate removes the
// "<cdPipeline>/<stage>" environment label from every CodebaseImageStream that is
// no longer referenced by the CDPipeline.
//
// PutEnvironmentLabelToCodebaseImageStreams only ever (re)labels the streams that
// are currently in spec.inputDockerStreams, so when an application is removed from
// a CDPipeline the label is orphaned on its CodebaseImageStream. codebase-operator
// then keeps creating a CDStageDeploy for that stage on every new image tag, i.e.
// it ghost-deploys an application that no longer belongs to the pipeline. This
// handler reconciles the label set to the current spec so the orphan is cleaned up
// regardless of how the application was removed (UI, kubectl or API).
type RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate struct {
	client client.Client
}

func (h RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate) ServeRequest(
	ctx context.Context,
	stage *cdPipeApi.Stage,
) error {
	log := ctrl.LoggerFrom(ctx)

	if stage.IsManualTriggerType() {
		log.Info("Trigger type is not auto deploy, skipping cleanup of environment labels")

		return nil
	}

	log.Info("Start cleaning up stale environment labels from CodebaseImageStream resources")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %s cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	// Guard against an empty desired set wiping every labelled stream. The CRD
	// enforces MinItems=1, but a bypassed/malformed spec must not trigger a purge.
	if len(pipe.Spec.InputDockerStreams) == 0 {
		return fmt.Errorf("pipeline %s doesn't contain codebase image streams", pipe.Name)
	}

	envLabel := createLabelName(pipe.Name, stage.Spec.Name)

	desired, err := h.desiredLabeledStreams(ctx, pipe, stage)
	if err != nil {
		return err
	}

	var streams codebaseApi.CodebaseImageStreamList
	if err = h.client.List(
		ctx,
		&streams,
		client.InNamespace(stage.Namespace),
		client.HasLabels{envLabel},
	); err != nil {
		return fmt.Errorf("failed to list CodebaseImageStreams labeled %q: %w", envLabel, err)
	}

	for i := range streams.Items {
		cis := &streams.Items[i]

		// Only act on streams that actually carry the label. client.HasLabels
		// silently widens the List to the whole namespace if envLabel is ever an
		// invalid selector key, so this guard keeps us from updating unrelated streams.
		if _, ok := cis.GetLabels()[envLabel]; !ok {
			continue
		}

		if _, ok := desired[cis.Name]; ok {
			continue
		}

		deleteLabel(&cis.ObjectMeta, envLabel)

		if err = h.client.Update(ctx, cis); err != nil {
			return fmt.Errorf("failed to remove label %q from CodebaseImageStream %s: %w", envLabel, cis.Name, err)
		}

		log.Info("Stale environment label has been removed from CodebaseImageStream",
			"label", envLabel, "codebaseImageStream", cis.Name)
	}

	log.Info("Stale environment labels have been cleaned up")

	return nil
}

// desiredLabeledStreams returns the set of CodebaseImageStream names that must
// carry the "<cdPipeline>/<stage>" label according to the current CDPipeline spec.
// It mirrors PutEnvironmentLabelToCodebaseImageStreams: a first stage (or a
// non-promoted application) labels the input stream itself, while a promoted
// application on a non-first stage labels the upstream
// "<cdPipeline>-<previousStage>-<codebase>-verified" stream.
func (h RemoveLabelsFromCodebaseDockerStreamsAfterCdPipelineUpdate) desiredLabeledStreams(
	ctx context.Context,
	pipe *cdPipeApi.CDPipeline,
	stage *cdPipeApi.Stage,
) (map[string]struct{}, error) {
	desired := make(map[string]struct{}, len(pipe.Spec.InputDockerStreams))

	// previousStageName is identical for every promoted stream, so resolve it at
	// most once and only when a promoted non-first stream actually needs it.
	var (
		previousStageName     string
		previousStageResolved bool
	)

	for _, name := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStreamByCodebaseBaseBranchName(ctx, h.client, name, stage.Namespace)
		if err != nil {
			// Fail (and requeue) rather than skip: a transient resolve error must not
			// drop a still-referenced stream from the desired set, or the caller would
			// wrongly strip its label. This matches Put/DeleteEnvironmentLabel.
			return nil, fmt.Errorf("failed to get %s codebase image stream: %w", name, err)
		}

		if stage.IsFirst() || !slices.Contains(pipe.Spec.ApplicationsToPromote, stream.Spec.Codebase) {
			desired[stream.Name] = struct{}{}

			continue
		}

		if !previousStageResolved {
			previousStageName, err = util.FindPreviousStageName(ctx, h.client, stage)
			if err != nil {
				return nil, fmt.Errorf("failed to get previous stage name: %w", err)
			}

			previousStageResolved = true
		}

		desired[createCisName(pipe.Name, previousStageName, stream.Spec.Codebase)] = struct{}{}
	}

	return desired, nil
}
