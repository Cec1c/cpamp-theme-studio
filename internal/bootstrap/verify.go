package bootstrap

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"
)

const (
	loaderStartMarker = "<!-- cpamp-theme-studio:start -->"
	loaderEndMarker   = "<!-- cpamp-theme-studio:end -->"
	maxPanelBytes     = 64 << 20
)

type Verifier struct {
	Systemd   Systemd
	Client    *http.Client
	PollEvery time.Duration
	Timeout   time.Duration
}

func (verifier Verifier) Verify(ctx context.Context, state State, oldPID int, expectedVersion string, expectPlugin bool) error {
	deadline := time.Now().Add(verifier.timeout())
	var lastError error
	for time.Now().Before(deadline) {
		service, errService := verifier.Systemd.Service(state.CPAService)
		if errService != nil {
			lastError = errService
			verifier.wait(ctx)
			continue
		}
		if oldPID > 0 && service.MainPID == oldPID {
			lastError = fmt.Errorf("CPA MainPID has not changed")
			verifier.wait(ctx)
			continue
		}
		if !verifier.Systemd.IsActive(state.CPAService) {
			lastError = fmt.Errorf("CPA systemd service is not active")
			verifier.wait(ctx)
			continue
		}
		if errHTTP := verifier.verifyHTTP(ctx, state, expectedVersion, expectPlugin); errHTTP != nil {
			lastError = errHTTP
			verifier.wait(ctx)
			continue
		}
		return nil
	}
	if lastError == nil {
		lastError = fmt.Errorf("verification timed out")
	}
	return lastError
}

func (verifier Verifier) verifyHTTP(ctx context.Context, state State, expectedVersion string, expectPlugin bool) error {
	client := verifier.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if expectPlugin {
		if strings.TrimSpace(state.ManagerService) != "" && strings.TrimSpace(state.PanelURL) == "" {
			return fmt.Errorf("Manager Server deployment has no public panel URL; refusing to verify an unbound panel")
		}
		resourceURL := strings.TrimRight(state.HealthURL, "/") + "/v0/resource/plugins/" + PluginID + "/studio"
		body, headers, status, errFetch := fetchLimited(ctx, client, resourceURL)
		if errFetch != nil {
			return fmt.Errorf("fetch plugin resource: %w", errFetch)
		}
		if status != http.StatusOK || !bytes.Contains(bytes.ToLower(body), []byte("cpamp theme studio")) {
			return fmt.Errorf("plugin resource returned HTTP %d or unexpected content", status)
		}
		if expectedVersion != "" && normalizeVersion(headers.Get("X-CPAMP-Theme-Studio-Version")) != normalizeVersion(expectedVersion) {
			return fmt.Errorf("plugin resource version %q does not match %q", headers.Get("X-CPAMP-Theme-Studio-Version"), expectedVersion)
		}
		panelRaw, errPanel := os.ReadFile(state.PanelPath)
		if errPanel != nil {
			return fmt.Errorf("read active panel: %w", errPanel)
		}
		if errMarkers := verifyPanelMarkers(panelRaw, expectedVersion); errMarkers != nil {
			return errMarkers
		}
		if strings.TrimSpace(state.PanelURL) != "" {
			publicRaw, _, publicStatus, errPublic := fetchLimited(ctx, client, state.PanelURL)
			if errPublic != nil {
				return fmt.Errorf("fetch public panel: %w", errPublic)
			}
			if publicStatus != http.StatusOK {
				return fmt.Errorf("public panel returned HTTP %d", publicStatus)
			}
			if errMarkers := verifyPanelMarkers(publicRaw, expectedVersion); errMarkers != nil {
				return fmt.Errorf("public panel validation: %w", errMarkers)
			}
		}
		return nil
	}
	_, _, status, errFetch := fetchLimited(ctx, client, state.HealthURL)
	if errFetch != nil {
		return fmt.Errorf("fetch CPA health URL: %w", errFetch)
	}
	if status >= 500 {
		return fmt.Errorf("CPA health URL returned HTTP %d", status)
	}
	return nil
}

func verifyPanelMarkers(raw []byte, version string) error {
	if bytes.Count(raw, []byte(loaderStartMarker)) != 1 || bytes.Count(raw, []byte(loaderEndMarker)) != 1 {
		return fmt.Errorf("Theme Studio marker count is not 1/1")
	}
	if version != "" && !bytes.Contains(raw, []byte("v="+normalizeVersion(version))) {
		return fmt.Errorf("panel loader version does not match %s", normalizeVersion(version))
	}
	return nil
}

func fetchLimited(ctx context.Context, client *http.Client, url string) ([]byte, http.Header, int, error) {
	request, errRequest := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if errRequest != nil {
		return nil, nil, 0, errRequest
	}
	request.Header.Set("Accept-Encoding", "identity")
	response, errDo := client.Do(request)
	if errDo != nil {
		return nil, nil, 0, errDo
	}
	defer response.Body.Close()
	reader := io.LimitReader(response.Body, maxPanelBytes+1)
	raw, errRead := io.ReadAll(reader)
	if errRead != nil {
		return nil, nil, response.StatusCode, errRead
	}
	if len(raw) > maxPanelBytes {
		return nil, nil, response.StatusCode, fmt.Errorf("response exceeds 64 MiB")
	}
	return raw, response.Header.Clone(), response.StatusCode, nil
}

func (verifier Verifier) timeout() time.Duration {
	if verifier.Timeout <= 0 {
		return 60 * time.Second
	}
	return verifier.Timeout
}

func (verifier Verifier) wait(ctx context.Context) {
	delay := verifier.PollEvery
	if delay <= 0 {
		delay = time.Second
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
	case <-timer.C:
	}
}
