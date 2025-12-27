package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type transcriptionJob struct {
	Title *string `json:"title"`
}

type listResponse struct {
	Jobs []transcriptionJob `json:"jobs"`
}

// isAlreadyUploaded checks if a file with the given title is already on the server
func isAlreadyUploaded(config *Config, fileName string) (bool, error) {
	u := fmt.Sprintf("%s/api/v1/transcription/list?q=%s", config.ServerURL, url.QueryEscape(fileName))
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return false, err
	}

	req.Header.Set("Authorization", "Bearer "+config.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("failed to fetch transcription list: %s", resp.Status)
	}

	var list listResponse
	if err := json.NewDecoder(resp.Body).Decode(&list); err != nil {
		return false, err
	}

	for _, job := range list.Jobs {
		if job.Title != nil && *job.Title == fileName {
			return true, nil
		}
	}

	return false, nil
}

// UploadFile uploads a file to the Scriberr server
func UploadFile(filePath string) error {
	config := GetConfig()
	if config.ServerURL == "" {
		return fmt.Errorf("server URL not configured. Please run 'scriberr login' or 'scriberr install'")
	}
	if config.Token == "" {
		return fmt.Errorf("not logged in (token missing). Please run 'scriberr login'")
	}

	fileName := filepath.Base(filePath)

	// Check if already uploaded
	alreadyUploaded, err := isAlreadyUploaded(config, fileName)
	if err == nil && alreadyUploaded {
		fmt.Printf("File '%s' already uploaded, skipping.\n", fileName)
		return nil
	}

	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("audio", fileName)
	if err != nil {
		return fmt.Errorf("failed to create form file: %w", err)
	}
	if _, err := io.Copy(part, file); err != nil {
		return fmt.Errorf("failed to copy file content: %w", err)
	}

	// Add title as filename
	if err := writer.WriteField("title", fileName); err != nil {
		return fmt.Errorf("failed to write title field: %w", err)
	}

	err = writer.Close()
	if err != nil {
		return fmt.Errorf("failed to close writer: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/transcription/upload", config.ServerURL)
	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Authorization", "Bearer "+config.Token)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("upload failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
