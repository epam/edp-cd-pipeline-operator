package service

import (
	"fmt"
	"log"
	"testing"
)

func TestExampleJenkins_CreateCDPipeline(t *testing.T) {
	url := "https://jenkins-mr-1717-8-edp-cicd.delivery.aws.main.edp.projects.epam.com/"
	appName := "test-12"
	folderName := "cdpipeline"
	username := "admin"
	token := ""

	jenkinsInstance, err := initJenkins(url, username, token)
	if err != nil {
		log.Print(err)
	}

	err = createCDPipeline(*jenkinsInstance, appName, folderName)
	if err != nil {
		fmt.Println(err)
	}
}