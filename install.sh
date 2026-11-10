#!/bin/sh
set -eu

agent_ping_version=${AGENT_PING_CURSOR_VERSION:-v0.1.0}
agent_ping_repo=sunnyby/agent-ping-cursor
agent_ping_bin_dir=${AGENT_PING_CURSOR_BIN_DIR:-"$HOME/.local/bin"}

case "$agent_ping_version" in
  v[0-9]*.[0-9]*.[0-9]*) ;;
  *) echo 'Invalid AGENT_PING_CURSOR_VERSION.' >&2; exit 1 ;;
esac
case "$agent_ping_version" in *[!A-Za-z0-9.-]*) echo 'Invalid release version.' >&2; exit 1 ;; esac
if [ "$(uname -s)" != Darwin ]; then
  echo 'Agent Ping for Cursor currently supports macOS only.' >&2
  exit 1
fi
case "$(uname -m)" in
  arm64) agent_ping_arch=arm64 ;;
  x86_64) agent_ping_arch=amd64 ;;
  *) echo 'Unsupported processor architecture.' >&2; exit 1 ;;
esac
agent_ping_major=$(sw_vers -productVersion | cut -d . -f 1)
if [ "$agent_ping_major" -lt 13 ]; then
  echo 'Agent Ping for Cursor requires macOS 13 or newer.' >&2
  exit 1
fi
for agent_ping_command in curl tar shasum; do
  command -v "$agent_ping_command" >/dev/null 2>&1 || { echo "Required command not found: $agent_ping_command" >&2; exit 1; }
done

agent_ping_temp=$(mktemp -d)
agent_ping_staged=
trap 'rm -rf "$agent_ping_temp"; if [ -n "$agent_ping_staged" ]; then rm -f "$agent_ping_staged"; fi' EXIT HUP INT TERM
agent_ping_asset="agent-ping-cursor_${agent_ping_version}_darwin_${agent_ping_arch}.tar.gz"
agent_ping_url="https://github.com/${agent_ping_repo}/releases/download/${agent_ping_version}"
echo "Downloading Agent Ping for Cursor ${agent_ping_version} for ${agent_ping_arch}..."
curl --proto '=https' --tlsv1.2 -fsSL --retry 3 "$agent_ping_url/$agent_ping_asset" -o "$agent_ping_temp/$agent_ping_asset"
curl --proto '=https' --tlsv1.2 -fsSL --retry 3 "$agent_ping_url/checksums.txt" -o "$agent_ping_temp/checksums.txt"
agent_ping_expected=$(awk -v asset="$agent_ping_asset" '$2 == asset { print $1 }' "$agent_ping_temp/checksums.txt")
agent_ping_actual=$(shasum -a 256 "$agent_ping_temp/$agent_ping_asset" | awk '{print $1}')
if [ -z "$agent_ping_expected" ] || [ "$agent_ping_actual" != "$agent_ping_expected" ]; then
  echo 'Download checksum verification failed; nothing was installed.' >&2
  exit 1
fi
tar -xzf "$agent_ping_temp/$agent_ping_asset" -C "$agent_ping_temp" agent-ping-cursor
mkdir -p "$agent_ping_bin_dir"
agent_ping_bin_dir=$(cd "$agent_ping_bin_dir" && pwd -P)
if [ -L "$agent_ping_bin_dir/agent-ping-cursor" ] || [ -d "$agent_ping_bin_dir/agent-ping-cursor" ]; then
  echo 'The destination agent-ping-cursor is a directory or symbolic link; nothing was overwritten.' >&2
  exit 1
fi
agent_ping_staged=$(mktemp "$agent_ping_bin_dir/.agent-ping-cursor.XXXXXX")
cp "$agent_ping_temp/agent-ping-cursor" "$agent_ping_staged"
chmod 755 "$agent_ping_staged"
mv -f "$agent_ping_staged" "$agent_ping_bin_dir/agent-ping-cursor"
agent_ping_staged=
echo "Installed: $agent_ping_bin_dir/agent-ping-cursor"
"$agent_ping_bin_dir/agent-ping-cursor" install "$@"
case ":$PATH:" in
  *":$agent_ping_bin_dir:"*) ;;
  *) printf '\nAdd this directory to your PATH to use agent-ping-cursor from any terminal:\n  %s\n' "$agent_ping_bin_dir" ;;
esac
