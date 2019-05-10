package jenkins

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"log"
	"net/http"
)

type Jenkins struct {
	client   gojenkins.Jenkins
	url      string
	username string
	token    string
}

func Init(url string, username string, token string) (*Jenkins, error) {
	log.Printf("Start initializing client for Jenkins: %v", url)

	jenkins := gojenkins.CreateJenkins(&http.Client{}, url, username, token)

	_, err := jenkins.Init()
	if err != nil {
		return nil, err
	}

	log.Printf("Client for Jenkins %v has been initialized", url)

	return &Jenkins{
		client:   *jenkins,
		url:      url,
		username: username,
		token:    token,
	}, nil
}

func (jenkins Jenkins) CreateFolder(name string) (*gojenkins.Folder, error) {
	log.Printf("Start creating folder %v in jenkins for job", name)

	folder, err := jenkins.client.CreateFolder(name)
	if err != nil {
		return nil, err
	}

	log.Printf("Folder %v in Jenkins for job has been created", name)
	return folder, nil
}

func (jenkins Jenkins) CreateJob(name string, folder string, jobConfig string) error {
	log.Printf("Start creating job %v in jenkins", name)
	jobFullName := fmt.Sprintf("%v/job/%v", folder, name)

	job, err := jenkins.getJob(jobFullName)
	if err != nil {
		return err
	}

	if job != nil {
		log.Printf("Job with name %v exist in jenkins. Creation skipped.", name)
	} else {
		_, err = jenkins.client.CreateJobInFolder(jobConfig, name, folder)
		if err != nil {
			return err
		}

		log.Printf("Job %v has been created", name)
	}

	return nil
}

func (jenkins Jenkins) getJob(name string) (*gojenkins.Job, error) {
	log.Printf("Start getting job %v in Jenkins", name)
	job, err := jenkins.client.GetJob(name)
	if err != nil {
		return nil, checkErrForNotFound(err)
	}

	log.Printf("Job %v in Jenkins has been recieved", name)

	return job, nil
}
func checkErrForNotFound(err error) error {
	if err.Error() == "404" {
		log.Printf("Job in Jenkins hasn't been found")
		return nil
	}
	return err
}