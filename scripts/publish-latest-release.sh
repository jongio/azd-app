#!/usr/bin/env bash

set -Eeuo pipefail

readonly REPO="${REPO:?REPO is required}"
readonly GITHUB_SHA="${GITHUB_SHA:?GITHUB_SHA is required}"
readonly STAGING_PREFIX="latest-staging-"
readonly BACKUP_PREFIX="latest-backup-"

HTTP_STATUS=
WORK_DIR=
STAGING_TAG=
BACKUP_TAG=
CUTOVER_STARTED=false
PUBLISH_COMPLETE=false
LATEST_EXISTED=false

probe_api() {
  local endpoint="$1"
  local headers
  local errors
  local status

  headers="$(mktemp)"
  errors="$(mktemp)"
  set +e
  gh api --include --silent "$endpoint" >"$headers" 2>"$errors"
  set -e

  status="$(awk 'NR == 1 && $1 ~ /^HTTP/ { print $2 }' "$headers")"
  if [[ ! "$status" =~ ^[0-9]{3}$ ]]; then
    echo "GitHub API request failed without an HTTP status: $endpoint" >&2
    cat "$errors" >&2
    rm -f "$headers" "$errors"
    return 1
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
    --jq '.[] | select(.draft and ((.tag_name | startswith("latest-staging-")) or (.tag_name | startswith("latest-backup-")))) | [.created_at, .tag_name] | @tsv' \
    | sort -r
}

delete_draft() {
  local tag="$1"
  gh release delete "$tag" --repo "$REPO" --yes
}

prune_recovery_artifacts() {
  local keep="${1:-}"
  local tag

  while IFS=$'\t' read -r _ tag; do
    if [[ -n "$tag" && "$tag" != "$keep" ]]; then
      echo "Deleting stale recovery draft $tag."
      delete_draft "$tag"
    fi
  done < <(list_recovery_artifacts)
}

delete_latest_objects() {
  latest_release_status
  if [[ "$HTTP_STATUS" == 200 ]]; then
    echo "Deleting the current latest release."
    gh release delete latest --repo "$REPO" --yes
  else
    echo "No latest release exists."
  fi

  latest_tag_status
  if [[ "$HTTP_STATUS" == 200 ]]; then
    echo "Deleting the current latest tag."
    gh api --method DELETE "repos/$REPO/git/refs/tags/latest"
  else
    echo "No latest tag exists."
  fi
}

release_target() {
  local tag="$1"
  gh api "repos/$REPO/releases/tags/$tag" --jq '.target_commitish'
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

  echo "Promoting recovery draft $tag to latest."
  gh release edit "$tag" \
    --repo "$REPO" \
    --tag latest \
    --draft=false \
    --prerelease
  verify_latest "$expected_sha"
}

select_recovery_artifact() {
  local artifacts="$1"
  local candidate

  candidate="$(printf '%s\n' "$artifacts" | awk -F '\t' '$2 ~ /^latest-backup-/ { print $2; exit }')"
  if [[ -z "$candidate" ]]; then
    candidate="$(printf '%s\n' "$artifacts" | awk -F '\t' '$2 ~ /^latest-staging-/ { print $2; exit }')"
  fi
  printf '%s' "$candidate"
}

recover_latest() {
  local artifacts
  local candidate
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
  candidate="$(select_recovery_artifact "$artifacts")"
  if [[ -z "$candidate" ]]; then
    if [[ "$release_status" == 404 && "$tag_status" == 404 ]]; then
      echo "No latest release or recovery draft exists."
      return
    fi
    echo "The latest release is incomplete and no recovery draft exists." >&2
    return 1
  fi

  target="$(release_target "$candidate")"
  if [[ "$release_status" == 200 ]]; then
    echo "Deleting the incomplete latest release before recovery."
    gh release delete latest --repo "$REPO" --yes
  fi
  if [[ "$tag_status" == 200 ]]; then
    echo "Deleting the surviving latest tag before recovery."
    gh api --method DELETE "repos/$REPO/git/refs/tags/latest"
  fi

  promote_draft "$candidate" "$target"
  prune_recovery_artifacts
  echo "Recovered the latest release from $candidate."
}

assert_main_tip() {
  local main_sha
  main_sha="$(gh api "repos/$REPO/git/ref/heads/main" --jq '.object.sha')"
  if [[ "$main_sha" != "$GITHUB_SHA" ]]; then
    echo "Skipping publication because main advanced to $main_sha."
    return 1
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
  else
    candidate="$STAGING_TAG"
  fi
  target="$(release_target "$candidate" 2>/dev/null || true)"

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

  if [[ "$CUTOVER_STARTED" == true ]]; then
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

publish_latest() {
  local -a files=("$@")
  local notes
  local short_sha="${GITHUB_SHA:0:7}"
  local build_date="${BUILD_DATE:-$(date -u +%Y-%m-%dT%H:%M:%SZ)}"

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

  if ! assert_main_tip; then
    return
  fi

  WORK_DIR="$(mktemp -d)"
  STAGING_TAG="${STAGING_PREFIX}${GITHUB_RUN_ID:?GITHUB_RUN_ID is required}-${GITHUB_RUN_ATTEMPT:?GITHUB_RUN_ATTEMPT is required}"
  BACKUP_TAG="${BACKUP_PREFIX}${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"
  notes="$WORK_DIR/release-notes.md"

  cat >"$notes" <<EOF
## Latest Build from main

**Commit:** [$short_sha](https://github.com/$REPO/commit/$GITHUB_SHA)
**Built:** $build_date

> This is an automatically built prerelease from the main branch.
> It may contain bugs. For stable releases, see the [Releases page](https://github.com/$REPO/releases).

### Install

\`azd extension install jongio.azd.app --version latest\`
EOF

  gh release create "$STAGING_TAG" "${files[@]}" \
    --repo "$REPO" \
    --title "Latest Build ($short_sha)" \
    --notes-file "$notes" \
    --draft \
    --target "$GITHUB_SHA"

  trap on_exit EXIT
  trap 'exit 130' INT
  trap 'exit 143' TERM

  latest_release_status
  if [[ "$HTTP_STATUS" == 200 ]]; then
    LATEST_EXISTED=true
    stage_backup
  else
    echo "No latest release exists; the staged release is the recovery artifact."
  fi

  if ! assert_main_tip; then
    return
  fi

  CUTOVER_STARTED=true
  delete_latest_objects
  promote_draft "$STAGING_TAG" "$GITHUB_SHA"
  PUBLISH_COMPLETE=true

  prune_recovery_artifacts
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
