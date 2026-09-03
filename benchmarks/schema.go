package main

import (
	"math/rand/v2"
	"strconv"
	"strings"
)

// Field is one logical field of a table.
//
// A field has two shapes: a spreadsheet shape, where it occupies Cols adjacent
// columns sharing a single Name, and a tree shape, where it is one key of a
// config entry object. Not every field supports both; the catalogue keeps the
// two pools apart.
type Field struct {
	Name string
	Desc string
	Kind string // catalogue key, used for coverage accounting

	// Spreadsheet shape.
	Type    string   // Type row text, default export value included
	Cols    int      // columns spanned; every column repeats Desc/Name/Type
	ColOpts []string // per-column Options cell; nil when the field needs none

	// Tree shape. TreeType is empty when Archmage's inference already lands on
	// the intended type and no meta node is needed.
	TreeType string
	TreeElem map[string]any // meta "elem" block, for [] / [N] / map

	// Cells renders the field's columns for one config entry. It returns
	// exactly Cols strings. Tree renders the field's value for one config
	// entry; a nil return means the key is omitted from the object.
	Cells func(row int) []string
	Tree  func(row int) any

	// Passive marks backref-n and origin fields: Archmage fills them in, so
	// their cells stay blank and their tree nodes are omitted entirely.
	Passive bool

	// demo suppresses the blank paths below. A companion demo file carries the
	// meta nodes of a tree data file, and a meta node must bind to a data node
	// that actually exists, so the demo entry has to be complete even where
	// the data files are sparse. It is per field, not global, so that files
	// can be generated in parallel.
	demo bool
}

// blanks returns Cols empty strings, the spreadsheet form of "not configured".
func (f *Field) blanks() []string { return make([]string, f.Cols) }

// skip decides whether a field leaves its slot empty for one config entry. It
// always draws from the random stream, so turning demo mode on and off never
// shifts the sequence a later entry sees.
func (f *Field) skip(r *rand.Rand, pct int) bool {
	roll := r.IntN(100) < pct
	return roll && !f.demo
}

// ---------------------------------------------------------------------------
// Scalars
// ---------------------------------------------------------------------------

// scalar is a basic type together with generators for both file shapes.
type scalar struct {
	typ  string                    // type string, default export value excluded
	cell func(r *rand.Rand) string // spreadsheet cell text
	tree func(r *rand.Rand) any    // tree node value
}

var (
	signedInts   = []string{"int", "int8", "int16", "int32", "int64"}
	unsignedInts = []string{"uint", "uint8", "uint16", "uint32", "uint64"}
	floats       = []string{"float", "float32", "float64"}
)

// intRange returns the value range this generator uses for an integer type.
// The bounds stay well inside Archmage's own limits so that no value can trip
// an overflow check.
func intRange(typ string) (lo, hi int64) {
	switch typ {
	case "int8":
		return -128, 127
	case "int16":
		return -32768, 32767
	case "int32":
		return -2000000, 2000000
	case "int", "int64":
		return -999999999, 999999999
	case "uint8":
		return 0, 255
	case "uint16":
		return 0, 65535
	case "uint32":
		return 0, 4000000
	case "uint", "uint64":
		return 0, 999999999
	}
	return 0, 1000
}

// newScalar builds a generator for one basic type. ctx supplies the corpus
// level pools that enum, ref and l10n values draw from.
func (c *Corpus) newScalar(typ string) scalar {
	switch {
	case typ == "string":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return c.word(r) },
			tree: func(r *rand.Rand) any { return c.word(r) },
		}
	case typ == "bool":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string {
				if r.IntN(2) == 0 {
					return "TRUE"
				}
				return "FALSE"
			},
			tree: func(r *rand.Rand) any { return r.IntN(2) == 0 },
		}
	case typ == "datetime":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return c.datetime(r) },
			tree: func(r *rand.Rand) any { return c.datetime(r) },
		}
	case typ == "duration":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return c.duration(r) },
			tree: func(r *rand.Rand) any { return c.duration(r) },
		}
	case typ == "path":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return c.path(r) },
			tree: func(r *rand.Rand) any { return c.path(r) },
		}
	case typ == "rgba":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return c.rgba(r) },
			tree: func(r *rand.Rand) any { return c.rgba(r) },
		}
	case typ == "l10n":
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return c.l10n(r) },
			tree: func(r *rand.Rand) any { return c.l10n(r) },
		}
	case strings.HasPrefix(typ, "enum@"):
		et := c.enumByRef(typ)
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return et.value(r, strings.Contains(typ, "@++")) },
			tree: func(r *rand.Rand) any { return et.value(r, strings.Contains(typ, "@++")) },
		}
	case strings.HasPrefix(typ, "ref@"):
		target := c.tableByName(strings.TrimPrefix(typ, "ref@"))
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return target.refValue(r) },
			tree: func(r *rand.Rand) any { return target.refValueTree(r) },
		}
	case strings.HasPrefix(typ, "float"):
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string {
				return strconv.FormatFloat(roundTo(r.Float64()*2000-500, 4), 'f', -1, 64)
			},
			tree: func(r *rand.Rand) any { return roundTo(r.Float64()*2000-500, 4) },
		}
	default: // integers
		lo, hi := intRange(typ)
		return scalar{typ: typ,
			cell: func(r *rand.Rand) string { return strconv.FormatInt(lo+r.Int64N(hi-lo+1), 10) },
			tree: func(r *rand.Rand) any { return lo + r.Int64N(hi-lo+1) },
		}
	}
}

func roundTo(v float64, digits int) float64 {
	p := 1.0
	for range digits {
		p *= 10
	}
	return float64(int64(v*p)) / p
}

// ---------------------------------------------------------------------------
// Element type pools
// ---------------------------------------------------------------------------

// elemPool describes which basic types may appear inside a given container.
type elemPool struct {
	allowRGBA bool
	allowRef  bool
	numeric   bool // integers and floats only
}

func (c *Corpus) pickElem(r *rand.Rand, p elemPool) string {
	if p.numeric {
		if r.IntN(3) == 0 {
			return floats[r.IntN(len(floats))]
		}
		return signedInts[r.IntN(len(signedInts))]
	}
	// Weighted towards the types that actually fill game containers.
	switch n := r.IntN(100); {
	case n < 34:
		return signedInts[r.IntN(len(signedInts))]
	case n < 44:
		return unsignedInts[r.IntN(len(unsignedInts))]
	case n < 60:
		return floats[r.IntN(len(floats))]
	case n < 76:
		return "string"
	case n < 82:
		return "bool"
	case n < 88:
		return c.randEnumRef(r, false)
	case n < 92:
		return "duration"
	case n < 95:
		return "path"
	case n < 97 && p.allowRef:
		return "ref@" + c.randTable(r).Name
	case n < 99 && p.allowRGBA:
		return "rgba"
	}
	return signedInts[r.IntN(len(signedInts))]
}

// mapKeyType picks a key type for a map container. Keys must be string, an
// integer type, or an enum type.
func (c *Corpus) mapKeyType(r *rand.Rand) string {
	switch n := r.IntN(10); {
	case n < 5:
		return "int"
	case n < 8:
		return "string"
	default:
		return c.randEnumRef(r, false)
	}
}

// ---------------------------------------------------------------------------
// Container element counts
// ---------------------------------------------------------------------------

// elemCount returns how many elements one container holds: 1 to 5, averaging 3.
func elemCount(r *rand.Rand) int { return 1 + r.IntN(5) }

// fixedCount returns a fixed container size in the same 2 to 5 band.
func fixedCount(r *rand.Rand) int { return 2 + r.IntN(4) }

// ---------------------------------------------------------------------------
// Naming
// ---------------------------------------------------------------------------

var fieldNouns = []string{
	"atk", "def", "hp", "mp", "crit", "dodge", "speed", "range", "cost", "gain",
	"rank", "tier", "grade", "slot", "stack", "charge", "cool", "cast", "guard", "pierce",
	"drop", "loot", "reward", "bonus", "malus", "scale", "growth", "decay", "burst", "aura",
	"icon", "model", "sfx", "vfx", "anim", "banner", "portrait", "badge", "frame", "tint",
	"label", "title", "note", "tag", "group", "family", "series", "chapter", "stage", "wave",
	"unlock", "expire", "start", "finish", "window", "period", "cycle", "phase", "step", "gate",
}

// nameGen hands out unique, valid field names for one table. Archmage detects
// conflicts on a normalized (lowercased, separator-stripped) form, so the
// generator tracks that same form.
type nameGen struct {
	used map[string]bool
	seq  int
}

func newNameGen() *nameGen { return &nameGen{used: map[string]bool{}} }

func (n *nameGen) next(r *rand.Rand) string {
	for {
		n.seq++
		name := fieldNouns[r.IntN(len(fieldNouns))] + strconv.Itoa(n.seq)
		key := strings.ToLower(strings.NewReplacer("-", "", "_", "").Replace(name))
		if n.used[key] {
			continue
		}
		n.used[key] = true
		return name
	}
}

// ---------------------------------------------------------------------------
// Default export values
// ---------------------------------------------------------------------------

// withDefault appends a default export value to a type string. Fields that
// carry one may leave cells blank; fields that do not must always be filled.
func withDefault(typ string, container bool, r *rand.Rand) string {
	if container && r.IntN(2) == 0 {
		return typ + "={}"
	}
	return typ + "=/"
}
