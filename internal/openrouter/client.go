package openrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nctqt/tc_timecapsule/internal/jsonhelp"
)

type Client struct {
	apiKey     string
	httpClient *http.Client
}

func NewClient(apiKey string) (*Client, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("api key required")
	}

	// standard library client w/ timeout
	lowLevelClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// our struct to hold the client plus the key
	c := &Client{
		apiKey:     apiKey,
		httpClient: lowLevelClient,
	}

	return c, nil
}

type AnalysisRequest struct {
	Title         string `json:"title"`
	Description   string `json:"description"`
	ChannelName   string `json:"channel_name"`
	RawTranscript string `json:"transcript"` // optional transcript text
}

type AnalysisResponse struct {
	Category           string   `json:"category"`             // e.g., "primary_source", "news", "commentary"
	Summary            string   `json:"summary"`              // 2-3 sentence neutral overview
	EstimatedEventDate string   `json:"estimated_event_date"` // YYYY-MM-DD string
	KeyEntities        []string `json:"key_entities"`         // names, locations mentioned
	SummarySource      string   `json:"summary_source"`       // "metadata" or "transcript"
}

func (c *Client) AnalyzeVideo(ctx context.Context, input AnalysisRequest) (*AnalysisResponse, error) {
	summarySource := "metadata"
	transcriptContext := "No transcript available. Rely on metadata."

	if strings.TrimSpace(input.RawTranscript) != "" {
		summarySource = "transcript"
		transcriptContext = fmt.Sprintf("\n%s", input.RawTranscript)
	}

	prompt := fmt.Sprintf(`
Analyze this true crime YouTube video and extract structured information.

Video Title: %s
Channel Name: %s
Description: %s
Transcript: %s

Do NOT include markdown formatting, code blocks, or preamble text.
Respond ONLY with a valid JSON object matching this schema:
{
  "category": "primary_source | news_report | commentary | legal_analysis",
  "summary": "Concise, factual 2-sentence summary of the specific event covered.",
  "estimated_event_date": "YYYY-MM-DD or empty string if unknown",
  "key_entities": ["names", "locations"]
}
`, input.Title, input.ChannelName, input.Description, transcriptContext)

	payload := map[string]any{
		"model": "openrouter/free", // free openRouter model
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
		"response_format": map[string]string{"type": "json_object"}, // forces clean JSON
	}

	// map -> json
	jsonBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("error marshalling json: %w", err)
	}

	// build request
	req, err := http.NewRequestWithContext(ctx, "POST", "https://openrouter.ai/api/v1/chat/completions", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("error creating openrouter request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("HTTP-Referer", "https://github.com/nctqt/tc_timecapsule") // optional: repo/site URL
	req.Header.Set("X-Title", "temporary name - true crime video timeline")   // optional: app name

	// send request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, fmt.Errorf("openrouter API returned status %d (failed to read error body: %w)", resp.StatusCode, err)
		}
		return nil, fmt.Errorf("openrouter API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// shape of openrouter response
	var apiResult struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	// decode into result
	err = json.NewDecoder(resp.Body).Decode(&apiResult)
	if err != nil || len(apiResult.Choices) == 0 {
		return nil, fmt.Errorf("failed to parse openrouter response: %w", err)
	}

	// sanitize
	rawContent := apiResult.Choices[0].Message.Content
	cleanedContent := jsonhelp.CleanJSONOutput(rawContent)

	// json -> struct
	var analysis AnalysisResponse
	err = json.Unmarshal([]byte(cleanedContent), &analysis)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal LLM JSON output: %w", err)
	}

	analysis.SummarySource = summarySource
	return &analysis, nil
}
