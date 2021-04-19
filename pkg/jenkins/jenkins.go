package jenkins

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
)

var log = ctrl.Log.WithName("jenkins")

type Jenkins struct {
	client   gojenkins.Jenkins
	url      string
	username string
	token    string
}

func Init(url string, username string, token string) (*Jenkins, error) {
	reqLog := log.WithValues("url", url, "username", username)
	reqLog.Info("Start initializing client for Jenkins")

	jenkins := gojenkins.CreateJenkins(&http.Client{}, url, username, token)
	_, err := jenkins.Init()
	if err != nil {
		return nil, err
	}
	reqLog.Info("Client for Jenkins has been initialized")
	return &Jenkins{
		client:   *jenkins,
		url:      url,
		username: username,
		token:    token,
	}, nil
}

func (jenkins Jenkins) CreateFolder(name string) (*gojenkins.Folder, error) {
	reqLog := log.WithValues("folder name", name)
	reqLog.Info("Start creating folder in jenkins for job")

	folder, err := jenkins.client.CreateFolder(name)
	if err != nil {
		return nil, err
	}
	reqLog.Info("Folder in Jenkins for job has been created")
	return folder, nil
}

func (jenkins Jenkins) CreateJob(name string, folder string, jobConfig string) error {
	reqLog := log.WithValues("job name", name, "folder name", folder)
	reqLog.Info("Start creating job in Jenkins")

	jobFullName := fmt.Sprintf("%v/job/%v", folder, name)
	job, err := jenkins.getJob(jobFullName)
	if err != nil {
		return err
	}
	if job != nil {
		reqLog.Info("Job with name exists in Jenkins. Creation skipped")
	} else {
		reqLog.Info("Job has not been found. Start creation")
		_, err = jenkins.client.CreateJobInFolder(jobConfig, name, folder)
		if err != nil {
			return err
		}
		reqLog.Info("Job has been created")
	}
	return nil
}

func (jenkins Jenkins) getJob(name string) (*gojenkins.Job, error) {
	reqLog := log.WithValues("job name", name)
	reqLog.Info("Start getting job in Jenkins")

	job, err := jenkins.client.GetJob(name)
	if err != nil {
		return nil, checkErrForNotFound(err)
	}
	reqLog.Info("Job in Jenkins has been received")
	return job, nil
}

func checkErrForNotFound(err error) error {
	if err.Error() == "404" {
		return nil
	}
	return err
}
