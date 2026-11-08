package main

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// maxDirectories bounds a scan (both directories together).
const maxDirectories = 256

// scanned is one entry of the scan result.
type scanned struct {
	ID             string         `json:"id"`
	Dir            string         `json:"dir"`
	DirName        string         `json:"dirName"`
	Origin         string         `json:"origin"`
	Valid          bool           `json:"valid"`
	DefaultEnabled bool           `json:"defaultEnabled"`
	Main           string         `json:"main,omitempty"`
	Manifest       map[string]any `json:"manifest"`
	Errors         []problem      `json:"errors"`
	Warnings       []problem      `json:"warnings"`
	// Interface 1.1. Only set for valid modules: an invalid module registers nothing.
	ExtensionPoints []pointDecl   `json:"extensionPoints,omitempty"`
	Contributions   []contribDecl `json:"contributions,omitempty"`
	Commands        []string      `json:"commands,omitempty"`

	points   []pointDecl
	contribs []contribDecl
	commands []string
}

var unsafeDirChars = regexp.MustCompile(`[^A-Za-z0-9._-]`)

// dirID is the id shown for a module whose manifest has no usable id:
// "dir:" plus the directory name with unsafe characters replaced by "_", cut to 64.
func dirID(dirName string) string {
	name := unsafeDirChars.ReplaceAllString(dirName, "_")
	if len(name) > 64 {
		name = name[:64]
	}
	return "dir:" + name
}

// listDirs returns the visible subdirectories of dir, sorted by name. A missing or
// unreadable directory gives none (it is not an error, and nothing is created).
func listDirs(dir string) []string {
	if dir == "" {
		return nil
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		// Follow symlinks to directories.
		st, err := os.Stat(filepath.Join(dir, e.Name()))
		if err != nil || !st.IsDir() {
			continue
		}
		names = append(names, e.Name())
	}
	sort.Strings(names)
	return names
}

// scanOne reads and validates one module directory.
func scanOne(root, name, origin string, lumen apiVersion) scanned {
	abs, err := filepath.Abs(filepath.Join(root, name))
	if err != nil {
		abs = filepath.Join(root, name)
	}
	s := scanned{
		Dir: abs, DirName: name, Origin: origin, DefaultEnabled: origin == "bundled",
		Manifest: map[string]any{}, Errors: []problem{}, Warnings: []problem{},
	}

	manifestPath := filepath.Join(abs, "module.json")
	st, err := os.Stat(manifestPath)
	switch {
	case err != nil:
		s.Errors = append(s.Errors, problem{"", "module.json not found in " + abs})
	case st.Size() > maxManifestBytes:
		s.Errors = append(s.Errors, problem{"", "manifest is larger than 64 KiB"})
	default:
		b, err := os.ReadFile(manifestPath)
		if err != nil {
			s.Errors = append(s.Errors, problem{"", "cannot read module.json: " + err.Error()})
			break
		}
		m, errs, warns := parseManifest(b, lumen)
		s.Manifest, s.Errors, s.Warnings = m.raw, append(s.Errors, errs...), append(s.Warnings, warns...)
		s.ID = m.id
		s.points, s.contribs, s.commands = m.points, m.contribs, m.commands
		s.checkFiles(m)
	}

	if s.ID == "" {
		s.ID = dirID(name)
		if s.ID != "dir:"+name {
			s.Errors = append(s.Errors, problem{"", fmt.Sprintf("directory name %q was shown as %s", name, s.ID)})
		}
	}
	return s
}

// checkFiles checks that the entry file and the declared helper files exist.
func (s *scanned) checkFiles(m *manifest) {
	if len(s.Errors) > 0 && hasFieldError(s.Errors, "main") {
		return
	}
	main := filepath.Join(s.Dir, m.main)
	if st, err := os.Stat(main); err != nil || !st.Mode().IsRegular() {
		s.Errors = append(s.Errors, problem{"main", fmt.Sprintf("entry file %s not found", m.main)})
	} else {
		s.Main = main
	}
	for i, c := range m.contribs {
		if c.File == "" {
			continue // already reported by the manifest rules
		}
		field := fmt.Sprintf("contributions[%d].file", i)
		if st, err := os.Stat(filepath.Join(s.Dir, c.File)); err != nil || !st.Mode().IsRegular() {
			s.Errors = append(s.Errors, problem{field, fmt.Sprintf("contribution file %s not found", c.File)})
		}
	}
	for i, h := range m.helpers {
		field := fmt.Sprintf("helpers[%d].path", i)
		st, err := os.Stat(filepath.Join(s.Dir, h.Path))
		switch {
		case err != nil || !st.Mode().IsRegular():
			s.Errors = append(s.Errors, problem{field, fmt.Sprintf("helper file %s not found", h.Path)})
		case st.Mode().Perm()&0o111 == 0:
			s.Errors = append(s.Errors, problem{field, fmt.Sprintf("helper file %s is not executable", h.Path)})
		}
	}
}

// publishInterface11 copies the extension points, contributions and commands of a valid
// module into the result: files made absolute, the long name set, and the short name given
// to the first-declared contribution per target point (specs/003-bar/contracts/helper-bar.md).
func (s *scanned) publishInterface11() {
	s.ExtensionPoints = s.points
	s.Commands = s.commands
	firstFor := map[string]bool{}
	for _, c := range s.contribs {
		c.File = filepath.Join(s.Dir, c.File)
		c.Key = s.ID + "/" + c.ID
		target := c.Target.Module + "/" + c.Target.Point
		if !firstFor[target] {
			firstFor[target] = true
			c.Short = s.ID
		}
		s.Contributions = append(s.Contributions, c)
	}
}

func hasFieldError(list []problem, field string) bool {
	for _, p := range list {
		if p.Field == field {
			return true
		}
	}
	return false
}

// scanAll scans both directories, bundled first, then user, each sorted by directory
// name, and applies the duplicate-id rule: the first module with an id keeps it.
func scanAll(bundledDir, userDir string, lumen apiVersion) []scanned {
	var out []scanned
	count := 0
	tooMany := false
	for _, set := range []struct{ dir, origin string }{{bundledDir, "bundled"}, {userDir, "user"}} {
		for _, name := range listDirs(set.dir) {
			if count >= maxDirectories {
				tooMany = true
				break
			}
			count++
			out = append(out, scanOne(set.dir, name, set.origin, lumen))
		}
	}

	owner := map[string]string{} // usable id -> directory of the first module with it
	for i := range out {
		m := &out[i]
		if strings.HasPrefix(m.ID, "dir:") {
			continue
		}
		if first, taken := owner[m.ID]; taken {
			m.Errors = append(m.Errors, problem{"id", fmt.Sprintf("duplicate of the module in %s (this one is in %s); the first one wins, bundled before user", first, m.Dir)})
			continue
		}
		owner[m.ID] = m.Dir
	}
	for i := range out {
		out[i].Valid = len(out[i].Errors) == 0
		if !out[i].Valid {
			out[i].Main = ""
			continue
		}
		out[i].publishInterface11()
	}
	if tooMany {
		out = append(out, scanned{
			ID: "dir:too-many-modules", Origin: "user", Valid: false, Manifest: map[string]any{},
			Errors:   []problem{{"", fmt.Sprintf("too many modules: only the first %d directories were scanned", maxDirectories)}},
			Warnings: []problem{},
		})
	}
	return out
}

func runScan(in map[string]any) {
	bundledDir, _ := in["bundledDir"].(string)
	userDir, _ := in["userDir"].(string)
	apiStr, _ := in["apiVersion"].(string)
	lumen, ok := parseAPI(apiStr)
	if !ok {
		fail("bad_input", "apiVersion must look like 1.0")
		return
	}
	mods := scanAll(bundledDir, userDir, lumen)
	logf("info", "scanned modules", map[string]any{"count": len(mods)})
	succeed(map[string]any{"apiVersion": lumen.String(), "modules": mods})
}
