package openshift

import (
	"crypto/rand"
	"errors"
	"fmt"
	projectV1 "github.com/openshift/api/project/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"math/big"
)

func CreateProject(clientSet *ClientSet, projectName string, projectDescription string) error {
	_, err := clientSet.ProjectClient.ProjectRequests().Create(
		&projectV1.ProjectRequest{
			ObjectMeta: metav1.ObjectMeta{
				Name: projectName,
			},
			Description: projectDescription,
		},
	)

	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}

	log.Printf("Project %v has been created in openshift", projectName)
	return nil
}

func CreateRoleBinding(clientSet *ClientSet, edpName string, namespace string, roleRef rbacV1.RoleRef, subjects []rbacV1.Subject) error {
	randPostfix, err := rand.Int(rand.Reader, big.NewInt(10000))
	_, err = clientSet.RbacClient.RoleBindings(namespace).Create(
		&rbacV1.RoleBinding{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s-%d", edpName, roleRef.Name, randPostfix),
			},
			RoleRef:  roleRef,
			Subjects: subjects,
		},
	)

	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return errors.New(errorMsg)
	}

	log.Printf("Role %v has been assigned for user(s) %v", roleRef.Name, subjects)
	return nil
}
