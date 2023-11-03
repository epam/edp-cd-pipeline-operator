<a name="unreleased"></a>
## [Unreleased]


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

[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.17.0...HEAD
[v2.17.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.1...v2.17.0
[v2.16.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.0...v2.16.1
[v2.16.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.15.0...v2.16.0
[v2.15.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.14.1...v2.15.0
