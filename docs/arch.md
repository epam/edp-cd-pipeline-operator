# Architecture Scheme of CD Pipeline Operator

This page contains a representation of the current CD Pipeline Operator architecture that is built using the plantUML capabilities.
All the diagrams sources are placed under the **/puml** directory of the current folder.

An Image of the HEAD of the current branch will be displayed as a result of an Image building with the plant uml proxy server.

If you are in the detached mode, use the sources to get the required version of diagrams.

![arch](https://www.plantuml.com/plantuml/proxy?src=https://raw.githubusercontent.com/epam/edp-cd-pipeline-operator/master/docs/puml/arch.puml)

## Autodeploy

The **cd-pipeline-operator** takes the primary role in processing the autodeploy functionality since it operates with stages and their parameters.
The **cd-pipeline-operator** checks the parameters of stages and triggers other resources that handle the feature logic.
When deploying several applications within a single CD pipeline, applications are managed individually, which means that each application is deployed separately.

### Autodeploy in Argo CD

The scheme below illustrates how autodeploy works in the Tekton deploy scenario:

![Autodeploy in Tekton deploy scenario](https://github.com/epam/edp-cd-pipeline-operator/blob/master/docs/puml/autodeploy_argo_cd.png)

Under the hood, the autodeploy logic is implemented in the following way:

1. User clicks the **Build** button or merges patch to VCS.
2. As a result of the build pipeline, a new version of the artifact is available for the application.
3. The **codebase-operator** detects the new tag and creates the **CDStageDeploy** with this tag.
4. The **codebase-operator** retrieves the new tag from the **CDStageDeploy** resource and updates the image tag in the Argo CD application.
5. Lastly, Argo CD deploys the newer image.

**Note:**  In Tekton deploy scenario, autodeploy will start working only after the first manual deploy.
