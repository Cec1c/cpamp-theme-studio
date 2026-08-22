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
	if plugin.Version != "0.1.5" {
		t.Fatalf("registry version = %q, want 0.1.5", plugin.Version)
	}
	if plugin.Install.Type != "direct" {
		t.Fatalf("install type = %q, want direct", plugin.Install.Type)
	}
	expected := map[string]struct {
		SHA256 string
		Size   int64
	}{
		"windows/amd64": {"e383ef3f38b5d659313134469f444d53303892ae0f74f0d527c0f396fc75d4a4", 2395699},
		"windows/arm64": {"eec7c3706e3f5859fe4ded32a52ce7ac5a96bcac701b25947b6f66042fbfabd3", 2350350},
		"linux/amd64":   {"490447b0e853950c14c1fbfc6274b725d7df5de8c00a0273af1c14bc100c9131", 2409705},
		"linux/arm64":   {"bac4ce3ba1cd3d9f1c332c1a135931155098abeba302cca0b8e5dcfcd09c4410", 2214473},
		"darwin/amd64":  {"b83f28c69e2cb03049329406fabb36a3bdab6276e23e75a64ed39525048c597a", 2285698},
		"darwin/arm64":  {"84fe0360b691be4551c3175761a97b0333c6683ec098f7e816586eed649ad8cc", 2133417},
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
	for _, asset := range []string{"dist/cpamp-theme-studio_*.zip", "dist/bootstrap-linux.sh", "dist/checksums.txt", "dist/registry.json"} {
		if !strings.Contains(workflow, asset) {
			t.Errorf("release workflow is missing asset rule %s", asset)
		}
	}
	for _, required := range []string{
		"install -m 0755 scripts/bootstrap-linux.sh dist/bootstrap-linux.sh",
		"sha256sum bootstrap-linux.sh cpamp-theme-studio_*.zip",
	} {
		if !strings.Contains(workflow, required) {
			t.Errorf("release workflow is missing bootstrap contract %q", required)
		}
	}
	if !strings.Contains(workflow, "python scripts/generate_registry.py") {
		t.Error("release workflow does not generate the pinned registry from packaged assets")
	}
}

func TestLinuxReleasePackagesIncludeBootstrap(t *testing.T) {
	raw, errRead := os.ReadFile("scripts/package.sh")
	if errRead != nil {
		t.Fatal(errRead)
	}
	script := string(raw)
	for _, required := range []string{
		"./cmd/cpamp-theme-bootstrap",
		"cpamp-theme-bootstrap",
		"scripts/bootstrap-linux.sh",
		"bootstrap-linux.sh",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("Linux package is missing bootstrap contract %q", required)
		}
	}
	if strings.Contains(script, "${extra_files[@]}") {
		t.Error("Linux package uses an empty array expansion that fails under macOS Bash 3.2 with nounset")
	}
}

func TestLinuxBootstrapDownloaderVerifiesPinnedRelease(t *testing.T) {
	raw, errRead := os.ReadFile("scripts/bootstrap-linux.sh")
	if errRead != nil {
		t.Fatal(errRead)
	}
	script := string(raw)
	for _, required := range []string{
		"--bootstrap-version",
		"--download-proxy",
		"--proto '=https'",
		"--tlsv1.2",
		"checksums.txt",
		"sha256sum",
		"SHA-256 mismatch",
		"export HTTP_PROXY=",
		"export HTTPS_PROXY=",
		"exec \"${temporary}/cpamp-theme-bootstrap\"",
	} {
		if !strings.Contains(script, required) {
			t.Errorf("Linux bootstrap downloader is missing %q", required)
		}
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
