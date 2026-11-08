package main

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

// Parsing of the interface 1.1 manifest fields: extensionPoints, contributions, commands.
// Rules: specs/003-bar/contracts/module-manifest-1.1.md.

type errFunc = func(field, format string, a ...any)

// parsePoints validates `extensionPoints` and returns the declared points. A manifest with
// any error is invalid as a whole, so the returned list is only used when there are none.
func parsePoints(obj map[string]any, addErr errFunc) []pointDecl {
	raw, present := obj["extensionPoints"]
	if !present {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		addErr("extensionPoints", "must be a list of {id, description}")
		return nil
	}
	if len(list) > maxPoints {
		addErr("extensionPoints", "at most %d extension points are allowed", maxPoints)
	}
	var out []pointDecl
	seen := map[string]int{}
	for i, item := range list {
		field := fmt.Sprintf("extensionPoints[%d]", i)
		p, ok := item.(map[string]any)
		if !ok {
			addErr(field, "must be an object with id and description")
			continue
		}
		id, _ := p["id"].(string)
		desc, hasDesc := p["description"].(string)
		switch {
		case !shortID.MatchString(id):
			addErr(field+".id", "must start with a lower-case letter and use only lower-case letters, digits, and '-' (1 to 32 characters)")
		default:
			if first, dup := seen[id]; dup {
				addErr(field+".id", "duplicate extension point id %q (also extensionPoints[%d])", id, first)
			} else {
				seen[id] = i
			}
		}
		if d, present := p["description"]; present && (!hasDesc || utf8.RuneCountInString(desc) > 160) {
			_ = d
			addErr(field+".description", "must be a string of at most 160 characters")
		}
		out = append(out, pointDecl{ID: id, Description: desc})
	}
	return out
}

// parseContributions validates `contributions`. ownID is the module's own id ("" when
// unusable); declared holds the module's permissions.
func parseContributions(obj map[string]any, ownID string, declared map[string]bool, addErr errFunc) []contribDecl {
	raw, present := obj["contributions"]
	if !present {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		addErr("contributions", "must be a list of {id, name, target, file}")
		return nil
	}
	if len(list) > maxContributions {
		addErr("contributions", "at most %d contributions are allowed", maxContributions)
	}
	var out []contribDecl
	seen := map[string]int{}
	for i, item := range list {
		field := fmt.Sprintf("contributions[%d]", i)
		c, ok := item.(map[string]any)
		if !ok {
			addErr(field, "must be an object with id, name, target and file")
			continue
		}
		var d contribDecl
		d.ID, _ = c["id"].(string)
		d.Name, _ = c["name"].(string)
		d.File, _ = c["file"].(string)

		if !shortID.MatchString(d.ID) {
			addErr(field+".id", "must start with a lower-case letter and use only lower-case letters, digits, and '-' (1 to 32 characters; no '/')")
		} else if first, dup := seen[d.ID]; dup {
			addErr(field+".id", "duplicate contribution id %q (also contributions[%d])", d.ID, first)
		} else {
			seen[d.ID] = i
		}
		if n := utf8.RuneCountInString(d.Name); n < 1 || n > 64 {
			addErr(field+".name", "must be 1 to 64 characters")
		}

		target, _ := c["target"].(map[string]any)
		d.Target.Module, _ = target["module"].(string)
		d.Target.Point, _ = target["point"].(string)
		switch {
		case target == nil:
			addErr(field+".target", "must be an object with module and point")
		case !idPattern.MatchString(d.Target.Module):
			addErr(field+".target.module", "must be a module id (lower-case letters, digits, '-'; 2 to 32 characters; no '/')")
		case !shortID.MatchString(d.Target.Point):
			addErr(field+".target.point", "must be an extension point id (lower-case letters, digits, '-'; 1 to 32 characters; no '/')")
		case d.Target.Module != ownID && !declared["contribute:"+d.Target.Module]:
			addErr(field+".target.module", "permission contribute:%s is not declared", d.Target.Module)
		}

		if d.File == "" || !strings.HasSuffix(d.File, ".qml") || !filepath.IsLocal(d.File) {
			addErr(field+".file", "must be a relative path inside the module directory that ends in .qml")
			d.File = ""
		} else {
			d.File = filepath.Clean(d.File)
		}
		if ownID != "" && d.ID != "" {
			d.Key = ownID + "/" + d.ID
		}
		out = append(out, d)
	}
	return out
}

// parseCommands validates `commands` (names a module handles for `lumen modules call`).
func parseCommands(obj map[string]any, addErr errFunc) []string {
	raw, present := obj["commands"]
	if !present {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		addErr("commands", "must be a list of command names")
		return nil
	}
	if len(list) > maxCommands {
		addErr("commands", "at most %d commands are allowed", maxCommands)
	}
	var out []string
	seen := map[string]int{}
	for i, item := range list {
		field := fmt.Sprintf("commands[%d]", i)
		s, ok := item.(string)
		switch {
		case !ok || !shortID.MatchString(s):
			addErr(field, "must start with a lower-case letter and use only lower-case letters, digits, and '-' (1 to 32 characters)")
			continue
		default:
			if first, dup := seen[s]; dup {
				addErr(field, "duplicate command %q (also commands[%d])", s, first)
				continue
			}
			seen[s] = i
		}
		out = append(out, s)
	}
	return out
}
