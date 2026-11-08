package main

import (
	"fmt"
	"regexp"
)

// `modules items`: resolves the user's lists of item names against the available
// contributions to one extension point. Pure: it reads its input and writes nothing.
// Contract: specs/003-bar/contracts/helper-bar.md, rules in specs/003-bar/contracts/bar-config.md.

const (
	maxItemEntries       = 64
	maxItemContributions = 512
)

// itemName is the long form module-id/item-id or the short form module-id.
var itemName = regexp.MustCompile(`^[a-z][a-z0-9-]{1,31}(/[a-z][a-z0-9-]{0,31})?$`)

// itemPlaces are the lists a bar-style host reads, in the order they are resolved.
var itemPlaces = []string{"left", "center", "right"}

type itemContribution struct {
	key, short, module, item, name, state, reason string
	enabled                                       bool
}

type itemRef struct {
	Name   string `json:"name"`
	Key    string `json:"key"`
	Module string `json:"module"`
	Item   string `json:"item"`
}

type itemWarning struct {
	Place   string `json:"place"`
	Index   int    `json:"index"`
	Entry   any    `json:"entry"`
	Problem string `json:"problem"`
}

type itemAvailable struct {
	Key   string `json:"key"`
	Short string `json:"short"`
	Name  string `json:"name"`
	State string `json:"state"`
}

func runItems(in map[string]any) {
	point, _ := in["point"].(map[string]any)
	wantModule, _ := point["module"].(string)
	wantPoint, _ := point["point"].(string)

	lists, ok := in["lists"].(map[string]any)
	if !ok {
		fail("invalid_input", "lists must be an object with the lists left, center and right")
		return
	}
	entries := map[string][]any{}
	for _, place := range itemPlaces {
		raw, present := lists[place]
		if !present {
			entries[place] = []any{}
			continue
		}
		list, isList := raw.([]any)
		if !isList {
			fail("invalid_input", "lists.%s must be a list", place)
			return
		}
		if len(list) > maxItemEntries {
			fail("invalid_input", "lists.%s has %d entries; at most %d are allowed", place, len(list), maxItemEntries)
			return
		}
		entries[place] = list
	}

	rawContribs, isList := in["contributions"].([]any)
	if !isList {
		fail("invalid_input", "contributions must be a list")
		return
	}
	if len(rawContribs) > maxItemContributions {
		fail("invalid_input", "%d contributions given; at most %d are allowed", len(rawContribs), maxItemContributions)
		return
	}
	var contribs []itemContribution
	for _, rc := range rawContribs {
		m, ok := rc.(map[string]any)
		if !ok {
			continue
		}
		// A contribution for another point is not an item of this host.
		if target, ok := m["target"].(map[string]any); ok && wantModule != "" {
			tm, _ := target["module"].(string)
			tp, _ := target["point"].(string)
			if tm != wantModule || tp != wantPoint {
				continue
			}
		}
		c := itemContribution{}
		c.key, _ = m["key"].(string)
		c.short, _ = m["short"].(string)
		c.module, _ = m["module"].(string)
		c.item, _ = m["item"].(string)
		c.name, _ = m["name"].(string)
		c.state, _ = m["moduleState"].(string)
		c.reason, _ = m["reason"].(string)
		c.enabled, _ = m["enabled"].(bool)
		if c.key == "" {
			continue
		}
		contribs = append(contribs, c)
	}

	byName := map[string]*itemContribution{}
	for i := range contribs {
		byName[contribs[i].key] = &contribs[i]
		if contribs[i].short != "" {
			// The long form always wins over a short form of another module.
			if _, taken := byName[contribs[i].short]; !taken {
				byName[contribs[i].short] = &contribs[i]
			}
		}
	}
	for i := range contribs { // long forms are unique; re-assert them over any short form
		byName[contribs[i].key] = &contribs[i]
	}

	placed := map[string]string{} // key -> "<place> entry <index>"
	result := map[string][]itemRef{}
	warnings := []itemWarning{}
	warn := func(place string, index int, entry any, format string, a ...any) {
		warnings = append(warnings, itemWarning{place, index, entry, fmt.Sprintf(format, a...)})
	}

	for _, place := range itemPlaces {
		result[place] = []itemRef{}
		for index, entry := range entries[place] {
			name, isString := entry.(string)
			switch {
			case !isString:
				// The config helper drops these before they get here; kept as a safety check.
				warn(place, index, entry, "not a string")
				continue
			case !itemName.MatchString(name):
				warn(place, index, entry, "not a valid item name")
				continue
			}
			c := byName[name]
			if c == nil {
				warn(place, index, entry, "no such item")
				continue
			}
			if first, dup := placed[c.key]; dup {
				warn(place, index, entry, "duplicate of %s", first)
				continue
			}
			switch {
			case !c.enabled || c.state == "disabled":
				warn(place, index, entry, "module %s is not enabled; run: lumen modules enable %s", c.module, c.module)
				continue
			case c.state == "failed":
				warn(place, index, entry, "module %s failed: %s", c.module, c.reason)
				continue
			}
			placed[c.key] = fmt.Sprintf("%s entry %d", place, index)
			result[place] = append(result[place], itemRef{Name: name, Key: c.key, Module: c.module, Item: c.item})
		}
	}

	available := []itemAvailable{}
	for _, c := range contribs {
		state := "unavailable"
		if _, isPlaced := placed[c.key]; isPlaced {
			state = "placed"
		} else if c.enabled && c.state == "loaded" {
			state = "available"
		}
		available = append(available, itemAvailable{Key: c.key, Short: c.short, Name: c.name, State: state})
	}

	succeed(map[string]any{
		"left": result["left"], "center": result["center"], "right": result["right"],
		"warnings": warnings, "available": available,
	})
}
