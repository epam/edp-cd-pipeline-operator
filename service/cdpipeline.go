package service

import (
	"bytes"
	edpv1alpha1 "cd-pipeline-handler-controller/pkg/apis/edp/v1alpha1"
	ClientSet "cd-pipeline-handler-controller/pkg/openshift"
	"errors"
	"fmt"
	"text/template"
	"github.com/bndr/gojenkins"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
	"net/http"
	"time"
)

type Jenkins struct {
	client   gojenkins.Jenkins
	Url      string
	Username string
	Token    string
}

const (
	StatusInit       = "initialized"
	StatusFailed     = "failed"
	StatusFinished   = "created"
	StatusInProgress = "in progress"
)

func CreateCDPipeline(cr *edpv1alpha1.CDPipeline) error {
	if cr.Status.Status != StatusInit {
		log.Printf("CD Pipeline %v is not in init status. Skipped", cr.Spec.Name)
		return errors.New(fmt.Sprintf("CD Pipeline %v is not in init status. Skipped", cr.Spec.Name))
	}

	setStatusFields(cr, StatusInProgress, time.Now())

	clientSet := ClientSet.CreateOpenshiftClients()
	jenkinsUrl := fmt.Sprintf("http://jenkins.%s:8080", cr.Namespace)
	jenkinsToken, jenkinsUsername, err := getJenkinsCreds(clientSet, cr.Namespace)
	if err != nil {
		rollback(cr)
		return err
	}

	jenkins, err := initJenkins(jenkinsUrl, jenkinsUsername, jenkinsToken)
	if err != nil {
		rollback(cr)
		return err
	}

	folder, err := createFolder(*jenkins, cr.Name)
	if err != nil {
		rollback(cr)
		return err
	}

	err = createCDPipeline(*jenkins, cr.Name, folder.GetName())
	if err != nil {
		rollback(cr)
		return err
	}

	setStatusFields(cr, StatusFinished, time.Now())
	log.Printf("CD pipeline has been created. Status: %v", StatusFinished)
	return nil
}

func initJenkins(url string, username string, token string) (*Jenkins, error) {
	log.Printf("Start initializing client for Jenkins: %v", url)

	jenkins := gojenkins.CreateJenkins(&http.Client{}, url, username, token)

	_, err := jenkins.Init()
	if err != nil {
		return nil, err
	}

	log.Printf("Client for Jenkins %v has been initialized", url)

	return &Jenkins{
		client:   *jenkins,
		Url:      url,
		Username: username,
		Token:    token,
	}, nil
}

func createFolder(jenkins Jenkins, name string) (*gojenkins.Folder, error) {
	log.Printf("Start creating folder %v in Jenkins for CD pipeline", name)

	folder, err := jenkins.client.CreateFolder(name)
	if err != nil {
		return nil, err
	}

	log.Printf("Folder %v in Jenkins for CD pipeline has been created", name)
	return folder, nil
}

func createCDPipeline(jenkins Jenkins, name string, folder string) error {
	log.Printf("Start creating CD pipeline %v in jenkins", name)
	jobFullName := fmt.Sprintf("%v/job/%v", folder, name)

	job, err := getJenkinsJob(jenkins, jobFullName)
	if err != nil {
		return err
	}

	cdPipelineConfig, err := createCDPipelineConfig(name)
	if err != nil {
		return err
	}

	if job != nil {
		log.Printf("CD pipeline with name %v exist in jenkins. Creation skipped.", name)
	} else {
		_, err = jenkins.client.CreateJobInFolder(*cdPipelineConfig, name, folder)
		if err != nil {
			return err
		}

		log.Printf("CD pipeline %v has been created", name)
	}

	return nil
}

func getJenkinsJob(jenkins Jenkins, name string) (*gojenkins.Job, error) {
	log.Printf("Start getting CD Pipeline %v in Jenkins", name)
	job, err := jenkins.client.GetJob(name)
	if err != nil {
		return nil, checkErrForNotFound(err)
	}

	log.Printf("CD Pipeline %v in Jenkins has been recieved", name)

	return job, nil
}

func getJenkinsCreds(clientSet *ClientSet.ClientSet, namespace string) (string, string, error) {
	log.Printf("Start recieving credentials for Jenkins in namespace %v", namespace)
	jenkinsTokenSecret, err := clientSet.CoreClient.Secrets(namespace).Get("jenkins-token", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprint(err)
		log.Println(errorMsg)
		return "", "", errors.New(errorMsg)
	}

	log.Printf("Credentials for Jenkins in namespace %v has been recieved", namespace)

	return string(jenkinsTokenSecret.Data["token"]), string(jenkinsTokenSecret.Data["username"]), nil
}

func rollback(cr *edpv1alpha1.CDPipeline) {
	setStatusFields(cr, StatusFailed, time.Now())
}

func setStatusFields(cr *edpv1alpha1.CDPipeline, status string, time time.Time) {
	cr.Status.Status = status
	cr.Status.LastTimeUpdated = time
	log.Printf("Status for CD pipeline %v has been updated to '%v' at %v.", cr.Spec.Name, status, time)
}

func checkErrForNotFound(err error) error {
	if err.Error() == "404" {
		log.Printf("CD pipeline in Jenkins hasn't been found")
		return nil
	}
	return err
}

func createCDPipelineConfig(name string) (*string, error) {
	var cdPipelineBuffer bytes.Buffer

	jenkinsName := map[string]interface{}{
		"name": name,
	}

	tmpl, err := template.New("cd-pipeline.tmpl").ParseFiles("/usr/local/bin/pipelines/cd-pipeline.tmpl")
	if err != nil {
		return nil, err
	}

	if err := tmpl.Execute(&cdPipelineBuffer, jenkinsName); err != nil {
		return nil, err
	}

	cdPipeline := cdPipelineBuffer.String()

	return &cdPipeline, nil
}