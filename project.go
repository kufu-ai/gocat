package main

import (
	"bytes"
	"fmt"
	yaml "gopkg.in/yaml.v2"
	"regexp"
	"strings"
	"text/template"
)

type PayloadVars struct {
	Tag string
}

func (self PayloadVars) Parse(s string) (string, error) {
	b := bytes.NewBuffer([]byte(""))
	tmpl, err := template.New("").Parse(s)
	if err != nil {
		return "", err
	}
	err = tmpl.Execute(b, self)
	return b.String(), err
}

type DeployPhase struct {
	Name          string      `yaml:"name"`
	Kind          string      `yaml:"kind"`
	Path          string      `yaml:"path"` // for job
	AutoDeploy    bool        `yaml:"autoDeploy"`
	NotifyChannel string      `yaml:"notifyChannel"`
	Payload       string      `yaml:"payload"`
	Destination   Destination `yaml:"destination"`
}

type DeployProject struct {
	ID               string
	Kind             string
	jenkinsJob       string
	funcName         string // for Lambda
	gitHubRepository string
	manifestPath     string
	dockerRegistry   string
	filterRegexp     string
	targetRegexp     string
	steps            []string
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

func (pj DeployProject) FuncName() string {
	return pj.funcName
}

func (pj DeployProject) Steps() []string {
	return pj.steps
}

func (pj DeployProject) FilterRegexp() string {
	if pj.filterRegexp == "" {
		return "^{{.Branch}}$"
	}
	return pj.filterRegexp
}

func (pj DeployProject) TargetRegexp() string {
	if pj.targetRegexp == "" {
		return `\b[0-9a-f]{5,40}\b`
	}
	return pj.targetRegexp
}

func (pj DeployProject) DockerRepository() string {
	return pj.dockerRegistry
}

func (pj DeployProject) ECRRepository() string {
	path := strings.Split(pj.dockerRegistry, "/")
	if len(path) < 2 {
		return ""
	}
	return strings.Join(path[1:], "/")
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
		pj.filterRegexp = cm.Data["FilterRegexp"]
		pj.targetRegexp = cm.Data["TargetRegexp"]
		pj.funcName = cm.Data["FuncName"]
		pj.Alias = cm.Data["Alias"]
		yaml.Unmarshal([]byte(cm.Data["Steps"]), &pj.steps)
		yaml.Unmarshal([]byte(cm.Data["Phases"]), &pj.Phases)
		for i, phase := range pj.Phases {
			if phase.Kind == "" {
				pj.Phases[i].Kind = pj.Kind
			}
			if phase.Destination.Kind == "" {
				pj.Phases[i].Destination.Kind = pj.Phases[i].Kind
			}
			if phase.Destination.Kustomize.Path == "" {
				pj.Phases[i].Destination.Kustomize.Path = phase.Path
			}
			if phase.Destination.Kustomize.Image == "" {
				pj.Phases[i].Destination.Kustomize.Image = pj.DockerRepository()
			}
			if phase.Destination.ECS.Image == "" {
				pj.Phases[i].Destination.ECS.Image = pj.DockerRepository()
			}
		}
		tmp = append(tmp, pj)
	}
	p.Items = tmp
}

func (p ProjectList) FindAll(ids []string) (o []DeployProject) {
	for _, id := range ids {
		for _, pj := range p.Items {
			if pj.ID == id {
				o = append(o, pj)
			}
		}
	}
	return o
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
