package models

import "edp-admin-console/models/dto"

type CDPipelineDTO struct {
	Name             string                  `json:"name"`
	Status           string                  `json:"status"`
	JenkinsLink      string                  `json:"jenkinsLink"`
	CodebaseBranches []dto.CodebaseBranchDTO `json:"codebaseBranches"`
	Stages           []CDPipelineStageView   `json:"stages"`
}

type CDPipelineView struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	JenkinsLink string `json:"jenkinsLink"`
}

type StageCreate struct {
	Name            string `json:"name" valid:"Required;Match(/^[a-z0-9]([-a-z0-9]*[a-z0-9])$/)"`
	Description     string `json:"description" valid:"Required"`
	StepName        string `json:"stepName" valid:"Required;Match(/^[A-z0-9-._]/)"`
	QualityGateType string `json:"qualityGateType" valid:"Required"`
	TriggerType     string `json:"triggerType" valid:"Required"`
	Order           int    `json:"order" valid:"Match(/^[0-9]$/)"`
}

type CDPipelineCreateCommand struct {
	Name               string                  `json:"name" valid:"Required;Match(/^[a-z0-9]([-a-z0-9]*[a-z0-9])$/)"`
	Applications       []ApplicationWithBranch `json:"applications" valid:"Required"`
	ThirdPartyServices []string                `json:"services"`
	Stages             []StageCreate           `json:"stages" valid:"Required"`
}

type ApplicationWithBranch struct {
	ApplicationName string `json:"appName" valid:"Required;Match(/^[a-z][a-z0-9-]*[a-z0-9]$/)"`
	BranchName      string `json:"branchName" valid:"Required;Match(/^[a-z0-9][a-z0-9-._]*[a-z0-9]$/)"`
}
