package transcript

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	ErrNoTranscriptFound = errors.New("no transcript available for this video")

	// Regex matching WebVTT timestamp lines (e.g. 00:01:20.500 --> 00:01:23.000)
	vttTimestampRegex = regexp.MustCompile(`(?m)^\d{2}:\d{2}(:\d{2})?\.\d{3}\s+-->\s+\d{2}:\d{2}(:\d{2})?\.\d{3}.*$\n?`)
	// Regex matching WebVTT header tags, positioning rules, or XML tags like <c> text </c>
	vttTagRegex = regexp.MustCompile(`<[^>]*>`)
)

type Client struct {
	httpClient *http.Client
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// FetchTranscript attempts to retrieve and clean captions for a given YouTube video ID.
func (c *Client) FetchTranscript(ctx context.Context, videoID string) (string, error) {
	if strings.TrimSpace(videoID) == "" {
		return "", errors.New("videoID cannot be empty")
	}

	pageURL := fmt.Sprintf("https://www.youtube.com/watch?v=%s", videoID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, pageURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to fetch video page: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("youtube returned status %d", resp.StatusCode)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	captionURL := extractCaptionURL(string(bodyBytes))
	if captionURL == "" {
		return "", ErrNoTranscriptFound
	}

	if !strings.Contains(captionURL, "fmt=") {
		captionURL += "&fmt=vtt"
	}

	vttReq, err := http.NewRequestWithContext(ctx, http.MethodGet, captionURL, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create caption request: %w", err)
	}

	vttResp, err := c.httpClient.Do(vttReq)
	if err != nil {
		return "", fmt.Errorf("failed to fetch captions: %w", err)
	}
	defer vttResp.Body.Close()

	if vttResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("caption endpoint returned status %d", vttResp.StatusCode)
	}

	vttBytes, err := io.ReadAll(vttResp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read caption body: %w", err)
	}

	cleanText := CleanVTT(string(vttBytes))
	if strings.TrimSpace(cleanText) == "" {
		return "", ErrNoTranscriptFound
	}

	return cleanText, nil
}

func extractCaptionURL(html string) string {
	idx := strings.Index(html, `"captionTracks":`)
	if idx == -1 {
		return ""
	}

	sub := html[idx:]
	urlIdx := strings.Index(sub, `"baseUrl":"`)
	if urlIdx == -1 {
		return ""
	}

	start := urlIdx + len(`"baseUrl":"`)
	end := strings.Index(sub[start:], `"`)
	if end == -1 {
		return ""
	}

	rawURL := sub[start : start+end]
	return strings.ReplaceAll(rawURL, `\u0026`, "&")
}

// CleanVTT strips WebVTT headers, timecodes, XML tags, numeric cue IDs, and deduplicates repeating lines.
func CleanVTT(vtt string) string {
	lines := strings.Split(vtt, "\n")
	var filteredLines []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" ||
			strings.HasPrefix(trimmed, "WEBVTT") ||
			strings.HasPrefix(trimmed, "Kind:") ||
			strings.HasPrefix(trimmed, "Language:") ||
			strings.HasPrefix(trimmed, "NOTE") {
			continue
		}
		filteredLines = append(filteredLines, line)
	}

	cleaned := strings.Join(filteredLines, "\n")

	// Strip timecodes (e.g. 00:00:01.000 --> 00:00:04.000)
	cleaned = vttTimestampRegex.ReplaceAllString(cleaned, "")

	// Strip XML inline tags (e.g. <c>text</c> or <00:00:01.500>)
	cleaned = vttTagRegex.ReplaceAllString(cleaned, "")

	rawLines := strings.Split(cleaned, "\n")
	var finalLines []string
	var lastLine string

	for _, line := range rawLines {
		t := strings.TrimSpace(line)

		// Skip empty lines or pure integer sequence markers
		if t == "" || isInteger(t) || t == lastLine {
			continue
		}

		finalLines = append(finalLines, t)
		lastLine = t
	}

	fullTranscript := strings.Join(finalLines, " ")

	// Truncate to maximum ~18,000 characters (~4,000 tokens) for LLM context safety
	const maxChars = 18000
	if len(fullTranscript) > maxChars {
		fullTranscript = fullTranscript[:maxChars] + "..."
	}

	return fullTranscript
}

func isInteger(s string) bool {
	_, err := strconv.Atoi(s)
	return err == nil
}
