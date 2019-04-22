package settings

import (
	"cd-pipeline-handler-controller/pkg/openshift"
	"errors"
	"fmt"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func GetUserSettingConfigMap(clientSet *openshift.ClientSet, namespace string, property string) (string, error) {
	userSettings, err := clientSet.CoreClient.ConfigMaps(namespace).Get("user-settings", metav1.GetOptions{})
	if err != nil {
		errorMsg := fmt.Sprintf("Unable to get user settings configmap: %v", err)
		log.Println(errorMsg)
		return "", errors.New(errorMsg)
	}

	return userSettings.Data[property], nil
}
