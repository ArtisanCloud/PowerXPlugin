#!/usr/bin/env bash
set -euo pipefail

root_dir=$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)
events_file="$root_dir/skeleton/plugin.d/events.yaml"

ruby -ryaml -e '
data = YAML.load_file(ARGV[0]) || {}
items = ((data["events"] || {})["topics"] || []) + ((data["events"] || {})["channels"] || [])
abort "realtime descriptors are required" if items.empty?
seen = {}
items.each do |item|
  key = item["key"].to_s.strip
  abort "invalid realtime descriptor key" unless key.start_with?("_topic.", "_channel.", "powerx.")
  abort "duplicate realtime descriptor #{key}" if seen[key]
  seen[key] = true
  protocols = Array(item["protocols"]).map { |value| value.to_s.strip }
  abort "descriptor #{key} missing protocols" if protocols.empty?
  abort "descriptor #{key} has unsupported protocol" unless protocols.all? { |value| ["ws", "sse"].include?(value) }
  abort "descriptor #{key} has duplicate protocol" unless protocols.uniq.length == protocols.length
  actions = Array(item["actions"]).map { |value| value.to_s.strip }
  abort "descriptor #{key} missing actions" if actions.empty?
  abort "descriptor #{key} has unsupported action" unless actions.all? { |value| ["publish", "subscribe"].include?(value) }
  abort "descriptor #{key} has duplicate action" unless actions.uniq.length == actions.length
  scope = item["scope"].to_s.strip
  abort "descriptor #{key} has unsupported scope" unless ["global", "tenant", "member"].include?(scope)
  event_types = Array(item["event_types"]).map { |value| value.to_s.strip }
  abort "descriptor #{key} has empty event type" if event_types.any?(&:empty?)
  abort "descriptor #{key} has duplicate event type" unless event_types.uniq.length == event_types.length
end
' "$events_file"

if rg -n 'new[[:space:]]+EventSource|new[[:space:]]+WebSocket|body\.getReader\(' \
  "$root_dir/skeleton/web-admin/nuxt/app" >/dev/null; then
  echo "realtime transport boundary violation in skeleton frontend" >&2
  exit 1
fi

if rg -n 'gin-contrib/sse' "$root_dir/skeleton/backend/go-gin/internal" --glob '*.go' >/dev/null; then
  echo "realtime transport boundary violation in skeleton backend" >&2
  exit 1
fi

template_frontend="$root_dir/tools/cli/internal/templates/data/web-admin/nuxt"
template_backend="$root_dir/tools/cli/internal/templates/data/backend/go-gin"
if rg -n 'new[[:space:]]+EventSource|new[[:space:]]+WebSocket|body\.getReader\(' "$template_frontend" --glob '*.tmpl' >/dev/null; then
  echo "realtime transport boundary violation in Nuxt scaffold templates" >&2
  exit 1
fi

if rg -n 'gin-contrib/sse' "$template_backend" --glob '*.go.tmpl' --glob '!go.sum.tmpl' >/dev/null; then
  echo "realtime transport boundary violation in Go scaffold templates" >&2
  exit 1
fi

echo "realtime transport boundary check passed"
