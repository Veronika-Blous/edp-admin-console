package query

type Stage struct {
	Id                   int         `json:"-" orm:"column(id)"`
	Name                 string      `json:"name" orm:"column(name)"`
	Description          string      `json:"description" orm:"column(description)"`
	TriggerType          string      `json:"triggerType" orm:"column(trigger_type)"`
	QualityGate          string      `json:"qualityGate" orm:"column(quality_gate)"`
	JenkinsStepName      string      `json:"jenkinsStepName" orm:"column(jenkins_step_name)"`
	Order                int         `json:"order" orm:"column(order)"`
	OpenshiftProjectLink string      `json:"openshiftProjectLink" orm:"-"`
	CDPipeline           *CDPipeline `json:"-" orm:"rel(fk);column(cd_pipeline_id)"`
}

func (cb *Stage) TableName() string {
	return "cd_stage"
}
