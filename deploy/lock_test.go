package deploy

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/require"
)

func TestLockUnlock(t *testing.T) {
	// Ensure you run `KUBECONFIG=gocat-test-kubeconfig kind export kubeconfig` before running this test
	kubeconfigPath := os.Getenv("GOCAT_TEST_KUBECONFIG")
	if kubeconfigPath == "" {
		t.Skip("GOCAT_TEST_KUBECONFIG is not set")
	}

	c := NewCoordinator("default", "gocat-test")
	defer func() {
		if c, _ := c.ClientSet(); c != nil {
			if err := c.CoreV1().ConfigMaps("default").Delete(context.Background(), "gocat-test", metav1.DeleteOptions{}); err != nil {
				t.Log(err)
			}
		}
	}()

	prevKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Setenv("KUBECONFIG", prevKubeconfig)

	ctx := context.Background()

	require.NoError(t, c.Lock(ctx, "myproject1", "prod", "user1", "for deployment of revision a"))
	require.ErrorIs(t, c.Lock(ctx, "myproject1", "prod", "user1", "for deployment of revision b"), ErrAlreadyLocked)
	require.ErrorIs(t, c.Lock(ctx, "myproject1", "prod", "user2", "for deployment of revision b"), ErrAlreadyLocked)
	require.ErrorIs(t, c.Unlock(ctx, "myproject1", "prod", "user2", false), newNotAllowedToUnlockError("user2"))
	require.NoError(t, c.Unlock(ctx, "myproject1", "prod", "user2", true))
	require.ErrorIs(t, c.Unlock(ctx, "myproject1", "prod", "user1", false), ErrAlreadyUnlocked)
	require.ErrorIs(t, c.Unlock(ctx, "myproject1", "prod", "user2", false), ErrAlreadyUnlocked)

	require.NoError(t, c.Lock(ctx, "myproject1", "prod", "user2", "for deployment of revision b"))
	require.NoError(t, c.Unlock(ctx, "myproject1", "prod", "user2", false))
	require.ErrorIs(t, c.Unlock(ctx, "myproject1", "prod", "user1", false), ErrAlreadyUnlocked)
	require.ErrorIs(t, c.Unlock(ctx, "myproject1", "prod", "user2", false), ErrAlreadyUnlocked)

	for u := 3; u < 10; u++ {
		user := fmt.Sprintf("user%d", u)
		require.NoError(t, c.Lock(ctx, "myproject1", "prod", user, fmt.Sprintf("for deployment of revision c+%d", u)))
		require.NoError(t, c.Unlock(ctx, "myproject1", "prod", user, false))
	}
}

type fakeClock struct {
	now metav1.Time
}

func (f *fakeClock) Now() metav1.Time {
	return f.now
}

func TestDescribeLocks(t *testing.T) {
	// Ensure you run `KUBECONFIG=gocat-test-kubeconfig kind export kubeconfig` before running this test
	kubeconfigPath := os.Getenv("GOCAT_TEST_KUBECONFIG")
	if kubeconfigPath == "" {
		t.Skip("GOCAT_TEST_KUBECONFIG is not set")
	}

	c := NewCoordinator("default", "gocat-test")
	defer func() {
		if c, _ := c.ClientSet(); c != nil {
			if err := c.CoreV1().ConfigMaps("default").Delete(context.Background(), "gocat-test", metav1.DeleteOptions{}); err != nil {
				t.Log(err)
			}
		}
	}()

	tim, err := time.Parse(time.RFC3339, "2021-09-01T00:00:00Z")
	require.NoError(t, err)
	tim = tim.In(time.Local)
	now := metav1.Time{Time: tim}
	c.Clock = &fakeClock{
		now: now,
	}

	prevKubeconfig := os.Getenv("KUBECONFIG")
	os.Setenv("KUBECONFIG", kubeconfigPath)
	defer os.Setenv("KUBECONFIG", prevKubeconfig)

	ctx := context.Background()

	require.NoError(t, c.Lock(ctx, "myproject1", "prod", "user1", "for deployment of revision a"))

	locks, err := c.DescribeLocks(ctx)
	require.NoError(t, err)
	require.Len(t, locks, 1)

	require.Equal(t, map[string]Phase{
		"prod": {
			Locked: true,
			LockHistory: []LockHistoryItem{
				{
					User:   "user1",
					At:     now,
					Reason: "for deployment of revision a",
					Action: LockActionLock,
				},
			},
		},
	}, locks["myproject1"])

	require.NoError(t, c.Unlock(ctx, "myproject1", "prod", "user1", false))

	locks, err = c.DescribeLocks(ctx)
	require.NoError(t, err)
	require.Len(t, locks, 1)
	require.Equal(t, map[string]Phase{
		"prod": {
			Locked: false,
			LockHistory: []LockHistoryItem{
				{
					User:   "user1",
					Action: LockActionLock,
					Reason: "for deployment of revision a",
					At:     now,
				},
				{
					User:   "user1",
					Action: LockActionUnlock,
					Reason: "",
					At:     now,
				},
			},
		},
	}, locks["myproject1"])
}
