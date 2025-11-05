# release workflow

## overview

the release process is split into two steps:

1. **changelog/version management** - automated via release-please
2. **creating releases** - manual trigger when ready

## changelog and version management

release-please automatically:
- tracks conventional commits on main branch
- creates/updates a PR with version bumps and changelog updates
- bundles multiple commits/PRs into a single version bump

## creating a release

when you're ready to release:

1. merge the release-please PR (after CI checks pass)
2. go to actions > release workflow
3. click "run workflow"
4. enter the version number (e.g., `0.2.1`)
5. the workflow will:
   - create a git tag
   - build binaries with goreleaser
   - create github release with assets
   - update the registry

## setup

### github personal access token

the release-please workflow requires a personal access token (PAT) to trigger CI on release PRs:

1. go to github settings > developer settings > personal access tokens > tokens (classic)
2. click "generate new token (classic)"
3. name: "azd-app release-please"
4. select scopes:
   - `repo` (full control of private repositories)
   - `workflow` (update github action workflows)
5. click "generate token"
6. copy the token

### add token to repository

1. go to repository settings > secrets and variables > actions
2. click "new repository secret"
3. name: `RELEASE_PLEASE_TOKEN`
4. value: paste the PAT
5. click "add secret"

## why this approach

- **bundles multiple changes**: release-please accumulates all commits since last release
- **manual control**: you decide when to actually cut a release
- **clean history**: one PR per release with complete changelog
- **flexible**: can merge many feature PRs before releasing
