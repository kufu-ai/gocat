package deploy

import (
	"context"
	"fmt"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Coordinator provides a way to lock and unlock deployments made via gocat.
// The lock is used to prevent multiple deployments from happening at the same time.
//
// The lock information is stored in a Kubernetes ConfigMap managed by the Coordinaator.
// The ConfigMap is created in the same namespace as gocat.
//
// The lock can be acquired by calling the Lock method. The lock is released by calling the Unlock method.
// Each method takes a user ID as an argument. The user ID is used to identify the user who acquired the lock,
// and also to verify that the user releasing the lock is the same user who acquired it, or has the necessary
// permissions to release it.
//
// The methods also take a project name and an environment name as arguments. These are used to identify the
// deployment that the lock is associated with.
type Coordinator struct {
	// Namespace is the namespace in which the ConfigMap is created.
	Namespace string

	// ConfigMapName is the name of the ConfigMap.
	ConfigMapName string

	Clock
	Kubernetes
}

type Clock interface {
	Now() metav1.Time
}

type systemClock struct{}

func (systemClock) Now() metav1.Time {
	return metav1.Now()
}

func NewCoordinator(ns, configMap string) *Coordinator {
	return &Coordinator{
		Namespace:     ns,
		ConfigMapName: configMap,
		Clock:         systemClock{},
	}
}

var ErrAlreadyLocked = fmt.Errorf("deployment is already locked")
var ErrAlreadyUnlocked = fmt.Errorf("deployment is already unlocked")

const (
	MaxConfigMapUpdateRetries = 3
)

// Lock acquires a lock for the given project and environment.
//
// Under the hood, this retries to update the ConfigMap if the update fails due to a conflict.
func (c *Coordinator) Lock(ctx context.Context, project, environment, user, reason string) error {
	var retried int
	for {
		err := c.lock(ctx, project, environment, user, reason)
		if err == nil {
			return nil
		}

		if kerrors.IsConflict(err) {
			if retried >= MaxConfigMapUpdateRetries {
				return fmt.Errorf("unable to acquire lock after %d retries: %w", MaxConfigMapUpdateRetries, err)
			}

			retried++
			continue
		} else {
			return err
		}
	}
}

func (c *Coordinator) lock(ctx context.Context, project, environment, user, reason string) error {
	configMap, err := c.getOrCreateConfigMap(ctx)
	if err != nil {
		return fmt.Errorf("unable to get or create configmap: %w", err)
	}

	key := c.configMapKey(project, environment)
	value, err := strToConfigMapValue(configMap.Data[key])
	if err != nil {
		return fmt.Errorf("unable to unmarshal str into value: %w", err)
	}

	if value.Locked {
		return ErrAlreadyLocked
	}

	if n := len(value.LockHistory); n >= MaxHistoryItems {
		value.LockHistory = value.LockHistory[n-MaxHistoryItems+1:]
	}

	value.LockHistory = append(value.LockHistory, LockHistoryItem{
		User:   user,
		Action: LockActionLock,
		At:     c.Now(),
		Reason: reason,
	})

	value.Locked = true

	configMap.Data[key], err = configMapValueToStr(value)
	if err != nil {
		return err
	}

	_, err = c.updateConfigMap(ctx, configMap)
	if err != nil {
		return err
	}

	return nil
}

// Unlock releases the lock for the given project and environment.
//
// The lock can be released by the same user who acquired it, or by anyone if the force argument is true.
//
// Under the hood, this retries to update the ConfigMap if the update fails due to a conflict.
func (c *Coordinator) Unlock(ctx context.Context, project, environment, user string, force bool) error {
	var retried int
	for {
		err := c.unlock(ctx, project, environment, user, force)
		if err == nil {
			return nil
		}

		if kerrors.IsConflict(err) {
			if retried >= MaxConfigMapUpdateRetries {
				return fmt.Errorf("unable to release lock after %d retries: %w", MaxConfigMapUpdateRetries, err)
			}

			retried++
			continue
		} else {
			return err
		}
	}
}

func (c *Coordinator) unlock(ctx context.Context, project, environment, user string, force bool) error {
	configMap, err := c.getOrCreateConfigMap(ctx)
	if err != nil {
		return err
	}

	key := c.configMapKey(project, environment)
	value, err := strToConfigMapValue(configMap.Data[key])
	if err != nil {
		return err
	}

	if !value.Locked {
		return ErrAlreadyUnlocked
	}

	if force {
		value.Locked = false
	} else {
		if len(value.LockHistory) == 0 || value.LockHistory[len(value.LockHistory)-1].User != user {
			return newNotAllowedToUnlockError(user)
		}

		if n := len(value.LockHistory); n >= MaxHistoryItems {
			value.LockHistory = value.LockHistory[n-MaxHistoryItems+1:]
		}

		value.Locked = false
		value.LockHistory = append(value.LockHistory, LockHistoryItem{
			User:   user,
			Action: LockActionUnlock,
			At:     c.Now(),
		})
	}

	configMap.Data[key], err = configMapValueToStr(value)
	if err != nil {
		return err
	}

	_, err = c.updateConfigMap(ctx, configMap)
	if err != nil {
		return err
	}

	return nil
}

// DescribeLocks returns a map of project names to a map of environment names to the lock information.
func (c *Coordinator) DescribeLocks(ctx context.Context) (map[string]map[string]Phase, error) {
	configMap, err := c.getOrCreateConfigMap(ctx)
	if err != nil {
		return nil, err
	}

	locks := make(map[string]map[string]Phase)
	for k, v := range configMap.Data {
		value, err := strToConfigMapValue(v)
		if err != nil {
			return nil, err
		}

		project, environment := splitConfigMapKey(k)
		if locks[project] == nil {
			locks[project] = make(map[string]Phase)
		}

		locks[project][environment] = value
	}

	return locks, nil
}

type NotAllowedTounlockError struct {
	User string
}

func (e NotAllowedTounlockError) Error() string {
	return fmt.Sprintf("user %s is not allowed to unlock this project", e.User)
}

func newNotAllowedToUnlockError(user string) NotAllowedTounlockError {
	return NotAllowedTounlockError{User: user}
}
