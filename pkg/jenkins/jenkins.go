package jenkins

import (
	"fmt"
	"github.com/bndr/gojenkins"
	"github.com/pkg/errors"
	"net/http"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"time"
)

var log = logf.Log.WithName("jenkins")

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

func (jenkins Jenkins) TriggerJobProvision(jpn string, jpc map[string]string) error {
	log.V(2).Info("start triggering job provisioner", "provisioner name", jpn)

	if _, err := jenkins.client.BuildJob(jpn, jpc); err != nil {
		return errors.Wrapf(err, "couldn't trigger %v job", jpn)
	}
	log.Info("provision job has been triggered", "provisioner name", jpn)
	return nil
}

func (jenkins Jenkins) GetJobStatus(name string, delay time.Duration, retryCount int) (string, error) {
	time.Sleep(delay)
	for i := 0; i < retryCount; i++ {
		isQueued, err := jenkins.IsJobQueued(name)
		isRunning, err := jenkins.IsJobRunning(name)
		if err != nil {
			job, err := jenkins.getJob(name)
			if job.Raw.Color == "notbuilt" {
				log.Info("Job didn't start yet", "name", name, "delay", delay, "attempts lasts", retryCount-i)
				time.Sleep(delay)
				continue
			}

			if err != nil {
				return "", err
			}
		}
		if *isRunning || *isQueued {
			log.Info("Job is running", "name", name, "delay", delay, "attempts lasts", retryCount-i)
			time.Sleep(delay)
		} else {
			job, err := jenkins.getJob(name)
			if err != nil {
				return "", err
			}

			return job.Raw.Color, nil
		}
	}

	return "", errors.Errorf("Job %v has not been finished after specified delay", name)
}

func (jenkins Jenkins) IsJobQueued(name string) (*bool, error) {
	job, err := jenkins.getJob(name)
	if err != nil {
		return nil, err
	}

	isQueued, err := job.IsQueued()
	if err != nil {
		return nil, err
	}

	return &isQueued, nil
}

func (jenkins Jenkins) IsJobRunning(name string) (*bool, error) {
	job, err := jenkins.getJob(name)
	if err != nil {
		return nil, err
	}

	isRunning, err := job.IsRunning()
	if err != nil {
		return nil, err
	}

	return &isRunning, nil
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
