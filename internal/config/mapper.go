package config

import (
	"fmt"

	"github.com/vagrant-mcp/server/internal/core"
)

// ConfigMapper provides unified configuration mapping functionality
type ConfigMapper struct {
	fieldMappings map[string]func(interface{}) (interface{}, error)
}

// NewConfigMapper creates a new configuration mapper
func NewConfigMapper() *ConfigMapper {
	mapper := &ConfigMapper{
		fieldMappings: make(map[string]func(interface{}) (interface{}, error)),
	}
	mapper.registerDefaultMappings()
	return mapper
}

// registerDefaultMappings registers the default field mappings
func (m *ConfigMapper) registerDefaultMappings() {
	m.fieldMappings["cpu"] = m.mapToInt
	m.fieldMappings["memory"] = m.mapToInt
	m.fieldMappings["box"] = m.mapToString
	m.fieldMappings["sync_type"] = m.mapToString
	m.fieldMappings["project_path"] = m.mapToString
	m.fieldMappings["environment"] = m.mapToStringSlice
	m.fieldMappings["sync_exclude_patterns"] = m.mapToStringSlice
	m.fieldMappings["ports"] = m.mapToPorts
}

// MapToVMConfig maps a generic configuration to a VMConfig struct
func (m *ConfigMapper) MapToVMConfig(configMap map[string]interface{}) (*core.VMConfig, error) {
	config := &core.VMConfig{}

	for key, value := range configMap {
		if mapper, exists := m.fieldMappings[key]; exists {
			mappedValue, err := mapper(value)
			if err != nil {
				return nil, fmt.Errorf("failed to map field %s: %w", key, err)
			}
			if err := m.setFieldValue(config, key, mappedValue); err != nil {
				return nil, fmt.Errorf("failed to set field %s: %w", key, err)
			}
		}
	}

	return config, nil
}

// ApplyCustomizations applies customizations to an existing config
func (m *ConfigMapper) ApplyCustomizations(config *core.VMConfig, customizations map[string]interface{}) error {
	for key, value := range customizations {
		if mapper, exists := m.fieldMappings[key]; exists {
			mappedValue, err := mapper(value)
			if err != nil {
				return fmt.Errorf("failed to map customization %s: %w", key, err)
			}
			if err := m.setFieldValue(config, key, mappedValue); err != nil {
				return fmt.Errorf("failed to apply customization %s: %w", key, err)
			}
		}
	}
	return nil
}

// setFieldValue sets a field value on the config struct
func (m *ConfigMapper) setFieldValue(config *core.VMConfig, key string, value interface{}) error {
	switch key {
	case "cpu":
		if v, ok := value.(int); ok {
			config.CPU = v
		}
	case "memory":
		if v, ok := value.(int); ok {
			config.Memory = v
		}
	case "box":
		if v, ok := value.(string); ok {
			config.Box = v
		}
	case "sync_type":
		if v, ok := value.(string); ok {
			config.SyncType = v
		}
	case "project_path":
		if v, ok := value.(string); ok {
			config.ProjectPath = v
		}
	case "environment":
		if v, ok := value.([]string); ok {
			config.Environment = v
		}
	case "sync_exclude_patterns":
		if v, ok := value.([]string); ok {
			config.SyncExcludePatterns = v
		}
	case "ports":
		if v, ok := value.([]core.Port); ok {
			config.Ports = v
		}
	default:
		return fmt.Errorf("unknown field: %s", key)
	}
	return nil
}

// Mapping functions
func (m *ConfigMapper) mapToInt(value interface{}) (interface{}, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case int64:
		return int(v), nil
	default:
		return nil, fmt.Errorf("cannot convert %v (%T) to int", value, value)
	}
}

func (m *ConfigMapper) mapToString(value interface{}) (interface{}, error) {
	if v, ok := value.(string); ok {
		return v, nil
	}
	return nil, fmt.Errorf("cannot convert %v (%T) to string", value, value)
}

func (m *ConfigMapper) mapToStringSlice(value interface{}) (interface{}, error) {
	if v, ok := value.([]string); ok {
		return v, nil
	}

	// Handle []interface{} containing strings
	if interfaceSlice, ok := value.([]interface{}); ok {
		stringSlice := make([]string, len(interfaceSlice))
		for i, item := range interfaceSlice {
			if str, ok := item.(string); ok {
				stringSlice[i] = str
			} else {
				return nil, fmt.Errorf("slice contains non-string element: %v (%T)", item, item)
			}
		}
		return stringSlice, nil
	}

	return nil, fmt.Errorf("cannot convert %v (%T) to []string", value, value)
}

func (m *ConfigMapper) mapToPorts(value interface{}) (interface{}, error) {
	var ports []core.Port

	switch v := value.(type) {
	case []interface{}:
		for _, portRaw := range v {
			if portMap, ok := portRaw.(map[string]interface{}); ok {
				port := core.Port{}
				if guest, exists := portMap["guest"]; exists {
					if guestInt, err := m.mapToInt(guest); err == nil {
						port.Guest = guestInt.(int)
					} else {
						return nil, fmt.Errorf("invalid guest port: %v", guest)
					}
				}
				if host, exists := portMap["host"]; exists {
					if hostInt, err := m.mapToInt(host); err == nil {
						port.Host = hostInt.(int)
					} else {
						return nil, fmt.Errorf("invalid host port: %v", host)
					}
				}
				ports = append(ports, port)
			}
		}
	case []map[string]int:
		for _, portMap := range v {
			port := core.Port{
				Guest: portMap["guest"],
				Host:  portMap["host"],
			}
			ports = append(ports, port)
		}
	default:
		return nil, fmt.Errorf("cannot convert %v (%T) to []core.Port", value, value)
	}

	return ports, nil
}

// Global mapper instance
var GlobalConfigMapper = NewConfigMapper()
