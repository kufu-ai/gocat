package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/slack-go/slack"
	"github.com/slack-go/slack/slackevents"
	"github.com/slack-go/slack/slacktest"
	"github.com/stretchr/testify/require"
	"github.com/zaiminc/gocat/deploy"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var project1ConfigMap = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "myproject1",
		Labels: map[string]string{
			"gocat.zaim.net/configmap-type": "project",
		},
	},
	Data: map[string]string{
		"Phases": `- name: production
- name: staging
`,
	},
}

var project2ConfigMap = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "myproject2",
		Labels: map[string]string{
			"gocat.zaim.net/configmap-type": "project",
		},
	},
	Data: map[string]string{
		"Phases": `- name: production
- name: staging
`,
	},
}

const (
	user1SlackDisplayName = "user1"
	user1GitHubLogin      = "user1"
)

var githubuserMappingConfigMap = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "githubuser-mapping",
		Labels: map[string]string{
			"gocat.zaim.net/configmap-type": "githubuser-mapping",
		},
	},
	Data: map[string]string{
		user1SlackDisplayName: user1GitHubLogin,
	},
}

var rolebindingConfigMap = corev1.ConfigMap{
	ObjectMeta: metav1.ObjectMeta{
		Name: "rolebinding",
		Labels: map[string]string{
			"gocat.zaim.net/configmap-type": "rolebinding",
		},
	},
	Data: map[string]string{
		"Developer": fmt.Sprintf(`%s
user2
user4
`, user1SlackDisplayName),
		"Admin": "user4",
	},
}

var configMaps = []corev1.ConfigMap{
	project1ConfigMap,
	project2ConfigMap,
	githubuserMappingConfigMap,
	rolebindingConfigMap,
}

// TestSlackLockUnlock tests the lock and unlock commands against
// a pre-configured Kubernetes cluster, fake Slack API, and fake GitHub API.
// It verifies that the lock and unlock commands work as expected, by sending
// messages to the fake Slack API and checking the messages that are sent from gocat.
func TestSlackLockUnlock(t *testing.T) {
	if testing.Short() {
		t.Skip()
	}

	var k = &deploy.Kubernetes{}
	clientset, err := k.ClientSet()
	require.NoError(t, err)

	setupConfigMaps(t, clientset, configMaps...)
	setupNamespace(t, clientset, "gocat")

	messages := make(chan message, 10)
	nextMessage := func() message {
		select {
		case <-time.After(5 * time.Second):
			t.Fatal("timed out waiting for message")
		case m := <-messages:
			return m
		}
		return message{}
	}
	ts := slacktest.NewTestServer(func(c slacktest.Customize) {
		c.Handle("/conversations.history", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte(``)); err != nil {
				t.Logf("failed to write response: %v", err)
			}
		})
		// List users
		c.Handle("/users.list", func(w http.ResponseWriter, r *http.Request) {
			if _, err := w.Write([]byte(`{"ok": true, "members": [{"id": "U1234", "name": "User 1", "profile": {"display_name": "user1"}}, {"id": "U1235", "name": "User 2", "profile": {"display_name": "user2"}}, {"id": "U1237", "name": "User 4", "profile": {"display_name": "user4"}}]}`)); err != nil {
				t.Logf("failed to write response: %v", err)
			}
		})
		// Message posted to the channel
		c.Handle("/chat.postMessage", func(w http.ResponseWriter, r *http.Request) {
			m := message{
				channel: r.FormValue("channel"),
				Blocks:  []block{},
			}
			blocksValue := r.FormValue("blocks")
			if err := json.Unmarshal([]byte(blocksValue), &m.Blocks); err != nil {
				t.Logf("failed to unmarshal blocks: %v", err)
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			messages <- m
			if _, err := w.Write([]byte(`{"ok": true}`)); err != nil {
				t.Logf("failed to write response: %v", err)
			}
		})
	})
	ts.Start()
	defer ts.Stop()

	// ghts is a test HTTP server that serves GitHub API v2 (GraphQL) responses.
	ghts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch r.URL.Path {
		case "/graphql":
			body, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			t.Logf("request body: %s", string(body))

			// Returns ids and logins of the users who are members of the organization

			if _, err := w.Write([]byte(`{"data": {"organization": {"membersWithRole": {"nodes": [{"id": "U1234", "login": "user1"}]}}}}`)); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		default:
			http.Error(w, "not found", http.StatusNotFound)
		}
	}))
	defer ghts.Close()

	s := slack.New("no-need-to-use-a-token-because-we-are-using-a-fake-server",
		slack.OptionAPIURL(ts.GetAPIURL()),
	)

	// You usually do:
	//   config, err := InitConfig()
	// But for this test, we'll just set the config manually.
	config := &CatConfig{
		ManifestRepository:    "no-need-to-use-a-token-because-this-test-does-not-use-the-manifest-repository",
		ManifestRepositoryOrg: "no-need-to-use-a-token-because-this-test-does-not-use-the-manifest-repository",
		GitHubUserName:        "no-need-for-github-user-name-because-this-test-does-not-use-the-manifest-repository",
		GitHubAccessToken:     "no-need-for-github-access-token-because-this-test-does-not-use-the-manifest-repository",
		GitHubDefaultBranch:   "no-need-for-github-default-branch-because-this-test-does-not-use-the-manifest-repository",
	}

	gh := CreateGitHubInstance(
		ghts.URL+"/graphql",
		config.GitHubAccessToken,
		config.ManifestRepositoryOrg,
		config.ManifestRepositoryName,
		config.GitHubDefaultBranch,
	)
	git := CreateGitOperatorInstance(
		config.GitHubUserName,
		config.GitHubAccessToken,
		config.ManifestRepository,
		config.GitHubDefaultBranch,
		os.Getenv("GOCAT_GITROOT"),
	)

	userList := UserList{github: gh, slackClient: s}
	// Set LOCAL so that NewProjectList uses the local kubeconfig
	local := os.Getenv("LOCAL")
	os.Setenv("LOCAL", "true")
	defer os.Setenv("LOCAL", local)
	projectList := NewProjectList()
	interactorContext := InteractorContext{
		projectList: &projectList,
		userList:    &userList,
		github:      gh,
		git:         git,
		client:      s,
		config:      *config,
	}
	interactorFactory := NewInteractorFactory(interactorContext)

	var l = &SlackListener{
		client:            s,
		verificationToken: "token",
		projectList:       &projectList,
		userList:          &userList,
		interactorFactory: &interactorFactory,
		mu:                &sync.Mutex{},
	}

	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1234",
		Channel: "C1234",
		Text:    "lock myproject1 production for deployment of revision a",
	}))
	require.Equal(t, "Locked myproject1 production", nextMessage().Text())

	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1234",
		Channel: "C1234",
		Text:    "lock myproject1 production for deployment of revision a",
	}))
	require.Equal(t, "deployment is already locked", nextMessage().Text())

	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1234",
		Channel: "C1234",
		Text:    "unlock myproject1 production",
	}))
	require.Equal(t, "Unlocked myproject1 production", nextMessage().Text())

	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1234",
		Channel: "C1234",
		Text:    "unlock myproject1 production",
	}))
	require.Equal(t, "deployment is already unlocked", nextMessage().Text())

	//
	// We assume that any users who aren't visible from the Slack API are not allowed to lock/unlock projects.
	//

	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1236",
		Channel: "C1234",
		Text:    "lock myproject1 production for deployment of revision a",
	}))
	require.Equal(t, "you are not allowed to lock/unlock projects: slack user id \"\"", nextMessage().Text())

	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1236",
		Channel: "C1234",
		Text:    "unlock myproject1 production",
	}))
	require.Equal(t, "you are not allowed to lock/unlock projects: slack user id \"\"", nextMessage().Text())

	// User 2 is a developer so can lock the project
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "lock myproject1 production for deployment of revision a",
	}))
	require.Equal(t, "Locked myproject1 production", nextMessage().Text())

	// Describe locks
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "describe locks",
	}))
	require.Equal(t, `myproject1
  production: Locked (by user2, for deployment of revision a)
`, nextMessage().Text())

	// Lock staging
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "lock myproject1 staging for deployment of revision b",
	}))
	require.Equal(t, "Locked myproject1 staging", nextMessage().Text())

	// Describe locks
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "describe locks",
	}))
	require.Equal(t, `myproject1
  production: Locked (by user2, for deployment of revision a)
  staging: Locked (by user2, for deployment of revision b)
`, nextMessage().Text())

	// Lock project 2 staging
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "lock myproject2 staging for deployment of revision c",
	}))
	require.Equal(t, "Locked myproject2 staging", nextMessage().Text())

	// Describe locks
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "describe locks",
	}))
	require.Equal(t, `myproject1
  production: Locked (by user2, for deployment of revision a)
  staging: Locked (by user2, for deployment of revision b)
myproject2
  staging: Locked (by user2, for deployment of revision c)
`, nextMessage().Text())

	// User 1 is a developer so cannot unlock the project forcefully
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1234",
		Channel: "C1234",
		Text:    "unlock myproject1 production",
	}))
	require.Equal(t, "user user1 is not allowed to unlock this project", nextMessage().Text())

	// User 4 is an admin and can unlock the project forcefully
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1237",
		Channel: "C1234",
		Text:    "unlock myproject1 production",
	}))
	require.Equal(t, "Unlocked myproject1 production", nextMessage().Text())

	// Describe locks
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "describe locks",
	}))
	require.Equal(t, `myproject1
  staging: Locked (by user2, for deployment of revision b)
myproject2
  staging: Locked (by user2, for deployment of revision c)
`, nextMessage().Text())

	// Unlock project 2 staging
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "unlock myproject2 staging",
	}))
	require.Equal(t, "Unlocked myproject2 staging", nextMessage().Text())

	// Describe locks
	require.NoError(t, l.handleMessageEvent(&slackevents.AppMentionEvent{
		User:    "U1235",
		Channel: "C1234",
		Text:    "describe locks",
	}))
	require.Equal(t, `myproject1
  staging: Locked (by user2, for deployment of revision b)
myproject2
`, nextMessage().Text())
}

// Message is a message posted to the fake Slack API's chat.postMessage endpoint
type message struct {
	Blocks  []block
	channel string
}

func (m message) Text() string {
	for _, b := range m.Blocks {
		if b.Type == "section" {
			return b.Text.Text
		}
	}
	return ""
}

type block struct {
	// For example, "section"
	Type string `json:"type"`
	Text text   `json:"text"`
}

type text struct {
	// For example, "mrkdwn"
	Type string `json:"type"`
	Text string `json:"text"`
}

func setupNamespace(t *testing.T, clientset kubernetes.Interface, name string) {
	ns := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
	}
	_, err := clientset.CoreV1().Namespaces().Create(context.Background(), ns, metav1.CreateOptions{})
	if kerrors.IsAlreadyExists(err) {
		_, err = clientset.CoreV1().Namespaces().Update(context.Background(), ns, metav1.UpdateOptions{})
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		if err := clientset.CoreV1().Namespaces().Delete(context.Background(), ns.Name, metav1.DeleteOptions{}); err != nil {
			t.Logf("failed to delete namespace %s: %v", ns.Name, err)
		}
	})
}

func setupConfigMaps(t *testing.T, clientset kubernetes.Interface, configMaps ...corev1.ConfigMap) {
	for _, cm := range configMaps {
		cm := cm
		_, err := clientset.CoreV1().ConfigMaps("default").Create(context.Background(), &cm, metav1.CreateOptions{})
		if kerrors.IsAlreadyExists(err) {
			_, err = clientset.CoreV1().ConfigMaps("default").Update(context.Background(), &cm, metav1.UpdateOptions{})
			require.NoError(t, err)
		}
		t.Cleanup(func() {
			if err := clientset.CoreV1().ConfigMaps("default").Delete(context.Background(), cm.Name, metav1.DeleteOptions{}); err != nil {
				t.Logf("failed to delete config map %s: %v", cm.Name, err)
			}
		})
	}
}
