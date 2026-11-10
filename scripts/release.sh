#!/bin/sh
set -eu

agent_ping_version=${1:?Usage: sh scripts/release.sh v0.1.0}
agent_ping_go=${AGENT_PING_CURSOR_GO:-go}
case "$agent_ping_version" in *[!A-Za-z0-9.-]*) echo 'Invalid version.' >&2; exit 1 ;; esac
mkdir -p dist
for agent_ping_arch in arm64 amd64; do
  agent_ping_stage=$(mktemp -d)
  trap 'rm -rf "$agent_ping_stage"' EXIT HUP INT TERM
  CGO_ENABLED=0 GOOS=darwin GOARCH="$agent_ping_arch" "$agent_ping_go" build -trimpath -ldflags "-s -w -X main.version=$agent_ping_version" -o "$agent_ping_stage/agent-ping-cursor" .
  cp LICENSE "$agent_ping_stage/"
  tar -czf "dist/agent-ping-cursor_${agent_ping_version}_darwin_${agent_ping_arch}.tar.gz" -C "$agent_ping_stage" agent-ping-cursor LICENSE
  rm -rf "$agent_ping_stage"
  trap - EXIT HUP INT TERM
done
(cd dist && shasum -a 256 "agent-ping-cursor_${agent_ping_version}_darwin_"*.tar.gz > checksums.txt)
