package helper

import (
	"context"
	"fmt"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/platform"
	jenkinsApi "github.com/epmd-edp/jenkins-operator/v2/pkg/apis/v2/v1alpha1"
	jenkinsOperatorSpec "github.com/epmd-edp/jenkins-operator/v2/pkg/service/jenkins/spec"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetJenkinsCreds(platform platform.PlatformService, k8sClient client.Client, namespace string) (string, string, error) {
	options := client.ListOptions{Namespace: namespace}
	jl := &jenkinsApi.JenkinsList{}

	err := k8sClient.List(context.TODO(), &options, jl)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Jenkins installation is not found in namespace %v", namespace)
		}
		return "", "", errors.Wrapf(err, "Unable to get Jenkins CRs in namespace %v", namespace)
	}

	if len(jl.Items) == 0 {
		errors.Wrapf(err, "Jenkins installation is not found in namespace %v", namespace)
	}

	jenkins := &jl.Items[0]
	annotationKey := fmt.Sprintf("%v/%v", jenkinsOperatorSpec.EdpAnnotationsPrefix, jenkinsOperatorSpec.JenkinsTokenAnnotationSuffix)
	jenkinsTokenSecretName := jenkins.Annotations[annotationKey]
	jenkinsTokenSecret, err := platform.GetSecretData(namespace, jenkinsTokenSecretName)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			return "", "", errors.Wrapf(err, "Secret %v in not found", jenkinsTokenSecretName)
		}
		return "", "", errors.Wrapf(err, "Getting secret %v failed", jenkinsTokenSecretName)
	}
	return string(jenkinsTokenSecret["password"]), string(jenkinsTokenSecret["username"]), nil
}
