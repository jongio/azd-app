#!/usr/bin/env bash

set -euo pipefail

command_text="$*"
printf '%s\n' "$command_text" >>"$FAKE_LOG"
if [[ -n "${FAIL_MATCH:-}" && "$command_text" == *"$FAIL_MATCH"* && ! -f "$FAKE_STATE/failure-used" ]]; then
  touch "$FAKE_STATE/failure-used"
  echo "injected failure" >&2
  exit 1
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
    api_status "$1"
    exit
  fi
  if [[ "$1" == --paginate ]]; then
    find "$FAKE_STATE/releases" -mindepth 1 -maxdepth 1 -type d \
      | while read -r dir; do
          tag="$(basename "$dir")"
          if [[ "$(cat "$dir/draft")" == true && "$tag" == latest-* ]]; then
            printf '2026-08-17T00:00:00Z\t%s\n' "$tag"
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
