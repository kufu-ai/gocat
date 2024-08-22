package deploy

import (
	"context"
	"encoding/json"
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Phase is a type to represent a phase in a project.
//
// Each phase can be locked by a user to prevent multiple deployments from happening at the same time.
// On locking and unlocking, the user ID and the reason for the action are recorded in the history.
type Phase struct {
	Locked      bool              `json:"locked"`
	LockHistory []LockHistoryItem `json:"lockHistory"`
}

type LockHistoryItem struct {
	User   string      `json:"user"`
	Action LockAction  `json:"action"`
	At     metav1.Time `json:"at"`
	Reason string      `json:"reason"`
}

type LockAction string

const (
	LockActionLock   LockAction = "lock"
	LockActionUnlock LockAction = "unlock"

	MaxHistoryItems = 3
)

func configMapValueToStr(value Phase) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func strToConfigMapValue(data string) (Phase, error) {
	if data == "" {
		return Phase{}, nil
	}

	var value Phase
	err := json.Unmarshal([]byte(data), &value)
	if err != nil {
		return Phase{}, err
	}

	return value, nil
}

const (
	Sep = "-"
)

// configMapKey is a helper function that returns the key within the ConfigMap for the project and the environment
// which is either locked or unlocked.
func (c *Coordinator) configMapKey(project, environment string) string {
	return project + Sep + environment
}

func splitConfigMapKey(key string) (string, string) {
	parts := strings.Split(key, Sep)
	return parts[0], parts[1]
}

func (c *Coordinator) getOrCreateConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	configMap, err := c.getConfigMap(ctx)
	if err != nil {
		configMap, err = c.createConfigMap(ctx)
		if err != nil {
			return nil, err
		}
	}

	return configMap, nil
}

// getConfigMap creates a Kubernetes API client, and use it to retrieve the ConfigMap.
func (c *Coordinator) getConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	clientset, err := c.ClientSet()
	if err != nil {
		return nil, err
	}

	configMap, err := clientset.CoreV1().ConfigMaps(c.Namespace).Get(ctx, c.ConfigMapName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	return configMap, nil
}

func (c *Coordinator) createConfigMap(ctx context.Context) (*corev1.ConfigMap, error) {
	clientset, err := c.ClientSet()
	if err != nil {
		return nil, err
	}

	configMap := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name: c.ConfigMapName,
		},
	}

	configMap, err = clientset.CoreV1().ConfigMaps(c.Namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	if configMap.Data == nil {
		configMap.Data = make(map[string]string)
	}

	return configMap, nil
}

func (c *Coordinator) updateConfigMap(ctx context.Context, configMap *corev1.ConfigMap) (*corev1.ConfigMap, error) {
	clientset, err := c.ClientSet()
	if err != nil {
		return nil, err
	}

	configMap, err = clientset.CoreV1().ConfigMaps(c.Namespace).Update(ctx, configMap, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}

	return configMap, nil
}
