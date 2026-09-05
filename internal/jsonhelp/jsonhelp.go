package jsonhelp

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"
)

func RespondWithError(w http.ResponseWriter, code int, message string, err error) {
	if err != nil {
		log.Printf("HTTP %d Error: %v", code, err)
	}
	RespondWithJSON(w, code, map[string]string{"error": message})
}

func RespondWithJSON(w http.ResponseWriter, code int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	if payload != nil {
		err := json.NewEncoder(w).Encode(payload)
		if err != nil {
			log.Printf("Failed to encode JSON response: %v", err)
		}
	}
}

// strips markdown code block fences and leading non-JSON text
func CleanJSONOutput(raw string) string {
	cleaned := strings.TrimSpace(raw)

	// Strip markdown fences
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned, "\n"); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
	}
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	// Trim leading non-JSON text up to the first '{'
	if startIdx := strings.Index(cleaned, "{"); startIdx != -1 {
		cleaned = cleaned[startIdx:]
	}

	// Trim trailing non-JSON text after the last '}'
	if endIdx := strings.LastIndex(cleaned, "}"); endIdx != -1 && endIdx > strings.Index(cleaned, "{") {
		cleaned = cleaned[:endIdx+1]
	}

	return cleaned
}
