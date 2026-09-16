package main

import (
	"testing"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
)

func newTestJob(initImage, image string, env ...corev1.EnvVar) *batchv1.Job {
	job := &batchv1.Job{}
	if initImage != "" {
		job.Spec.Template.Spec.InitContainers = []corev1.Container{{Name: "init", Image: initImage}}
	}
	job.Spec.Template.Spec.Containers = []corev1.Container{{Name: "main", Image: image, Env: env}}
	return job
}

func envMap(envs []corev1.EnvVar) map[string]string {
	m := map[string]string{}
	for _, e := range envs {
		m[e.Name] = e.Value
	}
	return m
}

func TestApplyDeployOption_ImageTag(t *testing.T) {
	image := "123456789012.dkr.ecr.ap-northeast-1.amazonaws.com/app"

	t.Run("untagged image matching DockerRegistry gets the tag", func(t *testing.T) {
		job := newTestJob(image, image)
		applyDeployOption(job, image, "abc1234", DeployOption{})
		if got := job.Spec.Template.Spec.Containers[0].Image; got != image+":abc1234" {
			t.Errorf("container image = %q", got)
		}
		if got := job.Spec.Template.Spec.InitContainers[0].Image; got != image+":abc1234" {
			t.Errorf("init container image = %q", got)
		}
	})

	t.Run("image already tagged in the manifest is left as is", func(t *testing.T) {
		job := newTestJob("", image+":pinned")
		applyDeployOption(job, image, "abc1234", DeployOption{})
		if got := job.Spec.Template.Spec.Containers[0].Image; got != image+":pinned" {
			t.Errorf("container image = %q", got)
		}
	})

	t.Run("no tag (project without DockerRegistry) leaves the image untouched", func(t *testing.T) {
		job := newTestJob("", image)
		applyDeployOption(job, "", "", DeployOption{})
		if got := job.Spec.Template.Spec.Containers[0].Image; got != image {
			t.Errorf("container image = %q", got)
		}
	})
}

func TestApplyDeployOption_Env(t *testing.T) {
	image := "123456789012.dkr.ecr.ap-northeast-1.amazonaws.com/app"

	t.Run("branch and user are passed to every container", func(t *testing.T) {
		job := newTestJob(image, image)
		applyDeployOption(job, image, "abc1234", DeployOption{
			Branch:   "feature/x",
			Assigner: User{SlackUserID: "U123", SlackDisplayName: "user1"},
		})
		for _, c := range append(job.Spec.Template.Spec.InitContainers, job.Spec.Template.Spec.Containers...) {
			env := envMap(c.Env)
			if env["BRANCH"] != "feature/x" {
				t.Errorf("%s: BRANCH = %q", c.Name, env["BRANCH"])
			}
			if env["DEPLOY_USER"] != "user1" {
				t.Errorf("%s: DEPLOY_USER = %q", c.Name, env["DEPLOY_USER"])
			}
		}
	})

	t.Run("manifest defaults are overridden, other variables are kept", func(t *testing.T) {
		job := newTestJob("", image,
			corev1.EnvVar{Name: "STAGE", Value: "staging"},
			corev1.EnvVar{Name: "BRANCH", Value: "master"},
			corev1.EnvVar{Name: "DEPLOY_USER", Value: "eks-deploy-job"},
		)
		applyDeployOption(job, image, "abc1234", DeployOption{
			Branch:   "feature/x",
			Assigner: User{SlackUserID: "U123"},
		})
		envs := job.Spec.Template.Spec.Containers[0].Env
		if len(envs) != 3 {
			t.Fatalf("env count = %d, want 3: %v", len(envs), envs)
		}
		env := envMap(envs)
		if env["STAGE"] != "staging" || env["BRANCH"] != "feature/x" || env["DEPLOY_USER"] != "U123" {
			t.Errorf("env = %v", env)
		}
	})

	t.Run("duplicate names in the manifest are all replaced", func(t *testing.T) {
		job := newTestJob("", image,
			corev1.EnvVar{Name: "BRANCH", Value: "master"},
			corev1.EnvVar{Name: "STAGE", Value: "staging"},
			corev1.EnvVar{Name: "BRANCH", Value: "develop"},
		)
		applyDeployOption(job, image, "abc1234", DeployOption{Branch: "feature/x"})
		envs := job.Spec.Template.Spec.Containers[0].Env
		if len(envs) != 2 {
			t.Fatalf("env count = %d, want 2: %v", len(envs), envs)
		}
		if envs[0].Name != "BRANCH" || envs[0].Value != "feature/x" || envs[1].Name != "STAGE" {
			t.Errorf("env = %v", envs)
		}
	})

	t.Run("replaced variable keeps its position so later references still expand", func(t *testing.T) {
		job := newTestJob("", image,
			corev1.EnvVar{Name: "BRANCH", Value: "master"},
			corev1.EnvVar{Name: "CHECKOUT_REF", Value: "refs/heads/$(BRANCH)"},
		)
		applyDeployOption(job, image, "abc1234", DeployOption{Branch: "feature/x"})
		envs := job.Spec.Template.Spec.Containers[0].Env
		if len(envs) != 2 || envs[0].Name != "BRANCH" || envs[0].Value != "feature/x" || envs[1].Name != "CHECKOUT_REF" {
			t.Errorf("env = %v", envs)
		}
	})

	t.Run("dollar signs are escaped so Kubernetes does not expand them", func(t *testing.T) {
		job := newTestJob("", image)
		applyDeployOption(job, image, "abc1234", DeployOption{
			Branch:   "$(DB_PASSWORD)",
			Assigner: User{SlackUserID: "U123", SlackDisplayName: "user$1"},
		})
		env := envMap(job.Spec.Template.Spec.Containers[0].Env)
		if env["BRANCH"] != "$$(DB_PASSWORD)" || env["DEPLOY_USER"] != "user$$1" {
			t.Errorf("env = %v", env)
		}
	})

	t.Run("empty branch and unknown user add nothing", func(t *testing.T) {
		job := newTestJob("", image, corev1.EnvVar{Name: "STAGE", Value: "staging"})
		applyDeployOption(job, image, "abc1234", DeployOption{})
		if envs := job.Spec.Template.Spec.Containers[0].Env; len(envs) != 1 {
			t.Errorf("env = %v, want only STAGE", envs)
		}
	})
}
