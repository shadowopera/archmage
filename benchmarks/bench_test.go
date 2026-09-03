package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

// The benchmark drives the archmage binary installed on this machine against a
// corpus produced by the generator in this same package. Generating and
// benchmarking are deliberately separate steps: `go run .` writes the corpus,
// `go test -bench=.` measures Archmage against it and refuses to start when
// the corpus is missing or has the wrong shape.

var (
	langFlag       = flag.String("language", "cs", `target language for archmage struct: "cs" or "go"`)
	wantFilesFlag  = flag.Int("want-files", 100, "corpus shape the benchmark expects: config files")
	wantRowsFlag   = flag.Int("want-rows", 3000, "corpus shape the benchmark expects: config entries per file")
	wantFieldsFlag = flag.Int("want-fields", 20, "corpus shape the benchmark expects: logical fields per file")
	warmupFlag     = flag.Duration("warmup-spin", 2*time.Second, "how long to spin every core before timing, to bring clocks up")
)

var (
	benchRoot string
	testdata  string
	outRoot   string
	manifest  *Manifest

	resultsMu sync.Mutex
	results   []benchResult

	roundsMu sync.Mutex
	rounds   = map[string]int{}
)

// benchResult is one measured case, kept for the summary report.
type benchResult struct {
	Name       string
	Command    string
	Iterations int
	Mean       time.Duration
	Best       time.Duration
	OutputSize int64
	OutputFile int
	Rates      []rate
}

// rate is a throughput metric: how many units of work one run gets through,
// reported per second of mean run time.
type rate struct {
	Unit  string // e.g. "entries/s"
	Count int
}

func TestMain(m *testing.M) {
	flag.Parse()

	// Caught here rather than after a full warm-up run of the export case.
	if *langFlag != "cs" && *langFlag != "go" {
		_, _ = fmt.Fprintf(os.Stderr, "magebench: -language is %q, want \"cs\" or \"go\"\n", *langFlag)
		os.Exit(2)
	}

	benchRoot = ResolveRoot()
	testdata = TestdataDir(benchRoot)
	outRoot = OutputDir(benchRoot)

	if err := checkCorpus(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "\nmagebench: %v\n\nGenerate the corpus first:\n    ./step1.sh\n\n", err)
		os.Exit(1)
	}

	progress("corpus %s: %d files x %d fields x %d entries", testdata, manifest.Files, manifest.Fields, manifest.Rows)
	code := m.Run()
	if code == 0 && len(results) > 0 {
		report := filepath.Join(benchRoot, "bench-report.md")
		if err := writeReport(report); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, "magebench: writing report:", err)
			code = 1
		} else {
			progress("report written to %s", report)
		}
	}
	os.Exit(code)
}

// checkCorpus refuses to benchmark anything but the corpus the caller asked
// for. Running the headline numbers against a smoke-sized corpus by accident
// is the failure mode this guards against.
func checkCorpus() error {
	m, err := readManifest(filepath.Join(testdata, "manifest.json"))
	if err != nil {
		return fmt.Errorf("no corpus manifest at %s: %w", filepath.Join(testdata, "manifest.json"), err)
	}
	if want := GeneratorFingerprint(); m.Generator != want {
		return fmt.Errorf("corpus was written by generator %q, but this generator is %q", m.Generator, want)
	}
	if m.Files != *wantFilesFlag || m.Rows != *wantRowsFlag || m.Fields != *wantFieldsFlag {
		return fmt.Errorf("corpus is %d files x %d fields x %d entries, want %d x %d x %d "+
			"(regenerate, or pass -want-files/-want-rows/-want-fields)",
			m.Files, m.Fields, m.Rows, *wantFilesFlag, *wantFieldsFlag, *wantRowsFlag)
	}
	configs := filepath.Join(testdata, "configs")
	for _, name := range m.Configs {
		if _, err := os.Stat(filepath.Join(configs, name)); err != nil {
			return fmt.Errorf("corpus file listed in the manifest is missing: %w", err)
		}
	}
	for _, name := range m.Enums {
		if _, err := os.Stat(filepath.Join(testdata, "enums", name)); err != nil {
			return fmt.Errorf("enum file listed in the manifest is missing: %w", err)
		}
	}
	if _, err := os.Stat(filepath.Join(configs, m.L10n)); err != nil {
		return fmt.Errorf("l10n table is missing: %w", err)
	}
	if _, err := exec.LookPath("archmage"); err != nil {
		return fmt.Errorf("archmage is not on PATH: %w", err)
	}
	manifest = m
	return nil
}

// ---------------------------------------------------------------------------
// The two cases
// ---------------------------------------------------------------------------

// commonArgs are the inputs both subcommands read: the enum definitions, the
// shared l10n table, and the config files themselves. Globs are passed through
// to archmage rather than expanded by the shell.
func commonArgs() []string {
	configs := filepath.Join(testdata, "configs")
	return []string{
		"--banner",
		"-e", filepath.Join(testdata, "enums", "*.yaml"),
		"--l10n", filepath.Join(configs, manifest.L10n),
		filepath.Join(configs, "*.xlsx"),
		filepath.Join(configs, "*.csv"),
		filepath.Join(configs, "*.yaml"),
		filepath.Join(configs, "*.json"),
	}
}

func BenchmarkExport(b *testing.B) {
	out := filepath.Join(outRoot, "export")
	// Export reads and writes every config entry.
	rates := []rate{
		{"entries/s", manifest.Stats.Entries},
		{"values/s", manifest.Filled},
	}
	benchCase(b, "export", out, rates, func() []string {
		return append([]string{"export", "-o", out}, commonArgs()...)
	})
}

func BenchmarkStruct(b *testing.B) {
	out := filepath.Join(outRoot, "struct")
	// The C# template takes a namespace; the Go template takes a package name,
	// not an import path — an import path would land in the package clause and
	// the generated file would not compile.
	namespace := "Conf"
	if *langFlag == "go" {
		namespace = "conf"
	}
	// Struct generates code from the schema, so its throughput is counted in
	// fields, not in config data.
	rates := []rate{{"fields/s", manifest.Stats.Files * manifest.Stats.Fields}}
	benchCase(b, "struct", out, rates, func() []string {
		return append([]string{
			"struct", "-o", out,
			"-t", "json-" + *langFlag,
			"--namespace", namespace,
		}, commonArgs()...)
	})
}

// ---------------------------------------------------------------------------
// Harness
// ---------------------------------------------------------------------------

func benchCase(b *testing.B, name, out string, rates []rate, build func() []string) {
	args := build()

	// Warm up: one full run, then a spin across every core so the measured
	// runs start with the CPU already at speed. `-count` calls this function
	// once per round; only the first round of each case warms up.
	round := nextRound(name)
	prefix := fmt.Sprintf("%s round %d/%s", name, round, flagValue("test.count"))
	if round == 1 {
		progress("%s: warming up (one untimed run, then a %s spin)", prefix, *warmupFlag)
		start := time.Now()
		clean(b, out)
		archmage(b, args)
		spin(*warmupFlag)
		progress("%s: warmed up in %s", prefix, time.Since(start).Round(100*time.Millisecond))
	}

	runs := "?"
	if bt := flagValue("test.benchtime"); strings.HasSuffix(bt, "x") {
		runs = strings.TrimSuffix(bt, "x")
	}
	var total, best time.Duration
	iterations := 0

	b.ResetTimer()
	for b.Loop() {
		b.StopTimer()
		clean(b, out)
		b.StartTimer()

		start := time.Now()
		archmage(b, args)
		d := time.Since(start)

		total += d
		if best == 0 || d < best {
			best = d
		}
		iterations++
		progress("%s: run %d/%s took %s", prefix, iterations, runs, d.Round(10*time.Millisecond))
	}
	b.StopTimer()

	if iterations == 0 {
		return
	}
	mean := total / time.Duration(iterations)
	secs := mean.Seconds()
	for _, r := range rates {
		b.ReportMetric(float64(r.Count)/secs, r.Unit)
	}

	size, files := measure(out)
	resultsMu.Lock()
	results = append(results, benchResult{
		Name:       name,
		Command:    "archmage " + strings.Join(redact(args), " "),
		Iterations: iterations,
		Mean:       mean,
		Best:       best,
		OutputSize: size,
		OutputFile: files,
		Rates:      rates,
	})
	resultsMu.Unlock()
}

// nextRound counts the calls for the named case, starting from 1.
func nextRound(name string) int {
	roundsMu.Lock()
	defer roundsMu.Unlock()
	rounds[name]++
	return rounds[name]
}

// progress prints a status line, so that a run whose iterations take seconds
// each does not look hung. It prints only under -v: go test merges the test
// binary's stderr into stdout, and without -v a benchmark's result line is
// written in two halves around the run, where a status line would split it
// and break benchstat.
func progress(format string, args ...any) {
	if !testing.Verbose() {
		return
	}
	_, _ = fmt.Fprintf(os.Stderr, "magebench: "+format+"\n", args...)
}

// flagValue returns the value of a registered flag, such as test.count.
func flagValue(name string) string {
	if f := flag.Lookup(name); f != nil {
		return f.Value.String()
	}
	return "?"
}

// clean removes the output directory. Archmage skips writing a file whose
// content has not changed, so a stale output directory would make every run
// after the first one artificially fast.
func clean(b *testing.B, out string) {
	b.Helper()
	if err := os.RemoveAll(out); err != nil {
		b.Fatalf("cleaning %s: %v", out, err)
	}
}

func archmage(b *testing.B, args []string) {
	b.Helper()
	cmd := exec.Command("archmage", args...)
	var log bytes.Buffer
	cmd.Stdout = &log
	cmd.Stderr = &log
	if err := cmd.Run(); err != nil {
		b.Fatalf("archmage %s: %v\n%s", args[0], err, tail(log.String(), 40))
	}
}

// spin keeps every core busy for d, so that the first measured iteration does
// not pay for a CPU that is still at its idle clock.
func spin(d time.Duration) {
	if d <= 0 {
		return
	}
	deadline := time.Now().Add(d)
	var wg sync.WaitGroup
	for range runtime.NumCPU() {
		wg.Go(func() {
			x := 1.0
			for time.Now().Before(deadline) {
				for range 1 << 16 {
					x = x*1.0000001 + 1
				}
			}
			_ = x
		})
	}
	wg.Wait()
}

func measure(dir string) (int64, int) {
	var size int64
	var count int
	_ = filepath.WalkDir(dir, func(_ string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if fi, err := d.Info(); err == nil {
			size += fi.Size()
			count++
		}
		return nil
	})
	return size, count
}

// redact shortens the corpus paths in a recorded command line so the report
// stays readable.
func redact(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = strings.ReplaceAll(a, testdata+string(filepath.Separator), "")
		out[i] = strings.ReplaceAll(out[i], outRoot+string(filepath.Separator), "output/")
	}
	return out
}

func tail(s string, lines int) string {
	parts := strings.Split(strings.TrimRight(s, "\n"), "\n")
	if len(parts) > lines {
		parts = parts[len(parts)-lines:]
	}
	return strings.Join(parts, "\n")
}
