package main

import (
	"fmt"
	"golang.org/x/xerrors"
	"gopkg.in/src-d/go-billy.v4/memfs"
	git "gopkg.in/src-d/go-git.v4"
	"gopkg.in/src-d/go-git.v4/config"
	"gopkg.in/src-d/go-git.v4/plumbing"
	"gopkg.in/src-d/go-git.v4/plumbing/object"
	"gopkg.in/src-d/go-git.v4/plumbing/transport"
	"gopkg.in/src-d/go-git.v4/plumbing/transport/http"
	"gopkg.in/src-d/go-git.v4/storage/memory"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"sigs.k8s.io/kustomize/api/types"
	"time"
)

type GitDocAWSOperator struct {
	auth       transport.AuthMethod
	repo       string
	repository *git.Repository
}

func CreateGitDocAWSOperatorInstance(token string, repo string) (g GitDocAWSOperator) {
	g.auth = &http.BasicAuth{
		Username: "bot-zaim", // yes, this can be anything except an empty string
		Password: token,
	}
	g.repo = repo
	g.Clone()
	return
}

func (g *GitDocAWSOperator) Clone() error {
	fs := memfs.New()
	r, err := git.Clone(memory.NewStorage(), fs, &git.CloneOptions{
		URL:  g.Repo(),
		Auth: g.auth,
	})
	g.repository = r

	if err != nil {
		fmt.Println("[ERROR] ", err)
	}
	return err
}

func (g GitDocAWSOperator) Repo() string {
	return g.repo
}

func (g GitDocAWSOperator) DeleteBranch(branch string) (err error) {
	return g.repository.Storer.RemoveReference(plumbing.ReferenceName(branch))
}

func (g GitDocAWSOperator) PushDockerImageTag(id string, target string, phase string, tag string, targetTag string) (branch string, err error) {
	branch = fmt.Sprintf("bot/docker-image-tag-%s-%s-%s", id, phase, tag)

	g.DeleteBranch(branch)
	if err != nil {
		fmt.Println("[ERROR] Failed to DeleteBranch: ", xerrors.New(err.Error()))
	}

	// checkout

	w, err := g.repository.Worktree()
	if err != nil {
		return
	}

	err = w.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: plumbing.Master,
	})
	if err != nil {
		fmt.Println("[ERROR] Failed to Checkout master: ", xerrors.New(err.Error()))
		return
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin", Auth: g.auth})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		fmt.Println("[ERROR] Failed to Pull origin/master: ", xerrors.New(err.Error()))
		g.Clone()
	}
	err = nil

	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.ReferenceName(branch),
	})
	if err != nil {
		fmt.Println("[ERROR] Failed to Checkout workbranch: ", xerrors.New(err.Error()))
		return
	}

	err = g.commit(w, fmt.Sprintf("%s/%s/kustomization.yaml", target, phase), KustomizationOverWrite{tag, targetTag})
	if err != nil {
		fmt.Println("[ERROR] Failed to Marshal kustomize.yaml: ", xerrors.New(err.Error()))
		return
	}

	err = g.commit(w, fmt.Sprintf("%s/%s/configmap.yaml", target, phase), MemcachedOverWrite{})
	if err != nil {
		fmt.Println("[ERROR] Failed to Write MEMCACHED_PREFIX \\n: ", xerrors.New(err.Error()))
		return
	}

	hash, _ := w.Commit(
		fmt.Sprintf("Change docker image tag. target: %s, phase: %s, tag: %s.", target, phase, tag),
		&git.CommitOptions{
			Author: &object.Signature{
				Name:  "bot-zaim",
				Email: "",
				When:  time.Now(),
			},
		})
	g.repository.Storer.SetReference(plumbing.NewReferenceFromStrings(branch, hash.String()))

	// push
	remote, err := g.repository.Remote("origin")
	if err != nil {
		fmt.Println("[ERROR] Failed to Add remote origin: ", xerrors.New(err.Error()))
		return
	}
	err = remote.Push(&git.PushOptions{
		Progress: os.Stdout,
		RefSpecs: []config.RefSpec{
			config.RefSpec(plumbing.ReferenceName(branch) + ":" + plumbing.ReferenceName(fmt.Sprintf("refs/heads/%s", branch))),
		},
		Auth: g.auth,
	})
	if err != nil {
		fmt.Println("[ERROR] Failed to Push origin: ", xerrors.New(err.Error()))
	}
	return
}

type ConfigMap struct {
	APIVersion string            `yaml:"apiVersion"`
	Kind       string            `yaml:"kind"`
	Metadata   map[string]string `yaml:"metadata"`
	Data       map[string]string `yaml:"data"`
}

type OverWrite interface {
	Update([]byte) (interface{}, error)
}

func (g GitDocAWSOperator) commit(w *git.Worktree, targetFilePath string, o OverWrite) (err error) {
	_, err = w.Filesystem.Stat(targetFilePath)
	if err != nil {
		fmt.Println("[INFO] The file does not exist: ", xerrors.New(err.Error()))
		return nil
	}

	file, err := w.Filesystem.Open(targetFilePath)
	if err != nil {
		fmt.Println("[ERROR] Failed to Open file: ", xerrors.New(err.Error()))
		return
	}
	b, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println("[ERROR] Failed to ReadAll file: ", xerrors.New(err.Error()))
		return
	}
	err = file.Close()
	if err != nil {
		fmt.Println("[ERROR] Failed to Close file: ", xerrors.New(err.Error()))
		return
	}

	obj, err := o.Update(b)
	if err != nil {
		return
	}

	rb, err := yaml.Marshal(&obj)
	if err != nil {
		fmt.Println("[ERROR] Failed to Marshal kustomize.yaml: ", xerrors.New(err.Error()))
		return
	}

	err = w.Filesystem.Remove(targetFilePath)
	if err != nil {
		fmt.Println("[ERROR] Failed to Remove kustomize.yaml: ", xerrors.New(err.Error()))
		return
	}

	file, err = w.Filesystem.OpenFile(targetFilePath, os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fmt.Println("[ERROR] Failed to Open kustomize.yaml: ", xerrors.New(err.Error()))
		return
	}

	_, err = file.Write(rb)
	if err != nil {
		fmt.Println("[ERROR] Failed to Write kustomize.yaml: ", xerrors.New(err.Error()))
		return
	}

	_, err = file.Write([]byte("\n"))
	if err != nil {
		fmt.Println("[ERROR] Failed to Write \\n: ", xerrors.New(err.Error()))
		return
	}

	// git add
	_, err = w.Add(targetFilePath)
	if err != nil {
		fmt.Println("[ERROR] Failed to Add file to Worktree: ", xerrors.New(err.Error()))
		return
	}
	return
}

type KustomizationOverWrite struct {
	tag       string
	targetTag string
}

func (o KustomizationOverWrite) Update(b []byte) (interface{}, error) {
	obj := types.Kustomization{}
	err := yaml.Unmarshal([]byte(b), &obj)
	if err != nil {
		return nil, err
	}
	updated := false
	for i, image := range obj.Images {
		if image.Name == o.targetTag {
			obj.Images[i].NewTag = o.tag
			updated = true
		}
	}

	if !updated {
		obj.Images = append(obj.Images, types.Image{
			Name:   o.targetTag,
			NewTag: o.tag,
		})
	}
	return obj, nil
}

type MemcachedOverWrite struct {
}

func (o MemcachedOverWrite) Update(b []byte) (interface{}, error) {
	obj := ConfigMap{}
	err := yaml.Unmarshal([]byte(b), &obj)
	if err != nil {
		return nil, err
	}
	if _, ok := obj.Data["MEMCACHED_PREFIX"]; ok {
		obj.Data["MEMCACHED_PREFIX"] = time.Now().Format("2006-01-02T15:04:05")
	}
	return obj, nil
}
