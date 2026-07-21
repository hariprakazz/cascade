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

	authDir := filepath.Join(dir, "auth")
	if err := os.Mkdir(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	authSrc := "package auth\n\nfunc Login() string { return \"ok\" }\n"
	if err := os.WriteFile(filepath.Join(authDir, "login.go"), []byte(authSrc), 0644); err != nil {
		t.Fatal(err)
	}

	apiDir := filepath.Join(dir, "api")
	if err := os.Mkdir(apiDir, 0755); err != nil {
		t.Fatal(err)
	}
	apiSrc := "package api\n\nimport \"github.com/hariprakazz/cascade/packages/auth\"\n\nfunc Serve() string { return auth.Login() }\n"
	if err := os.WriteFile(filepath.Join(apiDir, "server.go"), []byte(apiSrc), 0644); err != nil {
		t.Fatal(err)
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

func TestLoadGraphSkipsUnparseableFile(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.Mkdir(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "login.go"), []byte("package auth\n"), 0644); err != nil {
		t.Fatal(err)
	}

	brokenDir := filepath.Join(dir, "broken")
	if err := os.Mkdir(brokenDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(brokenDir, "bad.go"), []byte("not even go code {{{"), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := loadGraph(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"auth":   {},
		"broken": {},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadGraphSkipsBuildTaggedFile(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.Mkdir(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "login.go"), []byte("package auth\n"), 0644); err != nil {
		t.Fatal(err)
	}

	linuxOnly := "//go:build linux\n\npackage auth\n\nimport \"github.com/hariprakazz/cascade/packages/dashboard\"\n"
	if err := os.WriteFile(filepath.Join(authDir, "linux_only.go"), []byte(linuxOnly), 0644); err != nil {
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

func TestLoadGraphSkipsTestFileImports(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.Mkdir(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "login.go"), []byte("package auth\n"), 0644); err != nil {
		t.Fatal(err)
	}

	testSrc := "package auth\n\nimport (\n\t\"testing\"\n\n\t\"github.com/hariprakazz/cascade/packages/dashboard\"\n)\n\nfunc TestSomething(t *testing.T) {\n\t_ = dashboard.Foo\n}\n"
	if err := os.WriteFile(filepath.Join(authDir, "login_test.go"), []byte(testSrc), 0644); err != nil {
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
