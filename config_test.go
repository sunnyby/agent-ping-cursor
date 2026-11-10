package main

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestInstallUpdateUninstallPreservesOtherHooks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hooks.json")
	original := []byte(`{"version":1,"custom":{"large":9007199254740993},"hooks":{"stop":[{"command":"existing-stop","timeout":20,"matcher":"*"}],"afterFileEdit":[{"command":"existing-edit"}]}}`)
	if err := os.WriteFile(path, original, 0640); err != nil {
		t.Fatal(err)
	}
	ping := hookCommand("/path with spaces/agent-ping-cursor", "Ping")
	glass := hookCommand("/path with spaces/agent-ping-cursor", "Glass")
	for _, step := range []struct {
		command string
		changed bool
	}{{ping, true}, {ping, false}, {glass, true}} {
		changed, err := configure(dir, step.command)
		if err != nil || changed != step.changed {
			t.Fatalf("configure changed=%v, err=%v", changed, err)
		}
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	config, err := parseHooks(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(config.stop) != 2 || ownedHook(config.stop[0]) || !ownedHook(config.stop[1]) || !bytes.Contains(config.stop[1], []byte("Glass")) {
		t.Fatal("expected the original hook and one managed hook with the updated sound")
	}
	if !bytes.Contains(data, []byte("9007199254740993")) {
		t.Fatal("unrelated JSON value changed")
	}
	// Simulate Cursor or the user adding another entry while Agent Ping is installed.
	config.stop = append(config.stop, json.RawMessage(`{"command":"new-stop"}`))
	config.hooks["stop"], _ = json.Marshal(config.stop)
	config.root["hooks"], _ = json.Marshal(config.hooks)
	data, _ = json.Marshal(config.root)
	if err := os.WriteFile(path, data, 0640); err != nil {
		t.Fatal(err)
	}
	if changed, err := configure(dir, ""); err != nil || !changed {
		t.Fatalf("uninstall failed: %v", err)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	config, err = parseHooks(data)
	if err != nil {
		t.Fatal(err)
	}
	var commands []string
	for _, entry := range config.stop {
		var hook struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(entry, &hook); err != nil {
			t.Fatal(err)
		}
		commands = append(commands, hook.Command)
	}
	if !reflect.DeepEqual(commands, []string{"existing-stop", "new-stop"}) || !bytes.Contains(data, []byte("existing-edit")) {
		t.Fatal("an unrelated hook changed")
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0640 {
		t.Fatal("file permissions changed")
	}
}

func TestMalformedAndSymlinkConfigsArePreserved(t *testing.T) {
	for _, original := range []string{`{bad`, `null`, `{"version":2}`, `{"hooks":{"stop":null}}`} {
		dir := t.TempDir()
		path := filepath.Join(dir, "hooks.json")
		if err := os.WriteFile(path, []byte(original), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err := configure(dir, hookCommand("/app", "Ping")); err == nil {
			t.Fatal("invalid config accepted")
		}
		data, err := os.ReadFile(path)
		if err != nil || string(data) != original {
			t.Fatal("invalid config was overwritten")
		}
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "dotfiles.json")
	if err := os.WriteFile(target, []byte(`{"version":1,"hooks":{}}`), 0600); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "hooks.json")
	if err := os.Symlink(target, path); err != nil {
		t.Fatal(err)
	}
	if _, err := configure(dir, hookCommand("/app", "Ping")); err == nil {
		t.Fatal("symlink accepted")
	}
	if actual, err := os.Readlink(path); err != nil || actual != target {
		t.Fatal("symlink was replaced")
	}
}

func TestHookCommandQuotesShellArguments(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent's $test `path`")
	if err := os.WriteFile(path, []byte("#!/bin/sh\nprintf '%s\\n' \"$@\"\n"), 0700); err != nil {
		t.Fatal(err)
	}
	output, err := exec.Command("/bin/sh", "-c", hookCommand(path, "A sound's $name")).CombinedOutput()
	if err != nil {
		t.Fatalf("command failed: %v: %s", err, output)
	}
	if string(output) != "hook\n--sound\nA sound's $name\n" {
		t.Fatalf("unexpected arguments: %q", output)
	}
}

func TestStopEventPlaysOnlyOnSuccessfulCompletion(t *testing.T) {
	for _, test := range []struct {
		input string
		count int
	}{
		{`{"hook_event_name":"stop","status":"completed","loop_count":0}`, 1},
		{`{"status":"completed"}`, 1},
		{`{"hook_event_name":"subagentStop","status":"completed"}`, 0},
		{`{"status":"aborted"}`, 0}, {`{"status":"error"}`, 0},
		{`{"status":"completed"} {}`, 0}, {`{`, 0},
	} {
		var out bytes.Buffer
		count := 0
		handleStop(strings.NewReader(test.input), &out, func() { count++ })
		if count != test.count || out.String() != "{}\n" {
			t.Fatalf("input %q: played %d, output %q", test.input, count, out.String())
		}
	}
}
