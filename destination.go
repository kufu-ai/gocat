package main

import (
	"fmt"
	"strings"
)

type IDestination interface {
	GetCurrentRevision(input GetCurrentRevisionInput) (string, error)
}

type GetCurrentRevisionInput struct {
	github *GitHub
}

type DestinationKustomize struct {
	Path  string `yaml:"path"`
	Image string `yaml:"image"`
}

func (self DestinationKustomize) GetCurrentRevision(input GetCurrentRevisionInput) (string, error) {
	kf, err := input.github.GetKustomization(self.Path)
	if err != nil {
		return "", err
	}
	for _, image := range kf.Images {
		if image.Name == self.Image {
			return image.NewTag, nil
		}
	}
	return "", nil
}

type DestinationECS struct {
	TaskDefinitionArn string `yaml:"taskDefinitionArn"`
	Image             string `yaml:"image"`
}

func (self DestinationECS) GetCurrentRevision(input GetCurrentRevisionInput) (string, error) {
	ecs, err := CreateECSInstance()
	if err != nil {
		return "", err
	}
	td, err := ecs.DescribeTaskDefinition(self.TaskDefinitionArn)
	if err != nil {
		return "", err
	}
	for _, container := range td.ContainerDefinitions {
		if container.Image != nil && *container.Image == self.Image {
			im := strings.Split(*container.Image, ":")
			if len(im) != 2 {
				return "", fmt.Errorf("[ERROR] Invalid image name")
			}
			return im[1], nil
		}
	}
	return "", fmt.Errorf("[ERROR] NotFound specified image")
}

type DestinationAPI struct {
	RevisionURL string `yaml:"revisionURL"`
}

func (self DestinationAPI) GetCurrentRevision(input GetCurrentRevisionInput) (string, error) {
	return "", fmt.Errorf("[ERROR] API is not supported")

}

type Destination struct {
	Kind      string               `yaml:"kind"`
	Kustomize DestinationKustomize `yaml:"kustomize"`
	ECS       DestinationECS       `yaml:"ecs"`
	API       DestinationAPI       `yaml:"api"`
}

func (self Destination) GetDest() IDestination {
	switch self.Kind {
	case "kustomize":
		return self.Kustomize
	case "ecs":
		return self.ECS
	default:
		return self.API
	}
}

func (self Destination) GetCurrentRevision(input GetCurrentRevisionInput) (string, error) {
	dest := self.GetDest()
	return dest.GetCurrentRevision(input)
}
