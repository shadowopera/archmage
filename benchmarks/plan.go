package main

import (
	"fmt"
	"math/rand/v2"
)

// minPerKind is how many distinct tables every catalogue kind is forced into,
// so that a corpus always covers the whole type system regardless of how the
// weighted sampling falls.
const minPerKind = 2

// l10nBlankPercent is the knob that keeps the l10n aggregate near ten thousand
// strings: four tables carry one localizable field each, and a quarter of
// those cells are left blank.
const l10nBlankPercent = 25

var tableNouns = []string{
	"hero", "item", "monster", "skill", "quest", "shop", "buff", "drop", "stage", "level",
	"weapon", "armor", "gem", "rune", "pet", "mount", "title", "achieve", "mail", "guild",
	"arena", "dungeon", "event", "unlock", "growth", "biome", "weather", "trait", "element", "stance",
	"recipe", "craft", "forge", "enchant", "talent", "spell", "totem", "relic", "charm", "banner",
	"season", "pass", "daily", "weekly", "raid", "boss", "minion", "summon", "aura", "curse",
}

// NewCorpus plans the whole corpus. Nothing is written here; planning must
// finish first because ref and backref-n fields need to know which tables
// exist and which of them carry anchors.
func NewCorpus(cfg Config) *Corpus {
	c := &Corpus{
		cfg:    cfg,
		rng:    rand.New(rand.NewPCG(cfg.Seed, cfg.Seed^0x9e3779b97f4a7c15)),
		byName: map[string]*Table{},
	}
	c.buildPools()
	c.buildEnums(3, 30)
	c.buildTables()
	c.buildFields()
	return c
}

// ---------------------------------------------------------------------------
// Value pools
// ---------------------------------------------------------------------------

var (
	poolAdjectives = []string{
		"iron", "steel", "jade", "onyx", "amber", "ivory", "coral", "slate", "ember", "frost",
		"storm", "shadow", "solar", "lunar", "astral", "verdant", "crimson", "azure", "golden", "silver",
	}
	poolNouns = []string{
		"blade", "shield", "crown", "sigil", "totem", "charm", "banner", "helm", "cloak", "gauntlet",
		"orb", "shard", "relic", "tome", "scroll", "brew", "seal", "fang", "talon", "core",
	}
	poolVerbs = []string{
		"grants", "boosts", "restores", "reduces", "reflects", "absorbs", "unleashes", "channels",
	}
	poolDirs = []string{"prefabs", "textures", "audio", "anims", "shaders", "fonts"}
	poolExts = []string{".prefab", ".png", ".ogg", ".anim", ".shader", ".ttf"}
)

func (c *Corpus) buildPools() {
	r := c.rng
	// The numeric suffix is the pool index itself, which keeps every word
	// unique: distinctKeys hands out distinct indices as string map keys, and
	// two indices spelling the same word would be a duplicate key.
	for i := range 2000 {
		c.words = append(c.words, fmt.Sprintf("%s_%s%04d",
			poolAdjectives[r.IntN(len(poolAdjectives))], poolNouns[r.IntN(len(poolNouns))], i))
	}
	for i := range 2000 {
		c.sentences = append(c.sentences, fmt.Sprintf("The %s %s %s %d bonus %s for %d seconds.",
			poolAdjectives[r.IntN(len(poolAdjectives))], poolNouns[r.IntN(len(poolNouns))],
			poolVerbs[r.IntN(len(poolVerbs))], 1+i%99, poolNouns[r.IntN(len(poolNouns))], 1+i%30))
	}
	for i := range 1500 {
		d := r.IntN(len(poolDirs))
		c.assets = append(c.assets, fmt.Sprintf("%s/%s/%s_%04d%s",
			poolDirs[d], poolNouns[r.IntN(len(poolNouns))],
			poolAdjectives[r.IntN(len(poolAdjectives))], i, poolExts[d]))
	}
	for i := range 1000 {
		c.l10nKeys = append(c.l10nKeys, fmt.Sprintf("lk_%04d", i))
	}
}

// L10nText renders the shared string stored under the i-th l10n key.
func (c *Corpus) L10nText(i int) string {
	return c.sentences[i%len(c.sentences)]
}

// ---------------------------------------------------------------------------
// Tables
// ---------------------------------------------------------------------------

// formatSplit divides files into xlsx, csv, yaml and json counts at the
// 60:10:15:15 ratio the benchmark targets.
func formatSplit(files int) map[string]int {
	nx := files * 60 / 100
	nc := files * 10 / 100
	ny := files * 15 / 100
	if nx == 0 {
		nx = 1
	}
	nj := files - nx - nc - ny
	if nj < 0 {
		nj = 0
	}
	return map[string]int{"xlsx": nx, "csv": nc, "yaml": ny, "json": nj}
}

func (c *Corpus) buildTables() {
	r := c.rng
	split := formatSplit(c.cfg.Files)
	var formats []string
	for _, f := range []string{"xlsx", "csv", "yaml", "json"} {
		for range split[f] {
			formats = append(formats, f)
		}
	}
	r.Shuffle(len(formats), func(i, j int) { formats[i], formats[j] = formats[j], formats[i] })

	for i, format := range formats {
		name := tableNouns[i%len(tableNouns)] + fmt.Sprintf("%02d", i+1)
		t := &Table{Name: name, Format: format, rows: c.cfg.Rows}
		switch format {
		case "xlsx":
			t.File = name + ".xlsx"
		case "csv":
			t.File = name + ".csv"
		case "yaml":
			t.File, t.Demo = name+".yaml", name+".demo.yaml"
		case "json":
			t.File, t.Demo = name+".json", name+".demo.json"
		}
		c.tables = append(c.tables, t)
		c.byName[name] = t
	}

	// One table in five carries an anchor field, which lets half the
	// references to it be written as a name instead of a number.
	for _, i := range distinctIdx(r, len(c.tables), max(1, len(c.tables)/5)) {
		c.tables[i].HasAnchor = true
	}
}

// ---------------------------------------------------------------------------
// Field planning
// ---------------------------------------------------------------------------

// plan holds the per-table decisions made before field sampling starts.
type plan struct {
	forced      [][]kindSpec // forced kinds, by table index
	l10n        map[int]bool
	origin      map[int]bool
	backrefFrom map[int]*Table // table index -> source table of its backref-n
	refTo       map[int]*Table // table index -> table its forced ref must target
}

func (c *Corpus) buildFields() {
	r := c.rng
	p := c.newPlan(r)

	for i, t := range c.tables {
		cat := c.sheetCatalogue()
		if t.isTree() {
			cat = c.treeCatalogue()
		}
		ng := newNameGen()
		var fields []*Field
		// Every field draws from its own stream, so writing tables in parallel
		// still produces the same bytes for a given seed.
		next := func() *rand.Rand { return c.fieldRNG(i, len(fields)) }

		if t.HasAnchor {
			ng.used["anchor"] = true
			fields = append(fields, c.anchorField(t, "anchor"))
		}
		if p.l10n[i] {
			ng.used["blurb"] = true
			fields = append(fields, c.l10nField("blurb", next()))
		}
		if p.origin[i] {
			ng.used["source"] = true
			fields = append(fields, c.originField("source"))
		}
		if src, ok := p.backrefFrom[i]; ok {
			ng.used["referrers"] = true
			fields = append(fields, c.backrefField("referrers", src))
		}
		if target, ok := p.refTo[i]; ok {
			fields = append(fields, c.scalarField(ng.next(r), "ref@"+target.Name, "ref", next()))
		}
		for _, ks := range p.forced[i] {
			if len(fields) >= c.cfg.Fields {
				break
			}
			fields = append(fields, ks.build(ng.next(r), next()))
		}
		total := weightTotal(cat)
		for len(fields) < c.cfg.Fields {
			fields = append(fields, pickKind(cat, total, r).build(ng.next(r), next()))
		}
		t.Fields = fields[:c.cfg.Fields]
	}
}

// fieldRNG derives the random stream of one field from the corpus seed, the
// table index and the field index.
func (c *Corpus) fieldRNG(table, field int) *rand.Rand {
	return rand.New(rand.NewPCG(
		c.cfg.Seed*0x9e3779b97f4a7c15+uint64(table)*0xbf58476d1ce4e5b9,
		uint64(field)*0x94d049bb133111eb+0x2545f4914f6cdd1d))
}

// newPlan decides coverage forcing and the special field placements.
func (c *Corpus) newPlan(r *rand.Rand) *plan {
	p := &plan{
		forced:      make([][]kindSpec, len(c.tables)),
		l10n:        map[int]bool{},
		origin:      map[int]bool{},
		backrefFrom: map[int]*Table{},
		refTo:       map[int]*Table{},
	}

	var sheetIdx, treeIdx []int
	for i, t := range c.tables {
		if t.isTree() {
			treeIdx = append(treeIdx, i)
		} else {
			sheetIdx = append(sheetIdx, i)
		}
	}
	force := func(idx []int, cat []kindSpec) {
		if len(idx) == 0 {
			return
		}
		for _, ks := range cat {
			for _, j := range distinctIdx(r, len(idx), minPerKind) {
				p.forced[idx[j]] = append(p.forced[idx[j]], ks)
			}
		}
	}
	force(sheetIdx, c.sheetCatalogue())
	force(treeIdx, c.treeCatalogue())

	// l10n lands on exactly one table per format. Together with the blank rate
	// on that field, the aggregate stays near ten thousand strings.
	for _, format := range []string{"xlsx", "csv", "yaml", "json"} {
		for i, t := range c.tables {
			if t.Format == format {
				p.l10n[i] = true
				break
			}
		}
	}
	for _, i := range distinctIdx(r, len(c.tables), max(1, len(c.tables)/20)) {
		p.origin[i] = true
	}
	// Each backref-n target is paired with a source table that is forced to
	// carry a matching ref field, so the back-references are never empty.
	for _, i := range distinctIdx(r, len(c.tables), max(1, len(c.tables)*6/100)) {
		for range 8 {
			j := r.IntN(len(c.tables))
			if j == i {
				continue
			}
			if _, taken := p.refTo[j]; taken {
				continue
			}
			p.backrefFrom[i] = c.tables[j]
			p.refTo[j] = c.tables[i]
			break
		}
	}
	return p
}

func weightTotal(cat []kindSpec) int {
	n := 0
	for _, ks := range cat {
		n += ks.weight
	}
	return n
}

func pickKind(cat []kindSpec, total int, r *rand.Rand) kindSpec {
	n := r.IntN(total)
	for _, ks := range cat {
		n -= ks.weight
		if n < 0 {
			return ks
		}
	}
	return cat[len(cat)-1]
}

// ---------------------------------------------------------------------------
// Special fields
// ---------------------------------------------------------------------------

// anchorField gives every row a stable, human-readable alias. Anchors must be
// unique and must never be blank, so this field carries no default export
// value and no blanks.
func (c *Corpus) anchorField(t *Table, name string) *Field {
	f := &Field{Name: name, Desc: "row anchor", Kind: "anchor", Type: "anchor", Cols: 1, TreeType: "anchor"}
	f.Cells = func(row int) []string { return []string{t.Anchor(row)} }
	f.Tree = func(row int) any { return t.Anchor(row) }
	return f
}

// l10nField is the one localizable field a table may carry. Its blank rate is
// higher than the corpus default; that is the knob that keeps the l10n
// aggregate near ten thousand entries.
func (c *Corpus) l10nField(name string, r *rand.Rand) *Field {
	f := &Field{Name: name, Desc: "localizable blurb", Kind: "l10n", Type: "l10n=/", Cols: 1, TreeType: "l10n"}
	f.Cells = func(int) []string {
		if f.skip(r, l10nBlankPercent) {
			return []string{""}
		}
		return []string{c.l10n(r)}
	}
	f.Tree = func(int) any {
		if f.skip(r, l10nBlankPercent) {
			return nil
		}
		return c.l10n(r)
	}
	return f
}

// originField records which file a config entry came from. Passive fields take
// no input at all.
func (c *Corpus) originField(name string) *Field {
	return &Field{
		Name: name, Desc: "config entry source", Kind: "origin",
		Type: "origin", Cols: 1, TreeType: "origin", Passive: true,
		Cells: func(int) []string { return []string{""} },
	}
}

// backrefField collects the entries of src that point to this table.
func (c *Corpus) backrefField(name string, src *Table) *Field {
	typ := "backref-n@" + src.Name + "={}"
	return &Field{
		Name: name, Desc: "back-references from " + src.Name, Kind: "backref-n",
		Type: typ, Cols: 1, TreeType: typ, Passive: true,
		Cells: func(int) []string { return []string{""} },
	}
}

// ---------------------------------------------------------------------------
// Statistics
// ---------------------------------------------------------------------------

// Stats summarises the planned corpus for the manifest and the report.
type Stats struct {
	Files       int            `json:"files"`
	Rows        int            `json:"rows"`
	Fields      int            `json:"fields"`
	Entries     int            `json:"entries"`
	FieldValues int            `json:"fieldValues"`
	Columns     int            `json:"columns"`
	ByFormat    map[string]int `json:"byFormat"`
	ByKind      map[string]int `json:"byKind"`
}

func (c *Corpus) Stats() Stats {
	s := Stats{
		Files:    len(c.tables),
		Rows:     c.cfg.Rows,
		Fields:   c.cfg.Fields,
		ByFormat: map[string]int{},
		ByKind:   map[string]int{},
	}
	for _, t := range c.tables {
		s.ByFormat[t.Format]++
		for _, f := range t.Fields {
			s.ByKind[f.Kind]++
			if !t.isTree() {
				s.Columns += f.Cols
			}
		}
	}
	s.Entries = len(c.tables) * c.cfg.Rows
	s.FieldValues = s.Entries * c.cfg.Fields
	return s
}
