package main

import (
	"testing"

	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestCreateUserNamesInGroups(t *testing.T) {
	var (
		ul UserList

		rolebindings = &v1.ConfigMapList{
			Items: []v1.ConfigMap{
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rolebinding1",
						Labels: map[string]string{
							"gocat.zaim.net/configmap-type": "rolebinding",
						},
					},
					Data: map[string]string{
						"Developer": "user1\nuser2\n",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rolebinding2",
						Labels: map[string]string{
							"gocat.zaim.net/configmap-type": "rolebinding",
						},
					},
					Data: map[string]string{
						"Admin": "user3\nuser4",
					},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Name: "rolebinding3",
						Labels: map[string]string{
							"gocat.zaim.net/configmap-type": "rolebinding",
						},
					},
					Data: map[string]string{
						"Developer": "user5",
						"Admin":     "user6",
					},
				},
			},
		}
	)

	userNamesInGroups := ul.createUserNamesInGroups(rolebindings)

	if len(userNamesInGroups) != 2 {
		t.Fatalf("userNamesInGroups length is not 2")
	}

	require.Equal(t,
		map[string]struct{}{"user1": {}, "user2": {}, "user5": {}},
		userNamesInGroups[RoleDeveloper],
	)

	require.Equal(t,
		map[string]struct{}{"user3": {}, "user4": {}, "user6": {}},
		userNamesInGroups[RoleAdmin],
	)
}
