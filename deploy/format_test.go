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
	}

	require.Equal(t, `myproject1
  prod: Locked (by user1, for for deployment of revision a)
`, FormatProjectDescs(projects))

}
