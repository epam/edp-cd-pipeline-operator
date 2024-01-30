package argocd

import (
	"context"
	"encoding/json"
	"testing"

	argoApi "github.com/argoproj/argo-cd/v2/pkg/apis/application/v1alpha1"
	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	codebaseApi "github.com/epam/edp-codebase-operator/v2/api/v1"
)

func TestArgoApplicationSetManager_CreateApplicationSet(t *testing.T) {
	t.Parallel()

	ns := "default"
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
			name: "failed - codebases have different git servers",
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
								GitServer: "git-server2",
							},
						},
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "codebases have different git servers")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "failed - git server doesn't exist",
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
					).
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get GitServer")
			},
			wantAssert: func(t *testing.T, cl client.Client) {},
		},
		{
			name: "failed - codebases don't exist",
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
					WithObjects().
					Build()
			},
			wantErr: func(t require.TestingT, err error, i ...interface{}) {
				require.Error(t, err)
				require.Contains(t, err.Error(), "failed to get Codebase")
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
		tt := tt
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

	ns := "default"
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
									Type: codebaseApi.VersioningTypeEDP,
								},
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

				expected := map[string]string{
					"app1":   `{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", "imageRepository":"app1-main-image", "imageTag":"NaN", "namespace":"default", "stage":"stage1", "versionType":"default", "customValues":false}`,
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
				require.JSONEq(t,
					`{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", "imageRepository":"app1-main-image", "imageTag":"NaN", "namespace":"default", "stage":"stage1", "versionType":"default", "customValues":false}`,
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
													Raw: []byte(`{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", "imageRepository":"app1-main-image", "imageTag":"NaN", "namespace":"default", "stage":"stage1", "versionType":"default", "customValues":false}`),
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
				require.JSONEq(t,
					`{"cluster":"in-cluster", "codebase":"app1", "gitUrlPath":"company/app1", "imageRepository":"app1-main-image", "imageTag":"NaN", "namespace":"default", "stage":"stage1", "versionType":"default", "customValues":false}`,
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
		tt := tt
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

	ns := "default"
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
				require.JSONEq(t, string(appset.Spec.Generators[0].List.Elements[0].Raw), `{"stage":"should-skip-stage", "codebase": "go-app"}`)
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
	}

	for _, tt := range tests {
		tt := tt
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
