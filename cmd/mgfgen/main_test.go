package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/rdon-key/modgadget/internal/mgf"
)

func TestRunGeneratesEmptyMGF(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "empty.mgf")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-font-id", "sh12", "-subset-id", "full", "-region", "JP", "-o", output}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if len(data) != mgf.HeaderSize {
		t.Fatalf("size=%d", len(data))
	}
	header, err := mgf.DecodeHeader(string(data))
	if err != nil {
		t.Fatal(err)
	}
	if header.FontID != ([4]byte{'s', 'h', '1', '2'}) || header.SubsetID != ([4]byte{'f', 'u', 'l', 'l'}) || header.Region != ([2]byte{'J', 'P'}) || header.GlyphCount != 0 {
		t.Fatalf("header=%+v", header)
	}
	for _, text := range []string{"wrote " + output, "FontID: sh12", "SubsetID: full", "Region: JP", "GlyphCount: 0", "bytes: 36"} {
		if !strings.Contains(stdout.String(), text) {
			t.Fatalf("stdout %q lacks %q", stdout.String(), text)
		}
	}
	assertNoTemporaryFiles(t, directory)
}

func TestRunWithoutRegionAndReplaceExisting(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "replace.mgf")
	if err := os.WriteFile(output, []byte("old content"), 0o600); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	code := run([]string{"-font-id", "sp16", "-subset-id", "full", "-region", "", "-o", output}, &stdout, &stderr)
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), "Region: none") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	data, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	header, err := mgf.DecodeHeader(string(data))
	if err != nil || len(data) != mgf.HeaderSize || header.Region != ([2]byte{}) {
		t.Fatalf("size=%d header=%+v err=%v", len(data), header, err)
	}
	assertNoTemporaryFiles(t, directory)
}

func TestRunInputErrorsDoNotCreateOutput(t *testing.T) {
	tests := []struct {
		name string
		args []string
		text string
	}{
		{"font ID missing", []string{"-subset-id", "full"}, "font-id"},
		{"subset ID missing", []string{"-font-id", "sh12"}, "subset-id"},
		{"font ID three bytes", []string{"-font-id", "abc", "-subset-id", "full"}, "4 bytes"},
		{"font ID five bytes", []string{"-font-id", "abcde", "-subset-id", "full"}, "4 bytes"},
		{"font ID non-ASCII", []string{"-font-id", "aéx", "-subset-id", "full"}, "ASCII"},
		{"subset ID short", []string{"-font-id", "sh12", "-subset-id", "abc"}, "4 bytes"},
		{"subset ID non-ASCII", []string{"-font-id", "sh12", "-subset-id", "aéx"}, "ASCII"},
		{"region one byte", []string{"-font-id", "sh12", "-subset-id", "full", "-region", "J"}, "2 bytes"},
		{"region three bytes", []string{"-font-id", "sh12", "-subset-id", "full", "-region", "JPN"}, "2 bytes"},
		{"region non-ASCII", []string{"-font-id", "sh12", "-subset-id", "full", "-region", "é"}, "ASCII"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			directory := t.TempDir()
			output := filepath.Join(directory, "bad.mgf")
			args := append(append([]string(nil), tt.args...), "-o", output)
			var stdout, stderr bytes.Buffer
			code := run(args, &stdout, &stderr)
			if code == 0 || !strings.Contains(stderr.String(), tt.text) {
				t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
			}
			if _, err := os.Stat(output); !os.IsNotExist(err) {
				t.Fatalf("output exists or unexpected error: %v", err)
			}
			assertNoTemporaryFiles(t, directory)
		})
	}
}

func TestRunRequiresOutput(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-font-id", "sh12", "-subset-id", "full"}, &stdout, &stderr)
	if code == 0 || !strings.Contains(stderr.String(), "-o") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunOutputErrorsAndTemporaryCleanup(t *testing.T) {
	directory := t.TempDir()
	missingOutput := filepath.Join(directory, "missing", "out.mgf")
	var stdout, stderr bytes.Buffer
	code := run([]string{"-font-id", "sh12", "-subset-id", "full", "-o", missingOutput}, &stdout, &stderr)
	if code == 0 || stderr.Len() == 0 {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(missingOutput); !os.IsNotExist(err) {
		t.Fatalf("output exists or unexpected error: %v", err)
	}
	assertNoTemporaryFiles(t, directory)

	stdout.Reset()
	stderr.Reset()
	code = run([]string{"-font-id", "sh12", "-subset-id", "full", "-o", directory}, &stdout, &stderr)
	if code == 0 {
		t.Fatalf("directory replacement succeeded: stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
	assertNoTemporaryFiles(t, directory)
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"-h"}, &stdout, &stderr); code != 0 {
		t.Fatalf("code=%d stderr=%q", code, stderr.String())
	}
	for _, text := range []string{"header-only", "BDF conversion", "-font-id", "-subset-id", "-region", "-o"} {
		if !strings.Contains(stderr.String(), text) {
			t.Fatalf("help %q lacks %q", stderr.String(), text)
		}
	}
}

func assertNoTemporaryFiles(t *testing.T, directory string) {
	t.Helper()
	matches, err := filepath.Glob(filepath.Join(directory, ".*.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(matches) != 0 {
		t.Fatalf("temporary files remain: %v", matches)
	}
}
