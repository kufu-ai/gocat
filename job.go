package main

import (
	"context"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"log"
)

func createJob(job *batchv1.Job) (err error) {
	client, err := newKubernetesClient()
	if err != nil {
		log.Print(err)
		return
	}

	job, err = client.BatchV1().Jobs(job.Namespace).Create(context.Background(), job, metav1.CreateOptions{})
	if err != nil {
		log.Print(err)
		return
	}

	return
}

func getJob(jobName string, ns string) (job *batchv1.Job, err error) {
	client, err := newKubernetesClient()
	if err != nil {
		log.Print(err)
		return
	}

	job, err = client.BatchV1().Jobs(ns).Get(context.Background(), jobName, metav1.GetOptions{})
	if err != nil {
		log.Print(err)
		return
	}

	return
}
