package main

import (
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"encoding/json"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
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
	if tag == "" && pj.DockerRepository() != "" {
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
	if err != nil {
		return o, err
	}
	err = json.Unmarshal(j, &job)
	if err != nil {
		return o, err
	}

	if job.Namespace == "" {
		job.Namespace = "default"
	}
	job.Name = job.Name + "-" + RandString(10)
	applyDeployOption(&job, pj.DockerRepository(), tag, option)

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

// applyDeployOption fills in the image tag and passes the deploy context to
// every container of the Job (init containers included) as environment variables:
//
//   - BRANCH: the selected branch (the default branch for autodeploy)
//   - DEPLOY_USER: Slack display name of the approving user, or the Slack user ID
//
// Variables with the same name in the manifest are replaced. The values are
// passed verbatim: "$" is escaped so that Kubernetes does not expand "$(NAME)"
// references to other environment variables of the container.
func applyDeployOption(job *batchv1.Job, image string, tag string, option DeployOption) {
	envs := []corev1.EnvVar{}
	if option.Branch != "" {
		envs = append(envs, corev1.EnvVar{Name: "BRANCH", Value: escapeEnvValue(option.Branch)})
	}
	if user := deployUserName(option.Assigner); user != "" {
		envs = append(envs, corev1.EnvVar{Name: "DEPLOY_USER", Value: escapeEnvValue(user)})
	}

	apply := func(containers []corev1.Container) {
		for i, container := range containers {
			if tag != "" && container.Image == image {
				containers[i].Image = image + ":" + tag
			}
			for _, env := range envs {
				containers[i].Env = upsertEnv(containers[i].Env, env)
			}
		}
	}
	apply(job.Spec.Template.Spec.InitContainers)
	apply(job.Spec.Template.Spec.Containers)
}

func deployUserName(user User) string {
	if user.SlackDisplayName != "" {
		return user.SlackDisplayName
	}
	return user.SlackUserID
}

// escapeEnvValue escapes "$" as "$$", which Kubernetes reduces back to a
// single "$" without expanding "$(NAME)" references.
func escapeEnvValue(s string) string {
	return strings.ReplaceAll(s, "$", "$$")
}

// upsertEnv replaces the first variable named env.Name in place, drops any
// later duplicates (Kubernetes lets the last one win), and appends env when
// the name is not defined yet. Keeping the original position matters because
// Kubernetes expands "$(NAME)" only from variables defined earlier.
func upsertEnv(envs []corev1.EnvVar, env corev1.EnvVar) []corev1.EnvVar {
	out := make([]corev1.EnvVar, 0, len(envs)+1)
	replaced := false
	for _, e := range envs {
		if e.Name != env.Name {
			out = append(out, e)
			continue
		}
		if !replaced {
			out = append(out, env)
			replaced = true
		}
	}
	if !replaced {
		out = append(out, env)
	}
	return out
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
