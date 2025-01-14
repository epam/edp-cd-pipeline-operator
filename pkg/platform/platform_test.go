package platform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetPlatformTypeEnv_Success(t *testing.T) {
	stubPlatformType := "stubPlatformType"

	err := os.Setenv(TypeEnv, stubPlatformType)
	if err != nil {
		t.Fatalf("cannot set an env variable: %v", err)
	}

	defer func() {
		err := os.Unsetenv(TypeEnv)
		if err != nil {
			t.Fatalf("cannot unset an env variable: %v", err)
		}
	}()

	platformType := GetPlatformTypeEnv()
	assert.Equal(t, stubPlatformType, platformType)
}

func TestGetPlatformTypeEnv_PlatformTypeIsNotSet(t *testing.T) {
	err := os.Unsetenv(TypeEnv)
	if err != nil {
		t.Fatalf("cannot unset an env variable: %v", err)
	}

	platformType := GetPlatformTypeEnv()
	assert.Equal(t, Kubernetes, platformType)
}

func TestIsKubernetes(t *testing.T) {
	tests := []struct {
		name   string
		setEnv func(t *testing.T)
		want   bool
	}{
		{
			name: "platform type is kubernetes",
			setEnv: func(t *testing.T) {
				t.Setenv(TypeEnv, Kubernetes)
			},
			want: true,
		},
		{
			name: "platform type is openshift",
			setEnv: func(t *testing.T) {
				t.Setenv(TypeEnv, Openshift)
			},
			want: false,
		},
		{
			name: "platform type is not set",
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.setEnv != nil {
				tt.setEnv(t)
			}

			assert.Equal(t, IsKubernetes(), tt.want)
		})
	}
}

func TestManageNamespace(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T)
		want    bool
	}{
		{
			name: "manage namespace is enabled",
			prepare: func(t *testing.T) {
				t.Setenv(ManageNamespaceEnv, "true")
			},
			want: true,
		},
		{
			name: "manage namespace is disabled",
			prepare: func(t *testing.T) {
				t.Setenv(ManageNamespaceEnv, "false")
			},
			want: false,
		},
		{
			name: "manage namespace is not set",
			prepare: func(t *testing.T) {
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)
			assert.Equal(t, tt.want, ManageNamespace())
		})
	}
}

func TestGetOIDCAdminGroupName(t *testing.T) {
	tests := []struct {
		name    string
		prepare func(t *testing.T)
		want    string
	}{
		{
			name: "oidc admin group name is set",
			prepare: func(t *testing.T) {
				t.Setenv(OIDCAdminGroupName, "oidc-admin-group")
			},
			want: "oidc-admin-group",
		},
		{
			name:    "oidc admin group name is not set",
			prepare: func(t *testing.T) {},
			want:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.prepare(t)

			assert.Equal(t, tt.want, GetOIDCAdminGroupName())
		})
	}
}

func TestGetOIDCDeveloperGroupName(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T)
		want  string
	}{
		{
			name: "oidc developer group name is set",
			setup: func(t *testing.T) {
				t.Setenv(OIDCDeveloperGroupName, "oidc-developer-group")
			},
			want: "oidc-developer-group",
		},
		{
			name:  "oidc developer group name is not set",
			setup: func(t *testing.T) {},
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup(t)

			assert.Equal(t, tt.want, GetOIDCDeveloperGroupName())
		})
	}
}
