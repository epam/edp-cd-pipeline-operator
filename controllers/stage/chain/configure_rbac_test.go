package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/require"
	coreApi "k8s.io/api/core/v1"
	rbacApi "k8s.io/api/rbac/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/chain/util"
	"github.com/epam/edp-cd-pipeline-operator/v2/controllers/stage/rbac"
	"github.com/epam/edp-cd-pipeline-operator/v2/pkg/platform"
)

func TestConfigureRbac_ServeRequest(t *testing.T) {
	const namespace = "test-ns"

	scheme := runtime.NewScheme()
	require.NoError(t, cdPipeApi.AddToScheme(scheme))
	require.NoError(t, rbacApi.AddToScheme(scheme))
	require.NoError(t, coreApi.AddToScheme(scheme))

	tests := []struct {
		name      string
		stage     *cdPipeApi.Stage
		objects   []runtime.Object
		platform  string
		wantErr   require.ErrorAssertionFunc
		wantCheck func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client)
	}{
		{
			name: "rbac configuration for kubernetes platform is successful",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-stage",
				},
			},
			objects: []runtime.Object{
				&coreApi.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: namespace,
					},
				},
			},
			platform: platform.Kubernetes,
			wantErr:  require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				targetNamespace := util.GenerateNamespaceName(stage)

				rbJenkins := &rbacApi.RoleBinding{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      jenkinsAdminRbName,
					Namespace: targetNamespace,
				}, rbJenkins))

				viewGroup := &rbacApi.RoleBinding{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      generateViewGroupRoleBindingName(stage.Namespace),
					Namespace: targetNamespace,
				}, viewGroup)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name: "rbac for kubernetes platform already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-stage",
				},
			},
			objects: []runtime.Object{
				&coreApi.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: namespace,
					},
				},
				&rbacApi.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name: jenkinsAdminRbName,
						Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Namespace: namespace,
								Name:      "test-stage",
							},
						}),
					},
				},
			},
			platform: platform.Kubernetes,
			wantErr:  require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				targetNamespace := util.GenerateNamespaceName(stage)

				rbJenkins := &rbacApi.RoleBinding{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      jenkinsAdminRbName,
					Namespace: targetNamespace,
				}, rbJenkins))

				viewGroup := &rbacApi.RoleBinding{}
				err := k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      generateViewGroupRoleBindingName(stage.Namespace),
					Namespace: targetNamespace,
				}, viewGroup)
				require.Error(t, err)
				require.True(t, k8sErrors.IsNotFound(err))
			},
		},
		{
			name: "rbac configuration for openshift platform is successful",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-stage",
				},
			},
			objects: []runtime.Object{
				&coreApi.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: namespace,
					},
				},
			},
			platform: platform.Openshift,
			wantErr:  require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				targetNamespace := util.GenerateNamespaceName(stage)

				rbJenkins := &rbacApi.RoleBinding{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      jenkinsAdminRbName,
					Namespace: targetNamespace,
				}, rbJenkins))

				viewGroup := &rbacApi.RoleBinding{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      generateViewGroupRoleBindingName(stage.Namespace),
					Namespace: targetNamespace,
				}, viewGroup))
			},
		},
		{
			name: "rbac for openshift platform already exists",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Namespace: namespace,
					Name:      "test-stage",
				},
			},
			objects: []runtime.Object{
				&coreApi.Namespace{
					ObjectMeta: metaV1.ObjectMeta{
						Name: namespace,
					},
				},
				&rbacApi.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name: jenkinsAdminRbName,
						Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Namespace: namespace,
								Name:      "test-stage",
							},
						}),
					},
				},
				&rbacApi.RoleBinding{
					ObjectMeta: metaV1.ObjectMeta{
						Name: generateViewGroupRoleBindingName(namespace),
						Namespace: util.GenerateNamespaceName(&cdPipeApi.Stage{
							ObjectMeta: metaV1.ObjectMeta{
								Namespace: namespace,
								Name:      "test-stage",
							},
						}),
					},
				},
			},
			platform: platform.Openshift,
			wantErr:  require.NoError,
			wantCheck: func(t *testing.T, stage *cdPipeApi.Stage, k8sClient client.Client) {
				targetNamespace := util.GenerateNamespaceName(stage)

				rbJenkins := &rbacApi.RoleBinding{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      jenkinsAdminRbName,
					Namespace: targetNamespace,
				}, rbJenkins))

				viewGroup := &rbacApi.RoleBinding{}
				require.NoError(t, k8sClient.Get(context.Background(), client.ObjectKey{
					Name:      generateViewGroupRoleBindingName(stage.Namespace),
					Namespace: targetNamespace,
				}, viewGroup))
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv(platform.TypeEnv, tt.platform)

			k8sClient := fake.NewClientBuilder().WithScheme(scheme).WithRuntimeObjects(tt.objects...).Build()

			h := ConfigureRbac{
				client: k8sClient,
				log:    logr.Discard(),
				rbac:   rbac.NewRbacManager(k8sClient, logr.Discard()),
			}

			err := h.ServeRequest(tt.stage)
			tt.wantErr(t, err)
			tt.wantCheck(t, tt.stage, k8sClient)
		})
	}
}
