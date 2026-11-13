package runtimesecurity

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFromEnv(t *testing.T) {
	tests := []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults",
			want: Config{
				SecretName: "ci-defectdojo", ProductType: "KubeRocketCI", PollInterval: 5 * time.Minute, Concurrency: 2,
			},
		},
		{
			name: "all values",
			env: map[string]string{
				EnabledEnv: "true", SecretNameEnv: "dd", ProductTypeEnv: "Platform", PollIntervalEnv: "90s", ConcurrencyEnv: "4",
			},
			want: Config{Enabled: true, SecretName: "dd", ProductType: "Platform", PollInterval: 90 * time.Second, Concurrency: 4},
		},
		{
			name: "disabled explicitly",
			env:  map[string]string{EnabledEnv: "false"},
			want: Config{SecretName: "ci-defectdojo", ProductType: "KubeRocketCI", PollInterval: 5 * time.Minute, Concurrency: 2},
		},
		{name: "invalid enabled", env: map[string]string{EnabledEnv: "yes please"}, wantErr: EnabledEnv},
		{name: "invalid interval", env: map[string]string{PollIntervalEnv: "soon"}, wantErr: PollIntervalEnv},
		{name: "zero interval", env: map[string]string{PollIntervalEnv: "0s"}, wantErr: PollIntervalEnv},
		{name: "invalid concurrency", env: map[string]string{ConcurrencyEnv: "many"}, wantErr: ConcurrencyEnv},
		{name: "zero concurrency", env: map[string]string{ConcurrencyEnv: "0"}, wantErr: ConcurrencyEnv},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, name := range []string{EnabledEnv, SecretNameEnv, ProductTypeEnv, PollIntervalEnv, ConcurrencyEnv} {
				t.Setenv(name, tt.env[name])
			}

			got, err := ConfigFromEnv()

			if tt.wantErr != "" {
				require.ErrorContains(t, err, tt.wantErr)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
