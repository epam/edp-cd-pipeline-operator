# edp-cd-pipeline-operator

`github.com/epam/edp-cd-pipeline-operator/v2` — a Go kubebuilder operator that reconciles CD pipeline entities (`CDPipeline`, `Stage`) and manages environment promotion on Kubernetes and OpenShift. CRD API group: `v2.edp.epam.com/v1`.

## Build & Test

```bash
make build          # compile binary → dist/manager-<arch>
make test           # unit tests via setup-envtest (downloads envtest binaries automatically)
make lint           # golangci-lint (config: .golangci.yaml)
make lint-fix       # golangci-lint with auto-fix
make fmt            # go fmt
make vet            # go vet
```

Run a single test package:

```bash
go test ./internal/controller/stage/...
go test ./pkg/argocd/...
```

E2e tests (requires kind cluster):

```bash
make start-kind     # creates kind cluster named "cd-pipeline-operator"
make e2e            # builds image, loads into kind, runs Chainsaw tests
```

## Code Generation

```bash
make generate     # regenerate DeepCopy methods (run after editing api/v1/ types)
make manifests    # regenerate CRDs + RBAC → deploy-templates/crds/ and config/crd/bases/ (also calls make api-docs)
make mocks        # regenerate testify mocks via mockery (config: .mockery.yaml)
make api-docs     # regenerate docs/api.md from CRDs
make helm-docs    # regenerate deploy-templates/README.md from chart values
```

`api/v1/zz_generated.deepcopy.go` and `pkg/aws/mocks/` are generated — never edit by hand.
`deploy-templates/README.md` and `docs/api.md` are generated — run `make validate-docs` in CI to catch drift.

## Architecture

```
cmd/main.go
  └── registers three controllers + admission webhooks with controller-runtime manager

api/v1/
  ├── cdpipeline_types.go   — CDPipeline spec/status
  ├── stage_types.go        — Stage spec/status; trigger types (Auto, Manual, Auto-stable)
  └── zz_generated.deepcopy.go

internal/controller/
  ├── cdpipeline/           — ReconcileCDPipeline: creates Argo CD ApplicationSet per pipeline
  ├── stage/                — ReconcileStage: chain-of-responsibility for stage lifecycle
  │   ├── stage_controller.go
  │   ├── event_handler.go  — watches CDPipeline changes, enqueues owned Stages
  │   └── chain/
  │       ├── factory.go    — CreateChain() / CreateDeleteChain()
  │       ├── chain.go      — sequential handler runner
  │       └── <step>.go     — individual chain handlers (put_namespace, configure_*_rbac, etc.)
  └── clustersecret/        — watches Secrets labeled app.edp.epam.com/secret-type=cluster,
                              converts kubeconfig/IRSA secrets → Argo CD cluster secrets

pkg/
  ├── argocd/               — ArgoApplicationSetManager: create/update/remove ApplicationSet generators
  ├── aws/                  — EKS IRSA token generator (sigs.k8s.io/aws-iam-authenticator)
  ├── multiclusterclient/   — ClientProvider: resolves in-cluster or external cluster client from Secret
  ├── objectmodifier/       — StageBatchModifier: patches Stage label + CDPipeline owner ref pre-reconcile
  ├── platform/             — reads PLATFORM_TYPE / TENANCY_ENGINE / MANAGE_NAMESPACE env vars
  ├── rbac/                 — RbacManager: create/delete RoleBindings cross-namespace
  ├── webhook/              — admission webhooks for CDPipeline and Stage validation
  └── util/consts/          — shared constants (CDPipelineKind, status strings)

deploy-templates/           — Helm chart (crds/ auto-generated, README auto-generated)
config/                     — Kustomize manifests for out-of-cluster deployment
tests/e2e/chainsaw/         — Chainsaw-based e2e scenarios (webhooks, capsule-integration)
```

### Key Design Points

**Stage reconciliation chain** (`internal/controller/stage/chain/factory.go`): `CreateChain()` wires ~10 sequential handlers — namespace creation, codebase image stream labels, RBAC setup, Argo CD ApplicationSet generator injection, configmap creation. Each handler implements `handler.CdStageHandler`. Deletion runs `CreateDeleteChain()` which strips labels and removes Argo generators before deleting the namespace.

**Uncached client**: The manager cache is scoped to the operator namespace. A second `client.New` (uncached, cluster-scoped) is created in `cmd/main.go` for cross-namespace operations (RoleBindings in Stage namespaces, external cluster Secrets).

**Platform branching**: `pkg/platform/` reads `PLATFORM_TYPE` env var (`kubernetes` or `openshift`). OpenShift stages use `Project` instead of `Namespace` — see `put_openshift_project.go` vs `put_namespace.go`. Capsule tenancy is toggled via `TENANCY_ENGINE=capsule`.

**Multi-cluster Stages**: `Stage.Spec.ClusterName` defaults to `"in-cluster"`. When set to an external cluster name, `multiclusterclient.ClientProvider` looks up a Secret labeled `app.edp.epam.com/secret-type=cluster` and builds a remote client from the kubeconfig stored in it.

**Webhooks**: Disabled when `ENABLE_WEBHOOKS=false`. Registered for both `CDPipeline` and `Stage` — validate referential integrity (e.g., Stage references existing CDPipeline, no duplicate Stage namespaces).

**Stage deletion order**: Stages must be deleted last-to-first (highest `Spec.Order` first). The controller enforces this — a non-last Stage requeues its deletion until parent Stages are gone.

## Conventions

- Tests use Ginkgo v2 + Gomega; mock files end in `_generated.mock.go`.
- RBAC markers (`// +kubebuilder:rbac:...`) go above the owning controller's `Reconcile`, per operator-sdk convention; call sites outside a controller (chain handlers, `cmd/main.go`) declare their own. Run `make manifests` to regenerate. OpenShift-only grants are hand-written in the chart's `*_openshift.yaml` templates, not markers — `config/rbac/` stays a Kubernetes role.
- `Stage.Spec.Namespace` is immutable (CEL validation rule in the CRD).
- The `app.edp.epam.com/cdPipelineName` label is auto-set on Stage by `objectmodifier` before reconcile — used for listing Stages by pipeline.
