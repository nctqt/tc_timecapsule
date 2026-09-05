package youtube

// create a new http client
// extract video id
// extract video metadata

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
)

// the goal
type VideoMetadata struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	ChannelName string    `json:"channel_name"`
	Description string    `json:"description"`
	PublishedAt time.Time `json:"published_at"`
}

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
		Timeout: 10 * time.Second,
	}

	// our struct to hold the client plus the key
	c := &Client{
		apiKey:     apiKey,
		httpClient: lowLevelClient,
	}

	return c, nil
}

// extract the 11-character id from various formats
func ExtractVideoID(raw string) (string, error) {
	input := strings.TrimSpace(raw)
	if input == "" {
		return "", errors.New("invalid url")
	}

	// if a video id was passed in
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]{11}$`, input)
	if matched {
		return input, nil
	}

	// parse the url
	parsedURL, err := url.Parse(input)
	if err != nil {
		return "", errors.New("invalid url")
	}

	// standard urls
	// format is something like: https://www.youtube.com/watch?v=VIDEO_ID
	if strings.Contains(parsedURL.Host, "youtube.com") {
		videoID := parsedURL.Query().Get("v")
		if len(videoID) == 11 {
			return videoID, nil
		}
		// short links like youtube.com/embed/VIDEO_ID or youtube.com/v/VIDEO_ID
		pathSegments := strings.Split(strings.Trim(parsedURL.Path, "/"), "/")
		for i, seg := range pathSegments {
			if (seg == "embed" || seg == "v" || seg == "shorts") && i+1 < len(pathSegments) {
				// get string after confirmed segment
				if len(pathSegments[i+1]) == 11 {
					return pathSegments[i+1], nil
				}
			}
		}
	}

	// shortened URLs (youtu.be/VIDEO_ID)
	if strings.Contains(parsedURL.Host, "youtu.be") {
		path := strings.Trim(parsedURL.Path, "/")
		if len(path) == 11 {
			return path, nil
		}
	}

	return "", errors.New("invalid url")
}

func (c *Client) FetchVideoMetaData(ctx context.Context, inputURL string) (*VideoMetadata, error) {
	// get 11-char id
	videoID, err := ExtractVideoID(inputURL)
	if err != nil {
		return nil, err
	}

	endpoint := fmt.Sprintf(
		"https://www.googleapis.com/youtube/v3/videos?part=snippet&id=%s&key=%s",
		url.QueryEscape(videoID),
		url.QueryEscape(c.apiKey),
	)

	// build the request
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create youtube request: %w", err)
	}

	// send the request
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("youtube api network error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("youtube api returned non-200 status code: %d", resp.StatusCode)
	}

	// yt data api v3 response structure
	var ytResp struct {
		Items []struct {
			ID      string `json:"id"`
			Snippet struct {
				Title        string    `json:"title"`
				ChannelTitle string    `json:"channelTitle"`
				Description  string    `json:"description"`
				PublishedAt  time.Time `json:"publishedAt"`
			} `json:"snippet"`
		} `json:"items"`
	}

	// decoding the stream, does automatic unmarshal of json
	err = json.NewDecoder(resp.Body).Decode(&ytResp)
	if err != nil {
		return nil, fmt.Errorf("failed to decode youtube api response: %w", err)
	}

	if len(ytResp.Items) == 0 {
		return nil, errors.New("video not found")
	}

	item := ytResp.Items[0]
	meta := &VideoMetadata{
		ID:          item.ID,
		Title:       item.Snippet.Title,
		ChannelName: item.Snippet.ChannelTitle,
		Description: item.Snippet.Description,
		PublishedAt: item.Snippet.PublishedAt,
	}
	return meta, nil
}
