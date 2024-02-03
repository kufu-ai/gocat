package main

import (
	"fmt"
	"log"
	"math/rand"
	"time"

	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
	yaml "k8s.io/apimachinery/pkg/util/yaml"
)

type ModelJob struct {
	github *GitHub
}

func NewModelJob(github *GitHub) ModelJob {
	return ModelJob{github}
}

type ModelJobDeployOutput struct {
	status    DeployStatus
	Namespace string
	Name      string
	Path      string
	ImageTag  string
}

func (self ModelJobDeployOutput) Status() DeployStatus {
	return self.status
}

func (self ModelJobDeployOutput) Message() string {
	return fmt.Sprintf("*Namespace*: %s\n*Name*: %s\n*Path*: %s\n*ImageTag*: %s", self.Namespace, self.Name, self.Path, self.ImageTag)
}

func (self ModelJob) Deploy(pj DeployProject, phase string, option DeployOption) (DeployOutput, error) {
	o := ModelJobDeployOutput{status: DeployStatusFail}
	p := pj.FindPhase(phase)
	rawFile, err := self.github.GetFile(p.Path)
	if err != nil {
		return o, err
	}

	tag := option.Tag
	if tag == "" {
		ecr, err := CreateECRInstance()
		if err != nil {
			return o, err
		}
		tag, err = ecr.FindImageTagByRegexp(pj.ECRRegistryId(), pj.ECRRepository(), pj.ImageTagRegexp(), pj.TargetRegexp(), ImageTagVars{Branch: option.Branch, Phase: phase})
		if err != nil {
			return o, err
		}
	}

	job := batchv1.Job{}
	j, err := yaml.ToJSON(rawFile)
	err = json.Unmarshal(j, &job)
	if err != nil {
		return o, err
	}

	if job.Namespace == "" {
		job.Namespace = "default"
	}
	job.Name = job.Name + "-" + RandString(10)
	for i, container := range job.Spec.Template.Spec.Containers {
		if container.Image == pj.DockerRepository() {
			job.Spec.Template.Spec.Containers[i].Image = container.Image + ":" + tag
		}
	}

	if err = createJob(&job); err != nil {
		return o, err
	}

	if option.Wait {
		err = self.Watch(job.Name, job.Namespace)
		if err != nil {
			return o, err
		}
	}

	o.status = DeployStatusSuccess
	o.Namespace = job.Namespace
	o.Name = job.Name
	o.Path = p.Path
	o.ImageTag = tag
	return o, nil
}

func (self ModelJob) Watch(name, namespace string) error {
	// We don't stop the ticker as this is a long-running process
	// with no way to cancel it.
	t := time.NewTicker(time.Duration(20) * time.Second)
	log.Println("[INFO] Watch job", name)
	for range t.C {
		job, err := getJob(name, namespace)
		if err != nil {
			log.Println("[ERROR] Quit watching job ", job.Name)
			t.Stop()
			return err
		}
		if job.Status.Succeeded >= 1 {
			t.Stop()
			return nil
		}
		if job.Status.Failed >= 1 {
			t.Stop()
			return fmt.Errorf("[ERROR] Failed %s execution", job.Name)
		}
	}
	return nil
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letters = []rune("abcdefghijklmnopqrstuvwxyz1234567890")

func RandString(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
