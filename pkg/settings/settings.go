package settings

import (
	"errors"
	"fmt"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/openshift"
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
