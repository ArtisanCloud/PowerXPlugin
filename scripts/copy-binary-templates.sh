#!/usr/bin/env bash
set -euo pipefail
shopt -s nullglob
binary_exts=(png jpg jpeg gif webp avif bmp ico svg ttf woff woff2)
skeleton_root="skeleton"
scaffold_root="scaffold/templates"
cli_root="tools/cli/internal/templates/data"
for ext in "${binary_exts[@]}"; do
  while read -r -d '' file; do
    rel="${file#$skeleton_root/}"
    scaffold_target="$scaffold_root/${rel}.tmpl"
    cli_target="$cli_root/${rel}.tmpl"
    if [[ -f "$scaffold_target" ]]; then
      cp "$file" "$scaffold_target"
    fi
    if [[ -f "$cli_target" ]]; then
      cp "$file" "$cli_target"
    fi
  done < <(find "$skeleton_root" -type f -name "*.$ext" -print0)
done
