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

// CleanJSONOutput strips markdown code block fences and isolates raw JSON objects.
func CleanJSONOutput(raw string) string {
	cleaned := strings.TrimSpace(raw)

	// Strip markdown code fences
	if strings.HasPrefix(cleaned, "```") {
		if idx := strings.Index(cleaned, "\n"); idx != -1 {
			cleaned = cleaned[idx+1:]
		}
	}
	cleaned = strings.TrimSuffix(cleaned, "```")
	cleaned = strings.TrimSpace(cleaned)

	startIdx := strings.Index(cleaned, "{")
	if startIdx == -1 {
		return cleaned
	}

	endIdx := strings.LastIndex(cleaned, "}")
	if endIdx == -1 || endIdx < startIdx {
		return cleaned[startIdx:]
	}

	extracted := strings.TrimSpace(cleaned[startIdx : endIdx+1])

	// Verify if extracted string is valid JSON
	var js json.RawMessage
	if json.Unmarshal([]byte(extracted), &js) != nil {
		// If unmarshaling fails due to truncation within the object, return original raw output
		return cleaned
	}

	return extracted
}
