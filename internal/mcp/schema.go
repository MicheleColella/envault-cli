package mcp

import "encoding/json"

// Property describes one field of a tool's input JSON Schema.
type Property struct {
	Type        string // "string", "array", ...
	Items       string // element type, only used when Type == "array"
	Description string
}

// ObjectSchema builds a JSON Schema object (type: "object") from the given
// properties and required field names. Passing nil properties yields a
// schema for a tool that takes no arguments.
func ObjectSchema(properties map[string]Property, required []string) json.RawMessage {
	props := make(map[string]interface{}, len(properties))
	for name, p := range properties {
		prop := map[string]interface{}{"type": p.Type, "description": p.Description}
		if p.Type == "array" && p.Items != "" {
			prop["items"] = map[string]interface{}{"type": p.Items}
		}
		props[name] = prop
	}

	s := map[string]interface{}{
		"type":       "object",
		"properties": props,
	}
	if len(required) > 0 {
		s["required"] = required
	}

	b, _ := json.Marshal(s)
	return b
}
