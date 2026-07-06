// Package storage uploads files (e.g. onboarding logos) to a Supabase Storage
// bucket via its plain HTTP API (no SDK needed).
package storage

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	baseURL    string
	serviceKey string
	bucket     string
}

func New(baseURL, serviceKey, bucket string) *Client {
	return &Client{baseURL: baseURL, serviceKey: serviceKey, bucket: bucket}
}

// Upload PUTs data at path within the bucket and returns its public URL.
// Caller picks path (e.g. orgId/itemId/uuid.ext) to avoid collisions.
func (c *Client) Upload(path, contentType string, data []byte) (string, error) {
	url := fmt.Sprintf("%s/storage/v1/object/%s/%s", c.baseURL, c.bucket, path)
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.serviceKey)
	req.Header.Set("Content-Type", contentType)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("supabase upload failed: %s: %s", resp.Status, body)
	}
	return fmt.Sprintf("%s/storage/v1/object/public/%s/%s", c.baseURL, c.bucket, path), nil
}
