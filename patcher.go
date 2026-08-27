package main

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	loaderStartMarker = "<!-- cpamp-theme-studio:start -->"
	loaderEndMarker   = "<!-- cpamp-theme-studio:end -->"
	maxPanelSize      = 64 << 20
)

func loaderBlock() []byte {
	return []byte(loaderStartMarker + "\n" +
		`<script data-cpamp-theme-studio-loader src="/v0/resource/plugins/cpamp-theme-studio/studio?asset=loader&amp;v=` + pluginVersion + `"></script>` + "\n" +
		loaderEndMarker + "\n")
}

func injectLoaderHTML(input []byte) ([]byte, bool, error) {
	cleaned, _, errRemove := removeLoaderHTML(input)
	if errRemove != nil {
		return nil, false, errRemove
	}
	lower := bytes.ToLower(cleaned)
	if !bytes.Contains(lower, []byte("<html")) {
		return nil, false, fmt.Errorf("management panel does not contain an html element")
	}
	insertAt := bytes.Index(lower, []byte("</head>"))
	if insertAt < 0 {
		insertAt = bytes.Index(lower, []byte("</body>"))
	}
	if insertAt < 0 {
		return nil, false, fmt.Errorf("management panel has no closing head or body tag")
	}
	block := loaderBlock()
	out := make([]byte, 0, len(cleaned)+len(block))
	out = append(out, cleaned[:insertAt]...)
	out = append(out, block...)
	out = append(out, cleaned[insertAt:]...)
	return out, !bytes.Equal(out, input), nil
}

func removeLoaderHTML(input []byte) ([]byte, bool, error) {
	startToken := []byte(loaderStartMarker)
	endToken := []byte(loaderEndMarker)
	startCount := bytes.Count(input, startToken)
	endCount := bytes.Count(input, endToken)
	if startCount == 0 && endCount == 0 {
		return append([]byte(nil), input...), false, nil
	}
	if startCount != 1 || endCount != 1 {
		return nil, false, fmt.Errorf("management panel contains incomplete or duplicate Theme Studio markers")
	}
	start := bytes.Index(input, startToken)
	end := bytes.Index(input, endToken)
	if end < start {
		return nil, false, fmt.Errorf("management panel contains reversed Theme Studio markers")
	}
	end += len(endToken)
	for end < len(input) && (input[end] == '\r' || input[end] == '\n') {
		end++
	}
	out := make([]byte, 0, len(input)-(end-start))
	out = append(out, input[:start]...)
	out = append(out, input[end:]...)
	return out, true, nil
}

func patchPanelFile(path string) (bool, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	info, errValidate := validatePanelFile(path)
	if errValidate != nil {
		return false, errValidate
	}
	input, errRead := os.ReadFile(path)
	if errRead != nil {
		return false, errRead
	}
	out, changed, errInject := injectLoaderHTML(input)
	if errInject != nil || !changed {
		return changed, errInject
	}
	if errWrite := writePanelAtomically(path, out, info.Mode().Perm()); errWrite != nil {
		return false, errWrite
	}
	return true, nil
}

func unpatchPanelFile(path string) (bool, error) {
	path = filepath.Clean(strings.TrimSpace(path))
	info, errValidate := validatePanelFile(path)
	if errValidate != nil {
		return false, errValidate
	}
	input, errRead := os.ReadFile(path)
	if errRead != nil {
		return false, errRead
	}
	out, changed, errRemove := removeLoaderHTML(input)
	if errRemove != nil || !changed {
		return changed, errRemove
	}
	if errWrite := writePanelAtomically(path, out, info.Mode().Perm()); errWrite != nil {
		return false, errWrite
	}
	return true, nil
}

func validatePanelFile(path string) (os.FileInfo, error) {
	if path == "." || path == "" {
		return nil, fmt.Errorf("empty panel path")
	}
	linkInfo, errLstat := os.Lstat(path)
	if errLstat != nil {
		return nil, errLstat
	}
	if linkInfo.Mode()&os.ModeSymlink != 0 {
		return nil, fmt.Errorf("panel path must not be a symbolic link")
	}
	if !linkInfo.Mode().IsRegular() {
		return nil, fmt.Errorf("panel path is not a regular file")
	}
	if !strings.EqualFold(filepath.Ext(path), ".html") {
		return nil, fmt.Errorf("panel path must be an html file")
	}
	if linkInfo.Size() > maxPanelSize {
		return nil, fmt.Errorf("panel is larger than %d bytes", maxPanelSize)
	}
	return linkInfo, nil
}

func writePanelAtomically(path string, data []byte, mode os.FileMode) error {
	directory := filepath.Dir(path)
	temporary, errCreate := os.CreateTemp(directory, ".cpamp-theme-studio-*.tmp")
	if errCreate != nil {
		return errCreate
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		_ = os.Remove(temporaryPath)
	}()
	if _, errWrite := temporary.Write(data); errWrite != nil {
		return errWrite
	}
	if errSync := temporary.Sync(); errSync != nil {
		return errSync
	}
	if errChmod := temporary.Chmod(mode); errChmod != nil {
		return errChmod
	}
	if errClose := temporary.Close(); errClose != nil {
		return errClose
	}
	if errReplace := replacePathAtomic(temporaryPath, path); errReplace != nil {
		return errReplace
	}
	return nil
}

func resolvePanelCandidates(configuredPath string) ([]string, error) {
	normalizeCandidate := func(value string) string {
		value = strings.TrimSpace(value)
		if value == "" {
			return ""
		}
		value = filepath.Clean(value)
		if info, errStat := os.Stat(value); errStat == nil && info.IsDir() {
			value = filepath.Join(value, "management.html")
		} else if filepath.Ext(value) == "" {
			value = filepath.Join(value, "management.html")
		}
		if absolute, errAbs := filepath.Abs(value); errAbs == nil {
			value = absolute
		}
		return value
	}
	if configured := normalizeCandidate(configuredPath); configured != "" {
		return []string{configured}, nil
	}

	explicit := make([]string, 0, 3)
	for _, name := range []string{"CPAMP_THEME_PANEL_PATH", "MANAGEMENT_STATIC_PATH", "PANEL_PATH"} {
		if candidate := normalizeCandidate(os.Getenv(name)); candidate != "" {
			explicit = append(explicit, candidate)
		}
	}
	if result := uniquePanelCandidates(explicit); len(result) == 1 {
		return result, nil
	} else if len(result) > 1 {
		return nil, fmt.Errorf("multiple explicit panel paths are configured; set panel_path to the active management.html")
	}

	candidates := make([]string, 0, 4)
	appendExisting := func(value string) {
		candidate := normalizeCandidate(value)
		if candidate == "" {
			return
		}
		if _, errStat := os.Stat(candidate); errStat == nil {
			candidates = append(candidates, candidate)
		}
	}
	if cwd, errCWD := os.Getwd(); errCWD == nil {
		appendExisting(filepath.Join(cwd, "static", "management.html"))
		appendExisting(filepath.Join(cwd, "management.html"))
	}
	if executable, errExecutable := os.Executable(); errExecutable == nil {
		directory := filepath.Dir(executable)
		appendExisting(filepath.Join(directory, "static", "management.html"))
		appendExisting(filepath.Join(directory, "management.html"))
	}
	result := uniquePanelCandidates(candidates)
	if len(result) > 1 {
		return nil, fmt.Errorf("found %d possible panel files; set panel_path to the active management.html", len(result))
	}
	return result, nil
}

func uniquePanelCandidates(candidates []string) []string {
	seen := make(map[string]struct{}, len(candidates))
	result := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		key := strings.ToLower(candidate)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, candidate)
	}
	return result
}
