package main

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"
)

// Files (spec FR-012 to FR-018, research R9). A query is searched in stages; the source runs
// the stages in parallel and shows them in a fixed order.

const (
	storedMatches = 20000 // documented constant: matches kept for ranking per stage
	checkEvery    = 256   // documented constant: entries between two looks at the clock
)

// neverWalk are never searched, even if a setting lists them.
var neverWalk = []string{"/proc", "/sys", "/dev", "/run", "/tmp"}

var priorityDirs = []string{"Desktop", "Downloads", "Documents"}

var configDirs = []string{".cache", ".local", ".config"}

type fileMatch struct {
	path  string
	name  string
	kind  string
	group int
	key   rankKey
}

// walkState is shared with the walking goroutine, which may outlive the deadline.
type walkState struct {
	mu      sync.Mutex
	matches []fileMatch
	total   int
	scanned int
	cut     bool
	skipped int
	stop    bool
	seen    map[[2]uint64]bool
	dirs    map[[2]uint64]bool
}

func (w *walkState) stopped() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.stop
}

func opFiles(req *request) (any, *failure) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return map[string]any{"rows": []row{}, "total": 0, "cut": false, "hint": "type a file name"}, nil
	}
	words := queryWords(name)
	roots, ferr := stageRoots(req)
	if ferr != nil {
		return nil, ferr
	}
	maxFiles := req.MaxFiles
	if maxFiles <= 0 {
		maxFiles = defaultMaxFiles
	}
	st := &walkState{seen: map[[2]uint64]bool{}, dirs: map[[2]uint64]bool{}}
	done := make(chan struct{})
	deadline := time.Now().Add(budgetOf(req))
	go func() {
		defer close(done)
		for i, r := range roots {
			walkRoot(st, req, r, i, words, maxFiles, deadline)
			if st.stopped() {
				return
			}
		}
	}()
	select {
	case <-done:
	case <-time.After(time.Until(deadline) + 50*time.Millisecond):
		st.mu.Lock()
		st.cut = true
		st.stop = true
		st.mu.Unlock()
	}
	st.mu.Lock()
	matches := append([]fileMatch(nil), st.matches...)
	total, cut, skipped := st.total, st.cut, st.skipped
	st.mu.Unlock()
	if skipped > 0 {
		logDiag("warn", "skipped unreadable folders", map[string]any{"stage": req.Stage, "count": skipped})
	}
	return map[string]any{"rows": rowsOf(matches, limitOf(req)), "total": total, "cut": cut}, nil
}

type walkRootSpec struct {
	path  string
	depth int  // 0 means unlimited; 1 means only the entries directly inside
	root  bool // the root itself is searched even when its name is hidden
}

// stageRoots lists what a stage walks, in the order its results are grouped.
func stageRoots(req *request) ([]walkRootSpec, *failure) {
	h := userHome
	switch req.Stage {
	case "priority":
		var out []walkRootSpec
		for _, d := range priorityDirs {
			out = append(out, walkRootSpec{path: filepath.Join(h, d)})
		}
		return out, nil
	case "home":
		return []walkRootSpec{{path: h, depth: 1}}, nil
	case "homeDirs":
		entries, err := os.ReadDir(h)
		if err != nil {
			return nil, nil
		}
		var out []walkRootSpec
		for _, e := range entries {
			n := e.Name()
			if !e.IsDir() || contains(priorityDirs, n) {
				continue
			}
			if strings.HasPrefix(n, ".") {
				if contains(configDirs, n) || !req.HiddenHomeDirs {
					continue
				}
			}
			out = append(out, walkRootSpec{path: filepath.Join(h, n)})
		}
		return out, nil
	case "dotDirs":
		var out []walkRootSpec
		for _, d := range configDirs {
			out = append(out, walkRootSpec{path: filepath.Join(h, d), root: true})
		}
		return out, nil
	case "system":
		var out []walkRootSpec
		for _, r := range req.SystemRoots {
			if filepath.IsAbs(r) && !refusedRoot(r) {
				out = append(out, walkRootSpec{path: filepath.Clean(r)})
			}
		}
		return out, nil
	case "dir":
		if req.Dir == "" {
			return nil, fail("bad_request", "stage dir needs a directory")
		}
		return []walkRootSpec{{path: req.Dir, root: true}}, nil
	}
	return nil, fail("bad_request", "unknown stage")
}

func refusedRoot(p string) bool {
	p = filepath.Clean(p)
	for _, n := range neverWalk {
		if p == n || strings.HasPrefix(p, n+"/") {
			return true
		}
	}
	return false
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func identity(info fs.FileInfo) ([2]uint64, bool) {
	st, ok := info.Sys().(*syscall.Stat_t)
	if !ok {
		return [2]uint64{}, false
	}
	return [2]uint64{uint64(st.Dev), uint64(st.Ino)}, true
}

// walkRoot walks one root, adding matches. It never follows symbolic links.
func walkRoot(st *walkState, req *request, spec walkRootSpec, group int, words []string, maxFiles int, deadline time.Time) {
	hidden := req.Hidden
	rootInfo, err := os.Lstat(spec.path)
	if err != nil || !rootInfo.IsDir() {
		return
	}
	_ = filepath.WalkDir(spec.path, func(p string, d fs.DirEntry, err error) error {
		st.mu.Lock()
		if st.stop {
			st.mu.Unlock()
			return filepath.SkipAll
		}
		st.scanned++
		n := st.scanned
		if n > maxFiles {
			st.cut = true
			st.stop = true
			st.mu.Unlock()
			return filepath.SkipAll
		}
		st.mu.Unlock()
		if n%checkEvery == 0 && time.Now().After(deadline) {
			st.mu.Lock()
			st.cut = true
			st.stop = true
			st.mu.Unlock()
			return filepath.SkipAll
		}
		if err != nil {
			st.mu.Lock()
			st.skipped++
			st.mu.Unlock()
			if d != nil && d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if p == spec.path {
			return nil
		}
		depth := strings.Count(strings.TrimPrefix(p, spec.path), "/")
		if spec.depth > 0 && depth > spec.depth {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		name := d.Name()
		if d.IsDir() && contains(neverWalk, p) {
			// A walk that starts above /proc, /sys, /dev, /run or /tmp does not enter them
			// (a walk that starts inside one is the user's own choice).
			return filepath.SkipDir
		}
		if strings.HasPrefix(name, ".") && !hidden {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if d.IsDir() {
			if spec.depth == 1 {
				// directly inside the home: the folder itself can match, but is not entered
				recordMatch(st, p, name, "directory", group, words, d)
				return filepath.SkipDir
			}
			if info, ierr := d.Info(); ierr == nil {
				if id, ok := identity(info); ok {
					st.mu.Lock()
					if st.dirs[id] {
						st.mu.Unlock()
						return filepath.SkipDir
					}
					st.dirs[id] = true
					st.mu.Unlock()
				}
			}
		}
		kind := "file"
		if d.IsDir() {
			kind = "directory"
		}
		recordMatch(st, p, name, kind, group, words, d)
		return nil
	})
}

func recordMatch(st *walkState, p, name, kind string, group int, words []string, d fs.DirEntry) {
	ok, key := matchName(words, name)
	if !ok {
		return
	}
	info, err := d.Info()
	if err == nil {
		if id, has := identity(info); has && kind == "file" {
			st.mu.Lock()
			if st.seen[id] {
				st.mu.Unlock()
				return
			}
			st.seen[id] = true
			st.mu.Unlock()
		}
	}
	if d.Type()&fs.ModeSymlink != 0 {
		if target, err := os.Stat(p); err == nil && target.IsDir() {
			kind = "directory"
		}
	}
	st.mu.Lock()
	st.total++
	if len(st.matches) < storedMatches {
		st.matches = append(st.matches, fileMatch{path: p, name: name, kind: kind, group: group, key: key})
	}
	st.mu.Unlock()
}

// rowsOf orders matches by group, rank and path and turns the first limit into rows.
func rowsOf(matches []fileMatch, limit int) []row {
	list := make([]scored, 0, len(matches))
	for _, m := range matches {
		icon := "text-x-generic"
		if m.kind == "directory" {
			icon = "folder"
		}
		list = append(list, scored{
			r:    row{Key: m.path, Title: m.name, Subtitle: shorten(filepath.Dir(m.path)), Icon: icon},
			key:  rankKey{tier: m.group*10 + m.key.tier, pos: m.key.pos, length: m.key.length},
			sort: m.path,
		})
	}
	sortScored(list)
	rows := []row{}
	for i := 0; i < len(list) && i < limit; i++ {
		rows = append(rows, list[i].r)
	}
	return rows
}
