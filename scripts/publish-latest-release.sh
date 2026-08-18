#!/usr/bin/env bash

set -Eeuo pipefail

readonly REPO="${REPO:?REPO is required}"
readonly GITHUB_SHA="${GITHUB_SHA:?GITHUB_SHA is required}"
readonly STAGING_PREFIX="latest-staging-"
readonly BACKUP_PREFIX="latest-backup-"
readonly MAIN_TIP_STALE_STATUS=10
readonly MAIN_TIP_API_FAILURE_STATUS=11

HTTP_STATUS=
WORK_DIR=
STAGING_TAG=
BACKUP_TAG=
BACKUP_TARGET=
CUTOVER_STARTED=false
REMOTE_PROMOTION_APPLIED=false
PROMOTION_IN_PROGRESS=false
PENDING_SIGNAL_STATUS=0
PUBLISH_COMPLETE=false
LATEST_EXISTED=false

probe_api() {
  local endpoint="$1"
  local headers
  local errors
  local result
  local status
  local was_errexit=false

  headers="$(mktemp)"
  errors="$(mktemp)"
  HTTP_STATUS=
  [[ $- == *e* ]] && was_errexit=true
  set +e
  gh api --include --silent "$endpoint" >"$headers" 2>"$errors"
  result=$?
  if [[ "$was_errexit" == true ]]; then
    set -e
  fi

  status="$(awk 'NR == 1 && $1 ~ /^HTTP/ { print $2 }' "$headers")"
  if [[ ! "$status" =~ ^[0-9]{3}$ ]]; then
    echo "GitHub API request failed without an HTTP status: $endpoint" >&2
    cat "$errors" >&2
    rm -f "$headers" "$errors"
    if [[ "$result" -eq 0 ]]; then
      return 1
    fi
    return "$result"
  fi

  HTTP_STATUS="$status"
  rm -f "$headers" "$errors"
}

expect_present_or_missing() {
  local endpoint="$1"
  probe_api "$endpoint"
  case "$HTTP_STATUS" in
    200 | 404) ;;
    *)
      echo "GitHub API request returned HTTP $HTTP_STATUS: $endpoint" >&2
      return 1
      ;;
  esac
}

latest_release_endpoint() {
  printf 'repos/%s/releases/tags/latest' "$REPO"
}

latest_tag_endpoint() {
  printf 'repos/%s/git/ref/tags/latest' "$REPO"
}

latest_release_status() {
  expect_present_or_missing "$(latest_release_endpoint)"
}

latest_tag_status() {
  expect_present_or_missing "$(latest_tag_endpoint)"
}

list_recovery_artifacts() {
  gh api --paginate "repos/$REPO/releases?per_page=100" \
    --jq '.[] | select(.draft and ((.tag_name | startswith("latest-staging-")) or (.tag_name | startswith("latest-backup-")))) | [.created_at, .tag_name, .target_commitish] | @tsv' \
    | sort -r
}

delete_draft() {
  local tag="$1"
  gh release delete "$tag" --repo "$REPO" --yes
}

prune_recovery_artifacts() {
  local artifacts
  local tag

  artifacts="$(list_recovery_artifacts)" || return $?
  while IFS=$'\t' read -r _ tag _; do
    if [[ -n "$tag" ]]; then
      echo "Deleting stale recovery draft $tag."
      delete_draft "$tag" || return $?
    fi
  done <<<"$artifacts"
}

prune_after_success() {
  if ! prune_recovery_artifacts; then
    echo "The latest release is healthy, but stale recovery artifacts could not be pruned." >&2
    return 1
  fi
}

delete_latest_objects() {
  latest_release_status || return 1
  if [[ "$HTTP_STATUS" == 200 ]]; then
    echo "Deleting the current latest release."
    gh release delete latest --repo "$REPO" --yes || return 1
  else
    echo "No latest release exists."
  fi

  latest_tag_status || return 1
  if [[ "$HTTP_STATUS" == 200 ]]; then
    echo "Deleting the current latest tag."
    gh api --method DELETE "repos/$REPO/git/refs/tags/latest" || return 1
  else
    echo "No latest tag exists."
  fi
}

verify_latest() {
  local expected_sha="$1"
  local actual_sha

  latest_release_status
  if [[ "$HTTP_STATUS" != 200 ]]; then
    echo "The latest release was not published." >&2
    return 1
  fi

  latest_tag_status
  if [[ "$HTTP_STATUS" != 200 ]]; then
    echo "The latest tag was not published." >&2
    return 1
  fi

  actual_sha="$(gh api "$(latest_tag_endpoint)" --jq '.object.sha')"
  if [[ "$actual_sha" != "$expected_sha" ]]; then
    echo "The latest tag targets $actual_sha, expected $expected_sha." >&2
    return 1
  fi
}

promote_draft() {
  local tag="$1"
  local expected_sha="$2"
  local edit_result

  echo "Promoting recovery draft $tag to latest."
  PROMOTION_IN_PROGRESS=true
  if gh release edit "$tag" \
      --repo "$REPO" \
      --tag latest \
      --draft=false \
      --prerelease; then
    edit_result=0
    REMOTE_PROMOTION_APPLIED=true
  else
    edit_result=$?
  fi
  PROMOTION_IN_PROGRESS=false

  if [[ "$PENDING_SIGNAL_STATUS" -ne 0 ]]; then
    exit "$PENDING_SIGNAL_STATUS"
  fi
  if [[ "$edit_result" -ne 0 ]]; then
    return "$edit_result"
  fi

  verify_latest "$expected_sha"
}

select_recovery_artifact() {
  local artifacts="$1"
  local candidate

  candidate="$(printf '%s\n' "$artifacts" | awk -F '\t' '$2 ~ /^latest-backup-/ { print; exit }')"
  if [[ -z "$candidate" ]]; then
    candidate="$(printf '%s\n' "$artifacts" | awk -F '\t' -v sha="$GITHUB_SHA" '$2 ~ /^latest-staging-/ && $3 == sha { print; exit }')"
  fi
  printf '%s' "$candidate"
}

recover_latest() {
  local artifacts
  local candidate
  local candidate_row
  local release_status
  local tag_status
  local target

  latest_release_status
  release_status="$HTTP_STATUS"
  latest_tag_status
  tag_status="$HTTP_STATUS"
  if [[ "$release_status" == 200 && "$tag_status" == 200 ]]; then
    prune_recovery_artifacts
    echo "The latest release is healthy."
    return
  fi

  artifacts="$(list_recovery_artifacts)"
  candidate_row="$(select_recovery_artifact "$artifacts")"
  if [[ -z "$candidate_row" ]]; then
    if [[ "$release_status" == 200 || "$tag_status" == 200 ]]; then
      echo "Removing incomplete latest state so the current build can republish it."
      delete_latest_objects
    else
      echo "No latest release or matching recovery draft exists."
    fi
    prune_recovery_artifacts
    return
  fi

  IFS=$'\t' read -r _ candidate target <<<"$candidate_row"
  delete_latest_objects
  promote_draft "$candidate" "$target"
  prune_after_success
  echo "Recovered the latest release from $candidate."
}

assert_main_tip() {
  local main_sha

  if ! main_sha="$(gh api "repos/$REPO/git/ref/heads/main" --jq '.object.sha')"; then
    echo "Failed to determine the current main tip." >&2
    return "$MAIN_TIP_API_FAILURE_STATUS"
  fi
  if [[ "$main_sha" != "$GITHUB_SHA" ]]; then
    echo "Skipping publication because main advanced to $main_sha."
    return "$MAIN_TIP_STALE_STATUS"
  fi
}

stage_backup() {
  local backup_assets="$WORK_DIR/backup-assets"
  local backup_notes="$WORK_DIR/backup-notes.md"
  local backup_title
  local backup_target
  local asset_count
  local -a files=()

  latest_tag_status
  if [[ "$HTTP_STATUS" != 200 ]]; then
    echo "The latest release has no tag to back up." >&2
    return 1
  fi

  backup_target="$(gh api "$(latest_tag_endpoint)" --jq '.object.sha')"
  BACKUP_TARGET="$backup_target"
  backup_title="$(gh api "$(latest_release_endpoint)" --jq '.name // "Latest Build"')"
  gh api "$(latest_release_endpoint)" --jq '.body // ""' >"$backup_notes"
  asset_count="$(gh api "$(latest_release_endpoint)" --jq '.assets | length')"

  mkdir -p "$backup_assets"
  if [[ "$asset_count" -gt 0 ]]; then
    gh release download latest --repo "$REPO" --dir "$backup_assets"
    shopt -s nullglob
    files=("$backup_assets"/*)
    shopt -u nullglob
  fi

  gh release create "$BACKUP_TAG" "${files[@]}" \
    --repo "$REPO" \
    --title "$backup_title" \
    --notes-file "$backup_notes" \
    --draft \
    --target "$backup_target"
}

restore_after_failure() {
  local candidate
  local target

  if [[ "$LATEST_EXISTED" == true ]]; then
    candidate="$BACKUP_TAG"
    target="$BACKUP_TARGET"
  else
    candidate="$STAGING_TAG"
    target="$GITHUB_SHA"
  fi

  echo "Cutover failed. Restoring latest from $candidate." >&2
  set +e
  delete_latest_objects
  local delete_result=$?
  if [[ "$delete_result" -eq 0 && -n "$target" ]]; then
    promote_draft "$candidate" "$target"
    local promote_result=$?
  else
    local promote_result=1
  fi
  set -e

  if [[ "$promote_result" -ne 0 ]]; then
    echo "Automatic recovery failed. Recovery draft:" >&2
    echo "  https://github.com/$REPO/releases/tag/$candidate" >&2
    echo "After removing any partial latest release and tag, run:" >&2
    echo "  gh release edit $candidate --repo $REPO --tag latest --draft=false --prerelease" >&2
    return 1
  fi

  echo "Automatic recovery restored latest from $candidate." >&2
}

on_exit() {
  local status=$?
  trap - EXIT INT TERM

  if [[ "$PUBLISH_COMPLETE" == true ]]; then
    rm -rf "$WORK_DIR"
    exit "$status"
  fi

  if [[ "$REMOTE_PROMOTION_APPLIED" == true ]]; then
    echo "Promotion succeeded, but completion could not be verified." >&2
    echo "Preserving the promoted latest release and recovery artifacts." >&2
  elif [[ "$CUTOVER_STARTED" == true ]]; then
    if ! restore_after_failure; then
      status=1
    fi
  else
    set +e
    [[ -n "$STAGING_TAG" ]] && delete_draft "$STAGING_TAG" >/dev/null 2>&1
    [[ -n "$BACKUP_TAG" ]] && delete_draft "$BACKUP_TAG" >/dev/null 2>&1
    set -e
  fi

  [[ -n "$WORK_DIR" ]] && rm -rf "$WORK_DIR"
  exit "$status"
}

on_signal() {
  local status="$1"

  if [[ "$PROMOTION_IN_PROGRESS" == true ]]; then
    PENDING_SIGNAL_STATUS="$status"
    return
  fi
  exit "$status"
}

publish_latest() {
  local -a files=("$@")
  local notes
  local short_sha="${GITHUB_SHA:0:7}"
  local build_date="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"
  local main_tip_status

  if [[ "${#files[@]}" -eq 0 ]]; then
    echo "No release assets were provided." >&2
    return 1
  fi
  for file in "${files[@]}"; do
    if [[ ! -f "$file" ]]; then
      echo "Release asset does not exist: $file" >&2
      return 1
    fi
  done

  main_tip_status=0
  assert_main_tip || main_tip_status=$?
  case "$main_tip_status" in
    0) ;;
    "$MAIN_TIP_STALE_STATUS") return 0 ;;
    *) return "$main_tip_status" ;;
  esac

  WORK_DIR="$(mktemp -d)"
  STAGING_TAG="${STAGING_PREFIX}${GITHUB_RUN_ID:?GITHUB_RUN_ID is required}-${GITHUB_RUN_ATTEMPT:?GITHUB_RUN_ATTEMPT is required}"
  BACKUP_TAG="${BACKUP_PREFIX}${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"
  notes="$WORK_DIR/release-notes.md"

  trap on_exit EXIT
  trap 'on_signal 130' INT
  trap 'on_signal 143' TERM

  cat >"$notes" <<EOF
## Latest Build from main

**Commit:** [$short_sha](https://github.com/$REPO/commit/$GITHUB_SHA)
**Built:** $build_date

> This is an automatically built prerelease from the main branch.
> It may contain bugs. For stable releases, see the [Releases page](https://github.com/$REPO/releases).
EOF

  gh release create "$STAGING_TAG" "${files[@]}" \
    --repo "$REPO" \
    --title "Latest Build ($short_sha)" \
    --notes-file "$notes" \
    --draft \
    --target "$GITHUB_SHA"

  latest_release_status
  if [[ "$HTTP_STATUS" == 200 ]]; then
    LATEST_EXISTED=true
    stage_backup
  else
    echo "No latest release exists; the staged release is the recovery artifact."
  fi

  main_tip_status=0
  assert_main_tip || main_tip_status=$?
  case "$main_tip_status" in
    0) ;;
    "$MAIN_TIP_STALE_STATUS") return 0 ;;
    *) return "$main_tip_status" ;;
  esac

  CUTOVER_STARTED=true
  delete_latest_objects
  promote_draft "$STAGING_TAG" "$GITHUB_SHA"
  PUBLISH_COMPLETE=true

  prune_after_success
  echo "Latest release updated: https://github.com/$REPO/releases/tag/latest"
}

case "${1:-}" in
  recover)
    recover_latest
    ;;
  publish)
    shift
    publish_latest "$@"
    ;;
  *)
    echo "Usage: $0 recover | publish <asset>..." >&2
    exit 2
    ;;
esac
