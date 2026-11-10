package main

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

// Applications (spec FR-008 to FR-011, research R7): desktop entries of the user and the
// system, parsed here and cached under $XDG_CACHE_HOME/lumen.

type appEntry struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Generic    string `json:"generic"`
	Comment    string `json:"comment"`
	Keywords   string `json:"keywords"`
	Icon       string `json:"icon"`
	Exec       string `json:"exec"`
	Path       string `json:"path"`
	TryExec    string `json:"tryExec"`
	Terminal   bool   `json:"terminal"`
	Hidden     bool   `json:"hidden"`
	NoDisplay  bool   `json:"noDisplay"`
	OnlyShowIn string `json:"onlyShowIn"`
	NotShowIn  string `json:"notShowIn"`
	Type       string `json:"type"`
	SourcePath string `json:"sourcePath"`
}

type appCache struct {
	Stamp   string     `json:"stamp"`
	Entries []appEntry `json:"entries"`
}

// appDirs lists the application folders, the user's first (so that the user's entry wins).
func appDirs() []string {
	dataHome := os.Getenv("XDG_DATA_HOME")
	if dataHome == "" {
		dataHome = filepath.Join(userHome, ".local", "share")
	}
	dirs := []string{filepath.Join(dataHome, "applications")}
	data := os.Getenv("XDG_DATA_DIRS")
	if data == "" {
		data = "/usr/local/share:/usr/share"
	}
	for _, d := range strings.Split(data, ":") {
		if d != "" {
			dirs = append(dirs, filepath.Join(d, "applications"))
		}
	}
	return dirs
}

func cachePath() string {
	base := os.Getenv("XDG_CACHE_HOME")
	if base == "" {
		base = filepath.Join(userHome, ".cache")
	}
	return filepath.Join(base, "lumen", "prism-apps.json")
}

type desktopFile struct {
	id   string
	path string
	info fs.FileInfo
}

// desktopFiles lists the .desktop files of the folders, in order, with the id of each.
func desktopFiles(dirs []string) []desktopFile {
	var out []desktopFile
	for _, dir := range dirs {
		_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
			if err != nil {
				if d != nil && d.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			if d.IsDir() || !strings.HasSuffix(d.Name(), ".desktop") {
				return nil
			}
			info, err := os.Stat(p)
			if err != nil || !info.Mode().IsRegular() {
				return nil
			}
			rel, _ := filepath.Rel(dir, p)
			out = append(out, desktopFile{id: strings.ReplaceAll(rel, string(filepath.Separator), "-"), path: p, info: info})
			return nil
		})
	}
	return out
}

// stampOf identifies the state of the folders and of the settings that change what is parsed.
func stampOf(files []desktopFile) string {
	h := sha256.New()
	h.Write([]byte(os.Getenv("LANG") + "|" + os.Getenv("LC_MESSAGES") + "|" + os.Getenv("LANGUAGE") + "\n"))
	for _, f := range files {
		h.Write([]byte(f.path + "|" + strconv.FormatInt(f.info.Size(), 10) + "|" + strconv.FormatInt(f.info.ModTime().UnixNano(), 10) + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// loadApps returns the parsed entries (first id wins), using the cache while it is valid.
func loadApps() []appEntry {
	files := desktopFiles(appDirs())
	stamp := stampOf(files)
	if b, err := os.ReadFile(cachePath()); err == nil {
		var c appCache
		if json.Unmarshal(b, &c) == nil && c.Stamp == stamp {
			return c.Entries
		}
	}
	seen := map[string]bool{}
	var entries []appEntry
	bad := 0
	for _, f := range files {
		if seen[f.id] {
			continue
		}
		e, ok := parseDesktop(f)
		if !ok {
			bad++
			continue
		}
		seen[f.id] = true
		entries = append(entries, e)
	}
	if bad > 0 {
		logDiag("warn", "skipped unreadable desktop entries", map[string]any{"count": bad})
	}
	writeAppCache(appCache{Stamp: stamp, Entries: entries})
	return entries
}

func writeAppCache(c appCache) {
	b, err := json.Marshal(c)
	if err != nil {
		return
	}
	path := cachePath()
	if os.MkdirAll(filepath.Dir(path), 0o755) != nil {
		return
	}
	tmp := path + ".tmp"
	if os.WriteFile(tmp, b, 0o644) == nil {
		_ = os.Rename(tmp, path)
	}
}

// localeKeys lists the locale suffixes to try for localized keys, most specific first.
func localeKeys() []string {
	lang := os.Getenv("LC_MESSAGES")
	if lang == "" {
		lang = os.Getenv("LANG")
	}
	if i := strings.IndexAny(lang, ".@"); i >= 0 {
		lang = lang[:i]
	}
	if lang == "" || lang == "C" || lang == "POSIX" {
		return nil
	}
	keys := []string{lang}
	if i := strings.Index(lang, "_"); i > 0 {
		keys = append(keys, lang[:i])
	}
	return keys
}

func unescapeValue(v string) string {
	if !strings.Contains(v, `\`) {
		return v
	}
	r := strings.NewReplacer(`\s`, " ", `\n`, "\n", `\t`, "\t", `\r`, "\r", `\\`, `\`)
	return r.Replace(v)
}

// parseDesktop reads the [Desktop Entry] group of a file.
func parseDesktop(f desktopFile) (appEntry, bool) {
	fh, err := os.Open(f.path)
	if err != nil {
		return appEntry{}, false
	}
	defer fh.Close()
	e := appEntry{ID: f.id, SourcePath: f.path}
	locales := localeKeys()
	localized := map[string]map[string]string{}
	in := false
	found := false
	sc := bufio.NewScanner(fh)
	sc.Buffer(make([]byte, 64*1024), 1024*1024)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || line[0] == '#' {
			continue
		}
		if line[0] == '[' {
			in = line == "[Desktop Entry]"
			if in {
				found = true
			}
			continue
		}
		if !in {
			continue
		}
		eq := strings.Index(line, "=")
		if eq < 0 {
			continue
		}
		key, val := strings.TrimSpace(line[:eq]), unescapeValue(strings.TrimSpace(line[eq+1:]))
		base, loc := key, ""
		if i := strings.Index(key, "["); i > 0 && strings.HasSuffix(key, "]") {
			base, loc = key[:i], key[i+1:len(key)-1]
		}
		if loc != "" {
			if localized[base] == nil {
				localized[base] = map[string]string{}
			}
			localized[base][loc] = val
			continue
		}
		switch base {
		case "Type":
			e.Type = val
		case "Name":
			e.Name = val
		case "GenericName":
			e.Generic = val
		case "Comment":
			e.Comment = val
		case "Keywords":
			e.Keywords = strings.ReplaceAll(val, ";", " ")
		case "Icon":
			e.Icon = val
		case "Exec":
			e.Exec = val
		case "Path":
			e.Path = val
		case "TryExec":
			e.TryExec = val
		case "Terminal":
			e.Terminal = val == "true"
		case "Hidden":
			e.Hidden = val == "true"
		case "NoDisplay":
			e.NoDisplay = val == "true"
		case "OnlyShowIn":
			e.OnlyShowIn = val
		case "NotShowIn":
			e.NotShowIn = val
		}
	}
	if !found || sc.Err() != nil {
		return appEntry{}, false
	}
	pick := func(key string, dst *string) {
		for _, l := range locales {
			if v, ok := localized[key][l]; ok && v != "" {
				*dst = v
				return
			}
		}
	}
	pick("Name", &e.Name)
	pick("GenericName", &e.Generic)
	pick("Comment", &e.Comment)
	if v, ok := firstLocalized(localized["Keywords"], locales); ok {
		e.Keywords = strings.ReplaceAll(v, ";", " ")
	}
	return e, true
}

func firstLocalized(m map[string]string, locales []string) (string, bool) {
	for _, l := range locales {
		if v, ok := m[l]; ok && v != "" {
			return v, true
		}
	}
	return "", false
}

// visible applies the rules of research R7.
func (e appEntry) visible(desktops []string) bool {
	if e.Type != "Application" || e.Hidden || e.NoDisplay || e.Name == "" || e.Exec == "" {
		return false
	}
	if e.OnlyShowIn != "" && !listsAny(e.OnlyShowIn, desktops) {
		return false
	}
	if e.NotShowIn != "" && listsAny(e.NotShowIn, desktops) {
		return false
	}
	if e.TryExec != "" {
		if filepath.IsAbs(e.TryExec) {
			if _, err := os.Stat(e.TryExec); err != nil {
				return false
			}
		} else if _, err := exec.LookPath(e.TryExec); err != nil {
			return false
		}
	}
	return true
}

func listsAny(list string, desktops []string) bool {
	for _, item := range strings.Split(list, ";") {
		for _, d := range desktops {
			if item != "" && strings.EqualFold(item, d) {
				return true
			}
		}
	}
	return false
}

func currentDesktops() []string {
	var out []string
	for _, d := range strings.Split(os.Getenv("XDG_CURRENT_DESKTOP"), ":") {
		if d != "" {
			out = append(out, d)
		}
	}
	return out
}

func opApps(req *request) (any, *failure) {
	entries := loadApps()
	desktops := currentDesktops()
	words := queryWords(req.Query)
	var list []scored
	for _, e := range entries {
		if !e.visible(desktops) {
			continue
		}
		ok, key := matchFields(words, e.Name, e.Generic+" "+e.Keywords)
		if !ok {
			continue
		}
		sub := e.Comment
		if sub == "" {
			sub = e.Generic
		}
		list = append(list, scored{r: row{Key: e.ID, Title: e.Name, Subtitle: sub, Icon: e.Icon}, key: key, sort: fold(e.Name)})
	}
	sortScored(list)
	limit := limitOf(req)
	total := len(list)
	rows := []row{}
	for i := 0; i < len(list) && i < limit; i++ {
		rows = append(rows, list[i].r)
	}
	return map[string]any{"rows": rows, "total": total, "cut": total > limit}, nil
}

// opLaunch starts the application of a desktop entry (spec FR-010a).
func opLaunch(req *request) (any, *failure) {
	var entry *appEntry
	for _, e := range loadApps() {
		if e.ID == req.ID {
			e := e
			entry = &e
			break
		}
	}
	if entry == nil {
		return nil, fail("not_found", "The application is no longer installed.")
	}
	args := execArgs(*entry)
	if len(args) == 0 {
		return nil, fail("start_failed", "Could not start %s.", entry.Name)
	}
	prog := args[0]
	if entry.Terminal {
		term, pre, ferr := resolveTerminal(req.Terminal, req.TerminalExecFlag)
		if ferr != nil {
			return nil, ferr
		}
		prog, args = term, append(pre, args...)
	} else {
		args = args[1:]
	}
	dir := entry.Path
	if dir != "" {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			dir = ""
		}
	}
	if err := startDetached(prog, args, dir); err != nil {
		return nil, fail("start_failed", "Could not start %s.", entry.Name)
	}
	return map[string]any{"started": entry.Name}, nil
}

// execArgs splits the Exec key and removes the field codes (Desktop Entry specification).
func execArgs(e appEntry) []string {
	var out []string
	for _, w := range splitWords(e.Exec) {
		switch w {
		case "%f", "%F", "%u", "%U", "%d", "%D", "%n", "%N", "%v", "%m", "%k":
			continue
		case "%i":
			if e.Icon != "" {
				out = append(out, "--icon", e.Icon)
			}
			continue
		case "%c":
			out = append(out, e.Name)
			continue
		}
		out = append(out, replaceCodes(w, e))
	}
	return out
}

func replaceCodes(w string, e appEntry) string {
	var b strings.Builder
	rs := []rune(w)
	for i := 0; i < len(rs); i++ {
		if rs[i] != '%' || i+1 >= len(rs) {
			b.WriteRune(rs[i])
			continue
		}
		i++
		switch rs[i] {
		case '%':
			b.WriteRune('%')
		case 'c':
			b.WriteString(e.Name)
		}
	}
	return b.String()
}
