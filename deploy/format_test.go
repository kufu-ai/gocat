package deploy

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFormatProjectDescs(t *testing.T) {
	projects := []ProjectDesc{
		{
			Name: "myproject1",
			Phases: []PhaseDesc{
				{
					Name: "prod",
					Phase: Phase{
						Locked: true,
						LockHistory: []LockHistoryItem{
							{
								User:   "user1",
								Reason: "for deployment of revision a",
							},
						},
					},
				},
			},
		},
		{
			Name: "myproject1-api",
			Phases: []PhaseDesc{
				{
					Name: "staging",
					Phase: Phase{
						Locked: true,
						LockHistory: []LockHistoryItem{
							{
								User:   "user2",
								Reason: "for deployment of revision b",
							},
						},
					},
				},
			},
		},
		{
			Name: "myproject2",
			Phases: []PhaseDesc{
				{
					Name: "prod",
					Phase: Phase{
						Locked: true,
						// There should be 1 or more LockHistoryItem if Locked is true.
						// But we intentionally omit it here to test the case where
						// some bug causes LockHistoryItem to be empty.
					},
				},
			},
		},
	}

	require.Equal(t, `myproject1
  prod: Locked (by user1, for for deployment of revision a)
myproject1-api
  staging: Locked (by user2, for for deployment of revision b)
myproject2
  prod: Locked
`, FormatProjectDescs(projects))
}
