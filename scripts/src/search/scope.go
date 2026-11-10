package main

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// opScope finds the directory named in a files query (spec FR-014, research R9). A word that is
// "~", or starts with "/", "~/" or "./", and names an existing readable directory is the
// directory; the other words are the file name. A "./" path is taken from the home.
func opScope(req *request) (any, *failure) {
	words := splitWords(req.Query)
	dir := ""
	dirIndex := -1
	hint := ""
	for i, w := range words {
		if !pathLike(w) {
			continue
		}
		resolved := resolvePath(w)
		if info, err := os.Stat(resolved); err == nil && info.IsDir() && readableDir(resolved) {
			dir = resolved
			dirIndex = i
			break
		}
		if hint == "" {
			if info, err := os.Stat(resolved); err != nil || info.IsDir() {
				hint = "directory not found: " + w
			}
		}
	}
	var name []string
	for i, w := range words {
		if i != dirIndex {
			name = append(name, w)
		}
	}
	if dir != "" {
		hint = ""
	}
	return map[string]any{"dir": dir, "name": strings.Join(name, " "), "hint": hint}, nil
}

func pathLike(w string) bool {
	return w == "~" || strings.HasPrefix(w, "/") || strings.HasPrefix(w, "~/") || strings.HasPrefix(w, "./")
}

// resolvePath turns a path-like word into an absolute, cleaned path.
func resolvePath(w string) string {
	switch {
	case strings.HasPrefix(w, "./"):
		return filepath.Clean(filepath.Join(userHome, w[2:]))
	default:
		return filepath.Clean(expandHome(w))
	}
}

func readableDir(p string) bool {
	f, err := os.Open(p)
	if err != nil {
		return false
	}
	defer f.Close()
	_, err = f.ReadDir(1)
	return err == nil || errors.Is(err, io.EOF)
}
