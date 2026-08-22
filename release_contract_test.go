package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
)

type storeRegistry struct {
	SchemaVersion int           `json:"schema_version"`
	Plugins       []storePlugin `json:"plugins"`
}

type storePlugin struct {
	ID          string   `json:"id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Author      string   `json:"author"`
	Version     string   `json:"version"`
	Repository  string   `json:"repository"`
	Homepage    string   `json:"homepage"`
	License     string   `json:"license"`
	Tags        []string `json:"tags"`
	Install     struct {
		Type      string          `json:"type"`
		Artifacts []storeArtifact `json:"artifacts"`
	} `json:"install"`
}

type storeArtifact struct {
	GOOS   string `json:"goos"`
	GOARCH string `json:"goarch"`
	URL    string `json:"url"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
}

func TestRegistryMatchesCPAStoreV2DirectContract(t *testing.T) {
	raw, errRead := os.ReadFile("registry.json")
	if errRead != nil {
		t.Fatal(errRead)
	}
	var registry storeRegistry
	if errDecode := json.Unmarshal(raw, &registry); errDecode != nil {
		t.Fatal(errDecode)
	}
	if registry.SchemaVersion != 2 {
		t.Fatalf("schema_version = %d, want 2 for pinned direct installs", registry.SchemaVersion)
	}
	if len(registry.Plugins) != 1 {
		t.Fatalf("plugins = %d, want 1", len(registry.Plugins))
	}
	plugin := registry.Plugins[0]
	required := map[string]string{
		"id": plugin.ID, "name": plugin.Name, "description": plugin.Description,
		"author": plugin.Author, "version": plugin.Version, "repository": plugin.Repository,
	}
	for field, value := range required {
		if strings.TrimSpace(value) == "" {
			t.Fatalf("registry plugin is missing required field %s", field)
		}
	}
	if plugin.ID != pluginID {
		t.Fatalf("registry id = %q, want %q", plugin.ID, pluginID)
	}
	if plugin.Repository != pluginRepository {
		t.Fatalf("registry repository = %q, want %q", plugin.Repository, pluginRepository)
	}
	if strings.HasPrefix(strings.ToLower(plugin.Version), "v") {
		t.Fatalf("registry version %q must not include a v prefix", plugin.Version)
	}
	if plugin.Version != "0.1.4" {
		t.Fatalf("registry version = %q, want 0.1.4", plugin.Version)
	}
	if plugin.Install.Type != "direct" {
		t.Fatalf("install type = %q, want direct", plugin.Install.Type)
	}
	expected := map[string]struct {
		SHA256 string
		Size   int64
	}{
		"windows/amd64": {"10c67960f22bb057f681135abf9c4ef6fb7089e4d917ff8e8f007d42ca1d6359", 2393534},
		"windows/arm64": {"8c7a6f5f32808933a4bfc1b5f78be92c4646bf970b6217a02829c208752c7218", 2348279},
		"linux/amd64":   {"d870513e8c5afdca3d3816cfb2c7429a725561c32cd94c88400b1df8732429cd", 2407670},
		"linux/arm64":   {"c4137b5f30cae0e1d0c0993468b9fcdb5689fa422a43392bd6a1c7a476004695", 2212452},
		"darwin/amd64":  {"5786afff1da496692477d656b5b813d6a5f5e103b6e963ae26e4c958d030b803", 2283734},
		"darwin/arm64":  {"5d4bb5a6f5a7e98a68c034c06b53ef14b51f12b449c78b7bb9443710d5790ec3", 2131328},
	}
	if len(plugin.Install.Artifacts) != len(expected) {
		t.Fatalf("artifacts = %d, want %d", len(plugin.Install.Artifacts), len(expected))
	}
	hashPattern := regexp.MustCompile(`^[0-9a-f]{64}$`)
	seen := make(map[string]bool, len(expected))
	for _, artifact := range plugin.Install.Artifacts {
		platform := artifact.GOOS + "/" + artifact.GOARCH
		want, ok := expected[platform]
		if !ok {
			t.Errorf("unexpected artifact platform %q", platform)
			continue
		}
		if seen[platform] {
			t.Errorf("duplicate artifact platform %q", platform)
		}
		seen[platform] = true
		archive := fmt.Sprintf("cpamp-theme-studio_%s_%s_%s.zip", plugin.Version, artifact.GOOS, artifact.GOARCH)
		wantURL := fmt.Sprintf("%s/releases/download/v%s/%s", plugin.Repository, plugin.Version, archive)
		if artifact.URL != wantURL {
			t.Errorf("%s url = %q, want %q", platform, artifact.URL, wantURL)
		}
		if !hashPattern.MatchString(artifact.SHA256) || artifact.SHA256 != want.SHA256 {
			t.Errorf("%s sha256 = %q, want %q", platform, artifact.SHA256, want.SHA256)
		}
		if artifact.Size != want.Size {
			t.Errorf("%s size = %d, want %d", platform, artifact.Size, want.Size)
		}
	}
}

func TestReleaseWorkflowPublishesCPAStoreAssets(t *testing.T) {
	raw, errRead := os.ReadFile(".github/workflows/release.yml")
	if errRead != nil {
		t.Fatal(errRead)
	}
	workflow := string(raw)
	for _, target := range []string{
		"windows-amd64", "windows-arm64", "linux-amd64",
		"linux-arm64", "darwin-amd64", "darwin-arm64",
	} {
		if !strings.Contains(workflow, "target: "+target) {
			t.Errorf("release workflow is missing target %s", target)
		}
	}
	for _, asset := range []string{"dist/cpamp-theme-studio_*.zip", "dist/checksums.txt", "dist/registry.json"} {
		if !strings.Contains(workflow, asset) {
			t.Errorf("release workflow is missing asset rule %s", asset)
		}
	}
	if !strings.Contains(workflow, "python scripts/generate_registry.py") {
		t.Error("release workflow does not generate the pinned registry from packaged assets")
	}
}

func TestReleasePackagesIncludeJetBrainsMonoLicense(t *testing.T) {
	for _, testCase := range []struct {
		path     string
		required []string
	}{
		{
			path: "scripts/package.ps1",
			required: []string{
				"assets\\fonts\\OFL.txt",
				"JETBRAINS_MONO_OFL.txt",
			},
		},
		{
			path: "scripts/package.sh",
			required: []string{
				"assets/fonts/OFL.txt",
				"JETBRAINS_MONO_OFL.txt",
			},
		},
	} {
		t.Run(testCase.path, func(t *testing.T) {
			raw, errRead := os.ReadFile(testCase.path)
			if errRead != nil {
				t.Fatal(errRead)
			}
			for _, required := range testCase.required {
				if !strings.Contains(string(raw), required) {
					t.Fatalf("%s is missing %q", testCase.path, required)
				}
			}
		})
	}
}
