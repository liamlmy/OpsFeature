package feature

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestOutputAddTokenPreservesEmptyFields(t *testing.T) {
	for _, values := range [][]string{{"a", "b", "c"}, {"", "b", "c"}, {"a", "", "c"}, {"a", "b", ""}, {"", "", ""}} {
		input := map[string]string{"label": "1", "first": values[0], "second": values[1], "third": values[2]}
		got, err := OutputPerLineOp(input, []string{"label"}, []string{"first", "second", "third"}, nil)
		want := "1 #" + strings.Join(values, "\t") + "\t"
		if err != nil || got != want {
			t.Fatalf("values=%q: got %q, %v; want %q", values, got, err, want)
		}
	}
}

func TestOutputAddTokenRejectsMissingFieldAndDelimiters(t *testing.T) {
	for _, input := range []map[string]string{
		{"label": "1"},
		{"label": "1", "token": "a\tb"},
		{"label": "1", "token": "a\nb"},
		{"label": "1", "token": "a\rb"},
	} {
		got, err := OutputPerLineOp(input, []string{"label"}, []string{"token"}, nil)
		if err == nil || got != "" {
			t.Fatalf("input=%q: expected error and no output, got %q, %v", input, got, err)
		}
	}
}

func TestInitRejectsAmbiguousAddTokenLayout(t *testing.T) {
	for _, config := range []string{
		"fea_cols=a,b\nadd_token=missing\n",
		"fea_cols=a,b\nadd_token=a,a\n",
		"fea_cols=a,b\nadd_token=a, b\n",
		"fea_cols=a,b\nadd_token=a,\n",
		"fea_cols=a,a\nadd_token=a\n",
	} {
		dir := t.TempDir()
		path := filepath.Join(dir, "parse.conf")
		if err := os.WriteFile(path, []byte("feature_list=unused.conf\n"+config), 0600); err != nil {
			t.Fatal(err)
		}
		_, err := InitParser(path)
		if err == nil || strings.Contains(err.Error(), "open feature conf") {
			t.Fatalf("config=%q: expected layout rejection before feature loading, got %v", config, err)
		}
	}
}
