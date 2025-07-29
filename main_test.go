package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/spf13/viper"
)

func TestInitConfig(t *testing.T) {
	// Save original values
	originalPort := viper.GetString("port")
	originalMinIOURL := viper.GetString("minio_url")

	// Test default configuration
	viper.Reset()
	initConfig()

	if config.Port != "8080" {
		t.Errorf("Expected default port to be 8080, got %s", config.Port)
	}

	if config.MinIOURL != "localhost:9000" {
		t.Errorf("Expected default MinIO URL to be localhost:9000, got %s", config.MinIOURL)
	}

	// Test environment variable override
	os.Setenv("APP_PORT", "9090")
	os.Setenv("APP_MINIO_URL", "test.example.com:9000")

	viper.Reset()
	initConfig()

	if config.Port != "9090" {
		t.Errorf("Expected port from env var to be 9090, got %s", config.Port)
	}

	if config.MinIOURL != "test.example.com:9000" {
		t.Errorf("Expected MinIO URL from env var to be test.example.com:9000, got %s", config.MinIOURL)
	}

	// Cleanup
	os.Unsetenv("APP_PORT")
	os.Unsetenv("APP_MINIO_URL")
	viper.Set("port", originalPort)
	viper.Set("minio_url", originalMinIOURL)
}

func TestHealthEndpoint(t *testing.T) {
	// Initialize config for testing
	viper.Reset()
	initConfig()

	// Create test router and API
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("Test API", "1.0.0"))

	// Register health endpoint
	registerHealthEndpoint(api)

	// Create test request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Check response
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Parse response
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Verify response structure
	if status, ok := response["status"]; !ok || status != "healthy" {
		t.Errorf("Expected status to be 'healthy', got %v", status)
	}

	if services, ok := response["services"].(map[string]interface{}); !ok {
		t.Error("Expected services field to be present")
	} else {
		if _, ok := services["openai"]; !ok {
			t.Error("Expected openai service status to be present")
		}
		if _, ok := services["minio"]; !ok {
			t.Error("Expected minio service status to be present")
		}
	}

	if configData, ok := response["config"].(map[string]interface{}); !ok {
		t.Error("Expected config field to be present")
	} else {
		if port, ok := configData["port"]; !ok || port == "" {
			t.Error("Expected port in config to be present and non-empty")
		}
		if minioURL, ok := configData["minio_url"]; !ok || minioURL == "" {
			t.Error("Expected minio_url in config to be present and non-empty")
		}
	}
}

func TestChatEndpointWithoutClient(t *testing.T) {
	// Initialize config for testing
	viper.Reset()
	initConfig()

	// Ensure OpenAI client is not initialized
	openaiClient = nil

	// Create test router and API
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("Test API", "1.0.0"))

	// Register chat endpoint
	registerChatEndpoint(api)

	// Create test request
	requestBody := ChatRequest{Message: "Hello, world!"}
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/chat", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Should return 400 Bad Request since OpenAI client is not configured
	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status code 400, got %d", w.Code)
	}
}

func TestFileUploadEndpointWithoutClient(t *testing.T) {
	// Initialize config for testing
	viper.Reset()
	initConfig()

	// Ensure MinIO client is not initialized
	minioClient = nil

	// Create test router and API
	router := chi.NewMux()
	api := humachi.New(router, huma.DefaultConfig("Test API", "1.0.0"))

	// Register upload endpoint
	registerFileUploadEndpoint(api)

	// Create test request
	requestBody := FileUploadRequest{
		BucketName: "test-bucket",
		FileName:   "test.txt",
		Content:    "Hello, world!",
	}
	jsonBody, _ := json.Marshal(requestBody)
	req := httptest.NewRequest("POST", "/upload", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Execute request
	router.ServeHTTP(w, req)

	// Should return 200 OK but with success: false since MinIO client is not configured
	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	// Parse response
	var response FileUploadResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response JSON: %v", err)
	}

	// Should indicate failure due to missing client
	if response.Success {
		t.Error("Expected success to be false when MinIO client is not configured")
	}

	if response.Message != "MinIO client not configured" {
		t.Errorf("Expected specific error message, got: %s", response.Message)
	}
}

func TestConfigStructValidation(t *testing.T) {
	testConfig := Config{
		Port:        "8080",
		OpenAIKey:   "test-key",
		MinIOURL:    "localhost:9000",
		MinIOKey:    "test-access-key",
		MinIOSecret: "test-secret-key",
	}

	if testConfig.Port != "8080" {
		t.Errorf("Expected port to be 8080, got %s", testConfig.Port)
	}

	if testConfig.OpenAIKey != "test-key" {
		t.Errorf("Expected OpenAI key to be test-key, got %s", testConfig.OpenAIKey)
	}

	if testConfig.MinIOURL != "localhost:9000" {
		t.Errorf("Expected MinIO URL to be localhost:9000, got %s", testConfig.MinIOURL)
	}
}

func TestChatRequestResponseStructs(t *testing.T) {
	// Test ChatRequest
	chatReq := ChatRequest{Message: "Hello"}
	if chatReq.Message != "Hello" {
		t.Errorf("Expected message to be 'Hello', got %s", chatReq.Message)
	}

	// Test ChatResponse
	chatResp := ChatResponse{Reply: "Hi there"}
	if chatResp.Reply != "Hi there" {
		t.Errorf("Expected reply to be 'Hi there', got %s", chatResp.Reply)
	}

	// Test FileUploadRequest
	uploadReq := FileUploadRequest{
		BucketName: "test-bucket",
		FileName:   "test.txt",
		Content:    "test content",
	}
	if uploadReq.BucketName != "test-bucket" {
		t.Errorf("Expected bucket name to be 'test-bucket', got %s", uploadReq.BucketName)
	}

	// Test FileUploadResponse
	uploadResp := FileUploadResponse{
		Success: true,
		Message: "Upload successful",
	}
	if !uploadResp.Success {
		t.Error("Expected success to be true")
	}
	if uploadResp.Message != "Upload successful" {
		t.Errorf("Expected message to be 'Upload successful', got %s", uploadResp.Message)
	}
}
