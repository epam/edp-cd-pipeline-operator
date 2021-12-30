package jenkins

import (
	"net/http"
	"testing"

	"github.com/bndr/gojenkins"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
)

const (
	str    = `{"crumbRequestField": "file"}`
	name   = "jenkins-name"
	folder = "jenkins-folder"
)

func createJenkins(t *testing.T) Jenkins {
	t.Helper()
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder(http.MethodGet, "https://domain/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	client, err := gojenkins.CreateJenkins(http.DefaultClient, "https://domain").Init()
	if err != nil {
		t.Fatal(err)
	}
	return Jenkins{
		client: *client,
	}
}

func TestJenkins_CreateFolder(t *testing.T) {
	jenkins := createJenkins(t)
	httpmock.RegisterResponder(http.MethodGet, "https://domain/crumbIssuer/api/json/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodPost, "https://domain/createItem?Submit=OK&json=%7B%22mode%22%3A%22com.cloudbees.hudson.plugins.folder.Folder%22%2C%22name%22%3A%22jenkins-folder%22%7D&mode=com.cloudbees.hudson.plugins.folder.Folder&name=jenkins-folder", httpmock.NewStringResponder(http.StatusOK, ""))

	_, err := jenkins.CreateFolder(folder)
	assert.NoError(t, err)
}

func TestJenkins_GetJob(t *testing.T) {
	jenkins := createJenkins(t)
	httpmock.RegisterResponder(http.MethodGet, "https://domain/job/jenkins-name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))

	_, err := jenkins.getJob(name)
	assert.NoError(t, err)
}

func TestJenkins_GetJobError(t *testing.T) {
	jenkins := createJenkins(t)

	_, err := jenkins.getJob(name)
	assert.Error(t, err)
}

func TestJenkins_CreateJob_JobExists(t *testing.T) {
	jenkins := createJenkins(t)

	httpmock.RegisterResponder(http.MethodGet, "https://domain/job/jenkins-folder/job/jenkins-name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://domain/job/jenkins-name/api/json", httpmock.NewStringResponder(http.StatusOK, ""))

	_, err := jenkins.getJob(name)
	assert.NoError(t, err)

	err = jenkins.CreateJob(name, folder, "")
	assert.NoError(t, err)
}

func TestJenkins_CreateJob_Success(t *testing.T) {
	jenkins := createJenkins(t)

	httpmock.RegisterResponder(http.MethodGet, "https://domain/crumbIssuer/api/json/api/json", httpmock.NewStringResponder(http.StatusOK, str))
	httpmock.RegisterResponder(http.MethodGet, "https://domain/job/jenkins-folder/job/jenkins-name/api/json", httpmock.NewStringResponder(http.StatusNotFound, ""))
	httpmock.RegisterResponder(http.MethodPost, "https://domain/job/jenkins-folder/createItem?name=jenkins-name", httpmock.NewStringResponder(http.StatusOK, ""))

	err := jenkins.CreateJob(name, folder, "")
	assert.NoError(t, err)
}

func TestJenkins_InitSuccess(t *testing.T) {
	httpmock.DeactivateAndReset()
	httpmock.Activate()
	httpmock.RegisterResponder(http.MethodGet, "https://domain/api/json", httpmock.NewStringResponder(http.StatusOK, ""))
	httpmock.RegisterResponder(http.MethodGet, "https://domain/api/json/api/json", httpmock.NewStringResponder(http.StatusOK, str))

	jenkins, err := Init("https://domain/api/json", name, "")
	assert.NoError(t, err)
	assert.Equal(t, jenkins.username, name)
}
