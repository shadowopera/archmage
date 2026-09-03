package main

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// Config is the shape of the corpus the generator produces.
type Config struct {
	Seed   uint64
	Files  int
	Rows   int
	Fields int
	Root   string
}

// Corpus is the planned corpus: every table, its fields, and the shared pools
// that enum, ref and l10n values draw from. Planning happens in full before a
// single byte is written, because ref and backref-n fields need to know which
// tables exist and which of them carry anchors.
type Corpus struct {
	cfg    Config
	rng    *rand.Rand
	tables []*Table
	byName map[string]*Table

	enums        []*EnumType
	plainEnums   []*EnumType
	bitflagEnums []*EnumType
	enumFiles    []string

	words     []string
	sentences []string
	assets    []string
	l10nKeys  []string
}

// Table is one config table: one xlsx worksheet, one csv file, or one
// tree-backed regular table in yaml or json.
type Table struct {
	Name   string // export name; also the ref/backref target name
	Format string // xlsx, csv, yaml, json
	File   string // file name inside configs/
	Demo   string // companion demo file name; tree formats only

	HasAnchor bool
	Fields    []*Field

	rows int
}

// ID is the config ID of a row. Integer config IDs must be greater than zero.
func (t *Table) ID(row int) int64 { return 100001 + int64(row) }

// Anchor is the human-readable alias of a row. Anchors must be unique and must
// not look like a number.
func (t *Table) Anchor(row int) string { return fmt.Sprintf("%s-a%04d", t.Name, row) }

// refValue renders a reference to a random row of t, as a spreadsheet cell.
// Half the references to an anchored table go through the anchor name.
func (t *Table) refValue(r *rand.Rand) string {
	row := r.IntN(t.rows)
	if t.HasAnchor && r.IntN(2) == 0 {
		return t.Anchor(row)
	}
	return strconv.FormatInt(t.ID(row), 10)
}

// refValueTree renders the same reference as a tree node value.
func (t *Table) refValueTree(r *rand.Rand) any {
	row := r.IntN(t.rows)
	if t.HasAnchor && r.IntN(2) == 0 {
		return t.Anchor(row)
	}
	return t.ID(row)
}

func (t *Table) isTree() bool { return t.Format == "yaml" || t.Format == "json" }

// ---------------------------------------------------------------------------
// Shared value pools
// ---------------------------------------------------------------------------

func (c *Corpus) word(r *rand.Rand) string { return c.words[r.IntN(len(c.words))] }
func (c *Corpus) path(r *rand.Rand) string { return c.assets[r.IntN(len(c.assets))] }

func (c *Corpus) rgba(r *rand.Rand) string {
	if r.IntN(2) == 0 {
		return fmt.Sprintf("#%02X%02X%02X", r.IntN(256), r.IntN(256), r.IntN(256))
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", r.IntN(256), r.IntN(256), r.IntN(256), r.IntN(256))
}

func (c *Corpus) datetime(r *rand.Rand) string {
	return fmt.Sprintf("%04d-%02d-%02d %02d:%02d:%02d",
		2023+r.IntN(4), 1+r.IntN(12), 1+r.IntN(28), r.IntN(24), r.IntN(60), r.IntN(60))
}

func (c *Corpus) duration(r *rand.Rand) string {
	switch r.IntN(5) {
	case 0:
		return strconv.Itoa(50+r.IntN(950)) + "ms"
	case 1:
		return strconv.Itoa(1+r.IntN(120)) + "s"
	case 2:
		return strconv.Itoa(1+r.IntN(60)) + "m"
	case 3:
		return fmt.Sprintf("%dh%dm", 1+r.IntN(24), r.IntN(60))
	default:
		return fmt.Sprintf("%dd%dh", 1+r.IntN(14), r.IntN(24))
	}
}

// durationMillis renders a duration with an exact millisecond magnitude, used
// where minmax has to keep max-min inside Archmage's one-billion cap.
func durationMillis(ms int64) string { return strconv.FormatInt(ms, 10) + "ms" }

// l10n renders a localizable string. One in ten is a reference into the shared
// l10n.xlsx table; the rest are literal text.
func (c *Corpus) l10n(r *rand.Rand) string {
	if r.IntN(10) == 0 {
		return "{{" + c.l10nKeys[r.IntN(len(c.l10nKeys))] + "}}"
	}
	return c.sentences[r.IntN(len(c.sentences))]
}

// ---------------------------------------------------------------------------
// Lookups
// ---------------------------------------------------------------------------

func (c *Corpus) tableByName(name string) *Table {
	t, ok := c.byName[name]
	if !ok {
		panic("unknown ref target: " + name)
	}
	return t
}

func (c *Corpus) randTable(r *rand.Rand) *Table { return c.tables[r.IntN(len(c.tables))] }

func (c *Corpus) enumByRef(ref string) *EnumType {
	name := strings.TrimPrefix(strings.TrimPrefix(ref, "enum@"), "++")
	name = strings.TrimPrefix(strings.TrimPrefix(name, "bitflags@"), "++")
	for _, e := range c.enums {
		if e.Name == name {
			return e
		}
	}
	panic("unknown enum: " + ref)
}

// randEnumRef returns an enum@ type string. Strict mode rejects numeric input,
// so the generator only ever writes item names for those fields.
func (c *Corpus) randEnumRef(r *rand.Rand, strict bool) string {
	e := c.plainEnums[r.IntN(len(c.plainEnums))]
	if strict {
		return "enum@++" + e.Name
	}
	return "enum@" + e.Name
}

func (c *Corpus) randBitflagEnum(r *rand.Rand) *EnumType {
	return c.bitflagEnums[r.IntN(len(c.bitflagEnums))]
}
