module github.com/epmd-edp/cd-pipeline-operator/v2

go 1.14

replace github.com/openshift/api => github.com/openshift/api v0.0.0-20180801171038-322a19404e37

require (
	github.com/bndr/gojenkins v0.2.1-0.20181125150310-de43c03cf849
	github.com/epam/edp-codebase-operator/v2 v2.3.0-95.0.20210224132445-f851d06a3eab
	github.com/epmd-edp/edp-component-operator v0.1.1-0.20200827122548-e87429a916e0
	github.com/epmd-edp/jenkins-operator/v2 v2.3.0-130.0.20200525102742-f56cd8641faa
	github.com/go-openapi/spec v0.19.3
	github.com/operator-framework/operator-sdk v0.0.0-20190530173525-d6f9cdf2f52e
	github.com/pkg/errors v0.8.1
	github.com/spf13/pflag v1.0.3
	k8s.io/apimachinery v0.0.0-20190221213512-86fb29eff628
	k8s.io/client-go v0.0.0-20190228174230-b40b2a5939e4
	k8s.io/kube-openapi v0.0.0-20181109181836-c59034cc13d5
	sigs.k8s.io/controller-runtime v0.1.12
)
