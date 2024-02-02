package argocd

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	argoApi "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"golang.org/x/exp/maps"
	apiextensionsv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

type generatorElement struct {
	Stage           string `json:"stage"`
	Codebase        string `json:"codebase"`
	ImageTag        string `json:"imageTag"`
	ImageRepository string `json:"imageRepository"`
	Cluster         string `json:"cluster"`
	Namespace       string `json:"namespace"`
	GitUrlPath      string `json:"gitUrlPath"`
	VersionType     string `json:"versionType"`
	CustomValues    bool   `json:"customValues"`
}

const codebaseTypeSystem = "system"

var gitOpsCodebaseLabels = map[string]string{
	"app.edp.epam.com/codebaseType": "system",
	"app.edp.epam.com/systemType":   "gitops",
}

type ArgoApplicationSetManager struct {
	client client.Client
}

func NewArgoApplicationSetManager(k8sClient client.Client) *ArgoApplicationSetManager {
	return &ArgoApplicationSetManager{client: k8sClient}
}

func (c *ArgoApplicationSetManager) CreateApplicationSet(ctx context.Context, pipeline *cdPipeApi.CDPipeline) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating ArgoApplicationSet")

	appset := &argoApi.ApplicationSet{}

	err := c.client.Get(ctx, client.ObjectKey{
		Namespace: pipeline.Namespace,
		Name:      pipeline.Name,
	}, appset)
	if client.IgnoreNotFound(err) != nil {
		return fmt.Errorf("failed to get ArgoApplicationSet: %w", err)
	}

	if err == nil {
		log.Info("ArgoApplicationSet already exists. Skip creating")
		return nil
	}

	if len(pipeline.Spec.Applications) == 0 {
		log.Info("No applications specified. Skip creating ArgoApplicationSet")
		return nil
	}

	codebases, err := c.getPipelinesCodebasesMap(ctx, pipeline.Namespace, pipeline.Spec.Applications)
	if err != nil {
		return err
	}

	gitServer, err := c.getGitServer(ctx, pipeline.Namespace, codebases)
	if err != nil {
		return err
	}

	gitopsCodebase, err := c.getGitOpsRepo(ctx, pipeline.Namespace)
	if err != nil {
		return err
	}

	appset = generateApplicationSet(pipeline, gitServer, gitopsCodebase)

	if err = controllerutil.SetOwnerReference(pipeline, appset, c.client.Scheme()); err != nil {
		return fmt.Errorf("failed to set ApplicationSet owner reference: %w", err)
	}

	if err = c.client.Create(ctx, appset); err != nil {
		return fmt.Errorf("failed to create ArgoApplicationSet: %w", err)
	}

	log.Info("ArgoApplicationSet has been created")

	return nil
}

func (c *ArgoApplicationSetManager) CreateApplicationSetGenerators(ctx context.Context, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Creating ArgoApplicationSetGenerator")

	pipeline := &cdPipeApi.CDPipeline{}
	if err := c.client.Get(ctx, client.ObjectKey{
		Namespace: stage.Namespace,
		Name:      stage.Spec.CdPipeline,
	}, pipeline); err != nil {
		return fmt.Errorf("failed to get CDPipeline: %w", err)
	}

	appset := &argoApi.ApplicationSet{}
	if err := c.client.Get(ctx, client.ObjectKey{
		Namespace: stage.Namespace,
		Name:      pipeline.Name,
	}, appset); err != nil {
		return fmt.Errorf("failed to get ArgoApplicationSet: %w", err)
	}

	codebases, err := c.getPipelinesCodebasesMap(ctx, stage.Namespace, pipeline.Spec.Applications)
	if err != nil {
		return err
	}

	stageGenerators, err := c.makeStageGenerators(ctx, stage, maps.Values(codebases))
	if err != nil {
		return err
	}

	changed, err := setGenerators(stage.Spec.Name, appset, stageGenerators)
	if err != nil {
		return err
	}

	if changed {
		if err = c.client.Update(ctx, appset); err != nil {
			return fmt.Errorf("failed to update ArgoApplicationSet: %w", err)
		}

		log.Info("ArgoApplicationSet has been updated")

		return nil
	}

	log.Info("ArgoApplicationSet generators are already set")

	return nil
}

func (c *ArgoApplicationSetManager) RemoveApplicationSetGenerators(ctx context.Context, stage *cdPipeApi.Stage) error {
	log := ctrl.LoggerFrom(ctx)

	log.Info("Removing ArgoApplicationSetGenerator")

	appset := &argoApi.ApplicationSet{}
	if err := c.client.Get(ctx, client.ObjectKey{
		Namespace: stage.Namespace,
		Name:      stage.Labels[cdPipeApi.StageCdPipelineLabelName],
	}, appset); err != nil {
		if errors.IsNotFound(err) {
			log.Info("ArgoApplicationSet is not found. Skip removing generators")

			return nil
		}

		return fmt.Errorf("failed to get ArgoApplicationSet: %w", err)
	}

	for i := 0; i < len(appset.Spec.Generators); i++ {
		if appset.Spec.Generators[i].List == nil {
			continue
		}

		n := 0

		for _, rawel := range appset.Spec.Generators[i].List.Elements {
			el := &generatorElement{}
			if err := json.Unmarshal(rawel.Raw, el); err != nil {
				return fmt.Errorf("failed to unmarshal generator element: %w", err)
			}

			if el.Stage != stage.Spec.Name {
				appset.Spec.Generators[i].List.Elements[n] = rawel
				n++
			}
		}

		if len(appset.Spec.Generators[i].List.Elements) != n {
			appset.Spec.Generators[i].List.Elements = appset.Spec.Generators[i].List.Elements[:n]

			if err := c.client.Update(ctx, appset); err != nil {
				return fmt.Errorf("failed to update ArgoApplicationSet: %w", err)
			}

			log.Info("Stage generator was removed from ArgoApplicationSet")

			return nil
		}

		break
	}

	log.Info("Stage generators are not found in ArgoApplicationSet")

	return nil
}

func (c *ArgoApplicationSetManager) makeStageGenerators(
	ctx context.Context,
	stage *cdPipeApi.Stage,
	codebases []codebaseApi.Codebase,
) (map[string]apiextensionsv1.JSON, error) {
	stageGenerators := make(map[string]apiextensionsv1.JSON, len(codebases))

	for k := range codebases {
		image, err := c.getImageRepo(ctx, codebases[k].Namespace, codebases[k].Name, codebases[k].Spec.DefaultBranch)
		if err != nil {
			return nil, err
		}

		gen := generatorElement{
			Stage:           stage.Spec.Name,
			Codebase:        codebases[k].Name,
			ImageTag:        "NaN",
			ImageRepository: image,
			Cluster:         stage.Spec.ClusterName,
			Namespace:       stage.Spec.Namespace,
			GitUrlPath:      strings.TrimPrefix(codebases[k].Spec.GitUrlPath, "/"),
			VersionType:     string(codebases[k].Spec.Versioning.Type),
			CustomValues:    false,
		}

		var raw []byte

		if raw, err = json.Marshal(gen); err != nil {
			return nil, fmt.Errorf("failed to marshal generator element: %w", err)
		}

		stageGenerators[fmt.Sprintf("%s-%s", codebases[k].Name, stage.Spec.Name)] = apiextensionsv1.JSON{Raw: raw}
	}

	return stageGenerators, nil
}

func (c *ArgoApplicationSetManager) getImageRepo(ctx context.Context, ns, codebaseName, branch string) (string, error) {
	image := &codebaseApi.CodebaseImageStream{}
	if err := c.client.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      fmt.Sprintf("%s-%s", codebaseName, branch),
	}, image); err != nil {
		return "", fmt.Errorf("failed to get CodebaseImageStream: %w", err)
	}

	return image.Spec.ImageName, nil
}

// TODO: we can optimize this method by getting all codebases at once. We need to add label with name to codebase.
func (c *ArgoApplicationSetManager) getPipelinesCodebasesMap(ctx context.Context, ns string, apps []string) (map[string]codebaseApi.Codebase, error) {
	m := make(map[string]codebaseApi.Codebase, len(apps))

	for _, app := range apps {
		codebase := &codebaseApi.Codebase{}
		if err := c.client.Get(ctx, client.ObjectKey{
			Namespace: ns,
			Name:      app,
		}, codebase); err != nil {
			return nil, fmt.Errorf("failed to get Codebase: %w", err)
		}

		m[app] = *codebase
	}

	return m, nil
}

func (c *ArgoApplicationSetManager) getGitServer(ctx context.Context, ns string, codebases map[string]codebaseApi.Codebase) (*codebaseApi.GitServer, error) {
	if len(codebases) == 0 {
		return nil, fmt.Errorf("no codebases specified")
	}

	var gitServerName string
	for k := range codebases {
		if gitServerName != "" && gitServerName != codebases[k].Spec.GitServer {
			return nil, fmt.Errorf("codebases have different git servers")
		}

		gitServerName = codebases[k].Spec.GitServer
	}

	gitServer := &codebaseApi.GitServer{}
	if err := c.client.Get(ctx, client.ObjectKey{
		Namespace: ns,
		Name:      gitServerName,
	}, gitServer); err != nil {
		return nil, fmt.Errorf("failed to get GitServer: %w", err)
	}

	return gitServer, nil
}

func (c *ArgoApplicationSetManager) getGitOpsRepo(ctx context.Context, ns string) (*codebaseApi.Codebase, error) {
	codebaseList := &codebaseApi.CodebaseList{}
	if err := c.client.List(ctx, codebaseList, client.InNamespace(ns), client.MatchingLabels(gitOpsCodebaseLabels)); err != nil {
		return nil, fmt.Errorf("failed to list codebases: %w", err)
	}

	if len(codebaseList.Items) == 0 {
		return nil, fmt.Errorf("no GitOps codebases found")
	}

	if len(codebaseList.Items) > 1 {
		return nil, fmt.Errorf("found more than one GitOps codebase")
	}

	gitOpsCodebase := &codebaseList.Items[0]
	if gitOpsCodebase.Spec.Type != codebaseTypeSystem {
		return nil, fmt.Errorf("gitOps codebase does not have %q type", codebaseTypeSystem)
	}

	return gitOpsCodebase, nil
}

func generateTemplatePatch(pipeline, gitopsUrlPath, gitUser, gitHost string, sshPort int32) string {
	sshURL := fmt.Sprintf("ssh://%s@%s:%d", gitUser, gitHost, sshPort)
	gitopsRepoUrl := fmt.Sprintf("https://%s%s", gitHost, gitopsUrlPath)
	template := `
    {{- if .customValues }}
    spec:
      sources:
        - ref: values
          repoURL: %s
          targetRevision: main
        - helm:
            parameters:
              - name: image.tag
                value: '{{ .imageTag }}'
              - name: image.repository
                value: {{ .imageRepository }}
            releaseName: '{{ .codebase }}'
            valueFiles:
              - $values/%s/{{ .stage }}/{{ .codebase }}-values.yaml
          path: deploy-templates
          repoURL: %s/{{ .gitUrlPath }}
          targetRevision: '{{ if eq .versionType "edp" }}build/{{ .imageTag }}{{ else }}{{ .imageTag }}{{ end }}'
    {{- end }}`

	return fmt.Sprintf(template, gitopsRepoUrl, pipeline, sshURL)
}

func generateApplicationSet(
	pipeline *cdPipeApi.CDPipeline,
	gitServer *codebaseApi.GitServer,
	gitopsCodebase *codebaseApi.Codebase,
) *argoApi.ApplicationSet {
	templatePatch := generateTemplatePatch(
		pipeline.Name,
		gitopsCodebase.Spec.GitUrlPath,
		gitServer.Spec.GitUser,
		gitServer.Spec.GitHost,
		gitServer.Spec.SshPort)

	return &argoApi.ApplicationSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      pipeline.Name,
			Namespace: pipeline.Namespace,
		},
		Spec: argoApi.ApplicationSetSpec{
			Generators:        []argoApi.ApplicationSetGenerator{},
			GoTemplate:        true,
			GoTemplateOptions: []string{"missingkey=error"},
			TemplatePatch:     &templatePatch,
			Template: argoApi.ApplicationSetTemplate{
				ApplicationSetTemplateMeta: argoApi.ApplicationSetTemplateMeta{
					Name:       fmt.Sprintf("%s-{{ .stage }}-{{ .codebase }}", pipeline.Name),
					Finalizers: []string{"resources-finalizer.argocd.argoproj.io"}, // check if it is our or argo's responsibility
					Labels: map[string]string{
						"app.edp.epam.com/app-name": "{{ .codebase }}",
						"app.edp.epam.com/pipeline": pipeline.Name,
						"app.edp.epam.com/stage":    "{{ .stage }}",
					},
				},
				Spec: argoApi.ApplicationSpec{
					Destination: argoApi.ApplicationDestination{
						Name:      "{{ .cluster }}",
						Namespace: "{{ .namespace }}",
					},
					Project: pipeline.Namespace,
					Source: &argoApi.ApplicationSource{
						Helm: &argoApi.ApplicationSourceHelm{
							Parameters: []argoApi.HelmParameter{
								{
									Name:  "image.tag",
									Value: "{{ .imageTag }}",
								},
								{
									Name:  "image.repository",
									Value: "{{ .imageRepository }}",
								},
							},
							ReleaseName: "{{ .codebase }}",
						},
						Path:           "deploy-templates",
						RepoURL:        fmt.Sprintf("ssh://%s@%s:%d/{{ .gitUrlPath }}", gitServer.Spec.GitUser, gitServer.Spec.GitHost, gitServer.Spec.SshPort),
						TargetRevision: `{{ if eq .versionType "edp" }}build/{{ .imageTag }}{{ else }}{{ .imageTag }}{{ end }}`,
					},
				},
			},
		},
	}
}

func setGenerators(stageName string, appset *argoApi.ApplicationSet, stageGenerators map[string]apiextensionsv1.JSON) (bool, error) {
	if len(appset.Spec.Generators) == 0 {
		appset.Spec.Generators = []argoApi.ApplicationSetGenerator{
			{
				List: &argoApi.ListGenerator{},
			},
		}
	}

	for i := 0; i < len(appset.Spec.Generators); i++ {
		if appset.Spec.Generators[i].List == nil {
			continue
		}

		changed, err := processGeneratorListElements(stageName, &appset.Spec.Generators[i], stageGenerators)
		if err != nil {
			return false, err
		}

		return changed, nil
	}

	return false, nil
}

func processGeneratorListElements(stageName string, generator *argoApi.ApplicationSetGenerator, stageGenerators map[string]apiextensionsv1.JSON) (bool, error) {
	n := 0

	for _, rawel := range generator.List.Elements {
		el := &generatorElement{}
		if err := json.Unmarshal(rawel.Raw, el); err != nil {
			return false, fmt.Errorf("failed to unmarshal generator element: %w", err)
		}

		key := fmt.Sprintf("%s-%s", el.Codebase, el.Stage)
		_, ok := stageGenerators[key]

		if ok {
			delete(stageGenerators, key)
		}

		if el.Stage != stageName || (el.Stage == stageName && ok) {
			generator.List.Elements[n] = rawel
			n++
		}
	}

	if len(generator.List.Elements) != n || len(stageGenerators) > 0 {
		generator.List.Elements = generator.List.Elements[:n]
		generator.List.Elements = append(generator.List.Elements, maps.Values(stageGenerators)...)

		return true, nil
	}

	return false, nil
}
