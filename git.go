package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-git/go-billy/v5"
	"github.com/go-git/go-billy/v5/memfs"
	"github.com/go-git/go-billy/v5/osfs"
	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/config"
	"github.com/go-git/go-git/v5/plumbing"
	"github.com/go-git/go-git/v5/plumbing/cache"
	"github.com/go-git/go-git/v5/plumbing/object"
	"github.com/go-git/go-git/v5/plumbing/transport"
	"github.com/go-git/go-git/v5/plumbing/transport/http"
	"github.com/go-git/go-git/v5/storage"
	"github.com/go-git/go-git/v5/storage/filesystem"
	"github.com/go-git/go-git/v5/storage/memory"
	"golang.org/x/xerrors"
	yaml "gopkg.in/yaml.v2"
	"sigs.k8s.io/kustomize/api/types"
)

// GitOperator is our wrapper aroud go-git to do GitOps, and
// tagging commits to correlate them with the container image tags.
type GitOperator struct {
	auth          transport.AuthMethod
	repo          string
	repository    *git.Repository
	username      string
	defaultBranch string
	// gitRoot is the root of the local git repository, used to
	// clone and checkout the remote repository that contains the gitops config
	// or the kustomize config we are going to modify.
	// If empty, we will use in-memory filesystem.
	gitRoot string
}

func CreateGitOperatorInstance(username, token, repo, defaultBranch, gitRoot string) (g GitOperator) {
	g.auth = &http.BasicAuth{
		Username: username, // yes, this can be anything except an empty string
		Password: token,
	}
	g.repo = repo
	g.username = username
	g.defaultBranch = defaultBranch
	g.gitRoot = gitRoot
	if err := g.Clone(); err != nil {
		fmt.Println("[ERROR] Failed to Clone: ", xerrors.New(err.Error()))
	}
	return
}

func (g *GitOperator) Clone() error {
	var (
		storage storage.Storer
		fs      billy.Filesystem
	)
	if g.gitRoot != "" {
		fs = osfs.New(g.gitRoot)
		storage = filesystem.NewStorage(
			osfs.New(filepath.Join(g.gitRoot, ".git")),
			cache.NewObjectLRUDefault(),
		)
	} else {
		storage = memory.NewStorage()
		fs = memfs.New()
	}
	r, err := git.Clone(storage, fs, &git.CloneOptions{
		URL:  g.Repo(),
		Auth: g.auth,
	})
	g.repository = r

	if err != nil {
		fmt.Println("[ERROR] ", err)
	}
	return err
}

func (g GitOperator) Repo() string {
	return g.repo
}

func (g GitOperator) DeleteBranch(branch string) (err error) {
	return g.repository.Storer.RemoveReference(plumbing.ReferenceName(branch))
}

func (g GitOperator) PushDockerImageTag(id string, phase DeployPhase, tag string, targetTag string) (branch string, err error) {
	branch = fmt.Sprintf("bot/docker-image-tag-%s-%s-%s", id, phase.Name, tag)

	w, err := g.createAndCheckoutNewBranch(branch)
	if err != nil {
		return "", err
	}

	err = g.commit(w, phase.Path, KustomizationOverWrite{tag, targetTag})
	if err != nil {
		fmt.Println("[ERROR] Failed to Marshal kustomize.yaml: ", xerrors.New(err.Error()))
		return
	}

	err = g.commit(w, strings.Replace(phase.Path, "kustomization.yaml", "configmap.yaml", -1), MemcachedOverWrite{})
	if err != nil {
		fmt.Println("[ERROR] Failed to Write MEMCACHED_PREFIX \\n: ", xerrors.New(err.Error()))
		return
	}

	err = g.verify(w)
	if err != nil {
		return
	}

	hash, _ := w.Commit(
		fmt.Sprintf("Change docker image tag. target: %s, phase: %s, tag: %s.", phase.Path, phase.Name, tag),
		&git.CommitOptions{
			Author: &object.Signature{
				Name:  g.username,
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

func (g GitOperator) createAndCheckoutNewBranch(branch string) (*git.Worktree, error) {
	if err := g.DeleteBranch(branch); err != nil {
		fmt.Println("[ERROR] Failed to DeleteBranch: ", xerrors.New(err.Error()))
	}

	// checkout

	w, err := g.repository.Worktree()
	if err != nil {
		return nil, err
	}
	refName := plumbing.Master
	if g.defaultBranch != "" {
		refName = plumbing.ReferenceName(g.defaultBranch)
	}

	err = w.Checkout(&git.CheckoutOptions{
		Create: false,
		Branch: refName,
	})
	if err != nil {
		fmt.Println("[ERROR] Failed to Checkout master: ", xerrors.New(err.Error()))
		return nil, err
	}

	err = w.Pull(&git.PullOptions{RemoteName: "origin", Auth: g.auth})
	if err != nil && err != git.NoErrAlreadyUpToDate {
		fmt.Println("[ERROR] Failed to Pull origin/master: ", xerrors.New(err.Error()))
		fmt.Println("[INFO] Running Clone to see if it fixes the issue")
		if err := g.Clone(); err != nil {
			fmt.Println("[ERROR] Failed to Clone: ", xerrors.New(err.Error()))
		}
	}
	err = nil

	err = w.Checkout(&git.CheckoutOptions{
		Create: true,
		Branch: plumbing.ReferenceName(branch),
	})
	if err != nil {
		fmt.Println("[ERROR] Failed to Checkout workbranch: ", xerrors.New(err.Error()))
		return nil, err
	}

	return w, nil
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

func (g GitOperator) verify(w *git.Worktree) (err error) {
	status, err := w.Status()
	if err != nil {
		fmt.Println("[ERROR] Failed to get status: ", xerrors.New(err.Error()))
		return
	}

	for path, status := range status {
		if status.Staging != git.Modified {
			fmt.Printf("[ERROR] There are some extra file updates. File: %v %s", status, path)
			return xerrors.New("There are some extra file updates")
		}
	}
	return nil
}

func (g GitOperator) commit(w *git.Worktree, targetFilePath string, o OverWrite) (err error) {
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
	b, err := io.ReadAll(file)
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
