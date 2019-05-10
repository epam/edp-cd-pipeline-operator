package openshift

import (
	projectV1Client "github.com/openshift/client-go/project/clientset/versioned/typed/project/v1"
	coreV1Client "k8s.io/client-go/kubernetes/typed/core/v1"
	rbacV1Client "k8s.io/client-go/kubernetes/typed/rbac/v1"
	"k8s.io/client-go/tools/clientcmd"
	"log"
)

type ClientSet struct {
	CoreClient    *coreV1Client.CoreV1Client
	ProjectClient *projectV1Client.ProjectV1Client
	RbacClient    *rbacV1Client.RbacV1Client
}

func CreateOpenshiftClients() *ClientSet {
	config := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
		clientcmd.NewDefaultClientConfigLoadingRules(),
		&clientcmd.ConfigOverrides{},
	)
	restConfig, err := config.ClientConfig()
	if err != nil {
		log.Fatal(err)
	}
	coreClient, err := coreV1Client.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	projectClient, err := projectV1Client.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	rbacClient, err := rbacV1Client.NewForConfig(restConfig)
	if err != nil {
		log.Fatal(err)
	}
	log.Print("Openshift clients was successfully created")
	return &ClientSet{
		CoreClient:    coreClient,
		ProjectClient: projectClient,
		RbacClient:    rbacClient,
	}
}