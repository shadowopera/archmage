package main

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json/jsontext"
	"encoding/json/v2"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

// generatorSources is everything that decides the bytes of a corpus: the
// generator's own code and the pinned versions of the libraries it writes
// through. Test files are embedded too, since a pattern cannot exclude them,
// and are skipped when hashing.
//
//go:embed *.go go.mod go.sum
var generatorSources embed.FS

// GeneratorFingerprint hashes generatorSources, so that any change to the
// generator invalidates a corpus it wrote earlier.
func GeneratorFingerprint() string {
	h := sha256.New()
	// fs.WalkDir visits entries in lexical order, so the hash is stable.
	_ = fs.WalkDir(generatorSources, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || strings.HasSuffix(path, "_test.go") {
			return err
		}
		b, err := generatorSources.ReadFile(path)
		if err != nil {
			return err
		}
		h.Write([]byte(path))
		h.Write([]byte{0})
		h.Write(b)
		return nil
	})
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// writeL10n writes l10n.xlsx, the shared table of localizable strings.
//
// It is a regular table with string config IDs — the "$%" marker in the first
// row switches the ID type — and a single exportable column named
// "localizable" of type l10n. Config values written as {{key}} resolve here.
func (c *Corpus) writeL10n(dir string) error {
	f := excelize.NewFile()
	defer func() {
		_ = f.Close()
	}()
	const sheet = "l10n"
	idx, err := f.NewSheet(sheet)
	if err != nil {
		return err
	}
	f.SetActiveSheet(idx)
	if err := f.DeleteSheet("Sheet1"); err != nil {
		return err
	}
	sw, err := f.NewStreamWriter(sheet)
	if err != nil {
		return err
	}
	rows := [][]any{
		{"$%"},
		{},
		{"Description", "shared localizable string"},
		{"Name", "localizable"},
		{"Type", "l10n"},
	}
	for i, key := range c.l10nKeys {
		rows = append(rows, []any{key, c.L10nText(i)})
	}
	for i, row := range rows {
		cell, err := excelize.CoordinatesToCellName(1, i+1)
		if err != nil {
			return err
		}
		if len(row) == 0 {
			continue
		}
		if err := sw.SetRow(cell, row); err != nil {
			return err
		}
	}
	if err := sw.Flush(); err != nil {
		return err
	}
	return f.SaveAs(filepath.Join(dir, "l10n.xlsx"))
}

// writeEnums writes the enum definition files.
func (c *Corpus) writeEnums(dir string) error {
	for _, file := range c.enumFiles {
		path := filepath.Join(dir, file)
		if err := os.WriteFile(path, []byte(c.renderEnumFile(file)), 0o644); err != nil {
			return err
		}
	}
	return nil
}

// Manifest records what a generator run produced. The benchmark reads it back
// and refuses to run against a corpus that does not match, which keeps a
// smoke-sized corpus from quietly producing headline numbers.
type Manifest struct {
	// Generator fingerprints the generator that wrote the corpus. The shape
	// alone cannot tell a corpus from an older generator apart from a current
	// one, and both would pass a shape check.
	Generator   string   `json:"generator"`
	Seed        uint64   `json:"seed"`
	Files       int      `json:"files"`
	Rows        int      `json:"rows"`
	Fields      int      `json:"fields"`
	GeneratedAt string   `json:"generatedAt"`
	Configs     []string `json:"configs"`
	Enums       []string `json:"enums"`
	L10n        string   `json:"l10n"`
	InputBytes  int64    `json:"inputBytes"`
	// Filled counts populated spreadsheet cells and tree nodes. A
	// multi-column field occupies several cells, so this can exceed Files x
	// Rows x Fields.
	Filled int   `json:"populatedCells"`
	Stats  Stats `json:"stats"`
}

func writeManifest(path string, m *Manifest) error {
	b, err := json.Marshal(m, jsontext.WithIndent("  "), json.Deterministic(true))
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(b, '\n'), 0o644)
}

func readManifest(path string) (*Manifest, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(b, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// dirSize totals the bytes of every regular file under dir.
func dirSize(dir string) (int64, error) {
	var total int64
	err := filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		total += fi.Size()
		return nil
	})
	return total, err
}
