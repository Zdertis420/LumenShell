package main

import (
	"os"
	"path/filepath"
	"syscall"
)

// Commands (spec FR-019 to FR-021, research R11): executables in the default command folders,
// searched by name. Nothing is run.

func defaultCommandDirs() []string {
	return []string{
		filepath.Join(userHome, ".local", "bin"),
		filepath.Join(userHome, "bin"),
		"/usr/local/bin", "/usr/local/sbin", "/usr/bin", "/usr/sbin", "/bin", "/sbin", "/opt/bin",
	}
}

func opCommands(req *request) (any, *failure) {
	dirs := req.Dirs
	if len(dirs) == 0 {
		dirs = defaultCommandDirs()
	}
	for _, d := range req.ExtraDirs {
		if filepath.IsAbs(d) {
			dirs = append(dirs, d)
		}
	}
	words := queryWords(req.Query)
	seenDir := map[string]bool{}
	seenName := map[string]bool{}
	var list []scored
	for _, dir := range dirs {
		real, err := filepath.EvalSymlinks(dir)
		if err != nil || seenDir[real] {
			continue
		}
		seenDir[real] = true
		entries, err := os.ReadDir(real)
		if err != nil {
			continue
		}
		for _, e := range entries {
			name := e.Name()
			if seenName[name] {
				continue
			}
			path := filepath.Join(dir, name)
			if !executableFile(path) {
				continue
			}
			seenName[name] = true
			ok, key := matchName(words, name)
			if !ok {
				continue
			}
			list = append(list, scored{r: row{Key: path, Title: name, Subtitle: path, Icon: "utilities-terminal"}, key: key, sort: fold(name)})
		}
	}
	sortScored(list)
	limit := limitOf(req)
	rows := []row{}
	for i := 0; i < len(list) && i < limit; i++ {
		rows = append(rows, list[i].r)
	}
	return map[string]any{"rows": rows, "total": len(list), "cut": len(list) > limit}, nil
}

// executableFile reports whether the path is a regular file (or a link to one) that the user
// may execute.
func executableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return false
	}
	return syscall.Access(path, 1) == nil
}
