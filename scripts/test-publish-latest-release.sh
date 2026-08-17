#!/usr/bin/env bash

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
readonly SCRIPT_DIR
readonly PUBLISHER="$SCRIPT_DIR/publish-latest-release.sh"
PASSED=0

new_fixture() {
  FIXTURE="$(mktemp -d)"
  export FAKE_STATE="$FIXTURE/state"
  export FAKE_LOG="$FIXTURE/commands.log"
  export PATH="$FIXTURE/bin:$ORIGINAL_PATH"
  export REPO="example/repo"
  export GITHUB_SHA="new-sha"
  export GITHUB_RUN_ID="42"
  export GITHUB_RUN_ATTEMPT="1"
  export BUILD_DATE="2026-08-17T00:00:00Z"
  export MAIN_SHA="$GITHUB_SHA"
  unset FAIL_MATCH
  mkdir -p "$FAKE_STATE/releases" "$FAKE_STATE/tags" "$FIXTURE/bin"
  printf '%s' "$MAIN_SHA" >"$FAKE_STATE/main-sha"
  printf 'new asset' >"$FIXTURE/new.bin"
  cp "$SCRIPT_DIR/testdata/fake-gh.sh" "$FIXTURE/bin/gh"
  chmod +x "$FIXTURE/bin/gh"
}

cleanup_fixture() {
  rm -rf "$FIXTURE"
}

add_release() {
  local tag="$1"
  local draft="$2"
  local sha="$3"
  local dir="$FAKE_STATE/releases/$tag"
  mkdir -p "$dir/assets"
  printf '%s' "$draft" >"$dir/draft"
  printf '%s' "$sha" >"$dir/sha"
  printf 'Latest Build' >"$dir/title"
  printf 'notes' >"$dir/body"
  printf 'old asset' >"$dir/assets/app.bin"
  if [[ "$draft" == false ]]; then
    printf '%s' "$sha" >"$FAKE_STATE/tags/$tag"
  fi
}

assert_file() {
  [[ -f "$1" ]] || { echo "Expected file: $1" >&2; return 1; }
}

assert_no_file() {
  [[ ! -e "$1" ]] || { echo "Unexpected file: $1" >&2; return 1; }
}

run_case() {
  local name="$1"
  shift
  new_fixture
  if "$@"; then
    PASSED=$((PASSED + 1))
    echo "ok $PASSED - $name"
  else
    echo "not ok $((PASSED + 1)) - $name" >&2
    cleanup_fixture
    exit 1
  fi
  cleanup_fixture
}

case_first_publish() {
  bash "$PUBLISHER" publish "$FIXTURE/new.bin"
  assert_file "$FAKE_STATE/releases/latest/assets/new.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == new-sha ]]
}

case_replace_release() {
  add_release latest false old-sha
  bash "$PUBLISHER" publish "$FIXTURE/new.bin"
  assert_file "$FAKE_STATE/releases/latest/assets/new.bin"
  assert_no_file "$FAKE_STATE/releases/latest/assets/app.bin"
  assert_no_file "$FAKE_STATE/releases/latest-backup-42-1"
}

case_restore_on_promotion_failure() {
  add_release latest false old-sha
  export FAIL_MATCH="release edit latest-staging-42-1"
  if bash "$PUBLISHER" publish "$FIXTURE/new.bin"; then return 1; fi
  assert_file "$FAKE_STATE/releases/latest/assets/app.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == old-sha ]]
}

case_recover_backup() {
  add_release latest-backup-old true old-sha
  bash "$PUBLISHER" recover
  assert_file "$FAKE_STATE/releases/latest/assets/app.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == old-sha ]]
}

case_recover_staging() {
  add_release latest-staging-old true staged-sha
  bash "$PUBLISHER" recover
  [[ "$(cat "$FAKE_STATE/tags/latest")" == staged-sha ]]
}

case_recover_incomplete_release() {
  add_release latest false partial-sha
  rm -f "$FAKE_STATE/tags/latest"
  add_release latest-backup-old true old-sha
  bash "$PUBLISHER" recover
  assert_file "$FAKE_STATE/releases/latest/assets/app.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == old-sha ]]
}

case_prune_stale_drafts() {
  add_release latest false old-sha
  add_release latest-staging-old true stale-sha
  add_release latest-backup-old true stale-sha
  bash "$PUBLISHER" recover
  assert_no_file "$FAKE_STATE/releases/latest-staging-old"
  assert_no_file "$FAKE_STATE/releases/latest-backup-old"
}

case_surviving_tag() {
  printf 'old-sha' >"$FAKE_STATE/tags/latest"
  bash "$PUBLISHER" publish "$FIXTURE/new.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == new-sha ]]
}

case_stale_main() {
  printf 'newer-sha' >"$FAKE_STATE/main-sha"
  bash "$PUBLISHER" publish "$FIXTURE/new.bin"
  assert_no_file "$FAKE_STATE/releases/latest"
}

case_missing_asset() {
  if bash "$PUBLISHER" publish "$FIXTURE/missing.bin"; then return 1; fi
  assert_no_file "$FAKE_STATE/releases/latest"
}

case_empty_assets() {
  if bash "$PUBLISHER" publish; then return 1; fi
  assert_no_file "$FAKE_STATE/releases/latest"
}

case_backup_failure_preserves_latest() {
  add_release latest false old-sha
  export FAIL_MATCH="release create latest-backup-42-1"
  if bash "$PUBLISHER" publish "$FIXTURE/new.bin"; then return 1; fi
  assert_file "$FAKE_STATE/releases/latest/assets/app.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == old-sha ]]
}

case_recover_without_artifacts() {
  bash "$PUBLISHER" recover
  assert_no_file "$FAKE_STATE/releases/latest"
}

case_api_failure_stops_before_cutover() {
  add_release latest false old-sha
  export FAIL_MATCH="api --include --silent repos/example/repo/releases/tags/latest"
  if bash "$PUBLISHER" publish "$FIXTURE/new.bin"; then return 1; fi
  assert_file "$FAKE_STATE/releases/latest/assets/app.bin"
  [[ "$(cat "$FAKE_STATE/tags/latest")" == old-sha ]]
}

ORIGINAL_PATH="$PATH"
run_case "publishes when latest is absent" case_first_publish
run_case "replaces a complete existing release" case_replace_release
run_case "restores the prior release when promotion fails" case_restore_on_promotion_failure
run_case "recovers a retained backup" case_recover_backup
run_case "recovers a retained staging release" case_recover_staging
run_case "recovers when the latest tag is missing" case_recover_incomplete_release
run_case "prunes stale recovery drafts" case_prune_stale_drafts
run_case "replaces a surviving latest tag" case_surviving_tag
run_case "skips a stale main commit" case_stale_main
run_case "rejects a missing asset" case_missing_asset
run_case "rejects an empty asset list" case_empty_assets
run_case "preserves latest when backup staging fails" case_backup_failure_preserves_latest
run_case "allows an initial repository with no recovery artifacts" case_recover_without_artifacts
run_case "stops on an API failure before cutover" case_api_failure_stops_before_cutover

echo "1..$PASSED"
