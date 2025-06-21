package resources

import (
	"errors"
	"sync"
)

var (
	registeredTypes = make(map[string]bool)
	mutex           sync.RWMutex
)

var (
	// ErrInvalidResourceType is returned when attempting to use an invalid resource type
	ErrInvalidResourceType = errors.New("invalid resource type")
	// ErrResourceTypeExists is returned when attempting to register a duplicate resource type
	ErrResourceTypeExists = errors.New("resource type already registered")
	// ErrUnknownResourceType is returned when attempting to use an unregistered resource type
	ErrUnknownResourceType = errors.New("unknown resource type")
	// ErrMissingRequiredField is returned when a required field is missing from a resource
	ErrMissingRequiredField = errors.New("missing required field")
)

// RegisterResourceType registers a new resource type
func RegisterResourceType(resourceType string) error {
	if resourceType == "" {
		return ErrInvalidResourceType
	}

	mutex.Lock()
	defer mutex.Unlock()

	if _, exists := registeredTypes[resourceType]; exists {
		return ErrResourceTypeExists
	}

	registeredTypes[resourceType] = true
	return nil
}

// IsResourceTypeRegistered checks if a resource type is registered
func IsResourceTypeRegistered(resourceType string) bool {
	mutex.RLock()
	defer mutex.RUnlock()

	return registeredTypes[resourceType]
}

// ValidateResource validates a resource against its type's schema
func ValidateResource(resourceType string, data map[string]interface{}) error {
	if !IsResourceTypeRegistered(resourceType) {
		return ErrUnknownResourceType
	}

	// Type-specific validation
	switch resourceType {
	case "vm":
		return validateVMResource(data)
	default:
		return ErrUnknownResourceType
	}
}

// validateVMResource validates a VM resource
func validateVMResource(data map[string]interface{}) error {
	if _, ok := data["name"]; !ok {
		return errors.New("missing required field 'name'")
	}

	// Validate config if present
	if config, ok := data["config"]; ok {
		if configMap, ok := config.(map[string]interface{}); ok {
			return validateVMConfig(configMap)
		}
	}

	return nil
}

// validateVMConfig validates a VM configuration
func validateVMConfig(config map[string]interface{}) error {
	// Validate ports if present
	if ports, ok := config["ports"]; ok {
		if portsSlice, ok := ports.([]interface{}); ok {
			for _, p := range portsSlice {
				if portMap, ok := p.(map[string]interface{}); ok {
					if err := validatePortMapping(portMap); err != nil {
						return err
					}
				}
			}
		}
	}

	return nil
}

// validatePortMapping validates a port mapping configuration
func validatePortMapping(port map[string]interface{}) error {
	if guest, ok := port["guest"]; ok {
		if _, ok := guest.(float64); !ok {
			return errors.New("invalid guest port number")
		}
	}

	if host, ok := port["host"]; ok {
		if _, ok := host.(float64); !ok {
			return errors.New("invalid host port number")
		}
	}

	return nil
}

// ConvertResource converts a resource's data into its appropriate type
func ConvertResource(resourceType string, data map[string]interface{}) (interface{}, error) {
	if !IsResourceTypeRegistered(resourceType) {
		return nil, ErrUnknownResourceType
	}

	// Validate the resource first
	if err := ValidateResource(resourceType, data); err != nil {
		return nil, err
	}

	// Type-specific conversion
	switch resourceType {
	case "vm":
		return convertVMResource(data)
	default:
		return nil, ErrUnknownResourceType
	}
}

// convertVMResource converts VM resource data into a VM configuration
func convertVMResource(data map[string]interface{}) (interface{}, error) {
	// Extract VM configuration
	config := make(map[string]interface{})

	if name, ok := data["name"].(string); ok {
		config["name"] = name
	}

	if rawConfig, ok := data["config"].(map[string]interface{}); ok {
		// Convert memory and CPU if present
		if memory, ok := rawConfig["memory"].(float64); ok {
			config["memory"] = int(memory)
		}
		if cpu, ok := rawConfig["cpu"].(float64); ok {
			config["cpu"] = int(cpu)
		}

		// Convert ports if present
		if ports, ok := rawConfig["ports"].([]interface{}); ok {
			portConfigs := make([]map[string]int, 0)
			for _, p := range ports {
				if portMap, ok := p.(map[string]interface{}); ok {
					portConfig := make(map[string]int)
					if guest, ok := portMap["guest"].(float64); ok {
						portConfig["guest"] = int(guest)
					}
					if host, ok := portMap["host"].(float64); ok {
						portConfig["host"] = int(host)
					}
					portConfigs = append(portConfigs, portConfig)
				}
			}
			config["ports"] = portConfigs
		}
	}

	return config, nil
}
