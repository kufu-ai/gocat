package slackcmd

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

var (
	validProjects   = []string{"myproject1", "myproject-2"}
	invalidProjects = []string{"myproject#3", "myproject_4"}
	validEnvs       = []string{"staging", "production", "sandbox", "stg", "pro", "prd"}
	invalidENvs     = []string{"stg1", "pro1", "prd1", "prod", "test"}
)

func invalidPatternErr(text string) string {
	return fmt.Sprintf("invalid command %q: valid pattern is `help|ls|reload`, valid pattern is `lock|unlock <project> <env> [for <reason>]`, valid pattern is `describe locks`, valid pattern is `deploy <project> <env> [branch]|deploy <env>`", text)
}

func normalizedEnv(env string) string {
	switch env {
	case "stg":
		return "staging"
	case "pro", "prd":
		return "production"
	default:
		return env
	}
}

func TestParse(t *testing.T) {
	type test struct {
		name   string
		text   string
		want   Command
		errMsg string
	}

	var tests = []test{}

	tests = append(tests, test{
		name: "help",
		text: "<@U0LAN0Z89> help",
		want: &Help{},
	})
	tests = append(tests, test{
		name: "list projects",
		text: "<@U0LAN0Z89> ls",
		want: &ListProjects{},
	})
	tests = append(tests, test{
		name: "reload",
		text: "<@U0LAN0Z89> reload",
		want: &Reload{},
	})

	for i, p := range validProjects {
		for j, e := range validEnvs {
			tests = append(tests, test{
				name: fmt.Sprintf("lock with valid project %d and env %d", i, j),
				text: fmt.Sprintf("lock %s %s for deployment of revision a", p, e),
				want: &Lock{Project: p, Env: normalizedEnv(e), Reason: "deployment of revision a"},
			})
		}
	}

	for i, p := range invalidProjects {
		for j, e := range invalidENvs {
			tests = append(tests, test{
				name:   fmt.Sprintf("lock with invalid project %d and env %d", i, j),
				text:   fmt.Sprintf("lock %s %s for deployment of revision a", p, e),
				errMsg: invalidPatternErr(fmt.Sprintf("lock %s %s for deployment of revision a", p, e)),
			})
		}
	}

	for i, p := range validProjects {
		for j, e := range validEnvs {
			tests = append(tests, test{
				name: fmt.Sprintf("unlock with valid project %d and env %d", i, j),
				text: fmt.Sprintf("unlock %s %s", p, e),
				want: &Unlock{Project: p, Env: normalizedEnv(e)},
			})
		}
	}

	for i, p := range invalidProjects {
		for j, e := range invalidENvs {
			tests = append(tests, test{
				name:   fmt.Sprintf("unlock with invalid project %d and env %d", i, j),
				text:   fmt.Sprintf("unlock %s %s", p, e),
				errMsg: invalidPatternErr(fmt.Sprintf("unlock %s %s", p, e)),
			})
		}
	}

	tests = append(tests, test{
		name:   "unlock has redundant reason",
		text:   "unlock myproject1 production for deployment of revision a",
		errMsg: fmt.Sprintf("invalid command %q: unlock command does not accept reason", "unlock myproject1 production for deployment of revision a"),
	})

	tests = append(tests, test{
		name:   "lock missing reason",
		text:   "lock myproject1 production",
		errMsg: fmt.Sprintf("invalid command %q: lock command requires reason", "lock myproject1 production"),
	})

	tests = append(tests, test{
		name:   "unknown command",
		text:   "unknown myproject1 production for deployment of revision a",
		errMsg: invalidPatternErr("unknown myproject1 production for deployment of revision a"),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.text)
			assert.Equal(t, tt.want, got, "result")
			var errMsg string
			if err != nil {
				errMsg = err.Error()
			}
			assert.Equal(t, tt.errMsg, errMsg, "error")
		})
	}

	t.Run("describe locks", func(t *testing.T) {
		got, err := Parse("describe locks")
		assert.NoError(t, err)
		assert.IsType(t, &DescribeLocks{}, got)
	})

	t.Run("deploy project default branch", func(t *testing.T) {
		got, err := Parse("<@U0LAN0Z89> deploy myproject1 stg")
		assert.NoError(t, err)
		assert.Equal(t, &Deploy{Project: "myproject1", Env: "staging"}, got)
	})

	t.Run("deploy project selected branch", func(t *testing.T) {
		got, err := Parse("<@U0LAN0Z89> deploy myproject1 prd branch")
		assert.NoError(t, err)
		assert.Equal(t, &DeployBranchList{Project: "myproject1", Env: "production"}, got)
	})

	t.Run("select deploy target", func(t *testing.T) {
		got, err := Parse("<@U0LAN0Z89> deploy pro")
		assert.NoError(t, err)
		assert.Equal(t, &DeployTargetSelection{Env: "production"}, got)
	})
}
