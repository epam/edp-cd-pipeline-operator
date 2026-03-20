package argocd

import (
	"bytes"
	"context"
	"encoding/json"
	"testing"
	"text/template"

	argoApi "github.com/argoproj/argo-cd/v3/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/yaml"

	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

const (
	ns = "default"
)

func TestArgoApplicationSetManager_CreateApplicationSet(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, argoApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name       string
		pipeline   *cdPipeApi.CDPipeline
		client     func(t *testing.T) client.Client
		wantErr    require.ErrorAssertionFunc
		wantAssert func(t *testing.T, cl client.Client)
	}{
		{
			name: "application set is created successfully",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name:         "pipe1",
					Applications: []string{"app1", "app2"},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app2",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "gitops",
								Namespace: ns,
								Labels:    gitOpsCodebaseLabels,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitUrlPath: "/company/gitops",
								Type:       codebaseTypeSystem,
								GitServer:  "gitops-git-server",
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								GitUser: "git",
								SshPort: 22,
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "gitops-git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "gerrit.com",
								GitUser: "git",
								SshPort: 22,
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						&argoApi.ApplicationSet{},
					),
				)
			},
		},
		{
			name: "failed - multiple gitops codebases exist",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name:         "pipe1",
					Applications: []string{"app1", "app2"},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app2",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "gitops",
								Namespace: ns,
								Labels:    gitOpsCodebaseLabels,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitUrlPath: "/company/gitops",
								Type:       codebaseTypeSystem,
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "gitops2",
								Namespace: ns,
								Labels:    gitOpsCodebaseLabels,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitUrlPath: "/company/gitops",
								Type:       codebaseTypeSystem,
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								GitUser: "git",
								SshPort: 22,
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "found more than one GitOps codebase")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "failed - wrong gitops codebases type",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name:         "pipe1",
					Applications: []string{"app1", "app2"},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app2",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "gitops",
								Namespace: ns,
								Labels:    gitOpsCodebaseLabels,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitUrlPath: "/company/gitops",
								Type:       "wrong-type",
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								GitUser: "git",
								SshPort: 22,
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), `gitOps codebase does not have "system" type`)
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "failed - gitops codebase doesn't exist",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name:         "pipe1",
					Applications: []string{"app1", "app2"},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app2",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								GitUser: "git",
								SshPort: 22,
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "no GitOps codebases found")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "failed - gitops git server doesn't exist",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name:         "pipe1",
					Applications: []string{"app1"},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer: "git-server",
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "gitops",
								Namespace: ns,
								Labels:    gitOpsCodebaseLabels,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitUrlPath: "/company/gitops",
								Type:       codebaseTypeSystem,
								GitServer:  "gitops-gitserver",
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get gitops GitServer")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "pipeline doesn't contain applications",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name: "pipe1",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "application set already exists",
			pipeline: &cdPipeApi.CDPipeline{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1",
					Namespace: ns,
				},
				Spec: cdPipeApi.CDPipelineSpec{
					Name:         "pipe1",
					Applications: []string{"app1", "app2"},
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
						},
					).
					Build()
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cl := tt.client(t)
			m := NewArgoApplicationSetManager(cl)
			err := m.CreateApplicationSet(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.pipeline)

			tt.wantErr(t, err)
			tt.wantAssert(t, cl)
		})
	}
}

func TestArgoApplicationSetManager_CreateApplicationSetGenerators(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, argoApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      *cdPipeApi.Stage
		client     func(t *testing.T) client.Client
		wantErr    require.ErrorAssertionFunc
		wantAssert func(t *testing.T, cl client.Client)
	}{
		{
			name: "application set generator is created successfully",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{"app1", "app2"},
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer:     "git-server",
								DefaultBranch: "main",
								GitUrlPath:    "/company/app1",
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypDefault,
								},
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app2",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer:     "git-server",
								DefaultBranch: "main",
								GitUrlPath:    "/company/app1",
								Versioning: codebaseApi.Versioning{
									Type: "semver",
								},
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								SshPort: 22,
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1-main",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								ImageName: "app1-main-image",
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app2-main",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								ImageName: "app2-main-image",
							},
						},
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: argoApi.ApplicationSetSpec{
								Generators: []argoApi.ApplicationSetGenerator{
									{
										List: &argoApi.ListGenerator{
											Elements: []v1.JSON{
												{
													Raw: []byte(`{"stage": "stage1", "codebase": "app-to-be-removed"}`),
												},
												{
													Raw: []byte(`{"stage":"should-skip-stage", "codebase": "go-app"}`),
												},
												{
													Raw: []byte(`{"stage":"stage1", "codebase": "app2"}`),
												},
											},
										},
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				appset := &argoApi.ApplicationSet{}
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						appset,
					),
				)

				require.Len(t, appset.Spec.Generators, 1)
				require.Len(t, appset.Spec.Generators[0].List.Elements, 3)

				// Check that elements are sorted by stage, then codebase
				prevStage := ""
				prevCodebase := ""

				for i, rawel := range appset.Spec.Generators[0].List.Elements {
					el := &generatorElement{}
					require.NoError(t, json.Unmarshal(rawel.Raw, el))

					if i > 0 {
						if prevStage == el.Stage {
							require.LessOrEqual(t, prevCodebase, el.Codebase, "elements are not sorted by codebase")
						} else {
							require.Less(t, prevStage, el.Stage, "elements are not sorted by stage")
						}
					}

					prevStage = el.Stage
					prevCodebase = el.Codebase
				}

				expected := map[string]string{
					"app1": `{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", ` +
						`"imageRepository":"app1-main-image", "imageDigest":"", "imageTag":"NaN", "namespace":"default", ` +
						`"stage":"stage1", "versionType":"default", "customValues":false, ` +
						`"repoURL": "ssh://@github.com:22/company/app1"}`,
					"app2":   `{"stage":"stage1", "codebase": "app2"}`,
					"go-app": `{"stage":"should-skip-stage", "codebase": "go-app"}`,
				}

				for _, rawel := range appset.Spec.Generators[0].List.Elements {
					el := &generatorElement{}
					require.NoError(t, json.Unmarshal(rawel.Raw, el))
					require.Contains(t, expected, el.Codebase)
					require.JSONEq(t, expected[el.Codebase], string(rawel.Raw))
				}
			},
		},
		{
			name: "application set generator is created successfully with empty ApplicationSet",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{"app1"},
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer:     "git-server",
								DefaultBranch: "main",
								GitUrlPath:    "/company/app1",
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypDefault,
								},
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								SshPort: 22,
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1-main",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								ImageName: "app1-main-image",
							},
						},
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: argoApi.ApplicationSetSpec{},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				appset := &argoApi.ApplicationSet{}
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						appset,
					),
				)

				require.Len(t, appset.Spec.Generators, 1)
				require.Len(t, appset.Spec.Generators[0].List.Elements, 1)
				require.JSONEq(
					t,
					`{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", `+
						`"imageRepository":"app1-main-image", "imageDigest":"", "imageTag":"NaN", "namespace":"default", `+
						`"stage":"stage1", "versionType":"default", "customValues":false, `+
						`"repoURL": "ssh://@github.com:22/company/app1"}`,
					string(appset.Spec.Generators[0].List.Elements[0].Raw),
				)
			},
		},
		{
			name: "all application set generators already exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{"app1"},
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer:     "git-server",
								DefaultBranch: "main",
								GitUrlPath:    "/company/app1",
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypDefault,
								},
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								SshPort: 22,
							},
						},
						&codebaseApi.CodebaseImageStream{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1-main",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseImageStreamSpec{
								ImageName: "app1-main-image",
							},
						},
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:            "pipe1",
								Namespace:       ns,
								ResourceVersion: "1",
							},
							Spec: argoApi.ApplicationSetSpec{
								Generators: []argoApi.ApplicationSetGenerator{
									{
										List: &argoApi.ListGenerator{
											Elements: []v1.JSON{
												{
													Raw: []byte(
														`{"cluster":"in-cluster", "codebase":"app1", ` +
															`"gitUrlPath":"company/app1", ` +
															`"imageRepository":"app1-main-image", "imageTag":"NaN", ` +
															`"namespace":"default", "stage":"stage1", ` +
															`"versionType":"default", "customValues":false}`,
													),
												},
											},
										},
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				appset := &argoApi.ApplicationSet{}
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						appset,
					),
				)

				require.Equal(t, "1", appset.GetResourceVersion())
				require.Len(t, appset.Spec.Generators, 1)
				require.Len(t, appset.Spec.Generators[0].List.Elements, 1)
				require.JSONEq(
					t,
					`{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", `+
						`"imageRepository":"app1-main-image", "imageTag":"NaN", "namespace":"default", `+
						`"stage":"stage1", "versionType":"default", "customValues":false}`,
					string(appset.Spec.Generators[0].List.Elements[0].Raw),
				)
			},
		},
		{
			name: "empty cd pipeline",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{},
							},
						},
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:            "pipe1",
								Namespace:       ns,
								ResourceVersion: "1",
							},
							Spec: argoApi.ApplicationSetSpec{},
						},
					).
					Build()
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "codebase image stream doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{"app1"},
							},
						},
						&codebaseApi.Codebase{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "app1",
								Namespace: ns,
							},
							Spec: codebaseApi.CodebaseSpec{
								GitServer:     "git-server",
								DefaultBranch: "main",
								GitUrlPath:    "/company/app1",
								Versioning: codebaseApi.Versioning{
									Type: codebaseApi.VersioningTypDefault,
								},
							},
						},
						&codebaseApi.GitServer{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "git-server",
								Namespace: ns,
							},
							Spec: codebaseApi.GitServerSpec{
								GitHost: "github.com",
								SshPort: 22,
							},
						},
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:            "pipe1",
								Namespace:       ns,
								ResourceVersion: "1",
							},
							Spec: argoApi.ApplicationSetSpec{},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get CodebaseImageStream")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "codebase doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{"app1"},
							},
						},
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:            "pipe1",
								Namespace:       ns,
								ResourceVersion: "1",
							},
							Spec: argoApi.ApplicationSetSpec{},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get Codebase")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "application set doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&cdPipeApi.CDPipeline{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: cdPipeApi.CDPipelineSpec{
								Name:         "pipe1",
								Applications: []string{"app1"},
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get ArgoApplicationSet")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "pipeline doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
				},
				Spec: cdPipeApi.StageSpec{
					Name:        "stage1",
					CdPipeline:  "pipe1",
					Namespace:   ns,
					ClusterName: cdPipeApi.InCluster,
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get CDPipeline")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cl := tt.client(t)
			m := NewArgoApplicationSetManager(cl)
			err := m.CreateApplicationSetGenerators(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)

			tt.wantErr(t, err)
			tt.wantAssert(t, cl)
		})
	}
}

func TestArgoApplicationSetManager_RemoveApplicationSetGenerators(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()

	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, argoApi.AddToScheme(scheme))
	require.NoError(t, codebaseApi.AddToScheme(scheme))

	tests := []struct {
		name       string
		stage      *cdPipeApi.Stage
		client     func(t *testing.T) client.Client
		wantErr    require.ErrorAssertionFunc
		wantAssert func(t *testing.T, cl client.Client)
	}{
		{
			name: "application set generator is removed successfully",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
					Labels: map[string]string{
						cdPipeApi.StageCdPipelineLabelName: "pipe1",
					},
				},
				Spec: cdPipeApi.StageSpec{
					Name: "stage1",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: argoApi.ApplicationSetSpec{
								Generators: []argoApi.ApplicationSetGenerator{
									{
										List: &argoApi.ListGenerator{
											Elements: []v1.JSON{
												{
													Raw: []byte(`{"stage": "stage1", "codebase": "app-to-be-removed"}`),
												},
												{
													Raw: []byte(`{"stage":"should-skip-stage", "codebase": "go-app"}`),
												},
											},
										},
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				appset := &argoApi.ApplicationSet{}
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						appset,
					),
				)

				require.Len(t, appset.Spec.Generators, 1)
				require.Len(t, appset.Spec.Generators[0].List.Elements, 1)
				require.JSONEq(
					t,
					string(appset.Spec.Generators[0].List.Elements[0].Raw),
					`{"stage":"should-skip-stage", "codebase": "go-app"}`,
				)
			},
		},
		{
			name: "application set doesn't contain generators",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
					Labels: map[string]string{
						cdPipeApi.StageCdPipelineLabelName: "pipe1",
					},
				},
				Spec: cdPipeApi.StageSpec{
					Name: "stage1",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
						},
					).
					Build()
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "application set doesn't exist",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
					Labels: map[string]string{
						cdPipeApi.StageCdPipelineLabelName: "pipe1",
					},
				},
				Spec: cdPipeApi.StageSpec{
					Name: "stage1",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects().
					Build()
			},
			wantErr:    require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "element with imageDigest is removed",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
					Labels: map[string]string{
						cdPipeApi.StageCdPipelineLabelName: "pipe1",
					},
				},
				Spec: cdPipeApi.StageSpec{
					Name: "stage1",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: argoApi.ApplicationSetSpec{
								Generators: []argoApi.ApplicationSetGenerator{
									{
										List: &argoApi.ListGenerator{
											Elements: []v1.JSON{
												{
													Raw: []byte(`{"stage":"stage1","codebase":"app1","imageTag":"0.1.0","imageRepository":"registry/app1","imageDigest":"sha256:abc123","cluster":"in-cluster","namespace":"default","repoURL":"ssh://git@github.com:22/company/app1","gitUrlPath":"company/app1","versionType":"default","customValues":false}`),
												},
												{
													Raw: []byte(`{"stage":"stage2","codebase":"app1","imageTag":"0.2.0","imageRepository":"registry/app1","cluster":"in-cluster","namespace":"ns2","repoURL":"ssh://git@github.com:22/company/app1","gitUrlPath":"company/app1","versionType":"default","customValues":false}`),
												},
											},
										},
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				appset := &argoApi.ApplicationSet{}
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						appset,
					),
				)

				require.Len(t, appset.Spec.Generators[0].List.Elements, 1)

				var remaining generatorElement
				require.NoError(t, json.Unmarshal(appset.Spec.Generators[0].List.Elements[0].Raw, &remaining))
				require.Equal(t, "stage2", remaining.Stage)
			},
		},
		{
			name: "element without imageDigest is removed",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "pipe1-stage1",
					Namespace: ns,
					Labels: map[string]string{
						cdPipeApi.StageCdPipelineLabelName: "pipe1",
					},
				},
				Spec: cdPipeApi.StageSpec{
					Name: "stage1",
				},
			},
			client: func(t *testing.T) client.Client {
				return fake.NewClientBuilder().
					WithScheme(scheme).
					WithObjects(
						&argoApi.ApplicationSet{
							ObjectMeta: metav1.ObjectMeta{
								Name:      "pipe1",
								Namespace: ns,
							},
							Spec: argoApi.ApplicationSetSpec{
								Generators: []argoApi.ApplicationSetGenerator{
									{
										List: &argoApi.ListGenerator{
											Elements: []v1.JSON{
												{
													Raw: []byte(`{"stage":"stage1","codebase":"app1","imageTag":"0.1.0","imageRepository":"registry/app1","cluster":"in-cluster","namespace":"default","repoURL":"ssh://git@github.com:22/company/app1","gitUrlPath":"company/app1","versionType":"default","customValues":false}`),
												},
												{
													Raw: []byte(`{"stage":"stage2","codebase":"app2","imageTag":"0.2.0","imageRepository":"registry/app2","imageDigest":"sha256:def456","cluster":"in-cluster","namespace":"ns2","repoURL":"ssh://git@github.com:22/company/app2","gitUrlPath":"company/app2","versionType":"default","customValues":false}`),
												},
											},
										},
									},
								},
							},
						},
					).
					Build()
			},
			wantErr: require.NoError,
			wantAssert: func(t *testing.T, cl client.Client) {
				appset := &argoApi.ApplicationSet{}
				require.NoError(t,
					cl.Get(context.Background(),
						client.ObjectKey{
							Namespace: ns,
							Name:      "pipe1",
						},
						appset,
					),
				)

				require.Len(t, appset.Spec.Generators[0].List.Elements, 1)

				var remaining generatorElement
				require.NoError(t, json.Unmarshal(appset.Spec.Generators[0].List.Elements[0].Raw, &remaining))
				require.Equal(t, "stage2", remaining.Stage)
				require.Equal(t, "sha256:def456", remaining.ImageDigest)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cl := tt.client(t)
			m := NewArgoApplicationSetManager(cl)
			err := m.RemoveApplicationSetGenerators(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage)

			tt.wantErr(t, err)
			tt.wantAssert(t, cl)
		})
	}
}

func Test_generateTemplatePatch(t *testing.T) {
	appset := generateTemplatePatch("pipe1", "/company/edp-gitops")

	Create := func(name, t string) *template.Template {
		return template.Must(template.New(name).Option("missingkey=error").Parse(t))
	}

	tmpl := Create("appset", appset)

	buf := &bytes.Buffer{}
	err := tmpl.Execute(
		buf,
		map[string]any{
			"customValues":    true,
			"versionType":     "semver",
			"imageTag":        "NaN",
			"imageRepository": "repo1",
			"imageDigest":     "",
			"codebase":        "app1",
			"stage":           "stage1",
			"repoURL":         "ssh://github.com/company/app1",
		},
	)
	require.NoError(t, err)

	y, err := yaml.YAMLToJSON(buf.Bytes())
	require.NotNil(t, y)
	require.NoError(t, err)
}

func Test_generateTemplatePatch_containsImageDigestParam(t *testing.T) {
	patch := generateTemplatePatch("pipe1", "/company/edp-gitops")

	require.Contains(t, patch, "image.digest")
	require.Contains(t, patch, ".imageDigest")
}

func Test_generateTemplatePatch_missingImageDigest_errors(t *testing.T) {
	patch := generateTemplatePatch("pipe1", "/company/edp-gitops")

	tmpl := template.Must(template.New("test").Option("missingkey=error").Parse(patch))

	buf := &bytes.Buffer{}
	err := tmpl.Execute(
		buf,
		map[string]any{
			"customValues":    true,
			"versionType":     "semver",
			"imageTag":        "NaN",
			"imageRepository": "repo1",
			"codebase":        "app1",
			"stage":           "stage1",
			"repoURL":         "ssh://github.com/company/app1",
		},
	)
	require.Error(t, err)
}

func Test_generateTemplatePatch_withDigestValue(t *testing.T) {
	patch := generateTemplatePatch("pipe1", "/company/edp-gitops")

	tmpl := template.Must(template.New("test").Option("missingkey=error").Parse(patch))

	buf := &bytes.Buffer{}
	err := tmpl.Execute(
		buf,
		map[string]any{
			"customValues":    true,
			"versionType":     "default",
			"imageTag":        "0.1.0",
			"imageRepository": "registry.example.com/app1",
			"imageDigest":     "sha256:abc123def456",
			"codebase":        "app1",
			"stage":           "stage1",
			"repoURL":         "ssh://github.com/company/app1",
		},
	)
	require.NoError(t, err)

	rendered := buf.String()
	require.Contains(t, rendered, "sha256:abc123def456")
	require.Contains(t, rendered, "image.digest")

	y, err := yaml.YAMLToJSON([]byte(rendered))
	require.NoError(t, err)
	require.NotNil(t, y)
}

func Test_generateApplicationSet_containsImageDigestHelmParam(t *testing.T) {
	pipeline := &cdPipeApi.CDPipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipe1",
			Namespace: ns,
		},
		Spec: cdPipeApi.CDPipelineSpec{
			Name:         "pipe1",
			Applications: []string{"app1"},
		},
	}

	appset := generateApplicationSet(pipeline, "ssh://git@gerrit:22/gitops")

	helmParams := appset.Spec.Template.Spec.Source.Helm.Parameters
	require.Len(t, helmParams, 3)

	paramNames := make([]string, 0, len(helmParams))
	paramValues := make(map[string]string, len(helmParams))

	for _, p := range helmParams {
		paramNames = append(paramNames, p.Name)
		paramValues[p.Name] = p.Value
	}

	require.Contains(t, paramNames, "image.tag")
	require.Contains(t, paramNames, "image.repository")
	require.Contains(t, paramNames, "image.digest")

	require.Equal(t, "{{ .imageTag }}", paramValues["image.tag"])
	require.Equal(t, "{{ .imageRepository }}", paramValues["image.repository"])
	require.Equal(t, "{{ .imageDigest }}", paramValues["image.digest"])

	require.NotNil(t, appset.Spec.TemplatePatch)
	require.Contains(t, *appset.Spec.TemplatePatch, "image.digest")
	require.Contains(t, appset.Spec.GoTemplateOptions, "missingkey=error")
}

func Test_newGeneratorElement_hasEmptyImageDigest(t *testing.T) {
	gen := generatorElement{
		Stage:           "stage1",
		Codebase:        "app1",
		ImageTag:        "NaN",
		ImageRepository: "registry/app1",
		Cluster:         "in-cluster",
		Namespace:       "default",
		RepoURL:         "ssh://git@github.com:22/company/app1",
		GitUrlPath:      "company/app1",
		VersionType:     "default",
		CustomValues:    false,
	}

	raw, err := json.Marshal(gen)
	require.NoError(t, err)
	require.Contains(t, string(raw), `"imageDigest":""`)

	var decoded generatorElement
	require.NoError(t, json.Unmarshal(raw, &decoded))
	require.Equal(t, "", decoded.ImageDigest)
}

func Test_filterAndUpdateElements_preservesOldElementsWithoutImageDigest(t *testing.T) {
	oldJSON := []byte(`{"stage":"stage1","codebase":"app1","imageTag":"0.1.0","imageRepository":"registry/app1","cluster":"in-cluster","namespace":"default","repoURL":"ssh://git@github.com:22/company/app1","gitUrlPath":"company/app1","versionType":"default","customValues":false}`)

	elements := []v1.JSON{
		{Raw: oldJSON},
	}

	newGen := generatorElement{
		Stage:           "stage1",
		Codebase:        "app1",
		ImageTag:        "NaN",
		ImageRepository: "registry/app1",
		ImageDigest:     "",
		Cluster:         "in-cluster",
		Namespace:       "default",
		RepoURL:         "ssh://git@github.com:22/company/app1",
		GitUrlPath:      "company/app1",
		VersionType:     "default",
		CustomValues:    false,
	}
	newRaw, err := json.Marshal(newGen)
	require.NoError(t, err)

	stageGenerators := map[string]v1.JSON{
		"app1-stage1": {Raw: newRaw},
	}

	filtered, remaining := filterAndUpdateElements("stage1", elements, stageGenerators)

	require.Len(t, filtered, 1)
	require.Empty(t, remaining)
	require.Equal(t, oldJSON, filtered[0].Raw)
	require.NotContains(t, string(filtered[0].Raw), "imageDigest")
}

func Test_sortElementsByStageAndCodebase_mixedElements(t *testing.T) {
	oldElement := v1.JSON{
		Raw: []byte(`{"stage":"stage1","codebase":"app-old","imageTag":"0.1.0","imageRepository":"registry/app-old","cluster":"in-cluster","namespace":"default","repoURL":"ssh://git@github.com:22/company/app-old","gitUrlPath":"company/app-old","versionType":"default","customValues":false}`),
	}
	newElement := v1.JSON{
		Raw: []byte(`{"stage":"stage1","codebase":"app-new","imageTag":"NaN","imageRepository":"registry/app-new","imageDigest":"","cluster":"in-cluster","namespace":"default","repoURL":"ssh://git@github.com:22/company/app-new","gitUrlPath":"company/app-new","versionType":"default","customValues":false}`),
	}

	elements := []v1.JSON{newElement, oldElement}

	sorted, err := sortElementsByStageAndCodebase(elements)
	require.NoError(t, err)
	require.Len(t, sorted, 2)

	var el0, el1 generatorElement
	require.NoError(t, json.Unmarshal(sorted[0].Raw, &el0))
	require.NoError(t, json.Unmarshal(sorted[1].Raw, &el1))

	require.Equal(t, "app-new", el0.Codebase)
	require.Equal(t, "app-old", el1.Codebase)

	require.NotContains(t, string(sorted[1].Raw), "imageDigest")
	require.Contains(t, string(sorted[0].Raw), "imageDigest")
}

func Test_processGeneratorListElements_mixedOldAndNewGenerators(t *testing.T) {
	oldElement := v1.JSON{
		Raw: []byte(`{"stage":"stage1","codebase":"app1","imageTag":"0.1.0","imageRepository":"registry/app1","cluster":"in-cluster","namespace":"default","repoURL":"ssh://git@github.com:22/company/app1","gitUrlPath":"company/app1","versionType":"default","customValues":false}`),
	}
	otherStageElement := v1.JSON{
		Raw: []byte(`{"stage":"stage2","codebase":"app1","imageTag":"0.2.0","imageRepository":"registry/app1","cluster":"in-cluster","namespace":"ns2","repoURL":"ssh://git@github.com:22/company/app1","gitUrlPath":"company/app1","versionType":"default","customValues":false}`),
	}

	generator := &argoApi.ApplicationSetGenerator{
		List: &argoApi.ListGenerator{
			Elements: []v1.JSON{oldElement, otherStageElement},
		},
	}

	newGen := generatorElement{
		Stage:           "stage1",
		Codebase:        "app1",
		ImageTag:        "NaN",
		ImageRepository: "registry/app1",
		ImageDigest:     "",
		Cluster:         "in-cluster",
		Namespace:       "default",
		RepoURL:         "ssh://git@github.com:22/company/app1",
		GitUrlPath:      "company/app1",
		VersionType:     "default",
		CustomValues:    false,
	}
	newRaw, err := json.Marshal(newGen)
	require.NoError(t, err)

	stageGenerators := map[string]v1.JSON{
		"app1-stage1": {Raw: newRaw},
	}

	changed, err := processGeneratorListElements("stage1", generator, stageGenerators)
	require.NoError(t, err)
	require.False(t, changed)
	require.Len(t, generator.List.Elements, 2)

	foundStage2 := false

	for _, rawel := range generator.List.Elements {
		var el generatorElement
		require.NoError(t, json.Unmarshal(rawel.Raw, &el))

		if el.Stage == "stage2" {
			foundStage2 = true
		}
	}

	require.True(t, foundStage2, "stage2 element must be preserved")
}
