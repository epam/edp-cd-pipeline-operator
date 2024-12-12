<a name="unreleased"></a>
## [Unreleased]


<a name="v2.22.0"></a>
## v2.22.0 - 2024-12-11
### Features

- Add namespace validation in Stage CR ([#91](https://github.com/epam/edp-cd-pipeline-operator/issues/91))
- Add ConfigMap creation for Stage ([#78](https://github.com/epam/edp-cd-pipeline-operator/issues/78))
- Add new deploy trigger type Auto-stable ([#75](https://github.com/epam/edp-cd-pipeline-operator/issues/75))
- Add cleanTemplate field to the Stage CR ([#66](https://github.com/epam/edp-cd-pipeline-operator/issues/66))
- Remove deprecated v1alpha1 versions from the operator ([#64](https://github.com/epam/edp-cd-pipeline-operator/issues/64))
- Remove CodebaseImageStream if Stage is removed ([#60](https://github.com/epam/edp-cd-pipeline-operator/issues/60))
- Remove deprecated loft-sh kiosk ([#55](https://github.com/epam/edp-cd-pipeline-operator/issues/55))
- Narrow the scope of permissions for operator ([#52](https://github.com/epam/edp-cd-pipeline-operator/issues/52))
- Add support for multiple GitServers ([#37](https://github.com/epam/edp-cd-pipeline-operator/issues/37))
- Create ArgoCD cluster secret ([#30](https://github.com/epam/edp-cd-pipeline-operator/issues/30))
- Use kubeconfig format for external clusters ([#28](https://github.com/epam/edp-cd-pipeline-operator/issues/28))
- Add ArgoCD ApplicationSet customValues flag ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- Add triggerTemplate field to the Stage ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Use Argo CD ApplicationSet to manage deployments across CDPipeline ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Enable Capsule Tenant modification from values.yaml ([#13](https://github.com/epam/edp-cd-pipeline-operator/issues/13))
- Add multi-cluster support for the operator ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Create capsule tenant resource ([#4](https://github.com/epam/edp-cd-pipeline-operator/issues/4))
- Add capsule support for multi-tenancy ([#9](https://github.com/epam/edp-cd-pipeline-operator/issues/9))

### Bug Fixes

- Resolve repo path issue in ApplicationSet with GitServers ([#86](https://github.com/epam/edp-cd-pipeline-operator/issues/86))
- Fix string concatenation for generating gitopsRepoUrl ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- We have to use git over ssh for customValues in ApplicationSet ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- ArgoCD ApplicationSet customValues invalid patch ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- Generate ApplicationSet with pipeline name and namespace ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Generate ApplicationSet with pipeline name and namespace ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Deleting Stage with invalid cluster configuration ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Add access to namespace secrets to get external cluster access ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Stage creation failed with custom namespace ([#15](https://github.com/epam/edp-cd-pipeline-operator/issues/15))
- Namespace is not cleaned for the external cluster ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Use edp-config configmap for docker registry url ([#11](https://github.com/epam/edp-cd-pipeline-operator/issues/11))
- Skip multi-tenancy engines for external cluster ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))

### Code Refactoring

- Align default TriggerTemplate name ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Remove deprecated edpName parameter ([#5](https://github.com/epam/edp-cd-pipeline-operator/issues/5))
- Move tenancyEngine flag out of global section ([#9](https://github.com/epam/edp-cd-pipeline-operator/issues/9))

### Testing

- Update capsule to 0.7.2 in e2e tests ([#88](https://github.com/epam/edp-cd-pipeline-operator/issues/88))
- Ensure Argo CD ApplicationSet has expected values ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Update caspule version to the latest stable ([#28](https://github.com/epam/edp-cd-pipeline-operator/issues/28))
- Update caspule version to the latest stable ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))
- Add e2e for the custom namespace feature ([#15](https://github.com/epam/edp-cd-pipeline-operator/issues/15))
- Run e2e tests on Github PR/Merge ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))
- Add e2e tests. Start with capsule tenancy feature ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))

### Routine

- Update Pull Request Template ([#82](https://github.com/epam/edp-cd-pipeline-operator/issues/82))
- Update current development version ([#80](https://github.com/epam/edp-cd-pipeline-operator/issues/80))
- Update Kubernetes version ([#66](https://github.com/epam/edp-cd-pipeline-operator/issues/66))
- Update KubeRocketCI names and documentation links ([#69](https://github.com/epam/edp-cd-pipeline-operator/issues/69))
- Update current development version ([#58](https://github.com/epam/edp-cd-pipeline-operator/issues/58))
- Update helm-docs to the latest stable ([#44](https://github.com/epam/edp-cd-pipeline-operator/issues/44))
- Remove unused RBAC for secretManager own parameter ([#52](https://github.com/epam/edp-cd-pipeline-operator/issues/52))
- Bump to Go 1.22 ([#49](https://github.com/epam/edp-cd-pipeline-operator/issues/49))
- Use Go cache for helm-docs installation ([#44](https://github.com/epam/edp-cd-pipeline-operator/issues/44))
- Update kuttle to version 0.16 ([#44](https://github.com/epam/edp-cd-pipeline-operator/issues/44))
- Add codeowners file to the repo ([#40](https://github.com/epam/edp-cd-pipeline-operator/issues/40))
- Migrate from gerrit to github pipelines ([#35](https://github.com/epam/edp-cd-pipeline-operator/issues/35))
- Bump google.golang.org/protobuf from 1.31.0 to 1.33.0 ([#30](https://github.com/epam/edp-cd-pipeline-operator/issues/30))
- Update current development version ([#29](https://github.com/epam/edp-cd-pipeline-operator/issues/29))
- Add link to guide for managing namespace ([#162](https://github.com/epam/edp-cd-pipeline-operator/issues/162))
- Bump argo cd dependency ([#25](https://github.com/epam/edp-cd-pipeline-operator/issues/25))
- Bump github.com/argoproj/argo-cd/v2 ([#24](https://github.com/epam/edp-cd-pipeline-operator/issues/24))
- Bump github.com/cloudflare/circl from 1.3.3 to 1.3.7 ([#21](https://github.com/epam/edp-cd-pipeline-operator/issues/21))
- Bump github.com/go-git/go-git/v5 from 5.8.1 to 5.11.0 ([#21](https://github.com/epam/edp-cd-pipeline-operator/issues/21))
- Remove deprecated jobProvisioning field from Stage ([#20](https://github.com/epam/edp-cd-pipeline-operator/issues/20))
- Update current development version ([#19](https://github.com/epam/edp-cd-pipeline-operator/issues/19))
- Update release flow GH Action ([#17](https://github.com/epam/edp-cd-pipeline-operator/issues/17))
- Update GH Actions version of the steps ([#17](https://github.com/epam/edp-cd-pipeline-operator/issues/17))
- Update current development version ([#16](https://github.com/epam/edp-cd-pipeline-operator/issues/16))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#12](https://github.com/epam/edp-cd-pipeline-operator/issues/12))
- Remove jenkins admin-console perf operator logic ([#8](https://github.com/epam/edp-cd-pipeline-operator/issues/8))
- Upgrade Go to 1.20 ([#7](https://github.com/epam/edp-cd-pipeline-operator/issues/7))
- Update current development version ([#6](https://github.com/epam/edp-cd-pipeline-operator/issues/6))
- Update current development version ([#3](https://github.com/epam/edp-cd-pipeline-operator/issues/3))

### Documentation

- Update changelog file for release notes ([#73](https://github.com/epam/edp-cd-pipeline-operator/issues/73))
- Update CHANGELOG md ([#73](https://github.com/epam/edp-cd-pipeline-operator/issues/73))
- Add description for secretManager parameter ([#27](https://github.com/epam/edp-cd-pipeline-operator/issues/27))
- Add a link to the ESO configuration in the values.yaml file ([#26](https://github.com/epam/edp-cd-pipeline-operator/issues/26))
- Update README md file ([#132](https://github.com/epam/edp-cd-pipeline-operator/issues/132))

### BREAKING CHANGE:


Helm parameter kioskEnabled was removed. Use instead --set global.tenancyEngine=kiosk.


[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.22.0...HEAD
