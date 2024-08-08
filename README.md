[![codecov](https://codecov.io/gh/epam/edp-cd-pipeline-operator/branch/master/graph/badge.svg?token=T3XGW910VD)](https://codecov.io/gh/epam/edp-cd-pipeline-operator)

# CD Pipeline Operator

| :heavy_exclamation_mark: Please refer to [KubeRocketCI documentation](https://docs.kuberocketci.io/) to get the notion of the main concepts and guidelines. |
| --- |

Get acquainted with the CD Pipeline Operator and the installation process as well as the local development, and architecture scheme.

## Overview

CD Pipeline Operator is a KubeRocketCI operator that is responsible for provisioning continuous delivery pipeline entities. Operator installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Operator is platform-independent, that is why there is a unified instruction for deploying._

## Prerequisites

1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. KubeRocketCI project/namespace is deployed by following the [Install KubeRocketCI](https://docs.kuberocketci.io/docs/operator-guide/install-kuberocketci) instruction.

## Installation

In order to install the CD Pipeline Operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":

     ```bash
     helm repo add epamedp https://epam.github.io/edp-helm-charts/stable
     ```

2. Choose available Helm chart version:

     ```bash
     helm search repo epamedp/cd-pipeline-operator -l
     NAME                              CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/cd-pipeline-operator      2.20.0          2.20.0          A Helm chart for KubeRocketCI CD Pipeline Operator
     ```

   _**NOTE:** It is highly recommended to use the latest released version._

3. Full chart parameters available in [deploy-templates/README.md](deploy-templates/README.md).

4. Install operator in the edp namespace with the helm command; find below the installation command example:

    ```bash
    helm install cd-pipeline-operator epamedp/cd-pipeline-operator --version <chart_version> --namespace edp --set name=cd-pipeline-operator --set global.platform=<platform_type>
    ```

5. Check the edp namespace that should contain operator deployment with your operator in a running status.

## Local Development

In order to develop the operator, first set up a local environment. For details, please refer to the [Developer Guide](https://docs.kuberocketci.io/docs/developer-guide/local-development) page.

Development versions are also available, please refer to the [snapshot Helm Chart repository](https://epam.github.io/edp-helm-charts/snapshot/) page.

### Related Articles

- [Architecture Scheme of CD Pipeline Operator](docs/arch.md)
- [Install KubeRocketCI](https://docs.kuberocketci.io/docs/operator-guide/install-kuberocketci)
