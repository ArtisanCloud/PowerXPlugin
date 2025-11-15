#!/usr/bin/env bash
set -euo pipefail

cd "$(dirname "${BASH_SOURCE[0]}")"/../..

COMMITS=${AUDIT_COMMITS:-10}
BASE_REF=${AUDIT_BASE_REF:-origin/main}

if ! git rev-parse --is-inside-work-tree >/dev/null 2>&1; then
  echo "Error: not inside a git repository" >&2
  exit 1
fi

current_branch=$(git rev-parse --abbrev-ref HEAD)
merge_base=$(git merge-base "$BASE_REF" "$current_branch" 2>/dev/null || git rev-list --max-parents=0 HEAD | tail -n1)

printf "Analyzing last %s commits between %s and %s\n" "$COMMITS" "$merge_base" "$current_branch"

header=$(printf '%-12s %-8s %-8s %-30s\n' "Commit" "Go" "E2E" "Title")
echo "$header"
echo "$(printf '%.0s-' {1..70})"

count=0
go_hits=0
e2e_hits=0
while IFS= read -r line && [ "$count" -lt "$COMMITS" ]; do
  sha=${line%% *}
  title=${line#* }
  diff_files=$(git diff --name-only "$sha^" "$sha" || true)
  has_go=$(echo "$diff_files" | grep -E '_test\.go$' >/dev/null && echo "yes" || echo "no")
  has_e2e=$(echo "$diff_files" | grep -E 'tests/e2e/.*\.spec\.ts$' >/dev/null && echo "yes" || echo "no")
  printf '%-12s %-8s %-8s %-30s\n' "${sha:0:8}" "$has_go" "$has_e2e" "${title:0:30}"
  [ "$has_go" = "yes" ] && go_hits=$((go_hits+1))
  [ "$has_e2e" = "yes" ] && e2e_hits=$((e2e_hits+1))
  count=$((count+1))
done < <(git log "$merge_base..$current_branch" --pretty=format:'%H %s')

echo "$(printf '%.0s-' {1..70})"
printf 'Commits inspected: %s\n' "$count"
printf 'Go tests added:    %s\n' "$go_hits"
printf 'E2E tests added:   %s\n' "$e2e_hits"

if [ "$count" -gt 0 ]; then
  pct=$((100*(go_hits+e2e_hits)/count))
  printf 'Combined adoption: %s%%\n' "$pct"
else
  echo 'No commits to inspect.'
fi
