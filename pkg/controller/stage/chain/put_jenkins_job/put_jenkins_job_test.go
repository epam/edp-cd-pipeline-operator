package put_jenkins_job

import (
	"github.com/epam/edp-codebase-operator/v2/pkg/util"
	"github.com/epmd-edp/cd-pipeline-operator/v2/pkg/apis/edp/v1alpha1"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestGetQualityGateStagesMethod_ShouldReturnParsedStagesScenarioFirst(t *testing.T) {
	qualityGates := []v1alpha1.QualityGate{
		{
			QualityGateType: "autotests",
			StepName:        "aut1",
			AutotestName:    util.GetStringP("aut1"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "autotests",
			StepName:        "aut2",
			AutotestName:    util.GetStringP("aut2"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "autotests",
			StepName:        "aut3",
			AutotestName:    util.GetStringP("aut3"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "autotests",
			StepName:        "aut4",
			AutotestName:    util.GetStringP("aut4"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "manual",
			StepName:        "man1",
			AutotestName:    util.GetStringP("man1"),
			BranchName:      util.GetStringP("master"),
		},
	}
	stages, err := getQualityGateStages(qualityGates)
	assert.NoError(t, err)
	assert.NotNil(t, stages)
	expected := "[{\"name\":\"autotests\",\"step_name\":\"aut1\"},{\"name\":\"autotests\",\"step_name\":\"aut2\"},{\"name\":\"autotests\",\"step_name\":\"aut3\"},{\"name\":\"autotests\",\"step_name\":\"aut4\"}],{\"name\":\"manual\",\"step_name\":\"man1\"}"
	assert.Equal(t, expected, *stages)
}

func TestGetQualityGateStagesMethod_ShouldReturnParsedStagesScenarioSecond(t *testing.T) {
	qualityGates := []v1alpha1.QualityGate{
		{
			QualityGateType: "autotests",
			StepName:        "aut1",
			AutotestName:    util.GetStringP("aut1"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "autotests",
			StepName:        "aut3",
			AutotestName:    util.GetStringP("aut3"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "autotests",
			StepName:        "aut4",
			AutotestName:    util.GetStringP("aut4"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "manual",
			StepName:        "man1",
			AutotestName:    util.GetStringP("man1"),
			BranchName:      util.GetStringP("master"),
		},
		{
			QualityGateType: "autotests",
			StepName:        "aut2",
			AutotestName:    util.GetStringP("aut2"),
			BranchName:      util.GetStringP("master"),
		},
	}
	stages, err := getQualityGateStages(qualityGates)
	assert.NoError(t, err)
	assert.NotNil(t, stages)
	expected := "[{\"name\":\"autotests\",\"step_name\":\"aut1\"},{\"name\":\"autotests\",\"step_name\":\"aut3\"},{\"name\":\"autotests\",\"step_name\":\"aut4\"}],{\"name\":\"manual\",\"step_name\":\"man1\"},{\"name\":\"autotests\",\"step_name\":\"aut2\"}"
	assert.Equal(t, expected, *stages)
}

func TestGetQualityGateStagesMethod_ShouldReturnParsedStagesAsNilScenarioFirst(t *testing.T) {
	stages, err := getQualityGateStages(nil)
	assert.NoError(t, err)
	assert.Nil(t, stages)
}

func TestGetQualityGateStagesMethod_ShouldReturnParsedStagesAsNilScenarioSecond(t *testing.T) {
	qualityGates := []v1alpha1.QualityGate{}
	stages, err := getQualityGateStages(qualityGates)
	assert.NoError(t, err)
	assert.Nil(t, stages)
}
