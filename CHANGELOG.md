<a name="unreleased"></a>
## [Unreleased]

### Features

- Updated EDP components [EPMDEDP-11206](https://jiraeu.epam.com/browse/EPMDEDP-11206)
- Updated loft-sh/kiosk dependency [EPMDEDP-11274](https://jiraeu.epam.com/browse/EPMDEDP-11274)
- Removed loft-sh/kiosk lib direct dependency [EPMDEDP-11286](https://jiraeu.epam.com/browse/EPMDEDP-11286)
- Updated Operator SDK version [EPMDEDP-11363](https://jiraeu.epam.com/browse/EPMDEDP-11363)
- Replace Admin with self-provisioner Cluster Role [EPMDEDP-11426](https://jiraeu.epam.com/browse/EPMDEDP-11426)
- Create project for openshift platform [EPMDEDP-11441](https://jiraeu.epam.com/browse/EPMDEDP-11441)
- Remove admin-console-view RBAC [EPMDEDP-11486](https://jiraeu.epam.com/browse/EPMDEDP-11486)
- Remove Kiosk integration when using Openshift cluster [EPMDEDP-11489](https://jiraeu.epam.com/browse/EPMDEDP-11489)

### Bug Fixes

- Use ProjectRequest to create openshift Project [EPMDEDP-11441](https://jiraeu.epam.com/browse/EPMDEDP-11441)
- Create/delete openshift project without checking its existence [EPMDEDP-11441](https://jiraeu.epam.com/browse/EPMDEDP-11441)

### Code Refactoring

- Apply golangci-lint [EPMDEDP-10626](https://jiraeu.epam.com/browse/EPMDEDP-10626)
- Removed old api [EPMDEDP-11206](https://jiraeu.epam.com/browse/EPMDEDP-11206)

### Routine

- Update current development version [EPMDEDP-10610](https://jiraeu.epam.com/browse/EPMDEDP-10610)
- Change type of kioskEnabled parameter from string to bool [EPMDEDP-11426](https://jiraeu.epam.com/browse/EPMDEDP-11426)

### Documentation

- Update chart and application version in Readme file [EPMDEDP-11221](https://jiraeu.epam.com/browse/EPMDEDP-11221)
- Update diagram [EPMDEDP-11367](https://jiraeu.epam.com/browse/EPMDEDP-11367)


<a name="v2.13.0"></a>
## [v2.13.0] - 2022-12-06
### Features

- Added a stub linter [EPMDEDP-10536](https://jiraeu.epam.com/browse/EPMDEDP-10536)
- Align operator logic to work with tekton [EPMDEDP-11052](https://jiraeu.epam.com/browse/EPMDEDP-11052)

### Bug Fixes

- Provide namespace permissions for cd-pipeline operator [EPMDEDP-10661](https://jiraeu.epam.com/browse/EPMDEDP-10661)
- Reconcile stages when a new application was added to cdpipeline [EPMDEDP-11055](https://jiraeu.epam.com/browse/EPMDEDP-11055)
- Add access to jenkins resource [EPMDEDP-11093](https://jiraeu.epam.com/browse/EPMDEDP-11093)

### Code Refactoring

- Align ImageName in verified CBIS [EPMDEDP-11081](https://jiraeu.epam.com/browse/EPMDEDP-11081)

### Routine

- Update current development version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update current development version [EPMDEDP-10805](https://jiraeu.epam.com/browse/EPMDEDP-10805)


<a name="v2.12.1"></a>
## [v2.12.1] - 2022-10-28
### Bug Fixes

- Provide namespace permissions for cd-pipeline operator [EPMDEDP-10661](https://jiraeu.epam.com/browse/EPMDEDP-10661)


<a name="v2.12.0"></a>
## [v2.12.0] - 2022-08-26
### Features

- Download required tools for Makefile targets [EPMDEDP-10105](https://jiraeu.epam.com/browse/EPMDEDP-10105)
- Add CDPipeline label for Stage [EPMDEDP-10256](https://jiraeu.epam.com/browse/EPMDEDP-10256)
- Switch to use V1 version of CD Pipeline and Stage APIs [EPMDEDP-9214](https://jiraeu.epam.com/browse/EPMDEDP-9214)
- Switch to V1 of edp-component-operator CRDs [EPMDEDP-9747](https://jiraeu.epam.com/browse/EPMDEDP-9747)

### Bug Fixes

- Use separate client, which doesn't restrict namespaces [EPMDEDP-10037](https://jiraeu.epam.com/browse/EPMDEDP-10037)
- Make sure that during update of Stage CR, status field is ignored [EPMDEDP-10037](https://jiraeu.epam.com/browse/EPMDEDP-10037)
- Fix typo in openshift rolebinding [EPMDEDP-10055](https://jiraeu.epam.com/browse/EPMDEDP-10055)
- Incorrect subsequent CDPipeline Stage creation in Headlamp [EPMDEDP-10327](https://jiraeu.epam.com/browse/EPMDEDP-10327)
- Update SonarQube ignore list [EPMDEDP-9214](https://jiraeu.epam.com/browse/EPMDEDP-9214)

### Code Refactoring

- Switch internal APIs to V1 [EPMDEDP-10117](https://jiraeu.epam.com/browse/EPMDEDP-10117)
- Remove ClusterRole for admin console [EPMDEDP-10228](https://jiraeu.epam.com/browse/EPMDEDP-10228)
- Use repository and tag for image reference in chart [EPMDEDP-10389](https://jiraeu.epam.com/browse/EPMDEDP-10389)

### Routine

- Refactor RBAC [EPMDEDP-10055](https://jiraeu.epam.com/browse/EPMDEDP-10055)
- Upgrade go version to 1.18 [EPMDEDP-10110](https://jiraeu.epam.com/browse/EPMDEDP-10110)
- Fix Jira Ticket pattern for changelog generator [EPMDEDP-10159](https://jiraeu.epam.com/browse/EPMDEDP-10159)
- Update alpine base image to 3.16.2 version [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)
- Update alpine base image version [EPMDEDP-10280](https://jiraeu.epam.com/browse/EPMDEDP-10280)
- Change 'go get' to 'go install' for git-chglog [EPMDEDP-10337](https://jiraeu.epam.com/browse/EPMDEDP-10337)
- Remove VERSION file [EPMDEDP-10387](https://jiraeu.epam.com/browse/EPMDEDP-10387)
- Add gcflags for go build artifact [EPMDEDP-10411](https://jiraeu.epam.com/browse/EPMDEDP-10411)
- Update current development version [EPMDEDP-8832](https://jiraeu.epam.com/browse/EPMDEDP-8832)
- Update chart annotation [EPMDEDP-9515](https://jiraeu.epam.com/browse/EPMDEDP-9515)

### Documentation

- Align README.md [EPMDEDP-10274](https://jiraeu.epam.com/browse/EPMDEDP-10274)


<a name="v2.11.0"></a>
## [v2.11.0] - 2022-05-25
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

- Provide unique name of cluster RBAC resources [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Align RBAC according to kiosk usage [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace namespaces role to cluster for OKD [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Replace namespaces role to cluster for OKD [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Expand cd-pipeline-operator role [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
- Add namespace field in roleRef in OKD RB, align CRB name [EPMDEDP-7279](https://jiraeu.epam.com/browse/EPMDEDP-7279)
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
- Update codebase-operator to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
- Update jenkins-operator to the latest stable [EPMDEDP-7930](https://jiraeu.epam.com/browse/EPMDEDP-7930)
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

[Unreleased]: https://github.com/epam/edp-cd-pipeline-operator/compare/v2.13.0...HEAD
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
