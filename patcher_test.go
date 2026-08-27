package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInjectLoaderHTMLIsIdempotent(t *testing.T) {
	original := []byte("<!doctype html><html><head><title>CPAMP</title></head><body></body></html>")
	first, changed, errInject := injectLoaderHTML(original)
	if errInject != nil || !changed {
		t.Fatalf("first inject changed=%t err=%v", changed, errInject)
	}
	if strings.Count(string(first), loaderStartMarker) != 1 {
		t.Fatalf("first inject marker count mismatch: %s", first)
	}
	second, changed, errInject := injectLoaderHTML(first)
	if errInject != nil {
		t.Fatalf("second inject: %v", errInject)
	}
	if changed || string(second) != string(first) {
		t.Fatalf("second inject should be idempotent, changed=%t", changed)
	}
}

func TestInjectLoaderHTMLSupportsUppercaseHead(t *testing.T) {
	original := []byte("<HTML><HEAD></HEAD><BODY></BODY></HTML>")
	out, _, errInject := injectLoaderHTML(original)
	if errInject != nil {
		t.Fatalf("inject: %v", errInject)
	}
	if strings.Index(string(out), loaderStartMarker) > strings.Index(string(out), "</HEAD>") {
		t.Fatal("loader was not inserted inside head")
	}
}

func TestInjectLoaderHTMLRejectsIncompleteMarker(t *testing.T) {
	_, _, errInject := injectLoaderHTML([]byte("<html><head>" + loaderStartMarker + "</head></html>"))
	if errInject == nil {
		t.Fatal("expected incomplete marker error")
	}
}

func TestInjectLoaderHTMLRejectsDuplicateMarkers(t *testing.T) {
	input := "<html><head>" + loaderStartMarker + loaderEndMarker + loaderStartMarker + loaderEndMarker + "</head></html>"
	if _, _, errInject := injectLoaderHTML([]byte(input)); errInject == nil {
		t.Fatal("expected duplicate markers to fail")
	}
}

func TestInjectLoaderHTMLRejectsReversedMarkers(t *testing.T) {
	input := "<html><head>" + loaderEndMarker + loaderStartMarker + "</head></html>"
	if _, _, errInject := injectLoaderHTML([]byte(input)); errInject == nil {
		t.Fatal("expected reversed markers to fail")
	}
}

func TestInjectLoaderHTMLRejectsNonHTML(t *testing.T) {
	if _, _, errInject := injectLoaderHTML([]byte("<head></head>")); errInject == nil {
		t.Fatal("expected document without html element to fail")
	}
}

func TestPatchAndUnpatchPanelFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "management.html")
	original := "<html><head></head><body>panel</body></html>"
	if errWrite := os.WriteFile(path, []byte(original), 0o640); errWrite != nil {
		t.Fatal(errWrite)
	}
	changed, errPatch := patchPanelFile(path)
	if errPatch != nil || !changed {
		t.Fatalf("patch changed=%t err=%v", changed, errPatch)
	}
	changed, errPatch = patchPanelFile(path)
	if errPatch != nil || changed {
		t.Fatalf("second patch changed=%t err=%v", changed, errPatch)
	}
	changed, errUnpatch := unpatchPanelFile(path)
	if errUnpatch != nil || !changed {
		t.Fatalf("unpatch changed=%t err=%v", changed, errUnpatch)
	}
	got, errRead := os.ReadFile(path)
	if errRead != nil {
		t.Fatal(errRead)
	}
	if string(got) != original {
		t.Fatalf("unpatched panel mismatch\ngot:  %s\nwant: %s", got, original)
	}
}

func TestPatchPanelFileRejectsSymlink(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target.html")
	link := filepath.Join(dir, "management.html")
	if errWrite := os.WriteFile(target, []byte("<html><head></head><body></body></html>"), 0o600); errWrite != nil {
		t.Fatal(errWrite)
	}
	if errLink := os.Symlink(target, link); errLink != nil {
		t.Skipf("symbolic links are unavailable in this test environment: %v", errLink)
	}
	if _, errPatch := patchPanelFile(link); errPatch == nil || !strings.Contains(errPatch.Error(), "symbolic link") {
		t.Fatalf("patchPanelFile() error = %v, want symbolic-link rejection", errPatch)
	}
}

func TestPatchPanelFileRejectsOversizedPanel(t *testing.T) {
	path := filepath.Join(t.TempDir(), "management.html")
	file, errCreate := os.Create(path)
	if errCreate != nil {
		t.Fatal(errCreate)
	}
	if errTruncate := file.Truncate(maxPanelSize + 1); errTruncate != nil {
		_ = file.Close()
		t.Fatal(errTruncate)
	}
	if errClose := file.Close(); errClose != nil {
		t.Fatal(errClose)
	}
	if _, errPatch := patchPanelFile(path); errPatch == nil || !strings.Contains(errPatch.Error(), "larger") {
		t.Fatalf("patchPanelFile() error = %v, want size rejection", errPatch)
	}
}

func TestResolvePanelCandidatesUsesConfiguredDirectory(t *testing.T) {
	dir := t.TempDir()
	want, _ := filepath.Abs(filepath.Join(dir, "management.html"))
	candidates, errResolve := resolvePanelCandidates(dir)
	if errResolve != nil {
		t.Fatal(errResolve)
	}
	found := false
	for _, candidate := range candidates {
		if strings.EqualFold(candidate, want) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("configured panel candidate %q missing from %#v", want, candidates)
	}
}

func TestResolvePanelCandidatesRejectsConflictingExplicitEnvironmentPaths(t *testing.T) {
	t.Setenv("CPAMP_THEME_PANEL_PATH", filepath.Join(t.TempDir(), "one.html"))
	t.Setenv("PANEL_PATH", filepath.Join(t.TempDir(), "two.html"))
	if _, errResolve := resolvePanelCandidates(""); errResolve == nil {
		t.Fatal("conflicting explicit panel paths were accepted")
	}
}

func TestResolvePanelCandidatesPrefersConfiguredPath(t *testing.T) {
	t.Setenv("CPAMP_THEME_PANEL_PATH", filepath.Join(t.TempDir(), "wrong.html"))
	dir := t.TempDir()
	candidates, errResolve := resolvePanelCandidates(filepath.Join(dir, "management.html"))
	if errResolve != nil {
		t.Fatal(errResolve)
	}
	if len(candidates) != 1 || !strings.EqualFold(candidates[0], filepath.Join(dir, "management.html")) {
		t.Fatalf("candidates = %#v", candidates)
	}
}
