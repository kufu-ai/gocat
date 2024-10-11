package main

import (
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigInitGet(t *testing.T) {
	type testcase struct {
		subject                string
		env                    map[string]string
		secrets                map[string]string
		want                   CatConfig
		wantAppRepositoryOrg   string
		wantAppRepositoryToken string
	}

	secrets := map[string]string{
		"mysecret": `{
  "GITHUB_BOT_USER_TOKEN": "mysecret_token",
  "APP_REPOSITORY_GITHUB_ACCESS_TOKEN": "mysecret_apptoken"
}`,
	}

	tcs := []testcase{
		{
			subject: "minimal",
			env: map[string]string{
				"CONFIG_MANIFEST_REPOSITORY": "https://github.com/org/manifests.git",
				"CONFIG_GITHUB_ACCESS_TOKEN": "mytoken",
			},
			secrets: secrets,
			want: CatConfig{
				ManifestRepository:     "https://github.com/org/manifests.git",
				ManifestRepositoryName: "manifests",
				ManifestRepositoryOrg:  "org",
				GitHubAccessToken:      "mytoken",
				GitHubUserName:         "gocat",
			},
			wantAppRepositoryOrg:   "org",
			wantAppRepositoryToken: "mytoken",
		},
		{
			subject: "with app org and token",
			env: map[string]string{
				"CONFIG_MANIFEST_REPOSITORY":                "https://github.com/org/manifests.git",
				"CONFIG_GITHUB_ACCESS_TOKEN":                "mytoken",
				"CONFIG_APP_REPOSITORY_ORG":                 "apporg",
				"CONFIG_APP_REPOSITORY_GITHUB_ACCESS_TOKEN": "apptoken",
			},
			secrets: secrets,
			want: CatConfig{
				ManifestRepository:             "https://github.com/org/manifests.git",
				ManifestRepositoryName:         "manifests",
				ManifestRepositoryOrg:          "org",
				GitHubAccessToken:              "mytoken",
				GitHubUserName:                 "gocat",
				AppRepositoryOrg:               "apporg",
				AppRepositoryGitHubAccessToken: "apptoken",
			},
			wantAppRepositoryOrg:   "apporg",
			wantAppRepositoryToken: "apptoken",
		},
		{
			subject: "with secrets",
			env: map[string]string{
				"CONFIG_MANIFEST_REPOSITORY":                "https://github.com/org/manifests.git",
				"CONFIG_GITHUB_ACCESS_TOKEN":                "mytoken",
				"CONFIG_APP_REPOSITORY_ORG":                 "apporg",
				"CONFIG_APP_REPOSITORY_GITHUB_ACCESS_TOKEN": "apptoken",
				"SECRET_STORE":                              "aws/secrets-manager",
				"SECRET_NAME":                               "mysecret",
			},
			secrets: secrets,
			want: CatConfig{
				ManifestRepository:             "https://github.com/org/manifests.git",
				ManifestRepositoryName:         "manifests",
				ManifestRepositoryOrg:          "org",
				GitHubAccessToken:              "mysecret_token",
				GitHubUserName:                 "gocat",
				AppRepositoryOrg:               "apporg",
				AppRepositoryGitHubAccessToken: "mysecret_apptoken",
			},
			wantAppRepositoryOrg:   "apporg",
			wantAppRepositoryToken: "mysecret_apptoken",
		},
	}

	for _, tc := range tcs {
		t.Run(tc.subject, func(t *testing.T) {
			c, err := initConfig(func(secretName string) (*secretsmanager.GetSecretValueOutput, error) {
				return &secretsmanager.GetSecretValueOutput{
					SecretString: aws.String(tc.secrets[secretName]),
				}, nil
			}, func(s string) string {
				return tc.env[s]
			})
			require.NoError(t, err)

			assert.Equal(t, tc.want, *c, "CatConfig")
			assert.Equal(t, tc.wantAppRepositoryOrg, c.GetAppRepositoryOrg(), "AppRepositoryOrg")
			assert.Equal(t, tc.wantAppRepositoryToken, c.GetAppRepositoryGitHubAccessToken(), "AppRepositoryGitHubAccessToken")
		})
	}
}
