# How to Install Operator

EDP installation can be applied on two container orchestration platforms: OpenShift and Kubernetes.

_**NOTE:** Installation of operators is platform-independent, that is why there is a unified instruction for deploying._

### Prerequisites
1. Linux machine or Windows Subsystem for Linux instance with [Helm 3](https://helm.sh/docs/intro/install/) installed;
2. Cluster admin access to the cluster;
3. EDP project/namespace is deployed by following one of the instructions: [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/master/documentation/openshift_install_edp.md#edp-project) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/master/documentation/kubernetes_install_edp.md#edp-namespace).

### Installation
In order to install the CD Pipeline operator, follow the steps below:

1. To add the Helm EPAMEDP Charts for local client, run "helm repo add":
     ```bash
     helm repo add epamedp https://chartmuseum.demo.edp-epam.com/
     ```
2. Choose available Helm chart version:
     ```bash
     helm search repo epamedp/cd-pipeline-operator
     NAME                              CHART VERSION   APP VERSION     DESCRIPTION
     epamedp/cd-pipeline-operator      v2.4.0                          Helm chart for Golang application/service deplo...
     ```

Parameters:
 ```
    - chart_version                                 # a version of CD Pipeline operator Helm chart;
    - global.edpName                                # a namespace or a project name (in case of OpenShift);
    - global.platform                               # openShift or kubernetes;
    - image.name                                    # EDP image. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/cd-pipeline-operator);
    - image.version                                 # EDP tag. The released image can be found on [Dockerhub](https://hub.docker.com/repository/docker/epamedp/cd-pipeline-operator/tags);
 ```

_**NOTE:** Follow instruction to create namespace [edp-install-openshift](https://github.com/epmd-edp/edp-install/blob/master/documentation/openshift_install_edp.md#install-edp) or [edp-install-kubernetes](https://github.com/epmd-edp/edp-install/blob/master/documentation/kubernetes_install_edp.md#install-edp)._

Inspect the sample of launching a Helm template for CD Pipeline operator installation:
```bash
helm install cd-pipeline-operator epamedp/cd-pipeline-operator --version <chart_version> --namespace <edp_cicd_project> --set name=cd-pipeline-operator --set global.edpName=<edp_cicd_project> --set global.platform=<platform_type> deploy-templates
```

* Check the <edp_cicd_project> namespace that should contain Deployment with your operator in a running status

### Local Development
In order to develop the operator, first set up a local environment. For details, please refer to the [Local Development](documentation/local-development.md) page.
