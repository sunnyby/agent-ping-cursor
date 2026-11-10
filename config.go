package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"syscall"
)

const hookMarker = " # agent-ping-cursor:stop:v1"

type object = map[string]json.RawMessage

type hooksConfig struct {
	root  object
	hooks object
	stop  []json.RawMessage
}

func parseHooks(data []byte) (hooksConfig, error) {
	config := hooksConfig{root: object{}, hooks: object{}}
	if data == nil {
		config.root["version"] = json.RawMessage("1")
		return config, nil
	}
	if json.Unmarshal(data, &config.root) != nil || config.root == nil {
		return config, errors.New("Cursor hooks.json must be a valid JSON object")
	}
	if raw, ok := config.root["version"]; ok {
		var version int
		if json.Unmarshal(raw, &version) != nil || version != 1 {
			return config, errors.New("unsupported Cursor hooks.json version")
		}
	}
	if raw, ok := config.root["hooks"]; ok && (json.Unmarshal(raw, &config.hooks) != nil || config.hooks == nil) {
		return config, errors.New("Cursor hooks must be a JSON object")
	}
	if raw, ok := config.hooks["stop"]; ok && (json.Unmarshal(raw, &config.stop) != nil || config.stop == nil) {
		return config, errors.New("Cursor stop hooks must be a JSON array")
	}
	return config, nil
}

func ownedHook(entry json.RawMessage) bool {
	var hook struct {
		Command string `json:"command"`
	}
	return json.Unmarshal(entry, &hook) == nil && strings.HasSuffix(hook.Command, hookMarker)
}

func hookCommand(executable, sound string) string {
	quote := func(arg string) string {
		return "'" + strings.ReplaceAll(arg, "'", "'\"'\"'") + "'"
	}
	return quote(executable) + " hook --sound " + quote(sound) + hookMarker
}

// changeHooks adds or replaces the managed stop hook. An empty command removes it.
func changeHooks(data []byte, command string) ([]byte, bool, error) {
	config, err := parseHooks(data)
	if err != nil {
		return nil, false, err
	}
	var replacement json.RawMessage
	if command != "" {
		replacement, err = json.Marshal(struct {
			Command string `json:"command"`
			Timeout int    `json:"timeout"`
		}{command, 5})
		if err != nil {
			return nil, false, err
		}
	}
	stop := make([]json.RawMessage, 0, len(config.stop)+1)
	owned := 0
	for _, entry := range config.stop {
		if !ownedHook(entry) {
			stop = append(stop, entry)
			continue
		}
		owned++
		if owned == 1 && replacement != nil {
			stop = append(stop, replacement)
		}
	}
	if owned == 0 && replacement != nil {
		stop = append(stop, replacement)
	}
	if owned == 0 && replacement == nil {
		return data, false, nil
	}
	if len(stop) == 0 {
		delete(config.hooks, "stop")
	} else {
		config.hooks["stop"], err = json.Marshal(stop)
		if err != nil {
			return nil, false, err
		}
	}
	if _, ok := config.root["version"]; !ok {
		config.root["version"] = json.RawMessage("1")
	}
	config.root["hooks"], err = json.Marshal(config.hooks)
	if err != nil {
		return nil, false, err
	}
	updated, err := json.MarshalIndent(config.root, "", "  ")
	if err != nil {
		return nil, false, err
	}
	updated = append(updated, '\n')
	return updated, !bytes.Equal(data, updated), nil
}

func readConfigFile(path string) ([]byte, os.FileMode, error) {
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, 0600, nil
	}
	if err != nil {
		return nil, 0, err
	}
	if !info.Mode().IsRegular() {
		return nil, 0, errors.New("Cursor hooks.json must be a regular file")
	}
	data, err := os.ReadFile(path)
	return data, info.Mode().Perm(), err
}

func configure(dir, command string) (bool, error) {
	if command != "" {
		if err := os.MkdirAll(dir, 0700); err != nil {
			return false, err
		}
	}
	lock, err := os.OpenFile(filepath.Join(dir, ".agent-ping-cursor.lock"), os.O_CREATE|os.O_RDWR, 0600)
	if command == "" && errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	defer lock.Close()
	if err := syscall.Flock(int(lock.Fd()), syscall.LOCK_EX|syscall.LOCK_NB); err != nil {
		return false, errors.New("could not lock the Cursor hook configuration; try again")
	}
	path := filepath.Join(dir, "hooks.json")
	original, mode, err := readConfigFile(path)
	if err != nil {
		return false, err
	}
	updated, changed, err := changeHooks(original, command)
	if err != nil || !changed {
		return false, err
	}
	latest, latestMode, err := readConfigFile(path)
	if err != nil {
		return false, err
	}
	if !bytes.Equal(original, latest) || (original == nil) != (latest == nil) || mode != latestMode {
		return false, errors.New("Cursor hooks.json changed while it was being edited; try again")
	}
	if err := atomicWrite(path, updated, mode); err != nil {
		return false, err
	}
	return true, nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	file, err := os.CreateTemp(filepath.Dir(path), ".agent-ping-cursor-*")
	if err != nil {
		return err
	}
	name := file.Name()
	defer os.Remove(name)
	if err = file.Chmod(mode); err == nil {
		_, err = file.Write(data)
	}
	if err == nil {
		err = file.Sync()
	}
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	return os.Rename(name, path)
}
