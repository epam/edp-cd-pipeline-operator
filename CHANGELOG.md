<a name="unreleased"></a>
## [Unreleased]


<a name="v2.26.0"></a>
## [v2.26.0] - 2025-07-21
### Features

- Use label selector for CodebaseImageStream ([#135](https://github.com/epam/edp-cd-pipeline-operator/issues/135))

### Bug Fixes

- Improve error handling in ClusterSecret reconciliation ([#160](https://github.com/epam/edp-cd-pipeline-operator/issues/160))
- Use the lightweight /api endpoint to check cluster connection ([#140](https://github.com/epam/edp-cd-pipeline-operator/issues/140))

### Code Refactoring

- Update codebasebranch label to improve consistency ([#135](https://github.com/epam/edp-cd-pipeline-operator/issues/135))

### Testing

- Migrate e2e test from kuttl to chainsaw ([#153](https://github.com/epam/edp-cd-pipeline-operator/issues/153))

### Routine

- Bump Codecov version ([#164](https://github.com/epam/edp-cd-pipeline-operator/issues/164))
- Bump CodeQL version ([#164](https://github.com/epam/edp-cd-pipeline-operator/issues/164))
- Allow overriding securityContext ([#162](https://github.com/epam/edp-cd-pipeline-operator/issues/162))
- Update current development version ([#156](https://github.com/epam/edp-cd-pipeline-operator/issues/156))
- Make ApplicationSet generators sorted ([#151](https://github.com/epam/edp-cd-pipeline-operator/issues/151))
- Add Multi-architecture build support ([#149](https://github.com/epam/edp-cd-pipeline-operator/issues/149))
- Update current development version ([#133](https://github.com/epam/edp-cd-pipeline-operator/issues/133))
- Rename ConfigMap from 'edp-config' to 'krci-config' ([#124](https://github.com/epam/edp-cd-pipeline-operator/issues/124))
- Add ImagePullSecrets field support ([#131](https://github.com/epam/edp-cd-pipeline-operator/issues/131))
- Update current development version ([#128](https://github.com/epam/edp-cd-pipeline-operator/issues/128))


<a name="v2.25.2"></a>
## [v2.25.2] - 2025-05-26
### Bug Fixes

- Use the lightweight /api endpoint to check cluster connection ([#140](https://github.com/epam/edp-cd-pipeline-operator/issues/140))


<a name="v2.25.1"></a>
## [v2.25.1] - 2025-03-31
### Routine

- Rename ConfigMap from 'edp-config' to 'krci-config' ([#124](https://github.com/epam/edp-cd-pipeline-operator/issues/124))
- Add ImagePullSecrets field support ([#131](https://github.com/epam/edp-cd-pipeline-operator/issues/131))


<a name="v2.25.0"></a>
## [v2.25.0] - 2025-03-22
### Features

- Implement cluster secret connectivity status check ([#118](https://github.com/epam/edp-cd-pipeline-operator/issues/118))
- Enhance cluster secret processing with new label handling ([#111](https://github.com/epam/edp-cd-pipeline-operator/issues/111))
- Add validation webhook for protected resources ([#113](https://github.com/epam/edp-cd-pipeline-operator/issues/113))
- Improve namespace deletion handling ([#114](https://github.com/epam/edp-cd-pipeline-operator/issues/114))
- Implement AWS IRSA authentication for cluster access ([#111](https://github.com/epam/edp-cd-pipeline-operator/issues/111))

### Routine

- Align ApplicationSet to use semver versioning ([#122](https://github.com/epam/edp-cd-pipeline-operator/issues/122))
- Add support for ServiceAccount annotations ([#120](https://github.com/epam/edp-cd-pipeline-operator/issues/120))
- Update current development version ([#108](https://github.com/epam/edp-cd-pipeline-operator/issues/108))


<a name="v2.24.0"></a>
## [v2.24.0] - 2025-01-24
### Features

- Enable flexible naming for default OIDC groups ([#106](https://github.com/epam/edp-cd-pipeline-operator/issues/106))
- Add Description property to CDPipeline ([#104](https://github.com/epam/edp-cd-pipeline-operator/issues/104))

### Routine

- Update current development version ([#102](https://github.com/epam/edp-cd-pipeline-operator/issues/102))


<a name="v2.23.0"></a>
## [v2.23.0] - 2025-01-10
### Features

- Add Stage CR namespace validation for multitenancy ([#96](https://github.com/epam/edp-cd-pipeline-operator/issues/96))

### Routine

- Update Capsule CRD version ([#98](https://github.com/epam/edp-cd-pipeline-operator/issues/98))
- Update current development version ([#94](https://github.com/epam/edp-cd-pipeline-operator/issues/94))


<a name="v2.22.0"></a>
## [v2.22.0] - 2024-12-12
### Features

- Add namespace validation in Stage CR ([#91](https://github.com/epam/edp-cd-pipeline-operator/issues/91))

### Bug Fixes

- Resolve repo path issue in ApplicationSet with GitServers ([#86](https://github.com/epam/edp-cd-pipeline-operator/issues/86))

### Testing

- Update capsule to 0.7.2 in e2e tests ([#88](https://github.com/epam/edp-cd-pipeline-operator/issues/88))

### Routine

- Update Pull Request Template ([#82](https://github.com/epam/edp-cd-pipeline-operator/issues/82))
- Update current development version ([#80](https://github.com/epam/edp-cd-pipeline-operator/issues/80))


<a name="v2.21.0"></a>
## [v2.21.0] - 2024-10-18
### Features

- Add ConfigMap creation for Stage ([#78](https://github.com/epam/edp-cd-pipeline-operator/issues/78))
- Add new deploy trigger type Auto-stable ([#75](https://github.com/epam/edp-cd-pipeline-operator/issues/75))
- Add cleanTemplate field to the Stage CR ([#66](https://github.com/epam/edp-cd-pipeline-operator/issues/66))
- Remove deprecated v1alpha1 versions from the operator ([#64](https://github.com/epam/edp-cd-pipeline-operator/issues/64))
- Remove CodebaseImageStream if Stage is removed ([#60](https://github.com/epam/edp-cd-pipeline-operator/issues/60))

### Routine

- Update Kubernetes version ([#66](https://github.com/epam/edp-cd-pipeline-operator/issues/66))
- Update KubeRocketCI names and documentation links ([#69](https://github.com/epam/edp-cd-pipeline-operator/issues/69))
- Update current development version ([#58](https://github.com/epam/edp-cd-pipeline-operator/issues/58))

### Documentation

- Update changelog file for release notes ([#73](https://github.com/epam/edp-cd-pipeline-operator/issues/73))
- Update CHANGELOG md ([#73](https://github.com/epam/edp-cd-pipeline-operator/issues/73))


<a name="v2.20.0"></a>
## [v2.20.0] - 2024-06-12
### Features

- Remove deprecated loft-sh kiosk ([#55](https://github.com/epam/edp-cd-pipeline-operator/issues/55))
- Narrow the scope of permissions for operator ([#52](https://github.com/epam/edp-cd-pipeline-operator/issues/52))
- Add support for multiple GitServers ([#37](https://github.com/epam/edp-cd-pipeline-operator/issues/37))
- Create ArgoCD cluster secret ([#30](https://github.com/epam/edp-cd-pipeline-operator/issues/30))

### Routine

- Update helm-docs to the latest stable ([#44](https://github.com/epam/edp-cd-pipeline-operator/issues/44))
- Remove unused RBAC for secretManager own parameter ([#52](https://github.com/epam/edp-cd-pipeline-operator/issues/52))
- Bump to Go 1.22 ([#49](https://github.com/epam/edp-cd-pipeline-operator/issues/49))
- Use Go cache for helm-docs installation ([#44](https://github.com/epam/edp-cd-pipeline-operator/issues/44))
- Update kuttle to version 0.16 ([#44](https://github.com/epam/edp-cd-pipeline-operator/issues/44))
- Add codeowners file to the repo ([#40](https://github.com/epam/edp-cd-pipeline-operator/issues/40))
- Migrate from gerrit to github pipelines ([#35](https://github.com/epam/edp-cd-pipeline-operator/issues/35))
- Bump google.golang.org/protobuf from 1.31.0 to 1.33.0 ([#30](https://github.com/epam/edp-cd-pipeline-operator/issues/30))
- Update current development version ([#29](https://github.com/epam/edp-cd-pipeline-operator/issues/29))


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

<a name="v2.14.1"></a>
## [v2.14.1] - 2023-03-29

<a name="v2.14.0"></a>
## [v2.14.0] - 2023-03-24

<a name="v2.13.0"></a>
## [v2.13.0] - 2022-12-06

<a name="v2.12.1"></a>
## [v2.12.1] - 2022-10-28

<a name="v2.12.0"></a>
## [v2.12.0] - 2022-08-26

<a name="v2.11.0"></a>
## [v2.11.0] - 2022-05-25

<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06

<a name="v2.9.0"></a>
## [v2.9.0] - 2021-12-03

<a name="v2.8.2"></a>
## [v2.8.2] - 2021-12-03

<a name="v2.8.1"></a>
## [v2.8.1] - 2021-12-03

<a name="v2.8.0"></a>
## [v2.8.0] - 2021-12-03

<a name="v2.7.1"></a>
## [v2.7.1] - 2021-12-03

<a name="v2.7.0"></a>
## [v2.7.0] - 2021-12-03

<a name="v2.3.0-58.0.20210420131932-c2003069fbbd"></a>
## [v2.3.0-58.0.20210420131932-c2003069fbbd] - 2021-04-20

<a name="v2.3.0-58"></a>
## [v2.3.0-58] - 2020-01-21

<a name="v2.3.0-55"></a>
## [v2.3.0-55] - 2020-01-13

<a name="v2.3.0-54"></a>
## [v2.3.0-54] - 2019-12-25

<a name="v2.2.1-57"></a>
## [v2.2.1-57] - 2020-01-21

<a name="v2.2.1-56"></a>
## [v2.2.1-56] - 2020-01-13

<a name="v2.2.0-53"></a>
## [v2.2.0-53] - 2019-12-20

<a name="v2.2.0-52"></a>
## [v2.2.0-52] - 2019-12-13

<a name="v2.2.0-51"></a>
## [v2.2.0-51] - 2019-12-11

<a name="v2.2.0-50"></a>
## [v2.2.0-50] - 2019-12-10

<a name="v2.2.0-49"></a>
## [v2.2.0-49] - 2019-12-06

<a name="v2.2.0-48"></a>
## [v2.2.0-48] - 2019-12-05

<a name="v2.2.0-47"></a>
## [v2.2.0-47] - 2019-12-05

<a name="v2.2.0-46"></a>
## [v2.2.0-46] - 2019-12-04

<a name="v2.2.0-45"></a>
## [v2.2.0-45] - 2019-12-02

<a name="v2.2.0-44"></a>
## [v2.2.0-44] - 2019-11-22

<a name="v2.2.0-43"></a>
## [v2.2.0-43] - 2019-11-21

<a name="v2.2.0-42"></a>
## [v2.2.0-42] - 2019-11-13

<a name="v2.2.0-41"></a>
## [v2.2.0-41] - 2019-10-28

<a name="v2.2.0-40"></a>
## [v2.2.0-40] - 2019-10-25

<a name="v2.2.0-39"></a>
## [v2.2.0-39] - 2019-10-24

<a name="v2.2.0-38"></a>
## [v2.2.0-38] - 2019-09-30

<a name="v2.1.0-37"></a>
## [v2.1.0-37] - 2019-09-26

<a name="v2.1.0-36"></a>
## v2.1.0-36 - 2019-09-16

[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.26.0...HEAD
[v2.26.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.25.2...v2.26.0
[v2.25.2]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.25.1...v2.25.2
[v2.25.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.25.0...v2.25.1
[v2.25.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.24.0...v2.25.0
[v2.24.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.23.0...v2.24.0
[v2.23.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.22.0...v2.23.0
[v2.22.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.21.0...v2.22.0
[v2.21.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.20.0...v2.21.0
[v2.20.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.19.0...v2.20.0
[v2.19.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.18.0...v2.19.0
[v2.18.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.17.0...v2.18.0
[v2.17.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.1...v2.17.0
[v2.16.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.0...v2.16.1
[v2.16.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.15.0...v2.16.0
[v2.15.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.14.1...v2.15.0
[v2.14.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.14.0...v2.14.1
[v2.14.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.13.0...v2.14.0
[v2.13.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.12.1...v2.13.0
[v2.12.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.12.0...v2.12.1
[v2.12.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.11.0...v2.12.0
[v2.11.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.10.0...v2.11.0
[v2.10.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.2...v2.9.0
[v2.8.2]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.1...v2.8.2
[v2.8.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.1...v2.8.0
[v2.7.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.3.0-58.0.20210420131932-c2003069fbbd...v2.7.0
[v2.3.0-58.0.20210420131932-c2003069fbbd]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.3.0-58...v2.3.0-58.0.20210420131932-c2003069fbbd
[v2.3.0-58]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.3.0-55...v2.3.0-58
[v2.3.0-55]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.3.0-54...v2.3.0-55
[v2.3.0-54]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.1-57...v2.3.0-54
[v2.2.1-57]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.1-56...v2.2.1-57
[v2.2.1-56]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-53...v2.2.1-56
[v2.2.0-53]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-52...v2.2.0-53
[v2.2.0-52]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-51...v2.2.0-52
[v2.2.0-51]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-50...v2.2.0-51
[v2.2.0-50]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-49...v2.2.0-50
[v2.2.0-49]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-48...v2.2.0-49
[v2.2.0-48]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-47...v2.2.0-48
[v2.2.0-47]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-46...v2.2.0-47
[v2.2.0-46]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-45...v2.2.0-46
[v2.2.0-45]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-44...v2.2.0-45
[v2.2.0-44]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-43...v2.2.0-44
[v2.2.0-43]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-42...v2.2.0-43
[v2.2.0-42]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-41...v2.2.0-42
[v2.2.0-41]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-40...v2.2.0-41
[v2.2.0-40]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-39...v2.2.0-40
[v2.2.0-39]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.2.0-38...v2.2.0-39
[v2.2.0-38]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.1.0-37...v2.2.0-38
[v2.1.0-37]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.1.0-36...v2.1.0-37
