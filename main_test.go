package main

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunMinimalExample(t *testing.T) {
	input := readTestFile(t, "examples/minimal/input.tsv")
	want := readTestFile(t, "examples/minimal/output.txt")
	var stdout, stderr bytes.Buffer
	status := run([]string{"examples/minimal/parse.conf"}, strings.NewReader(input), &stdout, &stderr)
	if status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status=%d, stdout=%q, stderr=%q; want status=0, stdout=%q, no errors",
			status, stdout.String(), stderr.String(), want)
	}
}

func TestRunSkipsMalformedRows(t *testing.T) {
	config := writeTestConfig(t)
	input := "trace-1\t1\t10\n\nwrong\tcolumns\n\t1\t10\ntrace-2\t0\t20\n"
	var stdout, stderr bytes.Buffer
	status := run([]string{config}, strings.NewReader(input), &stdout, &stderr)
	want := "1 1:10 #trace-1\t\n0 1:20 #trace-2\t\n"
	if status != 0 || stdout.String() != want {
		t.Fatalf("status=%d, stdout=%q, stderr=%q; want %q", status, stdout.String(), stderr.String(), want)
	}
	if count := strings.Count(stderr.String(), "skip "); count != 3 {
		t.Fatalf("got %d skip diagnostics, want 3: %s", count, stderr.String())
	}
}

func TestRunLongLine(t *testing.T) {
	config := writeTestConfig(t)
	trace := strings.Repeat("x", initialInputBufferSize+1)
	var stdout, stderr bytes.Buffer
	status := run([]string{config}, strings.NewReader(trace+"\t1\t42\n"), &stdout, &stderr)
	if want := "1 1:42 #" + trace + "\t\n"; status != 0 || stdout.String() != want || stderr.Len() != 0 {
		t.Fatalf("status=%d, stdout length=%d, stderr=%q", status, stdout.Len(), stderr.String())
	}
}

func TestRunFatalErrors(t *testing.T) {
	config := writeTestConfig(t)
	for _, tc := range []struct {
		name string
		args []string
		in   io.Reader
		want string
	}{
		{"missing argument", nil, strings.NewReader(""), "cmd args len wrong"},
		{"missing config", []string{filepath.Join(t.TempDir(), "missing.conf")}, strings.NewReader(""), "init offline extractor error"},
		{"read failure", []string{config}, failedReader{}, "read stdin error"},
		{"oversized line", []string{config}, strings.NewReader(strings.Repeat("x", maxInputLineSize)), "read stdin error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			status := run(tc.args, tc.in, &stdout, &stderr)
			if status != 1 || stdout.Len() != 0 || !strings.Contains(stderr.String(), tc.want) {
				t.Fatalf("status=%d, stdout=%q, stderr=%q; want status=1 and %q", status, stdout.String(), stderr.String(), tc.want)
			}
		})
	}
}

type failedReader struct{}

func (failedReader) Read([]byte) (int, error) { return 0, errors.New("test read failure") }

func readTestFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func writeTestConfig(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	features := filepath.Join(dir, "features.conf")
	config := filepath.Join(dir, "parse.conf")
	for path, content := range map[string]string{
		features: "name=value;class=Direct;slot_id=1;depend=value;args=1,1,double\n",
		config:   "feature_list=" + features + "\nfea_cols=traceid,label,value\nlabel=label\nadd_token=traceid\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
	}
	return config
}
