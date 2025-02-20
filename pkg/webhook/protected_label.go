package webhook

import (
	"errors"
	"strings"

	"k8s.io/apimachinery/pkg/api/equality"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"slices"

	pipelineApi "github.com/epam/edp-cd-pipeline-operator/v2/api/v1"
)

const (
	protectedLabel  = "app.edp.epam.com/edit-protection"
	deleteOperation = "delete"
	updateOperation = "update"
)

func hasProtectedLabel(obj runtime.Object, operation string) bool {
	o, ok := obj.(metaV1.Object)
	if !ok {
		return false
	}

	return o.GetLabels()[protectedLabel] != "" &&
		slices.Contains(strings.Split(o.GetLabels()[protectedLabel], "-"), operation)
}

func isSpecUpdated(oldObj, newObj runtime.Object) bool {
	switch old := oldObj.(type) {
	case *pipelineApi.Stage:
		if newStage, ok := newObj.(*pipelineApi.Stage); ok {
			return !equality.Semantic.DeepEqual(old.Spec, newStage.Spec)
		}
	case *pipelineApi.CDPipeline:
		if newPipeline, ok := newObj.(*pipelineApi.CDPipeline); ok {
			return !equality.Semantic.DeepEqual(old.Spec, newPipeline.Spec)
		}
	}

	return false
}

func checkResourceProtectionFromModificationOnDelete(obj runtime.Object) error {
	if hasProtectedLabel(obj, deleteOperation) {
		return errors.New("resource contains label that protects it from deletion")
	}

	return nil
}

func checkResourceProtectionFromModificationOnUpdate(oldObj, newObj runtime.Object) error {
	if hasProtectedLabel(newObj, updateOperation) && isSpecUpdated(oldObj, newObj) {
		return errors.New("resource contains label that protects it from modification")
	}

	return nil
}
