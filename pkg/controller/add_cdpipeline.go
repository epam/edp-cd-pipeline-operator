package controller

import (
	"cd-pipeline-operator/pkg/controller/cdpipeline"
)

func init() {
	// AddToManagerFuncs is a list of functions to create controllers and add them to a manager.
	AddToManagerFuncs = append(AddToManagerFuncs, cdpipeline.Add)
}
