package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

// loadForWrite reads the user file for modification.
// A missing file gives an empty tree (exists=false). Any other problem is returned
// as an error and the file must not be touched.
func loadForWrite(path string) (tree map[string]any, exists bool, problem string) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, false, ""
	}
	t, p := readUserFile(path)
	if p != "" {
		return nil, true, p
	}
	return t, true, ""
}

// writeAtomic writes data to path through a temporary file in the same directory.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, ".config-*.tmp")
	if err != nil {
		return err
	}
	name := tmp.Name()
	cleanup := func() { _ = os.Remove(name) }
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		cleanup()
		return err
	}
	if err := tmp.Close(); err != nil {
		cleanup()
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		cleanup()
		return err
	}
	if err := os.Rename(name, path); err != nil {
		cleanup()
		return err
	}
	return nil
}

func saveTree(path string, tree map[string]any) error {
	b, err := json.MarshalIndent(tree, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(path, append(b, '\n'))
}

func runSet(in map[string]any) {
	userFile, _ := in["userFile"].(string)
	dataDir, _ := in["dataDir"].(string)
	changes, _ := in["changes"].([]any)
	if userFile == "" || len(changes) == 0 {
		fail("bad_input", "userFile and a non-empty changes list are required")
		return
	}

	type change struct {
		key   string
		value any
	}
	var list []change
	for i, c := range changes {
		m, ok := c.(map[string]any)
		key, _ := m["key"].(string)
		value, hasValue := m["value"]
		if !ok || key == "" || !hasValue {
			fail("bad_input", "changes[%d] needs a key and a value", i)
			return
		}
		r := findRule(key)
		if r == nil {
			fail("invalid_key", "unknown key %q", key)
			return
		}
		if p := r.check(value, dataDir); p != "" {
			fail("invalid_value", "%s: %s", key, p)
			return
		}
		list = append(list, change{key, value})
	}

	tree, _, problem := loadForWrite(userFile)
	if problem != "" {
		fail("file_invalid", "%s; not modified", problem)
		return
	}
	for _, c := range list {
		setPath(tree, c.key, c.value)
	}
	if err := saveTree(userFile, tree); err != nil {
		fail("write_failed", "cannot write %s: %v", userFile, err)
		return
	}
	logf("info", "saved overrides", map[string]any{"count": len(list)})
	succeed(map[string]any{"written": true})
}

func runUnset(in map[string]any) {
	userFile, _ := in["userFile"].(string)
	keys, _ := in["keys"].([]any)
	if userFile == "" || len(keys) == 0 {
		fail("bad_input", "userFile and a non-empty keys list are required")
		return
	}
	var names []string
	for _, k := range keys {
		s, ok := k.(string)
		if !ok || s == "" {
			fail("bad_input", "keys must be non-empty strings")
			return
		}
		names = append(names, s)
	}

	tree, exists, problem := loadForWrite(userFile)
	if problem != "" {
		fail("file_invalid", "%s; not modified", problem)
		return
	}
	if !exists {
		succeed(map[string]any{"written": false})
		return
	}
	changed := false
	for _, k := range names {
		if deletePath(tree, k) {
			changed = true
		}
	}
	if !changed {
		succeed(map[string]any{"written": false})
		return
	}
	if err := saveTree(userFile, tree); err != nil {
		fail("write_failed", "cannot write %s: %v", userFile, err)
		return
	}
	logf("info", "removed overrides", map[string]any{"count": len(names)})
	succeed(map[string]any{"written": true})
}
