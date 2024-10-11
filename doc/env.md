# ENV
## Config
|name|description|required|
|-|-|-|
|SECRET_STORE| Set to `aws/secrets-manager` if you use Secrets Manager as secret store.  |false|
|SECRET_NAME | Set Secrets Manager name if you set SECRET_STORE. |false|
|CONFIG_MANIFEST_REPOSITORY| Manifest repository (like `https://github.com/xxx/xxx.git`) |false|
|CONFIG_MANIFEST_REPOSITORY_ORG| Organization of manifest repository (like `zaiminc`)|false|
|APP_REPOSITORY_ORG | Set GitHub organization of the app repositories. Note it's used for listing branches | Defaults to the organization of the manifest repository |
|CONFIG_ARGOCD_HOST| Set your ArgoCD host. |false|
|CONFIG_JENKINS_HOST| Set your Jenkins host. |false|
|CONFIG_NAMESPACE| Set ConfigMap namespace |false|

## Secret
You can use env or AWS Secrets Manager as secret store (default: env).
To use AWS Secrets Manager, set SECRET_STORE env to `aws/secrets-manager`.

**AWS Secrets Manager**
|secret key|description|required|
|-|-|-|
|SLACK_BOT_OAUTH_TOKEN| Bot User OAuth Access Token |true|
|SLACK_BOT_API_VERIFICATION_TOKEN|Verification Token |true|
|GITHUB_BOT_USER_TOKEN| Set GitHub personal access token if your deploy with GitOps. |false|
|APP_REPOSITORY_GITHUB_ACCESS_TOKEN | Set GitHub personal access token for accessing app repositories. | Defaults to GITHUB_BOT_USER_TOKEN |
|JENKINS_BOT_USER_TOKEN| Set Jenkins token if you deploy through Jenkins. |false|
|JENKINS_JOB_TOKEN| Set Jenkins token if you deploy through Jenkins.|false|


**Environment variables**
|name |description|required|
|-|-|-|
|CONFIG_SLACK_OAUTH_TOKEN| Bot User OAuth Access Token |true|
|CONFIG_SLACK_VERIFICATION_TOKEN|Verification Token |true|
|CONFIG_GITHUB_ACCESS_TOKEN| Set GitHub personal access token if your deploy with GitOps. |false|
|CONFIG_JENKINS_BOT_TOKEN| Set Jenkins token if you deploy through Jenkins. |false|
|CONFIG_JENKINS_JOB_TOKEN| Set Jenkins token if you deploy through Jenkins.|false|
