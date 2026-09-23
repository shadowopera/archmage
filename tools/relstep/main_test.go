package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testSteps = "a,b,c"

func TestRun(t *testing.T) {
	tests := []struct {
		name    string
		initial string // release.json content; empty means no file
		args    []string
		out     string
		err     string
		final   string // expected release.json after the run; empty means unchanged
	}{
		{
			name:  "next creates release.json",
			args:  []string{"next", "-steps", testSteps, "v1.2.3"},
			out:   "a\n",
			final: "{\n\t\"version\": \"1.2.3\",\n\t\"steps\": {\n\t\t\"a\": 0,\n\t\t\"b\": 0,\n\t\t\"c\": 0\n\t}\n}\n",
		},
		{
			name: "next without file and version",
			args: []string{"next", "-steps", testSteps},
			err:  "release.json not found",
		},
		{
			name: "next with invalid semver",
			args: []string{"next", "-steps", testSteps, "1.2"},
			err:  "invalid semver: 1.2",
		},
		{
			name:    "next resumes",
			initial: `{"version":"1.2.3","steps":{"a":1,"b":0,"c":0}}`,
			args:    []string{"next", "-steps", testSteps},
			out:     "b\n",
		},
		{
			name:    "next with same version",
			initial: `{"version":"1.2.3","steps":{"a":1,"b":0,"c":0}}`,
			args:    []string{"next", "-steps", testSteps, "V1.2.3"},
			out:     "b\n",
		},
		{
			name:    "next done",
			initial: `{"version":"1.2.3","steps":{"a":1,"b":1,"c":1}}`,
			args:    []string{"next", "-steps", testSteps},
			out:     "DONE\n",
		},
		{
			name:    "next version mismatch",
			initial: `{"version":"1.2.3","steps":{"a":1,"b":0,"c":0}}`,
			args:    []string{"next", "-steps", testSteps, "1.2.4"},
			err:     "version mismatch",
		},
		{
			name:    "next after completed release",
			initial: `{"version":"1.2.3","steps":{"a":1,"b":1,"c":1}}`,
			args:    []string{"next", "-steps", testSteps, "1.2.4"},
			err:     "already completed",
		},
		{
			name:    "next with invalid semver in file",
			initial: `{"version":"x","steps":{"a":0,"b":0,"c":0}}`,
			args:    []string{"next", "-steps", testSteps},
			err:     "invalid semver format in release.json",
		},
		{
			name:    "next with invalid step value",
			initial: `{"version":"1.2.3","steps":{"a":2,"b":0,"c":0}}`,
			args:    []string{"next", "-steps", testSteps},
			err:     "invalid step value for a: 2",
		},
		{
			name:    "next with invalid step order",
			initial: `{"version":"1.2.3","steps":{"a":0,"b":1,"c":0}}`,
			args:    []string{"next", "-steps", testSteps},
			err:     "invalid step order: b",
		},
		{
			name:    "next with mismatched steps",
			initial: `{"version":"1.2.3","steps":{"a":1,"x":0,"y":0}}`,
			args:    []string{"next", "-steps", testSteps},
			err:     "missing: [b, c], extra: [x, y]",
		},
		{
			name:    "mark",
			initial: `{"version":"1.2.3","steps":{"c":0,"b":0,"a":1}}`,
			args:    []string{"mark", "-steps", testSteps, "b"},
			final:   "{\n\t\"version\": \"1.2.3\",\n\t\"steps\": {\n\t\t\"a\": 1,\n\t\t\"b\": 1,\n\t\t\"c\": 0\n\t}\n}\n",
		},
		{
			name:    "mark unknown step",
			initial: `{"version":"1.2.3","steps":{"a":0,"b":0,"c":0}}`,
			args:    []string{"mark", "-steps", testSteps, "x"},
			err:     "unknown step: x",
		},
		{
			name:    "mark out of order",
			initial: `{"version":"1.2.3","steps":{"a":0,"b":0,"c":0}}`,
			args:    []string{"mark", "-steps", testSteps, "b"},
			err:     "invalid step order: b",
		},
		{
			name:    "mark without step",
			initial: `{"version":"1.2.3","steps":{"a":0,"b":0,"c":0}}`,
			args:    []string{"mark", "-steps", testSteps},
			err:     "usage: relstep mark",
		},
		{
			name:    "version",
			initial: `{"version":"1.2.3","steps":{"a":0,"b":0,"c":0}}`,
			args:    []string{"version"},
			out:     "1.2.3\n",
		},
		{
			name: "missing -steps",
			args: []string{"next", "1.2.3"},
			err:  "missing -steps",
		},
		{
			name: "empty step name",
			args: []string{"next", "-steps", "a,,c", "1.2.3"},
			err:  "empty step name",
		},
		{
			name: "duplicate step",
			args: []string{"next", "-steps", "a,b,a", "1.2.3"},
			err:  "duplicate step in -steps: a",
		},
		{
			name: "no command",
			err:  "usage:",
		},
		{
			name: "unknown command",
			args: []string{"foo"},
			err:  "unknown command: foo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			file := filepath.Join(t.TempDir(), "release.json")
			if tt.initial != "" {
				if err := os.WriteFile(file, []byte(tt.initial), 0644); err != nil {
					t.Fatal(err)
				}
			}

			var out strings.Builder
			err := run(tt.args, &out, file)
			switch {
			case tt.err == "" && err != nil:
				t.Fatalf("unexpected error: %v", err)
			case tt.err != "" && err == nil:
				t.Fatalf("expected error containing %q, got nil", tt.err)
			case tt.err != "" && !strings.Contains(err.Error(), tt.err):
				t.Fatalf("expected error containing %q, got %q", tt.err, err)
			}
			if out.String() != tt.out {
				t.Errorf("output: got %q, want %q", out.String(), tt.out)
			}

			want := tt.final
			if want == "" {
				want = tt.initial
			}
			got, _ := os.ReadFile(file)
			if string(got) != want {
				t.Errorf("release.json: got %q, want %q", got, want)
			}
		})
	}
}
