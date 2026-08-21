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
	if plugin.Install.Type != "direct" {
		t.Fatalf("install type = %q, want direct", plugin.Install.Type)
	}
	expected := map[string]struct {
		SHA256 string
		Size   int64
	}{
		"windows/amd64": {"edeb34edd62d48dea727106c02795124f6bed91be82574b165635f7805d87fe0", 2299090},
		"windows/arm64": {"002f5742abb6f1806ca97cefcf8108ecf8ccc63ba4551cb8b1ca78766198f362", 2265601},
		"linux/amd64":   {"906de1ed47b5d32e1de982c1eee057de65aefb9cffdbda897f3b34d9e5dcbd1b", 2317116},
		"linux/arm64":   {"f95d10bc2d16138f0c3208c5f53eca322cf26ce4993e267e3a5dc3c4294c6a5b", 2127623},
		"darwin/amd64":  {"d859bda610c8c38fb1070f86faf6f2792015b7a0f67df4602abb620471b3c3e1", 2213144},
		"darwin/arm64":  {"281860d48e1ee83dde31ae6527342098179dac6f5b568b1ecabb79165608520f", 2064551},
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
