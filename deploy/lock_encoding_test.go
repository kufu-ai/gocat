package deploy

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKeysAndValuesEncoding(t *testing.T) {
	data := map[string]string{}
	enc := &keysAndValuesEncoding{
		data: data,
	}

	now, err := time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
	require.NoError(t, err)

	kNow := metav1.NewTime(now.Local())

	require.NoError(t, enc.lock("myproject1", "prod", "user1", "for deployment of revision 1a", kNow))
	require.ErrorIs(t, enc.lock("myproject1", "prod", "user1", "for deployment of revision 1b", kNow), ErrAlreadyLocked)
	require.ErrorIs(t, enc.lock("myproject1", "prod", "user2", "for deployment of revision 1b", kNow), ErrAlreadyLocked)

	require.ErrorIs(t, enc.unlock("myproject2", "prod", "user1", false, kNow), ErrAlreadyUnlocked)
	require.NoError(t, enc.lock("myproject2", "prod", "user1", "for deployment of revision 2a", kNow))

	require.NoError(t, enc.lock("myproject2-api", "prod", "user1", "for deployment of revision 3a", kNow))

	locks, err := enc.describeLocks("", "")
	require.NoError(t, err)

	assert.Equal(t, map[string]map[string]Phase{
		"myproject1": {
			"prod": {
				Locked: true,
				LockHistory: []LockHistoryItem{
					{
						User:   "user1",
						Action: LockActionLock,
						At:     kNow,
						Reason: "for deployment of revision 1a",
					},
				},
			},
		},
		"myproject2": {
			"prod": {
				Locked: true,
				LockHistory: []LockHistoryItem{
					{
						User:   "user1",
						Action: LockActionLock,
						At:     kNow,
						Reason: "for deployment of revision 2a",
					},
				},
			},
		},
		"myproject2-api": {
			"prod": {
				Locked: true,
				LockHistory: []LockHistoryItem{
					{
						User:   "user1",
						Action: LockActionLock,
						At:     kNow,
						Reason: "for deployment of revision 3a",
					},
				},
			},
		},
	}, locks)

	assert.Equal(t, map[string]string{
		"myproject1-prod":     `{"locked":true,"lockHistory":[{"user":"user1","action":"lock","at":"2021-09-01T00:00:00Z","reason":"for deployment of revision 1a"}]}`,
		"myproject2-prod":     `{"locked":true,"lockHistory":[{"user":"user1","action":"lock","at":"2021-09-01T00:00:00Z","reason":"for deployment of revision 2a"}]}`,
		"myproject2-api-prod": `{"locked":true,"lockHistory":[{"user":"user1","action":"lock","at":"2021-09-01T00:00:00Z","reason":"for deployment of revision 3a"}]}`,
	}, data)
}
