#!/usr/bin/env bash

set -euo pipefail

command_text="$*"
printf '%s\n' "$command_text" >>"$FAKE_LOG"
if [[ -n "${FAIL_MATCH:-}" && "$command_text" == *"$FAIL_MATCH"* && ! -f "$FAKE_STATE/failure-used" ]]; then
  failure_match_count=0
  if [[ -f "$FAKE_STATE/failure-match-count" ]]; then
    read -r failure_match_count <"$FAKE_STATE/failure-match-count"
  fi
  failure_match_count=$((failure_match_count + 1))
  printf '%s' "$failure_match_count" >"$FAKE_STATE/failure-match-count"
  if [[ "$failure_match_count" -eq "${FAIL_MATCH_OCCURRENCE:-1}" ]]; then
    touch "$FAKE_STATE/failure-used"
    echo "injected failure" >&2
    exit 1
  fi
fi
if [[ -n "${POST_EDIT_FAIL_MATCH:-}" &&
      -f "$FAKE_STATE/promotion-applied" &&
      "$command_text" == *"$POST_EDIT_FAIL_MATCH"* ]]; then
  echo "injected post-edit failure" >&2
  exit 1
fi
if [[ -n "${PAUSE_MATCH:-}" && "$command_text" == *"$PAUSE_MATCH"* ]]; then
  touch "$FAKE_STATE/pause-started"
  sleep 1
fi

release_dir() {
  printf '%s/releases/%s' "$FAKE_STATE" "$1"
}

api_status() {
  local endpoint="$1"
  local path=
  case "$endpoint" in
    repos/*/releases/tags/*)
      path="$(release_dir "${endpoint##*/}")"
      if [[ -e "$path" && "$(cat "$path/draft")" == true ]]; then
        path=
      fi
      ;;
    repos/*/git/ref/tags/*)
      path="$FAKE_STATE/tags/${endpoint##*/}"
      ;;
  esac
  if [[ -e "$path" ]]; then
    printf 'HTTP/2.0 200 OK\n'
    return
  fi
  printf 'HTTP/2.0 404 Not Found\n'
  return 1
}

if [[ "$1" == api ]]; then
  shift
  if [[ "$1" == --include ]]; then
    shift 2
    if [[ -n "${HTTP_FAIL_MATCH:-}" && "$1" == *"$HTTP_FAIL_MATCH"* ]]; then
      printf 'HTTP/2.0 500 Internal Server Error\n'
      exit 1
    fi
    api_status "$1"
    exit
  fi
  if [[ "$1" == --paginate ]]; then
    expected_filter='.[] | select(.draft and ((.tag_name | startswith("latest-staging-")) or (.tag_name | startswith("latest-backup-")))) | [.created_at, .tag_name, .target_commitish] | @tsv'
    [[ "$2" == "repos/example/repo/releases?per_page=100" ]]
    [[ "$3" == --jq ]]
    [[ "$4" == "$expected_filter" ]]
    find "$FAKE_STATE/releases" -mindepth 1 -maxdepth 1 -type d \
      | while read -r dir; do
          tag="$(basename "$dir")"
          if [[ "$(cat "$dir/draft")" == true && ("$tag" == latest-staging-* || "$tag" == latest-backup-*) ]]; then
            printf '2026-08-17T00:00:00Z\t%s\t%s\n' "$tag" "$(cat "$dir/sha")"
          fi
        done
    exit
  fi
  if [[ "$1" == --method && "$2" == DELETE ]]; then
    endpoint="$3"
    rm -f "$FAKE_STATE/tags/${endpoint##*/}"
    exit
  fi

  endpoint="$1"
  shift
  jq_filter=
  if [[ "${1:-}" == --jq ]]; then
    jq_filter="$2"
  fi
  case "$endpoint" in
    repos/*/git/ref/heads/main)
      cat "$FAKE_STATE/main-sha"
      ;;
    repos/*/git/ref/tags/*)
      cat "$FAKE_STATE/tags/${endpoint##*/}"
      ;;
    repos/*/releases/tags/*)
      dir="$(release_dir "${endpoint##*/}")"
      [[ "$(cat "$dir/draft")" == false ]]
      case "$jq_filter" in
        '.target_commitish') cat "$dir/sha" ;;
        '.name // "Latest Build"') cat "$dir/title" ;;
        '.body // ""') cat "$dir/body" ;;
        '.assets | length') find "$dir/assets" -type f | wc -l | tr -d ' ' ;;
        *) echo 1 ;;
      esac
      ;;
  esac
  exit
fi

if [[ "$1" == release ]]; then
  action="$2"
  tag="$3"
  shift 3
  case "$action" in
    create)
      dir="$(release_dir "$tag")"
      mkdir -p "$dir/assets"
      draft=false
      target=
      title=
      notes=
      while [[ "$#" -gt 0 ]]; do
        case "$1" in
          --repo) shift 2 ;;
          --title) title="$2"; shift 2 ;;
          --notes-file) notes="$2"; shift 2 ;;
          --draft) draft=true; shift ;;
          --target) target="$2"; shift 2 ;;
          --*) shift ;;
          *) cp "$1" "$dir/assets/"; shift ;;
        esac
      done
      printf '%s' "$draft" >"$dir/draft"
      printf '%s' "$target" >"$dir/sha"
      printf '%s' "$title" >"$dir/title"
      if [[ -n "$notes" ]]; then cp "$notes" "$dir/body"; else : >"$dir/body"; fi
      if [[ -n "${ADVANCE_MAIN_MATCH:-}" && "$command_text" == *"$ADVANCE_MAIN_MATCH"* ]]; then
        printf 'newer-sha' >"$FAKE_STATE/main-sha"
      fi
      ;;
    edit)
      new_tag=
      while [[ "$#" -gt 0 ]]; do
        case "$1" in
          --tag) new_tag="$2"; shift 2 ;;
          *) shift ;;
        esac
      done
      dir="$(release_dir "$tag")"
      target="$(cat "$dir/sha")"
      mv "$dir" "$(release_dir "$new_tag")"
      printf 'false' >"$(release_dir "$new_tag")/draft"
      printf '%s' "$target" >"$FAKE_STATE/tags/$new_tag"
      touch "$FAKE_STATE/promotion-applied"
      ;;
    delete)
      rm -rf "$(release_dir "$tag")"
      ;;
    download)
      dir=
      while [[ "$#" -gt 0 ]]; do
        case "$1" in
          --dir) dir="$2"; shift 2 ;;
          *) shift ;;
        esac
      done
      mkdir -p "$dir"
      cp "$(release_dir "$tag")/assets/"* "$dir/"
      ;;
  esac
fi
