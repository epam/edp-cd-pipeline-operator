package chain

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/util/cluster"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
	componentApi "github.com/epam/edp-component-operator/api/v1"
)

type PutCodebaseImageStream struct {
	client client.Client
}

const (
	dockerRegistryName              = "docker-registry"
	edpConfigMap                    = "edp-config"
	edpConfigContainerRegistryHost  = "container_registry_host"
	edpConfigContainerRegistrySpace = "container_registry_space"
)

func (h PutCodebaseImageStream) ServeRequest(ctx context.Context, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("start creating codebase image streams.")

	pipe, err := util.GetCdPipeline(h.client, stage)
	if err != nil {
		return fmt.Errorf("failed to get %v cd pipeline: %w", stage.Spec.CdPipeline, err)
	}

	registryUrl, err := h.getDockerRegistryUrl(ctx, stage.Namespace)
	if err != nil {
		return fmt.Errorf("failed to get container registry url: %w", err)
	}

	for _, ids := range pipe.Spec.InputDockerStreams {
		stream, err := cluster.GetCodebaseImageStream(h.client, ids, stage.Namespace)
		if err != nil {
			return fmt.Errorf("failed to get %v codebase image stream: %w", ids, err)
		}

		cisName := fmt.Sprintf("%v-%v-%v-verified", pipe.Name, stage.Spec.Name, stream.Spec.Codebase)
		image := fmt.Sprintf("%v/%v", registryUrl, stream.Spec.Codebase)

		if err := h.createCodebaseImageStreamIfNotExists(
			ctx,
			cisName,
			image,
			stream.Spec.Codebase,
			stage.Namespace,
			stage,
		); err != nil {
			return fmt.Errorf("failed to create %v codebase image stream: %w", cisName, err)
		}
	}

	log.Info("codebase image stream have been created.")

	return nil
}

func (h PutCodebaseImageStream) getDockerRegistryUrl(ctx context.Context, namespace string) (string, error) {
	config := &corev1.ConfigMap{}
	if err := h.client.Get(ctx, types.NamespacedName{
		Name:      edpConfigMap,
		Namespace: namespace,
	}, config); err != nil {
		if !k8sErrors.IsNotFound(err) {
			return "", fmt.Errorf("failed to get %s config map: %w", edpConfigMap, err)
		}

		// to save backward compatibility we need to get docker registry url
		// from edp component if config map doesn't exist. We can remove this in the future.
		ec := &componentApi.EDPComponent{}
		if err = h.client.Get(ctx, types.NamespacedName{
			Name:      dockerRegistryName,
			Namespace: namespace,
		}, ec); err != nil {
			return "", fmt.Errorf("failed to fetch %q resource %q: %w", ec.TypeMeta.Kind, dockerRegistryName, err)
		}

		return ec.Spec.Url, nil
	}

	if _, ok := config.Data[edpConfigContainerRegistryHost]; !ok {
		return "", fmt.Errorf("%s is not set in %s config map", edpConfigContainerRegistryHost, edpConfigMap)
	}

	if _, ok := config.Data[edpConfigContainerRegistrySpace]; !ok {
		return "", fmt.Errorf("%s is not set in %s config map", edpConfigContainerRegistrySpace, edpConfigMap)
	}

	return fmt.Sprintf("%s/%s", config.Data[edpConfigContainerRegistryHost], config.Data[edpConfigContainerRegistrySpace]), nil
}

func (h PutCodebaseImageStream) createCodebaseImageStreamIfNotExists(
	ctx context.Context,
	name, imageName, codebaseName, namespace string,
	stage *cdPipeApi.Stage,
) error {
	log := ctrl.LoggerFrom(ctx).WithValues("CodebaseImageStream", name)
	cis := &codebaseApi.CodebaseImageStream{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: codebaseApi.CodebaseImageStreamSpec{
			Codebase:  codebaseName,
			ImageName: imageName,
		},
	}

	if err := controllerutil.SetControllerReference(stage, cis, h.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set controller reference for CodebaseImageStream: %w", err)
	}

	if err := h.client.Create(ctx, cis); err != nil {
		if k8sErrors.IsAlreadyExists(err) {
			// For backward compatibility, we need to update the controller reference for the existing CodebaseImageStream.
			// We can remove this in the next releases.
			existingCIS := &codebaseApi.CodebaseImageStream{}
			if err = h.client.Get(ctx, types.NamespacedName{
				Namespace: namespace,
				Name:      name,
			}, existingCIS); err != nil {
				return fmt.Errorf("failed to get CodebaseImageStream: %w", err)
			}

			if metaV1.GetControllerOf(existingCIS) == nil {
				if err = controllerutil.SetControllerReference(stage, existingCIS, h.client.Scheme()); err != nil {
					return fmt.Errorf("failed to set controller reference for CodebaseImageStream: %w", err)
				}
			}

			if err = h.client.Update(ctx, existingCIS); err != nil {
				return fmt.Errorf("failed to update CodebaseImageStream controller reference: %w", err)
			}

			log.Info("CodebaseImageStream already exists. Skip creating")

			return nil
		}

		return fmt.Errorf("failed to create codebase stream: %w", err)
	}

	log.Info("CodebaseImageStream has been created")

	return nil
}
