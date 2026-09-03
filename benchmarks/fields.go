package main

import (
	"fmt"
	"math/rand/v2"
	"strconv"
	"strings"
)

// How often a field declares a default export value, and how often a field
// that declares one is actually left blank. Real config tables are sparse:
// designers fill in the columns they care about and leave the rest to the
// default.
const (
	optionalPercent = 60
	blankPercent    = 15
)

// kindSpec is one entry of a field catalogue: a stable key used for coverage
// accounting, a sampling weight in per mille, and a builder.
type kindSpec struct {
	kind   string
	weight int
	build  func(name string, r *rand.Rand) *Field
}

func desc(name string) string { return name + " value" }

// ---------------------------------------------------------------------------
// Distinct picking
// ---------------------------------------------------------------------------

// distinctIdx returns n distinct indices below limit. Container sizes here are
// tiny (at most five), so rejection sampling with a linear scan beats both a
// map and a full permutation.
func distinctIdx(r *rand.Rand, limit, n int) []int {
	if n > limit {
		n = limit
	}
	if n*2 > limit {
		return r.Perm(limit)[:n]
	}
	out := make([]int, 0, n)
outer:
	for len(out) < n {
		i := r.IntN(limit)
		for _, seen := range out {
			if seen == i {
				continue outer
			}
		}
		out = append(out, i)
	}
	return out
}

// distinctKeys returns n distinct map keys rendered as text.
func (c *Corpus) distinctKeys(keyType string, n int, r *rand.Rand) []string {
	switch {
	case keyType == "string":
		idx := distinctIdx(r, len(c.words), n)
		out := make([]string, len(idx))
		for i, j := range idx {
			out[i] = c.words[j]
		}
		return out
	case strings.HasPrefix(keyType, "enum@"):
		e := c.enumByRef(keyType)
		idx := distinctIdx(r, len(e.Items), n)
		out := make([]string, len(idx))
		for i, j := range idx {
			out[i] = e.Items[j].Name
		}
		return out
	default: // integer keys
		base := 1 + r.IntN(4000)
		out := make([]string, n)
		for i := range out {
			out[i] = strconv.Itoa(base + i*7)
		}
		return out
	}
}

// ---------------------------------------------------------------------------
// Scalar fields
// ---------------------------------------------------------------------------

// scalarField builds a single-column field of a basic type, usable in both
// spreadsheets and tree-structured data.
func (c *Corpus) scalarField(name, typ, kind string, r *rand.Rand) *Field {
	return c.scalarFieldOpt(name, typ, kind, r, true)
}

// scalarFieldOpt is scalarField with control over whether the field may be
// left empty.
//
// Tree rgba fields pass allowBlank=false. An absent tree node is resolved by
// the implicit presence rule, which fills it from the zero value — and rgba's
// zero value, the empty string, is not something the rgba parser accepts.
// Declaring `rgba=/` does not change that. Every other type resolves fine.
func (c *Corpus) scalarFieldOpt(name, typ, kind string, r *rand.Rand, allowBlank bool) *Field {
	s := c.newScalar(typ)
	optional := r.IntN(100) < optionalPercent && allowBlank
	sheetType := typ
	if optional {
		sheetType = typ + "=/"
	}
	// The tree shape declares the same default export value. Without it an
	// absent node is filled by the implicit presence rule, and for a type
	// whose zero value has no literal form — rgba — that fill fails to parse.
	f := &Field{Name: name, Desc: desc(name), Kind: kind, Type: sheetType, Cols: 1, TreeType: sheetType}
	f.Cells = func(int) []string {
		if optional && f.skip(r, blankPercent) {
			return []string{""}
		}
		return []string{s.cell(r)}
	}
	f.Tree = func(int) any {
		if optional && f.skip(r, blankPercent) {
			return nil
		}
		return s.tree(r)
	}
	return f
}

// ---------------------------------------------------------------------------
// Compact fields — one cell holds the whole container
// ---------------------------------------------------------------------------

// compactCell wraps a per-entry renderer with the blank handling every
// container field shares. Container fields always declare a default export
// value, so a blank cell is always legal.
func compactField(name, kind, typ string, r *rand.Rand, render func() string) *Field {
	f := &Field{Name: name, Desc: desc(name), Kind: kind, Type: typ, Cols: 1}
	f.Cells = func(int) []string {
		if f.skip(r, blankPercent) {
			return []string{""}
		}
		return []string{render()}
	}
	return f
}

func (c *Corpus) compactArray(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{})
	s := c.newScalar(et)
	return compactField(name, "compact-array", withDefault(">>[]"+et, true, r), r, func() string {
		n := elemCount(r)
		parts := make([]string, n)
		for i := range parts {
			parts[i] = s.cell(r)
		}
		return strings.Join(parts, "|")
	})
}

func (c *Corpus) compactFixedArray(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{})
	s := c.newScalar(et)
	n := fixedCount(r)
	typ := fmt.Sprintf(">>[%d]%s", n, et)
	return compactField(name, "compact-fixed-array", withDefault(typ, true, r), r, func() string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = s.cell(r)
		}
		return strings.Join(parts, "|")
	})
}

func (c *Corpus) compactMap(name string, r *rand.Rand) *Field {
	kt := c.mapKeyType(r)
	vt := c.pickElem(r, elemPool{})
	vs := c.newScalar(vt)
	typ := fmt.Sprintf(">>map[%s]%s", kt, vt)
	return compactField(name, "compact-map", withDefault(typ, true, r), r, func() string {
		keys := c.distinctKeys(kt, elemCount(r), r)
		parts := make([]string, len(keys))
		for i, k := range keys {
			// "=>" rather than ":" so that Excel never reads an int:int pair
			// as a clock time.
			parts[i] = k + "=>" + vs.cell(r)
		}
		return strings.Join(parts, "|")
	})
}

var objectFieldNouns = []string{"Id", "Qty", "Rate", "Lvl", "Kind", "Slot", "Bonus", "Cap"}

func (c *Corpus) compactObject(name string, r *rand.Rand) *Field {
	n := 2 + r.IntN(3)
	idx := distinctIdx(r, len(objectFieldNouns), n)
	names := make([]string, n)
	scalars := make([]scalar, n)
	var decl []string
	for i, j := range idx {
		names[i] = objectFieldNouns[j]
		et := c.pickElem(r, elemPool{})
		scalars[i] = c.newScalar(et)
		decl = append(decl, names[i]+" "+et)
	}
	typ := ">>object|" + strings.Join(decl, "|")
	return compactField(name, "compact-object", withDefault(typ, true, r), r, func() string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = names[i] + "=>" + scalars[i].cell(r)
		}
		return strings.Join(parts, "|")
	})
}

// minmaxElem picks an element type for a minmax and returns a renderer that
// produces an ordered pair whose span stays inside Archmage's limits.
func (c *Corpus) minmaxElem(r *rand.Rand) (string, func() (string, string)) {
	switch r.IntN(3) {
	case 0:
		return floats[r.IntN(len(floats))], func() (string, string) {
			lo := roundTo(r.Float64()*500, 3)
			hi := roundTo(lo+r.Float64()*500, 3)
			return strconv.FormatFloat(lo, 'f', -1, 64), strconv.FormatFloat(hi, 'f', -1, 64)
		}
	case 1:
		return "duration", func() (string, string) {
			lo := int64(r.IntN(60000))
			return durationMillis(lo), durationMillis(lo + int64(r.IntN(600000)))
		}
	default:
		et := signedInts[r.IntN(len(signedInts))]
		lo0, hi0 := intRange(et)
		// At most half the range, so that lo still varies on a narrow type;
		// a span covering all of int8 would pin lo to -128.
		span := min((hi0-lo0)/2, 1000)
		return et, func() (string, string) {
			lo := lo0 + r.Int64N(max(hi0-lo0-span, 1))
			return strconv.FormatInt(lo, 10), strconv.FormatInt(lo+r.Int64N(span), 10)
		}
	}
}

func (c *Corpus) compactMinmax(name string, r *rand.Rand) *Field {
	et, pair := c.minmaxElem(r)
	typ := ">>minmax|" + et
	return compactField(name, "compact-minmax", withDefault(typ, false, r), r, func() string {
		lo, hi := pair()
		return lo + "|" + hi
	})
}

func (c *Corpus) compactWtpool(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{allowRef: true})
	s := c.newScalar(et)
	typ := ">>wtpool|" + et
	return compactField(name, "compact-wtpool", withDefault(typ, true, r), r, func() string {
		n := elemCount(r)
		parts := make([]string, n)
		for i := range parts {
			parts[i] = s.cell(r) + "=>" + strconv.Itoa(1+r.IntN(100))
		}
		return strings.Join(parts, "|")
	})
}

func (c *Corpus) compactVector(name string, r *rand.Rand) *Field {
	dim := 2 + r.IntN(3)
	et := c.pickElem(r, elemPool{numeric: true})
	s := c.newScalar(et)
	typ := fmt.Sprintf(">>vector|%d|%s", dim, et)
	return compactField(name, "compact-vector", withDefault(typ, false, r), r, func() string {
		parts := make([]string, dim)
		for i := range parts {
			parts[i] = s.cell(r)
		}
		return strings.Join(parts, "|")
	})
}

// tupleTypes picks the field types of a tuple. rgba is excluded outright: it
// may not lead a compact tuple, and its leading '#' invites comment parsing.
func (c *Corpus) tupleTypes(r *rand.Rand) ([]string, []scalar) {
	n := 2 + r.IntN(3)
	types := make([]string, n)
	scalars := make([]scalar, n)
	for i := range types {
		types[i] = c.pickElem(r, elemPool{})
		scalars[i] = c.newScalar(types[i])
	}
	return types, scalars
}

func (c *Corpus) compactTuple(name string, r *rand.Rand) *Field {
	types, scalars := c.tupleTypes(r)
	typ := ">>tuple|" + strings.Join(types, "|")
	return compactField(name, "compact-tuple", withDefault(typ, true, r), r, func() string {
		parts := make([]string, len(scalars))
		for i := range parts {
			parts[i] = scalars[i].cell(r)
		}
		return strings.Join(parts, "|")
	})
}

// ---------------------------------------------------------------------------
// Multi-column fields — one logical field spread over several columns
// ---------------------------------------------------------------------------

func multiField(name, kind, typ string, cols int, r *rand.Rand, render func() []string) *Field {
	f := &Field{Name: name, Desc: desc(name), Kind: kind, Type: typ, Cols: cols}
	f.Cells = func(int) []string {
		if f.skip(r, blankPercent) {
			return f.blanks()
		}
		return render()
	}
	return f
}

func (c *Corpus) multiArray(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{})
	s := c.newScalar(et)
	cols := 2 + r.IntN(4)
	typ := withDefault("[]"+et, true, r)
	return multiField(name, "multi-array", typ, cols, r, func() []string {
		out := make([]string, cols)
		for i := range out {
			// Blank cells inside a []T span are skipped, which is exactly how
			// a variable-length array is written in a spreadsheet.
			if r.IntN(4) == 0 {
				continue
			}
			out[i] = s.cell(r)
		}
		return out
	})
}

func (c *Corpus) multiFixedArray(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{})
	s := c.newScalar(et)
	cols := fixedCount(r)
	typ := withDefault(fmt.Sprintf("[%d]%s", cols, et), true, r)
	return multiField(name, "multi-fixed-array", typ, cols, r, func() []string {
		out := make([]string, cols)
		for i := range out {
			out[i] = s.cell(r)
		}
		return out
	})
}

func (c *Corpus) multiMap(name string, r *rand.Rand) *Field {
	kt := c.mapKeyType(r)
	vt := c.pickElem(r, elemPool{})
	vs := c.newScalar(vt)
	pairs := 1 + r.IntN(3)
	cols := pairs * 2
	typ := withDefault(fmt.Sprintf("map[%s]%s", kt, vt), true, r)
	return multiField(name, "multi-map", typ, cols, r, func() []string {
		keys := c.distinctKeys(kt, pairs, r)
		out := make([]string, cols)
		for i, k := range keys {
			out[i*2] = k
			out[i*2+1] = vs.cell(r)
		}
		return out
	})
}

func (c *Corpus) multiMinmax(name string, r *rand.Rand) *Field {
	et, pair := c.minmaxElem(r)
	typ := withDefault("minmax|"+et, false, r)
	return multiField(name, "multi-minmax", typ, 2, r, func() []string {
		lo, hi := pair()
		return []string{lo, hi}
	})
}

func (c *Corpus) multiWtpool(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{allowRef: true})
	s := c.newScalar(et)
	pairs := 1 + r.IntN(3)
	cols := pairs * 2
	typ := withDefault("wtpool|"+et, true, r)
	return multiField(name, "multi-wtpool", typ, cols, r, func() []string {
		out := make([]string, cols)
		for i := range pairs {
			out[i*2] = s.cell(r)
			out[i*2+1] = strconv.Itoa(1 + r.IntN(100))
		}
		return out
	})
}

func (c *Corpus) multiVector(name string, r *rand.Rand) *Field {
	dim := 2 + r.IntN(3)
	et := c.pickElem(r, elemPool{numeric: true})
	s := c.newScalar(et)
	typ := withDefault(fmt.Sprintf("vector|%d|%s", dim, et), false, r)
	return multiField(name, "multi-vector", typ, dim, r, func() []string {
		out := make([]string, dim)
		for i := range out {
			out[i] = s.cell(r)
		}
		return out
	})
}

func (c *Corpus) multiTuple(name string, r *rand.Rand) *Field {
	types, scalars := c.tupleTypes(r)
	typ := withDefault("tuple|"+strings.Join(types, "|"), true, r)
	return multiField(name, "multi-tuple", typ, len(types), r, func() []string {
		out := make([]string, len(scalars))
		for i := range out {
			out[i] = scalars[i].cell(r)
		}
		return out
	})
}

func (c *Corpus) multiBitflags(name string, r *rand.Rand) *Field {
	e := c.randBitflagEnum(r)
	cols := 2 + r.IntN(4)
	// The zero-valued "none" item cannot back a flag column.
	idx := distinctIdx(r, len(e.Items)-1, cols)
	cols = len(idx)
	opts := make([]string, cols)
	for i, j := range idx {
		opts[i] = "e=" + e.Items[j+1].Name
	}
	f := multiField(name, "multi-bitflags", "bitflags@"+e.Name+"=/", cols, r, func() []string {
		out := make([]string, cols)
		for i := range out {
			if r.IntN(2) == 0 {
				out[i] = "1"
			} else {
				out[i] = "0"
			}
		}
		return out
	})
	f.ColOpts = opts
	return f
}

// ---------------------------------------------------------------------------
// Tree non-leaf fields
// ---------------------------------------------------------------------------

func treeField(name, kind, treeType string, r *rand.Rand, render func() any) *Field {
	f := &Field{Name: name, Desc: desc(name), Kind: kind, TreeType: treeType, Cols: 1}
	f.Tree = func(int) any {
		if f.skip(r, blankPercent) {
			return nil
		}
		return render()
	}
	return f
}

var treeObjectNouns = []string{"base", "peak", "cap", "step", "seed", "gain", "loss", "span"}

func (c *Corpus) treeObject(name string, r *rand.Rand) *Field {
	n := 2 + r.IntN(3)
	idx := distinctIdx(r, len(treeObjectNouns), n)
	keys := make([]string, n)
	scalars := make([]scalar, n)
	for i, j := range idx {
		keys[i] = treeObjectNouns[j]
		// Children stay on types Archmage infers correctly on its own, so the
		// object needs no nested meta nodes.
		scalars[i] = c.newScalar([]string{"int64", "float64", "string", "bool"}[r.IntN(4)])
	}
	return treeField(name, "tree-object", "{}", r, func() any {
		m := make(OM, n)
		for i := range m {
			m[i] = Pair{keys[i], scalars[i].tree(r)}
		}
		return m
	})
}

func (c *Corpus) treeArray(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{allowRGBA: true})
	s := c.newScalar(et)
	f := treeField(name, "tree-array", "[]", r, func() any {
		n := elemCount(r)
		out := make([]any, n)
		for i := range out {
			out[i] = s.tree(r)
		}
		return out
	})
	f.TreeElem = map[string]any{"type": et}
	return f
}

func (c *Corpus) treeFixedArray(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{allowRGBA: true})
	s := c.newScalar(et)
	n := fixedCount(r)
	f := treeField(name, "tree-fixed-array", fmt.Sprintf("[%d]", n), r, func() any {
		out := make([]any, n)
		for i := range out {
			out[i] = s.tree(r)
		}
		return out
	})
	f.TreeElem = map[string]any{"type": et}
	return f
}

func (c *Corpus) treeMap(name string, r *rand.Rand) *Field {
	kt := c.mapKeyType(r)
	vt := c.pickElem(r, elemPool{allowRGBA: true})
	vs := c.newScalar(vt)
	f := treeField(name, "tree-map", fmt.Sprintf("map[%s]", kt), r, func() any {
		keys := c.distinctKeys(kt, elemCount(r), r)
		m := make(OM, len(keys))
		for i, k := range keys {
			m[i] = Pair{k, vs.tree(r)}
		}
		return m
	})
	f.TreeElem = map[string]any{"type": vt}
	return f
}

func (c *Corpus) treeMinmax(name string, r *rand.Rand) *Field {
	et, pair := c.minmaxElem(r)
	numeric := et != "duration"
	return treeField(name, "tree-minmax", "minmax|"+et, r, func() any {
		lo, hi := pair()
		if numeric {
			return OM{{"min", jsonNumber(lo)}, {"max", jsonNumber(hi)}}
		}
		return OM{{"min", lo}, {"max", hi}}
	})
}

func (c *Corpus) treeWtpool(name string, r *rand.Rand) *Field {
	et := c.pickElem(r, elemPool{allowRef: true})
	s := c.newScalar(et)
	return treeField(name, "tree-wtpool", "wtpool|"+et, r, func() any {
		n := elemCount(r)
		out := make([]any, n)
		for i := range out {
			out[i] = OM{{"item", s.tree(r)}, {"weight", int64(1 + r.IntN(100))}}
		}
		return out
	})
}

func (c *Corpus) treeVector(name string, r *rand.Rand) *Field {
	dim := 2 + r.IntN(3)
	et := c.pickElem(r, elemPool{numeric: true})
	s := c.newScalar(et)
	axes := []string{"x", "y", "z", "w"}
	return treeField(name, "tree-vector", fmt.Sprintf("vector|%d|%s", dim, et), r, func() any {
		m := make(OM, dim)
		for i := range m {
			m[i] = Pair{axes[i], s.tree(r)}
		}
		return m
	})
}

// jsonNumber turns a rendered numeric literal back into a number so that tree
// files carry it as a number rather than a string.
func jsonNumber(s string) any {
	if i, err := strconv.ParseInt(s, 10, 64); err == nil {
		return i
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}

// ---------------------------------------------------------------------------
// Catalogues
// ---------------------------------------------------------------------------

// sheetCatalogue is the pool of field kinds available to xlsx and csv tables.
// Weights are per mille and reflect what game config tables actually hold:
// mostly scalars, with containers as seasoning.
func (c *Corpus) sheetCatalogue() []kindSpec {
	var ks []kindSpec
	add := func(kind string, weight int, b func(string, *rand.Rand) *Field) {
		ks = append(ks, kindSpec{kind, weight, b})
	}
	for _, s := range basicWeights {
		typ := s.typ
		add(typ, s.weight, func(name string, r *rand.Rand) *Field {
			return c.scalarField(name, typ, typ, r)
		})
	}
	add("enum", 64, func(name string, r *rand.Rand) *Field {
		return c.scalarField(name, c.randEnumRef(r, false), "enum", r)
	})
	add("enum-strict", 16, func(name string, r *rand.Rand) *Field {
		return c.scalarField(name, c.randEnumRef(r, true), "enum-strict", r)
	})
	add("ref", 70, func(name string, r *rand.Rand) *Field {
		return c.scalarField(name, "ref@"+c.randTable(r).Name, "ref", r)
	})

	add("compact-array", 30, c.compactArray)
	add("compact-map", 20, c.compactMap)
	add("compact-object", 20, c.compactObject)
	add("compact-vector", 15, c.compactVector)
	add("compact-wtpool", 10, c.compactWtpool)
	add("compact-minmax", 10, c.compactMinmax)
	add("compact-tuple", 10, c.compactTuple)
	add("compact-fixed-array", 5, c.compactFixedArray)

	add("multi-array", 20, c.multiArray)
	add("multi-map", 15, c.multiMap)
	add("multi-vector", 15, c.multiVector)
	add("multi-fixed-array", 10, c.multiFixedArray)
	add("multi-minmax", 10, c.multiMinmax)
	add("multi-wtpool", 10, c.multiWtpool)
	add("multi-tuple", 5, c.multiTuple)
	add("multi-bitflags", 5, c.multiBitflags)
	return ks
}

// treeCatalogue is the pool for yaml and json tables. Compact, multi-column
// and bitflags types are spreadsheet-only, and are replaced here by the
// non-leaf container types.
func (c *Corpus) treeCatalogue() []kindSpec {
	var ks []kindSpec
	add := func(kind string, weight int, b func(string, *rand.Rand) *Field) {
		ks = append(ks, kindSpec{kind, weight, b})
	}
	for _, s := range basicWeights {
		typ := s.typ
		allowBlank := typ != "rgba"
		add(typ, s.treeWeight, func(name string, r *rand.Rand) *Field {
			return c.scalarFieldOpt(name, typ, typ, r, allowBlank)
		})
	}
	add("enum", 64, func(name string, r *rand.Rand) *Field {
		return c.scalarField(name, c.randEnumRef(r, false), "enum", r)
	})
	add("enum-strict", 16, func(name string, r *rand.Rand) *Field {
		return c.scalarField(name, c.randEnumRef(r, true), "enum-strict", r)
	})
	add("ref", 70, func(name string, r *rand.Rand) *Field {
		return c.scalarField(name, "ref@"+c.randTable(r).Name, "ref", r)
	})

	add("tree-object", 40, c.treeObject)
	add("tree-array", 40, c.treeArray)
	add("tree-map", 30, c.treeMap)
	add("tree-vector", 20, c.treeVector)
	add("tree-wtpool", 10, c.treeWtpool)
	add("tree-minmax", 10, c.treeMinmax)
	add("tree-fixed-array", 10, c.treeFixedArray)
	return ks
}

// basicWeights is the shared scalar half of both catalogues.
var basicWeights = []struct {
	typ        string
	weight     int // spreadsheet, per mille
	treeWeight int
}{
	{"int32", 100, 105},
	{"int64", 50, 55},
	{"int", 40, 45},
	{"int16", 20, 20},
	{"uint32", 20, 20},
	{"int8", 10, 10},
	{"uint", 10, 15},
	{"uint8", 10, 10},
	{"uint16", 10, 10},
	{"uint64", 10, 10},
	{"float32", 60, 65},
	{"float", 30, 35},
	{"float64", 20, 20},
	{"string", 30, 35},
	{"bool", 70, 80},
	{"duration", 30, 30},
	{"path", 30, 30},
	{"datetime", 10, 10},
	{"rgba", 10, 10},
}
