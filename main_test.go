package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
)

func TestFindAffectedLinearChain(t *testing.T) {
	graph := map[string][]string{
		"auth":      {},
		"api":       {"auth"},
		"dashboard": {"api"},
	}

	got := findAffected([]string{"auth"}, graph)
	sort.Strings(got)

	want := []string{"api", "auth", "dashboard"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFindAffectedCycle(t *testing.T) {
	graph := map[string][]string{
		"a": {"b"},
		"b": {"c"},
		"c": {"a"},
	}

	got := findAffected([]string{"a"}, graph)
	sort.Strings(got)

	want := []string{"a", "b", "c"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestFindAffectedDisconnected(t *testing.T) {
	graph := map[string][]string{
		"auth":      {},
		"api":       {"auth"},
		"dashboard": {"api"},
		"billing":   {},
	}

	got := findAffected([]string{"auth"}, graph)
	sort.Strings(got)

	want := []string{"api", "auth", "dashboard"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadGraph(t *testing.T) {
	dir := t.TempDir()

	pkgs := map[string]string{
		"auth": `{"name":"auth","deps":[]}`,
		"api":  `{"name":"api","deps":["auth"]}`,
	}

	for name, content := range pkgs {
		pkgDir := filepath.Join(dir, name)
		if err := os.Mkdir(pkgDir, 0755); err != nil {
			t.Fatal(err)
		}
		cfgPath := filepath.Join(pkgDir, "cascade.json")
		if err := os.WriteFile(cfgPath, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	got, err := loadGraph(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"auth": {},
		"api":  {"auth"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadGraphSkipsInvalidConfig(t *testing.T) {
	dir := t.TempDir()

	good := filepath.Join(dir, "auth")
	if err := os.Mkdir(good, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(good, "cascade.json"), []byte(`{"name":"auth","deps":[]}`), 0644); err != nil {
		t.Fatal(err)
	}

	broken := filepath.Join(dir, "api")
	if err := os.Mkdir(broken, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(broken, "cascade.json"), []byte(`{not valid json`), 0644); err != nil {
		t.Fatal(err)
	}

	missing := filepath.Join(dir, "dashboard")
	if err := os.Mkdir(missing, 0755); err != nil {
		t.Fatal(err)
	}

	got, err := loadGraph(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"auth": {},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}
