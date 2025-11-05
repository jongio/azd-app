# release setup

## github personal access token

the release-please workflow requires a personal access token (PAT) with the following permissions:

1. go to github settings > developer settings > personal access tokens > tokens (classic)
2. click "generate new token (classic)"
3. set a descriptive name: "azd-app release-please"
4. select expiration: recommend "no expiration" for automation
5. select scopes:
   - `repo` (full control of private repositories)
   - `workflow` (update github action workflows)
6. click "generate token"
7. copy the token immediately (you won't see it again)

## add token to repository

1. go to repository settings > secrets and variables > actions
2. click "new repository secret"
3. name: `RELEASE_PLEASE_TOKEN`
4. value: paste the PAT from above
5. click "add secret"

## why this is needed

github prevents workflows using `GITHUB_TOKEN` from triggering other workflows. this is a security measure to prevent infinite workflow loops. by using a PAT, the release-please PR will trigger the CI workflow, allowing branch protection rules to work correctly.

## testing

after adding the token:
1. close PR #7
2. push a commit with a conventional commit message (e.g., `fix: test release-please`)
3. release-please will create a new PR
4. verify that CI checks run on the new PR
