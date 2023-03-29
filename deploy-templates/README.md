# cd-pipeline-operator

![Version: 2.14.1](https://img.shields.io/badge/Version-2.14.1-informational?style=flat-square) ![Type: application](https://img.shields.io/badge/Type-application-informational?style=flat-square) ![AppVersion: 2.14.1](https://img.shields.io/badge/AppVersion-2.14.1-informational?style=flat-square)

A Helm chart for EDP CD Pipeline Operator

**Homepage:** <https://epam.github.io/edp-install/>

## Maintainers

| Name | Email | Url |
| ---- | ------ | --- |
| epmd-edp | <SupportEPMD-EDP@epam.com> | <https://solutionshub.epam.com/solution/epam-delivery-platform> |
| sergk |  | <https://github.com/SergK> |

## Source Code

* <https://github.com/epam/edp-cd-pipeline-operator>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| affinity | string | `nil` |  |
| annotations | object | `{}` |  |
| global.edpName | string | `""` | namespace or a project name (in case of OpenShift) |
| global.kioskEnabled | bool | `false` |  |
| global.platform | string | `"kubernetes"` | platform type that can be "kubernetes" or "openshift" |
| image.repository | string | `"epamedp/cd-pipeline-operator"` | EDP cd-pipeline-operator Docker image name. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/cd-pipeline-operator) |
| image.tag | string | `nil` | EDP cd-pipeline-operator Docker image tag. The released image can be found on [Dockerhub](https://hub.docker.com/r/epamedp/cd-pipeline-operator/tags) |
| imagePullPolicy | string | `"IfNotPresent"` |  |
| manageNamespace | bool | `true` | should the operator manage(create/delete) namespaces for stages |
| name | string | `"cd-pipeline-operator"` | component name |
| nodeSelector | object | `{}` |  |
| resources.limits.memory | string | `"192Mi"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| tolerations | list | `[]` |  |

