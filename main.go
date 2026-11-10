package main

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"time"
)

var version = "dev"

const help = `Agent Ping for Cursor

  agent-ping-cursor install [--sound Ping] [--cursor-dir PATH]
  agent-ping-cursor status [--cursor-dir PATH]
  agent-ping-cursor test [--sound Ping]
  agent-ping-cursor uninstall [--cursor-dir PATH]
  agent-ping-cursor version

Plays a sound after a successful Cursor Agent response on macOS.
`

func main() {
	if err := run(os.Args[1:], os.Stdin, os.Stdout, os.Stderr); err != nil {
		fmt.Fprintln(os.Stderr, "agent-ping-cursor:", err)
		os.Exit(1)
	}
}

func run(args []string, in io.Reader, out, errOut io.Writer) error {
	if len(args) == 0 || args[0] == "--help" || args[0] == "-h" || args[0] == "help" {
		fmt.Fprint(out, help)
		return nil
	}
	command := args[0]
	if command == "version" || command == "--version" {
		fmt.Fprintln(out, "agent-ping-cursor", version)
		return nil
	}
	switch command {
	case "install", "uninstall", "status", "test", "hook":
	default:
		return errors.New("unknown command; run agent-ping-cursor --help")
	}
	if runtime.GOOS != "darwin" {
		return errors.New("this release supports macOS only")
	}
	flags := flag.NewFlagSet("agent-ping-cursor "+command, flag.ContinueOnError)
	flags.SetOutput(errOut)
	sound := "Ping"
	if command == "install" || command == "test" || command == "hook" {
		flags.StringVar(&sound, "sound", "Ping", "built-in macOS sound name")
	}
	dir := ""
	if command == "install" || command == "uninstall" || command == "status" {
		flags.StringVar(&dir, "cursor-dir", "", "directory containing hooks.json (default ~/.cursor)")
	}
	if err := flags.Parse(args[1:]); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	if flags.NArg() != 0 {
		return errors.New("unexpected argument")
	}
	if command == "hook" {
		handleStop(in, out, func() { startSound(sound) })
		return nil
	}
	if command == "test" {
		path, err := soundPath(sound)
		if err != nil {
			return err
		}
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := exec.CommandContext(ctx, "/usr/bin/afplay", path).Run(); err != nil {
			return errors.New("sound playback failed; check your Mac's audio output")
		}
		fmt.Fprintln(out, "Sound playback completed.")
		return nil
	}
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		dir = filepath.Join(home, ".cursor")
	}
	if command == "status" {
		data, _, err := readConfigFile(filepath.Join(dir, "hooks.json"))
		if err != nil {
			return err
		}
		config, err := parseHooks(data)
		if err != nil {
			return err
		}
		for _, entry := range config.stop {
			if ownedHook(entry) {
				fmt.Fprintln(out, "Agent Ping is configured in Cursor hooks.json.")
				return nil
			}
		}
		fmt.Fprintln(out, "Agent Ping is not configured in this Cursor directory.")
		return nil
	}
	if command == "uninstall" {
		changed, err := configure(dir, "")
		if err != nil {
			return err
		}
		if changed {
			fmt.Fprintln(out, "Removed the Agent Ping hook.")
		} else {
			fmt.Fprintln(out, "Agent Ping was already disconnected.")
		}
		return nil
	}
	if _, err := soundPath(sound); err != nil {
		return err
	}
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	changed, err := configure(dir, hookCommand(executable, sound))
	if err != nil {
		return err
	}
	if changed {
		fmt.Fprintln(out, "Connected to Cursor. Sound:", sound)
	} else {
		fmt.Fprintln(out, "Agent Ping is already configured.")
	}
	return nil
}

func soundPath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "/\\\x00\r\n") {
		return "", errors.New("use a built-in macOS sound name, such as Ping or Glass")
	}
	path := filepath.Join("/System/Library/Sounds", name+".aiff")
	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		return "", errors.New("that macOS sound was not found; try --sound Ping")
	}
	return path, nil
}

func handleStop(in io.Reader, out io.Writer, play func()) {
	defer fmt.Fprintln(out, "{}")
	const maxSize = 1 << 20
	data, err := io.ReadAll(io.LimitReader(in, maxSize+1))
	if err != nil || len(data) > maxSize {
		return
	}
	var event struct {
		Name   string `json:"hook_event_name"`
		Status string `json:"status"`
	}
	if json.Unmarshal(data, &event) == nil && event.Status == "completed" && (event.Name == "" || event.Name == "stop") {
		play()
	}
}

func startSound(name string) {
	path, err := soundPath(name)
	if err != nil {
		return
	}
	player := exec.Command("/usr/bin/afplay", path)
	player.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if player.Start() == nil {
		_ = player.Process.Release()
	}
}
