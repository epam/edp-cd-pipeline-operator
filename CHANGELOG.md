<a name="unreleased"></a>
## [Unreleased]

### Features

- Update Makefile changelog target [EPMDEDP-8218](https://jiraeu.epam.com/browse/EPMDEDP-8218)
- Generate CRDs and helm docs automatically [EPMDEDP-8385](https://jiraeu.epam.com/browse/EPMDEDP-8385)
- Add aplication field to CDPipeline CRD [EPMDEDP-8929](https://jiraeu.epam.com/browse/EPMDEDP-8929)

### Bug Fixes

- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix changelog generation in GH Release Action [EPMDEDP-8468](https://jiraeu.epam.com/browse/EPMDEDP-8468)
- Correct image version [EPMDEDP-8471](https://jiraeu.epam.com/browse/EPMDEDP-8471)

### Code Refactoring

- Remove deprecated parameter [EPMDEDP-8168](https://jiraeu.epam.com/browse/EPMDEDP-8168)
- Switch from Virtual resources to CRD one [EPMDEDP-8287](https://jiraeu.epam.com/browse/EPMDEDP-8287)

### Testing

- Add tests [EPMDEDP-7993](https://jiraeu.epam.com/browse/EPMDEDP-7993)

### Routine

- Update release CI pipelines [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Align codecov generation flow [EPMDEDP-7993](https://jiraeu.epam.com/browse/EPMDEDP-7993)
- Populate chart with Artifacthub annotations [EPMDEDP-8049](https://jiraeu.epam.com/browse/EPMDEDP-8049)
- Update changelog [EPMDEDP-8227](https://jiraeu.epam.com/browse/EPMDEDP-8227)
- Update base docker image to alpine 3.15.4 [EPMDEDP-8853](https://jiraeu.epam.com/browse/EPMDEDP-8853)
- Update changelog [EPMDEDP-9185](https://jiraeu.epam.com/browse/EPMDEDP-9185)

### Documentation

- Align diagram to the current state [EPMDEDP-7970](https://jiraeu.epam.com/browse/EPMDEDP-7970)
- Updates architecture diagram [EPMDEDP-8255](https://jiraeu.epam.com/browse/EPMDEDP-8255)
- Update documentation section [EPMDEDP-8255](https://jiraeu.epam.com/browse/EPMDEDP-8255)


<a name="v2.10.0"></a>
## [v2.10.0] - 2021-12-06
### Features

- Provide operator's build information [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)

### Bug Fixes

- Fix panic issue with non-existing cbis [EPMDEDP-7470](https://jiraeu.epam.com/browse/EPMDEDP-7470)
- Changelog links [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Fix links in changelog [EPMDEDP-7847](https://jiraeu.epam.com/browse/EPMDEDP-7847)
- Extend cd-operator permissions on EKS [EPMDEDP-7935](https://jiraeu.epam.com/browse/EPMDEDP-7935)
- Extend cd-operator permissions on EKS [EPMDEDP-7935](https://jiraeu.epam.com/browse/EPMDEDP-7935)
- Extend cd-operator permissions on EKS [EPMDEDP-7935](https://jiraeu.epam.com/browse/EPMDEDP-7935)

### Code Refactoring

- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Provide unique name of cluster RBAC resources [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Align RBAC according to kiosk usage [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace namespaces role to cluster for OKD [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace namespaces role to cluster for OKD [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Expand cd-pipeline-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace cluster-wide role/rolebinding to namespaced [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Remake condition for simplicity [EPMDEDP-7945](https://jiraeu.epam.com/browse/EPMDEDP-7945)

### Testing

- Increase coverage for put_codebase_image_stream [EPMDEDP-7470](https://jiraeu.epam.com/browse/EPMDEDP-7470)

### Routine

- Fix link in changelog config [EPMDEDP-7874](https://jiraeu.epam.com/browse/EPMDEDP-7874)
- Add changelog generator [EPMDEDP-7874](https://jiraeu.epam.com/browse/EPMDEDP-7874)
- Add codecov report [EPMDEDP-7885](https://jiraeu.epam.com/browse/EPMDEDP-7885)
- Update docker image [EPMDEDP-7895](https://jiraeu.epam.com/browse/EPMDEDP-7895)
- Update go.sum and go.mod. [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update codebase-operator to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update jenkins-operator to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update codebase-operator to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Use custom go build step for operator [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)
- Update go to version 1.17 [EPMDEDP-7932](https://jiraeu.epam.com/browse/EPMDEDP-7932)

### Documentation

- Update the links on GitHub [EPMDEDP-7781](https://jiraeu.epam.com/browse/EPMDEDP-7781)


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

[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.10.0...HEAD
[v2.10.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.9.0...v2.10.0
[v2.9.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.2...v2.9.0
[v2.8.2]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.1...v2.8.2
[v2.8.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.8.0...v2.8.1
[v2.8.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.1...v2.8.0
[v2.7.1]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.7.0...v2.7.1
[v2.7.0]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.3.0-58...v2.7.0
