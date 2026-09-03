package main

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"os"
	"path/filepath"
	"strconv"

	"github.com/goccy/go-yaml"
)

// treeData builds the config entry object for one row: a tree object whose
// keys are field names. Passive fields contribute nothing — Archmage fills
// them in — and blank fields are simply absent, which the default `implicit`
// presence resolves to the zero value.
func (c *Corpus) treeData(t *Table, row int) OM {
	entry := make(OM, 0, len(t.Fields))
	for _, f := range t.Fields {
		if f.Passive || f.Tree == nil {
			continue
		}
		v := f.Tree(row)
		if v == nil {
			continue
		}
		entry = append(entry, Pair{f.Name, v})
	}
	return entry
}

// treeDemo builds the companion demo file: the root map type, one complete
// config entry, and a meta node for every field.
func (c *Corpus) treeDemo(t *Table) OM {
	for _, f := range t.Fields {
		f.demo = true
	}
	defer func() {
		for _, f := range t.Fields {
			f.demo = false
		}
	}()

	entry := make(OM, 0, len(t.Fields)*2)
	for _, f := range t.Fields {
		meta := OM{{"desc", f.Desc}}
		if f.TreeType != "" {
			meta = append(meta, Pair{"type", f.TreeType})
		}
		if f.TreeElem != nil {
			meta = append(meta, Pair{"elem", OM{{"type", f.TreeElem["type"]}}})
		}
		if !f.Passive && f.Tree != nil {
			if v := f.Tree(0); v != nil {
				entry = append(entry, Pair{f.Name, v})
			}
		}
		entry = append(entry, Pair{f.Name + "__meta__", meta})
	}
	return OM{
		{"__meta__", OM{{"type", "map[int64]"}}},
		{strconv.FormatInt(t.ID(0), 10), entry},
	}
}

// treeRoot builds the data file: a map of config ID to config entry, which is
// exactly the shape Archmage recognises as a tree-backed regular table.
func (c *Corpus) treeRoot(t *Table) (OM, int) {
	root := make(OM, 0, t.rows)
	nonEmpty := 0
	for row := range t.rows {
		entry := c.treeData(t, row)
		nonEmpty += len(entry)
		root = append(root, Pair{strconv.FormatInt(t.ID(row), 10), entry})
	}
	return root, nonEmpty
}

func (c *Corpus) writeYAML(dir string, t *Table) (int, error) {
	demo := c.treeDemo(t)
	root, nonEmpty := c.treeRoot(t)
	if err := writeYAMLFile(filepath.Join(dir, t.Demo), demo); err != nil {
		return 0, err
	}
	return nonEmpty, writeYAMLFile(filepath.Join(dir, t.File), root)
}

func (c *Corpus) writeJSON(dir string, t *Table) (int, error) {
	demo := c.treeDemo(t)
	root, nonEmpty := c.treeRoot(t)
	if err := writeJSONFile(filepath.Join(dir, t.Demo), demo); err != nil {
		return 0, err
	}
	return nonEmpty, writeJSONFile(filepath.Join(dir, t.File), root)
}

func writeYAMLFile(path string, v any) error {
	b, err := yaml.Marshal(v)
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func writeJSONFile(path string, v any) error {
	b, err := json.Marshal(v, jsontext.WithIndent("  "))
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}
