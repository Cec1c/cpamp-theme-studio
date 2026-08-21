package main

/*
#include <stdint.h>
#include <stdlib.h>

typedef struct {
	void* ptr;
	size_t len;
} cliproxy_buffer;

typedef int (*cliproxy_host_call_fn)(void*, const char*, const uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_host_free_fn)(void*, size_t);

typedef struct {
	uint32_t abi_version;
	void* host_ctx;
	cliproxy_host_call_fn call;
	cliproxy_host_free_fn free_buffer;
} cliproxy_host_api;

typedef int (*cliproxy_plugin_call_fn)(char*, uint8_t*, size_t, cliproxy_buffer*);
typedef void (*cliproxy_plugin_free_fn)(void*, size_t);
typedef void (*cliproxy_plugin_shutdown_fn)(void);

typedef struct {
	uint32_t abi_version;
	cliproxy_plugin_call_fn call;
	cliproxy_plugin_free_fn free_buffer;
	cliproxy_plugin_shutdown_fn shutdown;
} cliproxy_plugin_api;

extern int cliproxyPluginCall(char*, uint8_t*, size_t, cliproxy_buffer*);
extern void cliproxyPluginFree(void*, size_t);
extern void cliproxyPluginShutdown(void);
*/
import "C"

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unsafe"
)

const (
	pluginID         = "cpamp-theme-studio"
	pluginName       = "CPAMP Theme Studio"
	pluginRepository = "https://github.com/Cec1c/cpamp-theme-studio"
	abiVersion       = 1
	// Schema v2 keeps the plugin compatible with CPA versions before schema v3.
	// Theme Studio only uses the management resource contract shared by both.
	schemaVersion = 2

	methodPluginRegister     = "plugin.register"
	methodPluginReconfigure  = "plugin.reconfigure"
	methodManagementRegister = "management.register"
	methodManagementHandle   = "management.handle"
)

// pluginVersion is injected by release builds with -ldflags -X=main.pluginVersion=<version>.
var pluginVersion = "0.0.0-dev"

//go:embed assets/loader.js
var loaderJavaScript []byte

type envelope struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  *envelopeError  `json:"error,omitempty"`
}

type envelopeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type lifecycleRequest struct {
	ConfigYAML []byte `json:"config_yaml"`
}

type registration struct {
	SchemaVersion uint32                   `json:"schema_version"`
	Metadata      metadata                 `json:"metadata"`
	Capabilities  registrationCapabilities `json:"capabilities"`
}

type metadata struct {
	Name             string        `json:"Name"`
	Version          string        `json:"Version"`
	Author           string        `json:"Author"`
	GitHubRepository string        `json:"GitHubRepository"`
	Logo             string        `json:"Logo"`
	ConfigFields     []configField `json:"ConfigFields"`
}

type configField struct {
	Name        string   `json:"Name"`
	Type        string   `json:"Type"`
	EnumValues  []string `json:"EnumValues,omitempty"`
	Description string   `json:"Description"`
}

type registrationCapabilities struct {
	ManagementAPI bool `json:"management_api"`
}

type managementRegistration struct {
	Resources []managementResource `json:"resources"`
}

type managementResource struct {
	Path        string `json:"Path"`
	Menu        string `json:"Menu"`
	Description string `json:"Description"`
}

type managementRequest struct {
	Method  string
	Path    string
	Headers http.Header
	Query   url.Values
	Body    []byte
}

type managementResponse struct {
	StatusCode int         `json:"StatusCode"`
	Headers    http.Header `json:"Headers"`
	Body       []byte      `json:"Body"`
}

func main() {}

//export cliproxy_plugin_init
func cliproxy_plugin_init(_ *C.cliproxy_host_api, plugin *C.cliproxy_plugin_api) C.int {
	if plugin == nil {
		return 1
	}
	plugin.abi_version = C.uint32_t(abiVersion)
	plugin.call = C.cliproxy_plugin_call_fn(C.cliproxyPluginCall)
	plugin.free_buffer = C.cliproxy_plugin_free_fn(C.cliproxyPluginFree)
	plugin.shutdown = C.cliproxy_plugin_shutdown_fn(C.cliproxyPluginShutdown)
	return 0
}

//export cliproxyPluginCall
func cliproxyPluginCall(method *C.char, request *C.uint8_t, requestLen C.size_t, response *C.cliproxy_buffer) C.int {
	if response != nil {
		response.ptr = nil
		response.len = 0
	}
	if method == nil {
		writeResponse(response, errorEnvelope("invalid_method", "method is required"))
		return 1
	}
	var rawRequest []byte
	if request != nil && requestLen > 0 {
		rawRequest = C.GoBytes(unsafe.Pointer(request), C.int(requestLen))
	}
	rawResponse, errHandle := handleMethod(C.GoString(method), rawRequest)
	if errHandle != nil {
		writeResponse(response, errorEnvelope("plugin_error", errHandle.Error()))
		return 1
	}
	writeResponse(response, rawResponse)
	return 0
}

//export cliproxyPluginFree
func cliproxyPluginFree(ptr unsafe.Pointer, _ C.size_t) {
	if ptr != nil {
		C.free(ptr)
	}
}

//export cliproxyPluginShutdown
func cliproxyPluginShutdown() {
	stopInjector(true)
}

func handleMethod(method string, request []byte) ([]byte, error) {
	switch method {
	case methodPluginRegister, methodPluginReconfigure:
		var lifecycle lifecycleRequest
		if len(request) > 0 {
			if errUnmarshal := json.Unmarshal(request, &lifecycle); errUnmarshal != nil {
				return nil, fmt.Errorf("decode lifecycle request: %w", errUnmarshal)
			}
		}
		cfg, errConfig := parseInjectorConfig(lifecycle.ConfigYAML)
		if errConfig != nil {
			return nil, errConfig
		}
		configureInjector(cfg)
		return okEnvelope(pluginRegistration())
	case methodManagementRegister:
		return okEnvelope(managementRegistration{Resources: []managementResource{{
			Path:        "/studio",
			Menu:        "Theme Studio",
			Description: "Customize CPAMP color, shape, density, typography, layout, and visual effects.",
		}}})
	case methodManagementHandle:
		return handleManagement(request)
	default:
		return errorEnvelope("unknown_method", "unknown method: "+method), nil
	}
}

func pluginRegistration() registration {
	return registration{
		SchemaVersion: schemaVersion,
		Metadata: metadata{
			Name:             pluginName,
			Version:          pluginVersion,
			Author:           "Cec1c",
			GitHubRepository: pluginRepository,
			ConfigFields: []configField{
				{Name: "auto_inject", Type: "boolean", Description: "Inject the Theme Studio loader into a writable official management.html."},
				{Name: "panel_path", Type: "string", Description: "Optional management.html file or directory. Required when CPAMP Manager Server uses PANEL_PATH."},
				{Name: "host_config_path", Type: "string", Description: "Optional CPA config.yaml used to remove the loader promptly when the plugin is disabled."},
				{Name: "watch_seconds", Type: "integer", Description: "Panel update check interval in seconds (1-300, default 3)."},
			},
		},
		Capabilities: registrationCapabilities{ManagementAPI: true},
	}
}

func handleManagement(raw []byte) ([]byte, error) {
	var req managementRequest
	if len(raw) > 0 {
		if errUnmarshal := json.Unmarshal(raw, &req); errUnmarshal != nil {
			return nil, fmt.Errorf("decode management request: %w", errUnmarshal)
		}
	}
	method := strings.ToUpper(strings.TrimSpace(req.Method))
	if method == "" {
		method = http.MethodGet
	}
	if method != http.MethodGet && method != http.MethodHead {
		return okEnvelope(resourceResponse(http.StatusMethodNotAllowed, "text/plain; charset=utf-8", []byte("method not allowed"), map[string]string{
			"Allow": "GET, HEAD",
		}))
	}

	asset := strings.TrimSpace(req.Query.Get("asset"))
	switch asset {
	case "loader":
		body := loaderJavaScript
		if method == http.MethodHead {
			body = nil
		}
		return okEnvelope(resourceResponse(http.StatusOK, "application/javascript; charset=utf-8", body, map[string]string{
			"Cache-Control":                "public, max-age=31536000, immutable",
			"Cross-Origin-Resource-Policy": "same-origin",
			"ETag":                         loaderETag(),
		}))
	case "":
		body := renderStudioLanding(snapshotInjectorStatus())
		if method == http.MethodHead {
			body = nil
		}
		return okEnvelope(resourceResponse(http.StatusOK, "text/html; charset=utf-8", body, map[string]string{
			"Cache-Control":           "no-store",
			"Content-Security-Policy": "default-src 'self'; script-src 'self'; style-src 'unsafe-inline'; img-src 'self' data:; frame-ancestors 'self'; base-uri 'none'; form-action 'none'",
			"Referrer-Policy":         "no-referrer",
			"X-Frame-Options":         "SAMEORIGIN",
		}))
	default:
		return okEnvelope(resourceResponse(http.StatusNotFound, "text/plain; charset=utf-8", []byte("asset not found"), map[string]string{
			"Cache-Control": "no-store",
		}))
	}
}

func resourceResponse(status int, contentType string, body []byte, extra map[string]string) managementResponse {
	headers := http.Header{
		"Content-Type":                 []string{contentType},
		"X-Content-Type-Options":       []string{"nosniff"},
		"X-CPAMP-Theme-Studio-Version": []string{pluginVersion},
		"Cross-Origin-Opener-Policy":   []string{"same-origin"},
		"Cross-Origin-Resource-Policy": []string{"same-origin"},
	}
	for key, value := range extra {
		headers.Set(key, value)
	}
	return managementResponse{StatusCode: status, Headers: headers, Body: body}
}

func loaderETag() string {
	digest := sha256.Sum256(append(append([]byte(nil), loaderJavaScript...), []byte(pluginVersion)...))
	return `"` + hex.EncodeToString(digest[:16]) + `"`
}

func renderStudioLanding(status injectorStatus) []byte {
	stateEN := "Waiting for a writable official management.html."
	stateZH := "正在等待可写的官方 management.html。"
	if !status.Configured {
		stateEN = "Automatic panel injection is disabled. The editor can only affect this frame."
		stateZH = "自动面板注入已禁用；编辑器只能影响当前插件页面。"
	} else if status.PatchedCount > 0 {
		stateEN = "Theme Studio is active in the host panel."
		stateZH = "主题工作室已在宿主面板中启用。"
	} else if status.LastError != "" {
		stateEN = "The configured panel could not be patched. Check the path and file permissions."
		stateZH = "无法注入已配置的面板，请检查路径和文件权限。"
	}
	lastCheck := "never"
	if !status.LastCheck.IsZero() {
		lastCheck = status.LastCheck.UTC().Format(time.RFC3339)
	}
	page := `<!doctype html>
<html lang="zh-CN">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width,initial-scale=1">
  <title>CPAMP Theme Studio</title>
  <style>
    :root{color-scheme:light dark;font-family:Inter,-apple-system,BlinkMacSystemFont,"Segoe UI",sans-serif}
    *{box-sizing:border-box}body{display:grid;min-height:100vh;margin:0;place-items:center;background:var(--app-bg,#f3f6fa);color:var(--app-text-primary,#243041)}
    main{width:min(620px,calc(100% - 32px));padding:28px;border:1px solid var(--app-border,#dbe2ea);border-radius:18px;background:var(--app-surface-strong,#fff);box-shadow:0 18px 55px rgb(15 23 42/.10)}
    h1{margin:0 0 8px;font-size:22px}p{margin:8px 0;color:var(--app-text-regular,#58677a);line-height:1.65}.meta{font-family:ui-monospace,SFMono-Regular,Consolas,monospace;font-size:12px;color:var(--app-text-muted,#7b8797)}
  </style>
</head>
<body>
  <main>
    <h1>CPAMP Theme Studio</h1>
    <p>` + html.EscapeString(stateZH) + `</p>
    <p lang="en">` + html.EscapeString(stateEN) + `</p>
    <p class="meta">version ` + html.EscapeString(pluginVersion) + ` · patched ` + fmt.Sprintf("%d/%d", status.PatchedCount, status.CandidateCount) + ` · checked ` + html.EscapeString(lastCheck) + `</p>
  </main>
  <script src="?asset=loader&amp;open=1&amp;v=` + url.QueryEscape(pluginVersion) + `"></script>
</body>
</html>`
	return []byte(page)
}

func okEnvelope(value any) ([]byte, error) {
	result, errMarshal := json.Marshal(value)
	if errMarshal != nil {
		return nil, errMarshal
	}
	return json.Marshal(envelope{OK: true, Result: result})
}

func errorEnvelope(code, message string) []byte {
	raw, _ := json.Marshal(envelope{OK: false, Error: &envelopeError{Code: code, Message: message}})
	return raw
}

func writeResponse(response *C.cliproxy_buffer, raw []byte) {
	if response == nil || len(raw) == 0 {
		return
	}
	ptr := C.CBytes(raw)
	if ptr == nil {
		return
	}
	response.ptr = ptr
	response.len = C.size_t(len(raw))
}
