package main

import (
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"regexp"
	"strings"
)

type DeployPhase struct {
	Name          string `yaml:"name"`
	Kind          string `yaml:"kind"`
	Path          string `yaml:"path"`
	AutoDeploy    bool   `yaml:"autoDeploy"`
	NotifyChannel string `yaml:"notifyChannel"`
}

type DeployProject struct {
	ID               string
	Kind             string
	jenkinsJob       string
	gitHubRepository string
	manifestPath     string
	dockerRegistry   string
	Alias            string
	Phases           []DeployPhase
}

func (p DeployProject) FindPhase(name string) DeployPhase {
	for _, phase := range p.Phases {
		if phase.Name == name {
			return phase
		}
	}
	return DeployPhase{}
}

func (pj DeployProject) JenkinsJob() string {
	return pj.jenkinsJob
}

func (pj DeployProject) GitHubRepository() string {
	return pj.gitHubRepository
}

func (pj DeployProject) K8SMetadata() string {
	return pj.manifestPath
}

func (pj DeployProject) DockerRepository() string {
	return pj.dockerRegistry
}

func (pj DeployProject) ECRRepository() string {
	path := strings.Split(pj.dockerRegistry, "/")
	return path[1]
}

type ProjectList struct {
	Items []DeployProject
}

func NewProjectList() (pl ProjectList) {
	pl.Reload()
	return
}

func (p *ProjectList) Reload() {
	var tmp []DeployProject
	cml := getConfigMapList("project")
	for _, cm := range cml.Items {
		pj := DeployProject{}
		pj.ID = cm.Name
		pj.Kind = cm.Data["Kind"]
		pj.jenkinsJob = cm.Data["JenkinsJob"]
		pj.gitHubRepository = cm.Data["GitHubRepository"]
		pj.manifestPath = cm.Data["ManifestPath"]
		pj.dockerRegistry = cm.Data["DockerRegistry"]
		pj.Alias = cm.Data["Alias"]
		yaml.Unmarshal([]byte(cm.Data["Phases"]), &pj.Phases)
		tmp = append(tmp, pj)
	}
	p.Items = tmp
}

func (p ProjectList) Find(id string) DeployProject {
	for _, pj := range p.Items {
		if pj.ID == id {
			return pj
		}
	}
	fmt.Printf("[ERROR] No Such Project. ID: %s", id)
	return DeployProject{}
}

func (p ProjectList) FindByAlias(id string) (DeployProject, error) {
	for _, pj := range p.Items {
		if regexp.MustCompile(pj.Alias).Match([]byte(id)) {
			return pj, nil
		}
	}
	return DeployProject{}, fmt.Errorf("[ERROR] No Such Project. ID: %s", id)
}
