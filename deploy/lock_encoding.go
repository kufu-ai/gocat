package deploy

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type keysAndValuesEncoding struct {
	data map[string]string
}

func (e *keysAndValuesEncoding) lock(project, environment, user, reason string, at metav1.Time) error {
	key := configMapKey(project, environment)
	value, err := strToConfigMapValue(e.data[key])
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
		At:     at,
		Reason: reason,
	})

	value.Locked = true

	e.data[key], err = configMapValueToStr(value)
	if err != nil {
		return err
	}

	return nil
}

func (e *keysAndValuesEncoding) unlock(project, environment, user string, force bool, at metav1.Time) error {
	key := configMapKey(project, environment)
	value, err := strToConfigMapValue(e.data[key])
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
			At:     at,
		})
	}

	e.data[key], err = configMapValueToStr(value)
	if err != nil {
		return err
	}

	return nil
}

func (e *keysAndValuesEncoding) describeLocks(projectFilter, phaseFilter string) (map[string]map[string]Phase, error) {
	locks := make(map[string]map[string]Phase)
	for k, v := range e.data {
		value, err := strToConfigMapValue(v)
		if err != nil {
			return nil, err
		}

		project, environment := splitConfigMapKey(k)

		if projectFilter != "" && project != projectFilter {
			continue
		}

		if phaseFilter != "" && environment != phaseFilter {
			continue
		}

		if locks[project] == nil {
			locks[project] = make(map[string]Phase)
		}

		locks[project][environment] = value
	}

	return locks, nil
}
