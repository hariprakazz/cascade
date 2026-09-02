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

func TestParseCargoDeps(t *testing.T) {
	got, err := parseCargoDeps("packages/worker/Cargo.toml")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"auth"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadGraphIncludesCargoDeps(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.Mkdir(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "login.go"), []byte("package auth\n"), 0644); err != nil {
		t.Fatal(err)
	}

	workerDir := filepath.Join(dir, "worker")
	if err := os.Mkdir(workerDir, 0755); err != nil {
		t.Fatal(err)
	}
	cargoSrc := "[package]\nname = \"worker\"\nversion = \"0.1.0\"\n\n[dependencies]\nauth = { path = \"../auth\" }\n"
	if err := os.WriteFile(filepath.Join(workerDir, "Cargo.toml"), []byte(cargoSrc), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := loadGraph(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"auth":   {},
		"worker": {"auth"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestGetPackageFromCargoToml(t *testing.T) {
	got := getPackage("packages/worker/Cargo.toml")
	want := "worker"

	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestParseDubDeps(t *testing.T) {
	got, err := parseDubDeps("packages/tool/dub.json")
	if err != nil {
		t.Fatal(err)
	}

	want := []string{"auth"}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadGraphIncludesDubDeps(t *testing.T) {
	dir := t.TempDir()

	authDir := filepath.Join(dir, "auth")
	if err := os.Mkdir(authDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(authDir, "login.go"), []byte("package auth\n"), 0644); err != nil {
		t.Fatal(err)
	}

	toolDir := filepath.Join(dir, "tool")
	if err := os.Mkdir(toolDir, 0755); err != nil {
		t.Fatal(err)
	}
	dubSrc := `{"name":"tool","dependencies":{"auth":"~>1.0.0"}}`
	if err := os.WriteFile(filepath.Join(toolDir, "dub.json"), []byte(dubSrc), 0644); err != nil {
		t.Fatal(err)
	}

	got, err := loadGraph(dir)
	if err != nil {
		t.Fatal(err)
	}

	want := map[string][]string{
		"auth": {},
		"tool": {"auth"},
	}

	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %v, want %v", got, want)
	}
}

func TestLoadGraphHandlesRenamedFile(t *testing.T) {
	dir := t.TempDir()
	pkgDir := filepath.Join(dir, "renamed")
	if err := os.MkdirAll(pkgDir, 0755); err != nil {
		t.Fatal(err)
	}
	oldPath := filepath.Join(pkgDir, "old.go")
	newPath := filepath.Join(pkgDir, "new.go")
	src := []byte("package renamed\n\nimport \"github.com/hariprakazz/cascade/packages/auth\"\n")
	if err := os.WriteFile(oldPath, src, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(oldPath, newPath); err != nil {
		t.Fatal(err)
	}

	graph, err := loadGraph(dir)
	if err != nil {
		t.Fatalf("loadGraph failed: %v", err)
	}

	deps, ok := graph["renamed"]
	if !ok {
		t.Fatal("expected package 'renamed' in graph after file rename")
	}
	found := false
	for _, dep := range deps {
		if dep == "auth" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected renamed package to still show import of auth, got deps: %v", deps)
	}
}

func TestFindAffectedRejectsPhantomPackage(t *testing.T) {
	graph := map[string][]string{
		"auth": {"api"},
		"api":  {},
	}
	changed := []string{"ghost"}

	affected := findAffected(changed, graph)

	for _, pkg := range affected {
		if pkg == "ghost" {
			t.Fatalf("findAffected returned phantom package %q not present in graph", "ghost")
		}
	}
}
