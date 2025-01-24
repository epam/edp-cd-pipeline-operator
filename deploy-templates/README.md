# cd-pipeline-operator

![Version: 2.25.0-SNAPSHOT](https://img.shields.io/badge/Version-2.25.0--SNAPSHOT-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.25.0-SNAPSHOT](https://img.shields.io/badge/AppVersion-2.25.0--SNAPSHOT-informational?style=flat-square)

A Helm chart for KubeRocketCI CD Pipeline Operator

**Homepage:** <https://docs.kuberocketci.io/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/kuberocketci> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-cd-pipeline-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | string | `nil` |  |
| annotations | object | `{}` |  |
| capsuleTenant | object | `{"create":true,"spec":null}` | Required tenancyEngine: capsule. Specify Capsule Tenant specification for Environments. |
| global.adminGroupName | string | `""` | specify the admin OIDC group name. If empty, default {{ .Release.Namespace }}-oidc-admins. |
| global.developerGroupName | string | `""` | specify the developer OIDC group name. If empty, default {{ .Release.Namespace }}-oidc-developers. |
| global.platform | string | `"kubernetes"` | platform type that can be "kubernetes" or "openshift" |
| image.repository | string | `"epamedp/cd-pipeline-operator"` | KubeRocketCI cd-pipeline-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/cd-pipeline-operator) |
| image.tag | string | `nil` | KubeRocketCI cd-pipeline-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/cd-pipeline-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| manageNamespace | bool | `true` | should the operator manage(create/delete) namespaces for stages Refer to the guide for managing namespace (https://docs.kuberocketci.io/docs/operator-guide/auth/namespace-management) |
| name | string | `"cd-pipeline-operator"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| secretManager | string | `"none"` | Flag indicating whether the operator should manage secrets for stages. This parameter controls the provisioning of the 'regcred' secret within deployed environments, facilitating access to private container registries. Set the parameter to "none" under the following conditions:   - If 'global.dockerRegistry.type=ecr' and IRSA is enabled, or   - If 'global.dockerRegistry.type=openshift'. For private registries, choose the most appropriate method to provide credentials to deployed environments. Refer to the guide for managing container registries (https://docs.kuberocketci.io/docs/user-guide/manage-container-registries). Possible values: own/eso/none.   - own: Copies the secret once from the parent namespace, without subsequent reconciliation. If updated in the parent namespace, manual updating in all created namespaces is required.   - eso: The secret will be managed by the External Secrets Operator (requires installation and configuration in the cluster: https://docs.kuberocketci.io/docs/operator-guide/secrets-management/install-external-secrets-operator).   - none: Disables secrets management logic. |
| tenancyEngine | string | `"none"` | defines the type of the tenant engine that can be "none" or "capsule"; for Stages with external cluster tenancyEngine will be ignored |
| tolerations | list | `[]` |  |

