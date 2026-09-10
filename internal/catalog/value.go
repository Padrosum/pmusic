package catalog

import (
	"encoding/json"
	"strings"
)

func objectValue(raw json.RawMessage) map[string]any {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}
	var v any
	if err := json.Unmarshal(raw, &v); err != nil {
		return nil
	}
	obj, _ := v.(map[string]any)
	return obj
}

func stringField(obj map[string]any, keys ...string) string {
	if obj == nil {
		return ""
	}
	for _, key := range keys {
		if s, ok := obj[key].(string); ok && s != "" {
			return s
		}
	}
	return ""
}

func refName(v any) string {
	m, ok := v.(map[string]any)
	if !ok {
		return ""
	}
	s, _ := m["$ref"].(string)
	return strings.TrimSpace(s)
}

func entityTitle(obj map[string]any, fallback string) string {
	if s := stringField(obj, "baslik", "isim", "title", "name"); s != "" {
		return s
	}
	return fallback
}

func entityPath(obj map[string]any) string {
	return stringField(obj, "yol", "path")
}

func entityArtist(obj map[string]any) string {
	return stringField(obj, "sanatci", "artist")
}
