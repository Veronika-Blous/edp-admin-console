/*
 * Copyright 2019 EPAM Systems.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package service

import (
	"edp-admin-console/context"
	"edp-admin-console/k8s"
	"edp-admin-console/models"
	"edp-admin-console/models/query"
	"edp-admin-console/repository"
	"fmt"
	"github.com/astaxie/beego"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/rest"
	"log"
	"sort"
	"strings"
	"time"
)

type CDPipelineService struct {
	Clients               k8s.ClientSet
	ICDPipelineRepository repository.ICDPipelineRepository
	CodebaseService       CodebaseService
	BranchService         CodebaseBranchService
}

type ErrMsg struct {
	Message    string
	StatusCode int
}

const OpenshiftProjectLink = "https://master.%s/console/project/"

func (s *CDPipelineService) CreatePipeline(cdPipeline models.CDPipelineCreateCommand) (*k8s.CDPipeline, error) {
	log.Printf("Start creating CD Pipeline: %v", cdPipeline)

	exist := s.CodebaseService.checkAppAndBranch(cdPipeline.Applications)
	if !exist {
		return nil, models.ErrNonValidRelatedBranch
	}

	cdPipelineReadModel, err := s.GetCDPipelineByName(cdPipeline.Name)
	if err != nil {
		return nil, err
	}

	if cdPipelineReadModel != nil {
		log.Printf("CD Pipeline %s is already exists in DB.", cdPipeline.Name)
		return nil, models.ErrCDPipelineIsExists
	}

	edpRestClient := s.Clients.EDPRestClient
	pipelineCR, err := getCDPipelineCR(edpRestClient, cdPipeline.Name, context.Namespace)
	if err != nil {
		return nil, err
	}

	if pipelineCR != nil {
		log.Printf("CD Pipeline %s is already exists in k8s.", cdPipeline.Name)
		return nil, models.ErrCDPipelineIsExists
	}

	crd := &k8s.CDPipeline{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "edp.epam.com/v1alpha1",
			Kind:       "CDPipeline",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cdPipeline.Name,
			Namespace: context.Namespace,
		},
		Spec: convertPipelineData(cdPipeline),
		Status: k8s.CDPipelineStatus{
			LastTimeUpdated: time.Now(),
			Status:          "initialized",
		},
	}

	cdPipelineCr := &k8s.CDPipeline{}
	err = edpRestClient.Post().Namespace(context.Namespace).Resource("cdpipelines").Body(crd).Do().Into(cdPipelineCr)

	if err != nil {
		log.Printf("An error has occurred while creating CD Pipeline object in k8s: %s", err)
		return nil, err
	}
	log.Printf("Pipeline CR %v is saved into k8s", cdPipeline)

	_, err = s.CreateStages(edpRestClient, cdPipeline)
	if err != nil {
		log.Printf("An error has occurred while creating Stages in k8s: %s", err)
		return nil, err
	}
	log.Printf("Stages for CD Pipeline %s were created in k8s: %v", cdPipeline.Name, cdPipeline.Stages)

	return cdPipelineCr, nil
}

func (s *CDPipelineService) GetCDPipelineByName(pipelineName string) (*query.CDPipeline, error) {
	log.Println("Start execution of GetCDPipelineByName method...")
	cdPipeline, err := s.ICDPipelineRepository.GetCDPipelineByName(pipelineName)
	if err != nil {
		log.Printf("An error has occurred while getting CD Pipeline from database: %s", err)
		return nil, err
	}
	if cdPipeline != nil {
		createJenkinsLink(cdPipeline)
		if len(cdPipeline.Stage) != 0 {
			sortStagesByOrder(cdPipeline.Stage)
			createOpenshiftProjectLinks(cdPipeline.Stage, cdPipeline.Name)
			log.Printf("Fetched Stages. Count: {%v}. Rows: {%v}", len(cdPipeline.Stage), cdPipeline.Stage)
		}
		for i, branch := range cdPipeline.CodebaseBranch {
			branch.AppName = branch.Codebase.Name
			cdPipeline.CodebaseBranch[i] = branch
		}
		log.Printf("Fetched CD Pipeline from DB: %v", cdPipeline)
	}

	return cdPipeline, nil
}

func (s *CDPipelineService) CreateStages(edpRestClient *rest.RESTClient, cdPipeline models.CDPipelineCreateCommand) ([]k8s.Stage, error) {
	log.Printf("Start creating CR stages: %+v\n", cdPipeline.Stages)

	err := checkStagesInK8s(edpRestClient, cdPipeline.Name, cdPipeline.Stages)
	if err != nil {
		log.Println(err)
		return nil, err
	}

	stagesCr, err := saveStagesIntoK8s(edpRestClient, cdPipeline.Name, cdPipeline.Stages)
	if err != nil {
		return nil, err
	}

	return stagesCr, nil
}

func (s *CDPipelineService) GetAllPipelines(criteria query.CDPipelineCriteria) ([]*query.CDPipeline, error) {
	log.Println("Start fetching all CD Pipelines...")
	cdPipelines, err := s.ICDPipelineRepository.GetCDPipelines(criteria)
	if err != nil {
		log.Printf("An error has occurred while getting CD Pipelines from database: %s", err)
		return nil, err
	}

	if len(cdPipelines) != 0 {
		createJenkinsLinks(cdPipelines)
	}
	log.Printf("Fetched CD Pipelines. Count: {%v}. Rows: {%v}", len(cdPipelines), cdPipelines)

	return cdPipelines, nil
}

func sortStagesByOrder(stages []*query.Stage) {
	sort.Slice(stages, func(i, j int) bool {
		return stages[i].Order < stages[j].Order
	})
}

func (s *CDPipelineService) GetStage(cdPipelineName, stageName string) (*models.StageView, error) {
	log.Printf("Start fetching Stage by CD Pipeline %s and Stage %s names...", cdPipelineName, stageName)
	stage, err := s.ICDPipelineRepository.GetStage(cdPipelineName, stageName)
	if err != nil {
		log.Printf("An error has occurred while getting Stage from database: %s", err)
		return nil, err
	}
	log.Printf("Fetched Stage: {%v}", stage)
	return stage, nil
}

func createOpenshiftProjectLinks(stages []*query.Stage, cdPipelineName string) {
	for index, stage := range stages {
		stage.OpenshiftProjectLink = fmt.Sprintf(OpenshiftProjectLink+"%s-%s-%s", beego.AppConfig.String("dnsWildcard"), context.Tenant, cdPipelineName, stage.Name)
		stages[index] = stage
	}
}

func convertPipelineData(cdPipeline models.CDPipelineCreateCommand) k8s.CDPipelineSpec {
	var codebaseBranches []string
	for _, v := range cdPipeline.Applications {
		codebaseBranches = append(codebaseBranches, fmt.Sprintf("%s-%s", v.ApplicationName, v.BranchName))
	}
	return k8s.CDPipelineSpec{
		Name:               cdPipeline.Name,
		CodebaseBranch:     codebaseBranches,
		ThirdPartyServices: cdPipeline.ThirdPartyServices,
	}
}

func getCDPipelineCR(edpRestClient *rest.RESTClient, pipelineName string, namespace string) (*k8s.CDPipeline, error) {
	cdPipeline := &k8s.CDPipeline{}
	err := edpRestClient.Get().Namespace(namespace).Resource("cdpipelines").Name(pipelineName).Do().Into(cdPipeline)

	if k8serrors.IsNotFound(err) {
		log.Printf("Pipeline resource %s doesn't exist.", pipelineName)
		return nil, nil
	}

	if err != nil {
		log.Printf("An error has occurred while getting Pipeline CR from k8s: %s", err)
		return nil, err
	}

	return cdPipeline, nil
}

func createJenkinsLinks(cdPipelines []*query.CDPipeline) {
	wildcard := beego.AppConfig.String("dnsWildcard")
	for index, pipeline := range cdPipelines {
		pipeline.JenkinsLink = fmt.Sprintf("https://%s-%s-edp-cicd.%s/job/%s", "jenkins", context.Tenant, wildcard, fmt.Sprintf("%s-%s", pipeline.Name, "cd-pipeline"))
		cdPipelines[index] = pipeline
		log.Printf("Created Jenkins link %v", pipeline.JenkinsLink)
	}
}

func createJenkinsLink(cdPipeline *query.CDPipeline) {
	wildcard := beego.AppConfig.String("dnsWildcard")
	cdPipeline.JenkinsLink = fmt.Sprintf("https://%s-%s-edp-cicd.%s/job/%s", "jenkins", context.Tenant, wildcard, fmt.Sprintf("%s-%s", cdPipeline.Name, "cd-pipeline"))
	log.Printf("Created CD Pipeline Jenkins link %v", cdPipeline.JenkinsLink)
	createLinksForBranchEntities(cdPipeline.CodebaseBranch)
}

func createLinksForBranchEntities(branchEntities []*query.CodebaseBranch) {
	wildcard := beego.AppConfig.String("dnsWildcard")
	for index, branch := range branchEntities {
		branch.VCSLink = fmt.Sprintf("https://%s-%s-edp-cicd.%s/gitweb?p=%s.git;a=shortlog;h=refs/heads/%s", "gerrit", context.Tenant, wildcard, branch.Codebase.Name, branch.Name)
		branch.CICDLink = fmt.Sprintf("https://%s-%s-edp-cicd.%s/job/%s/view/%s", "jenkins", context.Tenant, wildcard, branch.Codebase.Name, strings.ToUpper(branch.Name))
		branchEntities[index] = branch
	}
}

func createCrd(cdPipelineName string, stage models.StageCreate) k8s.Stage {
	return k8s.Stage{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "edp.epam.com/v1alpha1",
			Kind:       "Stage",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", cdPipelineName, stage.Name),
			Namespace: context.Namespace,
		},
		Spec: k8s.StageSpec{
			Name:        stage.Name,
			Description: stage.Description,
			QualityGate: stage.QualityGateType,
			JenkinsStep: stage.StepName,
			TriggerType: stage.TriggerType,
			Order:       stage.Order,
			CdPipeline:  cdPipelineName,
		},
		Status: k8s.StageStatus{
			LastTimeUpdated: time.Now(),
			Status:          "initialized",
		},
	}
}

func saveStagesIntoK8s(edpRestClient *rest.RESTClient, cdPipelineName string, stages []models.StageCreate) ([]k8s.Stage, error) {
	var stagesCr []k8s.Stage
	for _, stage := range stages {
		crd := createCrd(cdPipelineName, stage)
		stageCr := k8s.Stage{}
		err := edpRestClient.Post().Namespace(context.Namespace).Resource("stages").Body(&crd).Do().Into(&stageCr)
		if err != nil {
			log.Printf("An error has occurred while creating Stage object in k8s: %s", err)
			return nil, err
		}
		log.Printf("Stage is saved into k8s: %+v\n", stage.Name)
		stagesCr = append(stagesCr, stageCr)
	}
	return stagesCr, nil
}

func checkStagesInK8s(edpRestClient *rest.RESTClient, cdPipelineName string, stages []models.StageCreate) error {
	for _, stage := range stages {
		stagesCr := &k8s.Stage{}
		stageName := fmt.Sprintf("%s-%s", cdPipelineName, stage.Name)
		err := edpRestClient.Get().Namespace(context.Namespace).Resource("stages").Name(stageName).Do().Into(stagesCr)

		if k8serrors.IsNotFound(err) {
			log.Printf("Stage %s doesn't exist.", stage.Name)
			continue
		}

		if err != nil {
			log.Printf("An error has occurred while getting Stage from k8s: %s", err)
			return err
		}

		if stagesCr != nil {
			log.Printf("CR Stage %s is already exists in k8s: %s", stageName, err)
			return fmt.Errorf("stage %s is already exists", stage.Name)
		}
	}
	return nil
}
