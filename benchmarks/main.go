// Command magebench generates the benchmark corpus for Archmage.
//
// Generating the corpus and running the benchmark are two separate steps. This
// program only writes files; `go test -bench=.` in the same directory runs
// Archmage against them and refuses to start if the corpus is missing.
//
//	go run .                        # 100 files x 20 fields x 3000 entries
//	go run . -files=5 -rows=50      # a smoke-sized corpus
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"sync"
	"time"
)

func main() {
	cfg := Config{}
	flag.Uint64Var(&cfg.Seed, "seed", 42, "random seed; the same seed always yields the same corpus")
	flag.IntVar(&cfg.Files, "files", 100, "number of config files, split 60:10:15:15 across xlsx, csv, yaml and json")
	flag.IntVar(&cfg.Rows, "rows", 3000, "config entries per file")
	flag.IntVar(&cfg.Fields, "fields", 20, "logical fields per file")
	out := flag.String("out", "", "corpus directory (default: <root>/testdata, root being /Volumes/Fuzz/archmage_bench when present)")
	flag.Parse()

	root := ResolveRoot()
	cfg.Root = root
	testdata := TestdataDir(root)
	if *out != "" {
		testdata = *out
	}

	if err := run(cfg, testdata); err != nil {
		_, _ = fmt.Fprintln(os.Stderr, "magebench:", err)
		os.Exit(1)
	}
}

func run(cfg Config, testdata string) error {
	start := time.Now()
	configs := filepath.Join(testdata, "configs")
	enums := filepath.Join(testdata, "enums")

	fmt.Printf("corpus   %s\n", testdata)
	fmt.Printf("plan     %d files x %d fields x %d entries, seed %d\n", cfg.Files, cfg.Fields, cfg.Rows, cfg.Seed)

	// A stale corpus is worse than no corpus: leftover files from an earlier
	// shape would be picked up by the input globs. The directory is wiped, so
	// it must be empty or hold an earlier corpus; -out pointing at anything
	// else is almost certainly a mistake.
	if err := checkWipeable(testdata); err != nil {
		return err
	}
	if err := os.RemoveAll(testdata); err != nil {
		return err
	}
	for _, d := range []string{configs, enums} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return err
		}
	}

	c := NewCorpus(cfg)
	if err := c.writeEnums(enums); err != nil {
		return err
	}
	if err := c.writeL10n(configs); err != nil {
		return err
	}

	filled, err := c.writeTables(configs)
	if err != nil {
		return err
	}

	size, err := dirSize(testdata)
	if err != nil {
		return err
	}
	stats := c.Stats()
	m := &Manifest{
		Generator:   GeneratorFingerprint(),
		Seed:        cfg.Seed,
		Files:       cfg.Files,
		Rows:        cfg.Rows,
		Fields:      cfg.Fields,
		GeneratedAt: time.Now().Format(time.RFC3339),
		Enums:       c.enumFiles,
		L10n:        "l10n.xlsx",
		InputBytes:  size,
		Filled:      filled,
		Stats:       stats,
	}
	for _, t := range c.tables {
		m.Configs = append(m.Configs, t.File)
		if t.Demo != "" {
			m.Configs = append(m.Configs, t.Demo)
		}
	}
	sort.Strings(m.Configs)
	if err := writeManifest(filepath.Join(testdata, "manifest.json"), m); err != nil {
		return err
	}

	fmt.Printf("wrote    %d config files (+%d enum files, l10n.xlsx), %s\n",
		len(m.Configs), len(m.Enums), humanBytes(size))
	// A multi-column field occupies several cells, so the populated count can
	// exceed the field-slot count.
	fmt.Printf("data     %d config entries, %d field slots, %d populated cells/nodes, %d spreadsheet columns\n",
		stats.Entries, stats.FieldValues, filled, stats.Columns)
	fmt.Printf("elapsed  %s\n", time.Since(start).Round(time.Millisecond))
	return nil
}

// writeTables writes every config file. Tables are independent and each field
// owns its own random stream, so the work parallelises without changing the
// bytes any given seed produces.
func (c *Corpus) writeTables(dir string) (int, error) {
	type result struct {
		filled int
		err    error
	}
	results := make([]result, len(c.tables))

	sem := make(chan struct{}, runtime.GOMAXPROCS(0))
	var wg sync.WaitGroup
	for i, t := range c.tables {
		wg.Go(func() {
			sem <- struct{}{}
			defer func() { <-sem }()
			var n int
			var err error
			switch t.Format {
			case "xlsx":
				n, err = c.writeXLSX(dir, t)
			case "csv":
				n, err = c.writeCSV(dir, t)
			case "yaml":
				n, err = c.writeYAML(dir, t)
			case "json":
				n, err = c.writeJSON(dir, t)
			}
			results[i] = result{n, err}
		})
	}
	wg.Wait()

	total := 0
	for i, r := range results {
		if r.err != nil {
			return 0, fmt.Errorf("%s: %w", c.tables[i].File, r.err)
		}
		total += r.filled
	}
	return total, nil
}

// checkWipeable reports an error unless dir is missing or holds nothing but
// what the generator writes, which also covers a run that failed halfway.
func checkWipeable(dir string) error {
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	for _, e := range entries {
		switch e.Name() {
		case "configs", "enums", "manifest.json":
		default:
			return fmt.Errorf("refusing to wipe %s: %s is not something the generator writes", dir, e.Name())
		}
	}
	return nil
}

func humanBytes(n int64) string {
	const unit = 1024
	if n < unit {
		return fmt.Sprintf("%d B", n)
	}
	div, exp := int64(unit), 0
	for v := n / unit; v >= unit; v /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(n)/float64(div), "KMGT"[exp])
}
