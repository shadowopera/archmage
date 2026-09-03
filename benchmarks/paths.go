package main

import (
	"os"
	"path/filepath"
)

// FuzzVolume is probed at runtime. When it is present, both the generated
// corpus and the benchmark output live on it, keeping several hundred
// megabytes of churn off the repository volume.
const FuzzVolume = "/Volumes/Fuzz"

// FuzzRoot is the directory used on FuzzVolume.
const FuzzRoot = FuzzVolume + "/archmage_bench"

// ResolveRoot returns the directory that holds testdata/ and output/.
//
// It is FuzzRoot when /Volumes/Fuzz is a mounted directory, and the
// benchmarks directory itself otherwise. Any probing error falls back to the
// local directory rather than failing.
func ResolveRoot() string {
	fi, err := os.Stat(FuzzVolume)
	if err != nil || !fi.IsDir() {
		return localRoot()
	}
	if err := os.MkdirAll(FuzzRoot, 0o755); err != nil {
		return localRoot()
	}
	return FuzzRoot
}

// localRoot is the benchmarks directory, resolved from the working directory.
// Both `go run .` and `go test` run with it as their working directory.
func localRoot() string {
	wd, err := os.Getwd()
	if err != nil {
		return "."
	}
	return wd
}

// TestdataDir holds the generated corpus.
func TestdataDir(root string) string { return filepath.Join(root, "testdata") }

// ConfigsDir holds the 130 config files plus l10n.xlsx.
func ConfigsDir(root string) string { return filepath.Join(TestdataDir(root), "configs") }

// EnumsDir holds the enum definition files.
func EnumsDir(root string) string { return filepath.Join(TestdataDir(root), "enums") }

// ManifestPath records what the generator produced.
func ManifestPath(root string) string { return filepath.Join(TestdataDir(root), "manifest.json") }

// L10nPath is the shared localizable string table.
func L10nPath(root string) string { return filepath.Join(ConfigsDir(root), "l10n.xlsx") }

// OutputDir holds everything Archmage writes during a benchmark run.
func OutputDir(root string) string { return filepath.Join(root, "output") }
