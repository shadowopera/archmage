package main

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"
)

// writeReport writes bench-report.md next to the output directory: corpus
// shape, machine, Archmage build, and the measured numbers. It is meant to be
// quotable as-is.
func writeReport(path string) error {
	var b strings.Builder
	s := manifest.Stats

	b.WriteString("# Archmage benchmark\n\n")
	_, _ = fmt.Fprintf(&b, "Generated %s by `magebench`.\n\n", time.Now().Format(time.RFC3339))

	b.WriteString("## Corpus\n\n")
	b.WriteString("| Metric | Value |\n|---|---|\n")
	_, _ = fmt.Fprintf(&b, "| Config files | %d (%d xlsx, %d csv, %d yaml, %d json) |\n",
		s.Files, s.ByFormat["xlsx"], s.ByFormat["csv"], s.ByFormat["yaml"], s.ByFormat["json"])
	_, _ = fmt.Fprintf(&b, "| Config entries | %s |\n", comma(s.Entries))
	_, _ = fmt.Fprintf(&b, "| Logical fields per file | %d |\n", s.Fields)
	_, _ = fmt.Fprintf(&b, "| Logical fields | %s |\n", comma(s.Files*s.Fields))
	_, _ = fmt.Fprintf(&b, "| Field slots | %s |\n", comma(s.FieldValues))
	_, _ = fmt.Fprintf(&b, "| Populated cells and tree nodes | %s |\n", comma(manifest.Filled))
	_, _ = fmt.Fprintf(&b, "| Spreadsheet columns | %s |\n", comma(s.Columns))
	_, _ = fmt.Fprintf(&b, "| Distinct data types used | %d |\n", len(s.ByKind))
	_, _ = fmt.Fprintf(&b, "| Enum definition files | %d |\n", len(manifest.Enums))
	_, _ = fmt.Fprintf(&b, "| Input size on disk | %s |\n", humanBytes(manifest.InputBytes))
	_, _ = fmt.Fprintf(&b, "| Corpus seed | %d |\n\n", manifest.Seed)

	b.WriteString("## Machine\n\n")
	b.WriteString("| Component | Value |\n|---|---|\n")
	_, _ = fmt.Fprintf(&b, "| CPU | %s (%d logical cores) |\n", sysctl("machdep.cpu.brand_string"), runtime.NumCPU())
	_, _ = fmt.Fprintf(&b, "| Memory | %s |\n", memSize())
	_, _ = fmt.Fprintf(&b, "| Platform | %s/%s |\n", runtime.GOOS, runtime.GOARCH)
	_, _ = fmt.Fprintf(&b, "| Go | %s |\n", runtime.Version())
	_, _ = fmt.Fprintf(&b, "| Archmage | %s |\n\n", archmageVersion())

	cases := aggregate(results)
	b.WriteString("## Results\n\n")
	// One column per throughput unit, in first-seen order. A case that does
	// not measure a unit shows a dash: struct generates code and touches no
	// config data, so entries/s and values/s mean nothing for it.
	var units []string
	for _, r := range cases {
		for _, rt := range r.Rates {
			if !slices.Contains(units, rt.Unit) {
				units = append(units, rt.Unit)
			}
		}
	}
	b.WriteString("| Case | Runs | Mean | Best |")
	for _, u := range units {
		b.WriteString(" " + strings.ToUpper(u[:1]) + u[1:] + " |")
	}
	b.WriteString(" Output |\n|---|---:|---:|---:|" + strings.Repeat("---:|", len(units)) + "---|\n")
	for _, r := range cases {
		secs := r.Mean.Seconds()
		_, _ = fmt.Fprintf(&b, "| %s | %d | %s | %s |",
			r.Name, r.Iterations, r.Mean.Round(time.Millisecond), r.Best.Round(time.Millisecond))
		for _, u := range units {
			cell := "—"
			for _, rt := range r.Rates {
				if rt.Unit == u {
					cell = comma(int(float64(rt.Count) / secs))
				}
			}
			b.WriteString(" " + cell + " |")
		}
		_, _ = fmt.Fprintf(&b, " %s in %d files |\n", humanBytes(r.OutputSize), r.OutputFile)
	}
	b.WriteString("\nMean is the mean of every timed run; best is the fastest single run.\n")

	b.WriteString("\n## Commands\n\n")
	for _, r := range cases {
		_, _ = fmt.Fprintf(&b, "**%s**\n\n```\n%s\n```\n\n", r.Name, wrapArgs(r.Command))
	}

	b.WriteString("## Type mix\n\n")
	b.WriteString("Field kinds across the whole corpus, most frequent first.\n\n")
	b.WriteString("| Kind | Fields | Share |\n|---|---:|---:|\n")
	type kv struct {
		k string
		v int
	}
	var kinds []kv
	total := 0
	for k, v := range s.ByKind {
		kinds = append(kinds, kv{k, v})
		total += v
	}
	sort.Slice(kinds, func(i, j int) bool {
		if kinds[i].v != kinds[j].v {
			return kinds[i].v > kinds[j].v
		}
		return kinds[i].k < kinds[j].k
	})
	for _, e := range kinds {
		_, _ = fmt.Fprintf(&b, "| `%s` | %d | %.1f%% |\n", e.k, e.v, float64(e.v)/float64(total)*100)
	}

	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// aggregate folds the rounds of one case into a single row: `go test -count`
// runs each benchmark several times, and the report wants one line per case,
// not one per round.
func aggregate(rs []benchResult) []benchResult {
	byName := map[string]*benchResult{}
	var order []string
	for _, r := range rs {
		acc, ok := byName[r.Name]
		if !ok {
			cp := r
			byName[r.Name] = &cp
			order = append(order, r.Name)
			continue
		}
		// Weight each round's mean by its iteration count.
		total := time.Duration(acc.Iterations)*acc.Mean + time.Duration(r.Iterations)*r.Mean
		acc.Iterations += r.Iterations
		acc.Mean = total / time.Duration(acc.Iterations)
		if r.Best < acc.Best {
			acc.Best = r.Best
		}
	}
	sort.Strings(order)
	out := make([]benchResult, 0, len(order))
	for _, name := range order {
		out = append(out, *byName[name])
	}
	return out
}

func wrapArgs(cmd string) string {
	parts := strings.Fields(cmd)
	var lines []string
	var cur string
	for _, p := range parts {
		if cur != "" && len(cur)+len(p)+1 > 72 {
			lines = append(lines, cur+" \\")
			cur = "    "
		}
		if cur == "" || cur == "    " {
			cur += p
		} else {
			cur += " " + p
		}
	}
	return strings.Join(append(lines, cur), "\n")
}

func sysctl(key string) string {
	out, err := exec.Command("sysctl", "-n", key).Output()
	if err != nil {
		return "unknown"
	}
	return strings.TrimSpace(string(out))
}

func memSize() string {
	n, err := strconv.ParseInt(sysctl("hw.memsize"), 10, 64)
	if err != nil {
		return "unknown"
	}
	return humanBytes(n)
}

func archmageVersion() string {
	out, err := exec.Command("archmage", "version").Output()
	if err != nil {
		return "unknown"
	}
	for line := range strings.SplitSeq(string(out), "\n") {
		return strings.TrimSpace(strings.TrimPrefix(line, "archmage version:"))
	}
	return "unknown"
}

func comma(n int) string {
	s := strconv.Itoa(n)
	if n < 0 {
		return "-" + comma(-n)
	}
	var out []byte
	for i, c := range []byte(s) {
		if i > 0 && (len(s)-i)%3 == 0 {
			out = append(out, ',')
		}
		out = append(out, c)
	}
	return string(out)
}
