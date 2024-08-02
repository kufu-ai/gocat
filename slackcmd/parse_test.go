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

func TestParse(t *testing.T) {
	type test struct {
		name string
		text string
		want Command
		err  error
	}

	var tests = []test{}

	for i, p := range validProjects {
		for j, e := range validEnvs {
			tests = append(tests, test{
				name: fmt.Sprintf("lock with valid project %d and env %d", i, j),
				text: fmt.Sprintf("lock %s %s for deployment of revision a", p, e),
				want: &Lock{Project: p, Env: e, Reason: "deployment of revision a"},
			})
		}
	}

	for i, p := range invalidProjects {
		for j, e := range invalidENvs {
			tests = append(tests, test{
				name: fmt.Sprintf("lock with invalid project %d and env %d", i, j),
				text: fmt.Sprintf("lock %s %s for deployment of revision a", p, e),
				err:  fmt.Errorf("invalid command %q: valid pattern is 'lock|unlock <project> <env> [for <reason>]", fmt.Sprintf("lock %s %s for deployment of revision a", p, e)),
			})
		}
	}

	for i, p := range validProjects {
		for j, e := range validEnvs {
			tests = append(tests, test{
				name: fmt.Sprintf("unlock with valid project %d and env %d", i, j),
				text: fmt.Sprintf("unlock %s %s", p, e),
				want: &Unlock{Project: p, Env: e},
			})
		}
	}

	for i, p := range invalidProjects {
		for j, e := range invalidENvs {
			tests = append(tests, test{
				name: fmt.Sprintf("unlock with invalid project %d and env %d", i, j),
				text: fmt.Sprintf("unlock %s %s", p, e),
				err:  fmt.Errorf("invalid command %q: valid pattern is 'lock|unlock <project> <env> [for <reason>]", fmt.Sprintf("unlock %s %s", p, e)),
			})
		}
	}

	tests = append(tests, test{
		name: "unlock has redundant reason",
		text: "unlock myproject1 production for deployment of revision a",
		err:  fmt.Errorf("invalid command %q: unlock command does not accept reason", "unlock myproject1 production for deployment of revision a"),
	})

	tests = append(tests, test{
		name: "lock missing reason",
		text: "lock myproject1 production",
		err:  fmt.Errorf("invalid command %q: lock command requires reason", "lock myproject1 production"),
	})

	tests = append(tests, test{
		name: "unknown command",
		text: "unknown myproject1 production for deployment of revision a",
		err:  fmt.Errorf("invalid command %q: valid pattern is 'lock|unlock <project> <env> [for <reason>]", "unknown myproject1 production for deployment of revision a"),
	})

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.text)
			assert.Equal(t, tt.want, got, "result")
			assert.Equal(t, tt.err, err, "error")
		})
	}
}
