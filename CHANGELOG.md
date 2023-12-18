<a name="unreleased"></a>
## [Unreleased]


<a name="v2.18.0"></a>
## v2.18.0 - 2023-11-25
### Features

- Enable Capsule Tenant modification from values.yaml ([#13](https://github.com/epam/edp-cd-pipeline-operator/issues/13))
- Add multi-cluster support for the operator ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Create capsule tenant resource ([#4](https://github.com/epam/edp-cd-pipeline-operator/issues/4))
- Add capsule support for multi-tenancy ([#9](https://github.com/epam/edp-cd-pipeline-operator/issues/9))

### Bug Fixes

- Deleting Stage with invalid cluster configuration ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Add access to namespace secrets to get external cluster access ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Stage creation failed with custom namespace ([#15](https://github.com/epam/edp-cd-pipeline-operator/issues/15))
- Namespace is not cleaned for the external cluster ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))
- Use edp-config configmap for docker registry url ([#11](https://github.com/epam/edp-cd-pipeline-operator/issues/11))
- Skip multi-tenancy engines for external cluster ([#10](https://github.com/epam/edp-cd-pipeline-operator/issues/10))

### Code Refactoring

- Remove deprecated edpName parameter ([#5](https://github.com/epam/edp-cd-pipeline-operator/issues/5))
- Move tenancyEngine flag out of global section ([#9](https://github.com/epam/edp-cd-pipeline-operator/issues/9))

### Testing

- Update caspule version to the latest stable ([#28](https://github.com/epam/edp-cd-pipeline-operator/issues/28))
- Update caspule version to the latest stable ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))
- Add e2e for the custom namespace feature ([#15](https://github.com/epam/edp-cd-pipeline-operator/issues/15))
- Run e2e tests on Github PR/Merge ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))
- Add e2e tests. Start with capsule tenancy feature ([#14](https://github.com/epam/edp-cd-pipeline-operator/issues/14))

### Routine

- Update release flow GH Action ([#17](https://github.com/epam/edp-cd-pipeline-operator/issues/17))
- Update GH Actions version of the steps ([#17](https://github.com/epam/edp-cd-pipeline-operator/issues/17))
- Update current development version ([#16](https://github.com/epam/edp-cd-pipeline-operator/issues/16))
- Bump golang.org/x/net from 0.8.0 to 0.17.0 ([#12](https://github.com/epam/edp-cd-pipeline-operator/issues/12))
- Remove jenkins admin-console perf operator logic ([#8](https://github.com/epam/edp-cd-pipeline-operator/issues/8))
- Upgrade Go to 1.20 ([#7](https://github.com/epam/edp-cd-pipeline-operator/issues/7))
- Update current development version ([#6](https://github.com/epam/edp-cd-pipeline-operator/issues/6))
- Update current development version ([#3](https://github.com/epam/edp-cd-pipeline-operator/issues/3))

### BREAKING CHANGE:


Helm parameter kioskEnabled was removed. Use instead --set global.tenancyEngine=kiosk.


[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.18.0...HEAD
