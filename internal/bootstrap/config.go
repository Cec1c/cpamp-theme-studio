package bootstrap

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

const StoreSource = "https://raw.githubusercontent.com/Cec1c/cpamp-theme-studio/main/registry.json"

type ConfigInfo struct {
	PluginsDir          string
	Port                int
	DesiredVersion      string
	PluginConfigYAML    string
	PluginConfigPresent bool
}

type ConfigPatch struct {
	PluginsDir     string
	PanelPath      string
	ConfigPath     string
	RestartMode    string
	RestartService string
}

func InspectConfig(raw []byte, workingDir string) (ConfigInfo, error) {
	doc, root, err := parseDocument(raw)
	if err != nil {
		return ConfigInfo{}, err
	}
	_ = doc
	info := ConfigInfo{Port: 8317}
	if portNode := mappingValue(root, "port"); portNode != nil {
		port, errPort := strconv.Atoi(strings.TrimSpace(portNode.Value))
		if errPort != nil || port < 1 || port > 65535 {
			return ConfigInfo{}, fmt.Errorf("invalid CPA port %q", portNode.Value)
		}
		info.Port = port
	}
	plugins := mappingValue(root, "plugins")
	if plugins == nil || plugins.Kind != yaml.MappingNode {
		info.PluginsDir = filepath.Join(workingDir, "plugins")
		return info, nil
	}
	dir := strings.TrimSpace(scalarValue(mappingValue(plugins, "dir")))
	if dir == "" {
		dir = "plugins"
	}
	if !filepath.IsAbs(dir) {
		dir = filepath.Join(workingDir, dir)
	}
	info.PluginsDir = filepath.Clean(dir)
	configs := mappingValue(plugins, "configs")
	if configs == nil || configs.Kind != yaml.MappingNode {
		return info, nil
	}
	plugin := mappingValue(configs, PluginID)
	if plugin == nil {
		return info, nil
	}
	info.PluginConfigPresent = true
	pluginRaw, errEncode := encodeNode(plugin)
	if errEncode != nil {
		return ConfigInfo{}, fmt.Errorf("encode plugin config: %w", errEncode)
	}
	info.PluginConfigYAML = string(pluginRaw)
	if store := mappingValue(plugin, "store"); store != nil && store.Kind == yaml.MappingNode {
		info.DesiredVersion = normalizeVersion(scalarValue(mappingValue(store, "version")))
	}
	return info, nil
}

func PatchConfig(raw []byte, patch ConfigPatch) ([]byte, ConfigInfo, error) {
	doc, root, err := parseDocument(raw)
	if err != nil {
		return nil, ConfigInfo{}, err
	}
	plugins := ensureMapping(root, "plugins")
	setMappingValue(plugins, "enabled", boolNode(true))
	setMappingValue(plugins, "dir", stringNode(filepath.Clean(patch.PluginsDir)))
	ensureStoreSource(plugins)
	configs := ensureMapping(plugins, "configs")
	plugin := mappingValue(configs, PluginID)
	if plugin == nil || plugin.Kind != yaml.MappingNode {
		plugin = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMappingValue(configs, PluginID, plugin)
	}
	setMappingValue(plugin, "enabled", boolNode(true))
	setMappingValue(plugin, "auto_inject", boolNode(true))
	setMappingValue(plugin, "panel_path", stringNode(filepath.Clean(patch.PanelPath)))
	setMappingValue(plugin, "host_config_path", stringNode(filepath.Clean(patch.ConfigPath)))
	setMappingValue(plugin, "watch_seconds", intNode(3))
	setMappingValue(plugin, "restart_mode", stringNode(strings.TrimSpace(patch.RestartMode)))
	setMappingValue(plugin, "restart_service", stringNode(strings.TrimSpace(patch.RestartService)))

	encoded, errEncode := encodeDocument(doc)
	if errEncode != nil {
		return nil, ConfigInfo{}, errEncode
	}
	info, errInspect := InspectConfig(encoded, filepath.Dir(patch.ConfigPath))
	if errInspect != nil {
		return nil, ConfigInfo{}, errInspect
	}
	return encoded, info, nil
}

// ReassertBootstrapConfig restores only the settings owned by the bootstrap
// after a marketplace write. It intentionally preserves the plugin's enabled
// flag, auto_inject choice, store manifest, and all unrelated configuration.
func ReassertBootstrapConfig(raw []byte, patch ConfigPatch) ([]byte, ConfigInfo, error) {
	doc, root, err := parseDocument(raw)
	if err != nil {
		return nil, ConfigInfo{}, err
	}
	plugins := ensureMapping(root, "plugins")
	if strings.TrimSpace(patch.PluginsDir) != "" {
		setMappingValue(plugins, "dir", stringNode(filepath.Clean(patch.PluginsDir)))
	}
	ensureStoreSource(plugins)
	configs := ensureMapping(plugins, "configs")
	plugin := mappingValue(configs, PluginID)
	if plugin == nil || plugin.Kind != yaml.MappingNode {
		plugin = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setMappingValue(configs, PluginID, plugin)
	}
	setMappingValue(plugin, "panel_path", stringNode(filepath.Clean(patch.PanelPath)))
	setMappingValue(plugin, "host_config_path", stringNode(filepath.Clean(patch.ConfigPath)))
	setMappingValue(plugin, "watch_seconds", intNode(3))
	setMappingValue(plugin, "restart_mode", stringNode(strings.TrimSpace(patch.RestartMode)))
	setMappingValue(plugin, "restart_service", stringNode(strings.TrimSpace(patch.RestartService)))

	encoded, errEncode := encodeDocument(doc)
	if errEncode != nil {
		return nil, ConfigInfo{}, errEncode
	}
	info, errInspect := InspectConfig(encoded, filepath.Dir(patch.ConfigPath))
	if errInspect != nil {
		return nil, ConfigInfo{}, errInspect
	}
	return encoded, info, nil
}

func ensureStoreSource(plugins *yaml.Node) {
	sources := mappingValue(plugins, "store-sources")
	if sources == nil || sources.Kind != yaml.SequenceNode {
		sources = &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
		setMappingValue(plugins, "store-sources", sources)
	}
	for _, item := range sources.Content {
		if strings.TrimSpace(scalarValue(item)) == StoreSource {
			return
		}
	}
	sources.Content = append(sources.Content, stringNode(StoreSource))
}

func RestorePluginConfig(raw []byte, snapshot string, present bool) ([]byte, error) {
	doc, root, err := parseDocument(raw)
	if err != nil {
		return nil, err
	}
	plugins := ensureMapping(root, "plugins")
	configs := ensureMapping(plugins, "configs")
	if !present {
		deleteMappingValue(configs, PluginID)
	} else {
		var restored yaml.Node
		if errDecode := yaml.Unmarshal([]byte(snapshot), &restored); errDecode != nil {
			return nil, fmt.Errorf("decode accepted plugin config: %w", errDecode)
		}
		if restored.Kind == yaml.DocumentNode && len(restored.Content) == 1 {
			setMappingValue(configs, PluginID, cloneNode(restored.Content[0]))
		} else {
			return nil, fmt.Errorf("accepted plugin config is not a YAML document")
		}
	}
	return encodeDocument(doc)
}

func parseDocument(raw []byte) (*yaml.Node, *yaml.Node, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		raw = []byte("{}\n")
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		return nil, nil, fmt.Errorf("decode CPA config: %w", err)
	}
	if doc.Kind != yaml.DocumentNode || len(doc.Content) != 1 {
		return nil, nil, fmt.Errorf("CPA config must contain one YAML document")
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return nil, nil, fmt.Errorf("CPA config root must be a YAML mapping")
	}
	return &doc, root, nil
}

func encodeDocument(doc *yaml.Node) ([]byte, error) {
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(doc); err != nil {
		return nil, fmt.Errorf("encode CPA config: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("finish CPA config: %w", err)
	}
	return out.Bytes(), nil
}

func encodeNode(node *yaml.Node) ([]byte, error) {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{cloneNode(node)}}
	return encodeDocument(doc)
}

func mappingValue(mapping *yaml.Node, key string) *yaml.Node {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index] != nil && mapping.Content[index].Value == key {
			return mapping.Content[index+1]
		}
	}
	return nil
}

func ensureMapping(parent *yaml.Node, key string) *yaml.Node {
	value := mappingValue(parent, key)
	if value != nil && value.Kind == yaml.MappingNode {
		return value
	}
	value = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setMappingValue(parent, key, value)
	return value
}

func setMappingValue(mapping *yaml.Node, key string, value *yaml.Node) {
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index] != nil && mapping.Content[index].Value == key {
			mapping.Content[index+1] = value
			return
		}
	}
	mapping.Content = append(mapping.Content, stringNode(key), value)
}

func deleteMappingValue(mapping *yaml.Node, key string) {
	if mapping == nil || mapping.Kind != yaml.MappingNode {
		return
	}
	for index := 0; index+1 < len(mapping.Content); index += 2 {
		if mapping.Content[index] != nil && mapping.Content[index].Value == key {
			mapping.Content = append(mapping.Content[:index], mapping.Content[index+2:]...)
			return
		}
	}
}

func scalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return node.Value
}

func stringNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func boolNode(value bool) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!bool", Value: strconv.FormatBool(value)}
}

func intNode(value int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(value)}
}

func cloneNode(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	cloned := *node
	if len(node.Content) > 0 {
		cloned.Content = make([]*yaml.Node, len(node.Content))
		for index, child := range node.Content {
			cloned.Content[index] = cloneNode(child)
		}
	}
	return &cloned
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(value), "v")
}
