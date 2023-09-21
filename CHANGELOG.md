<a name="unreleased"></a>
## [Unreleased]


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

[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.16.0...HEAD
[v2.16.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.15.0...v2.16.0
[v2.15.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.14.1...v2.15.0
