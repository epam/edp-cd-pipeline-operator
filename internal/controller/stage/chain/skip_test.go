package chain

import (
	"context"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"

	cdPipeApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

func TestSkip_ServeRequest(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		stage   *cdPipeApi.Stage
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "skip chain",
			stage: &cdPipeApi.Stage{
				ObjectMeta: metaV1.ObjectMeta{
					Name:      "stage-1",
					Namespace: "default",
				},
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tt.wantErr(t, Skip{}.ServeRequest(ctrl.LoggerInto(context.Background(), logr.Discard()), tt.stage))
		})
	}
}
