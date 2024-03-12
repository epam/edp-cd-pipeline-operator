<a name="unreleased"></a>
## [Unreleased]


<a name="v2.19.0"></a>
## [v2.19.0] - 2024-03-12
### Features

- Use kubeconfig format for external clusters ([#28](https://github.com/epam/edp-cd-pipeline-operator/issues/28))
- Add ArgoCD ApplicationSet customValues flag ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- Add triggerTemplate field to the Stage ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Use Argo CD ApplicationSet to manage deployments across CDPipeline ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))

### Bug Fixes

- Fix string concatenation for generating gitopsRepoUrl ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- We have to use git over ssh for customValues in ApplicationSet ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- ArgoCD ApplicationSet customValues invalid patch ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- Generate ApplicationSet with pipeline name and namespace ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Generate ApplicationSet with pipeline name and namespace ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))

### Code Refactoring

- Align default TriggerTemplate name ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))

### Testing

- Ensure Argo CD ApplicationSet has expected values ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))

### Routine

- Add link to guide for managing namespace ([#162](https://github.com/epam/edp-cd-pipeline-operator/issues/162))
- Bump argo cd dependency ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- Bump github.com/argoproj/argo-cd/v2 ([#24](https://github.com/epam/edp-cd-pipeline-operator/issues/24))
- Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#21](https://github.com/epam/edp-cd-pipeline-operator/issues/21))
- Bump github.com/go-git/go-git/v5 from 5.8.1 to 5.11.0 ([#21](https://github.com/epam/edp-cd-pipeline-operator/issues/21))
- Remove deprecated jobProvisioning field from Stage ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Update current development version ([#19](https://github.com/epam/edp-cd-pipeline-operator/issues/19))

### Documentation

- Add description for secretManager parameter ([#27](https://github.com/epam/edp-cd-pipeline-operator/issues/27))
- Add a link to the ESO configuration in the values.yaml file ([#26](https://github.com/epam/edp-cd-pipeline-operator/issues/26))
- Update README md file ([#132](https://github.com/epam/edp-cd-pipeline-operator/issues/132))


<a name="v2.18.0"></a>
## [v2.18.0] - 2023-12-18
### Bug Fixes

- Deleting Stage with invalid cluster configuration ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))

### Testing

- Update caspule version to the latest stable ([#28](https://github.com/epam/edp-cd-pipeline-operator/issues/28))
- Update caspule version to the latest stable ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))

### Routine

- Update release flow GH Action ([#17](https://github.com/epam/edp-cd-pipeline-operator/issues/17))
- Update GH Actions version of the steps ([#17](https://github.com/epam/edp-cd-pipeline-operator/issues/17))
- Update current development version ([#16](https://github.com/epam/edp-cd-pipeline-operator/issues/16))


<a name="v2.17.0"></a>
## [v2.17.0] - 2023-11-03
### Features

- Enable Capsule Tenant modification from values.yaml ([#13](https://github.com/epam/edp-cd-pipeline-operator/issues/13))
- Add multi-cluster support for the operator ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))

### Bug Fixes

- Add access to namespace secrets to get external cluster access ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Stage creation failed with custom namespace ([#15](https://github.com/epam/edp-cd-pipeline-operator/issues/15))
- Namespace is not cleaned for the external cluster ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Use edp-config configmap for docker registry url ([#11](https://github.com/epam/edp-cd-pipeline-operator/issues/11))
- Skip multi-tenancy engines for external cluster ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))

### Testing

- Add e2e for the custom namespace feature ([#15](https://github.com/epam/edp-cd-pipeline-operator/issues/15))
- Run e2e tests on Github PR/Merge ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))
- Add e2e tests. Start with capsule tenancy feature ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))

### Routine

- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#12](https://github.com/epam/edp-cd-pipeline-operator/issues/12))
- Remove jenkins admin-console perf operator logic ([#8](https://github.com/epam/edp-cd-pipeline-operator/issues/8))
- Upgrade Go to 1.20 ([#7](https://github.com/epam/edp-cd-pipeline-operator/issues/7))
- Update current development version ([#6](https://github.com/epam/edp-cd-pipeline-operator/issues/6))


<a name="v2.16.1"></a>
## [v2.16.1] - 2023-09-25
### Routine

- Upgrade Go to 1.20 ([#7](https://github.com/epam/edp-cd-pipeline-operator/issues/7))


<a name="v2.16.0"></a>
## [v2.16.0] - 2023-09-21
### Features

- Create capsule tenant resource ([#4](https://github.com/epam/edp-cd-pipeline-operator/issues/4))
- Add capsule support for multi-tenancy ([#9](https://github.com/epam/edp-cd-pipeline-operator/issues/9))

### Code Refactoring

- Remove deprecated edpName parameter ([#5](https://github.com/epam/edp-cd-pipeline-operator/issues/5))
- Move tenancyEngine flag out of global section ([#9](https://github.com/epam/edp-cd-pipeline-operator/issues/9))

### Routine

- Update current development version ([#3](https://github.com/epam/edp-cd-pipeline-operator/issues/3))

### BREAKING CHANGE:


Helm parameter kioskEnabled was removed. Use instead --set global.tenancyEngine=kiosk.


<a name="v2.15.0"></a>
## [v2.15.0] - 2023-08-17

[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.19.0...HEAD
[v2.19.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.18.0...v2.19.0
[v2.18.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.17.0...v2.18.0
[v2.17.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.1...v2.17.0
[v2.16.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.0...v2.16.1
[v2.16.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.15.0...v2.16.0
[v2.15.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.14.1...v2.15.0
