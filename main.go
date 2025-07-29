package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

// Config structure for our application
type Config struct {
	Port        string `mapstructure:"port"`
	OpenAIKey   string `mapstructure:"openai_key"`
	MinIOURL    string `mapstructure:"minio_url"`
	MinIOKey    string `mapstructure:"minio_key"`
	MinIOSecret string `mapstructure:"minio_secret"`
}

// API Input/Output structures
type ChatRequest struct {
	Message string `json:"message" doc:"Message to send to OpenAI"`
}

type ChatResponse struct {
	Reply string `json:"reply" doc:"Response from OpenAI"`
}

type FileUploadRequest struct {
	BucketName string `json:"bucket_name" doc:"MinIO bucket name"`
	FileName   string `json:"file_name" doc:"File name to create"`
	Content    string `json:"content" doc:"File content"`
}

type FileUploadResponse struct {
	Success bool   `json:"success" doc:"Upload success status"`
	Message string `json:"message" doc:"Upload result message"`
}

var (
	config       Config
	openaiClient *openai.Client
	minioClient  *minio.Client
)

func initConfig() {
	// Initialize Viper for configuration management
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	// Set defaults
	viper.SetDefault("port", "8080")
	viper.SetDefault("minio_url", "localhost:9000")

	// Enable environment variable binding
	viper.SetEnvPrefix("APP")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Try to read config file (optional)
	if err := viper.ReadInConfig(); err != nil {
		log.Printf("Config file not found, using defaults and environment variables: %v", err)
	}

	// Unmarshal config
	if err := viper.Unmarshal(&config); err != nil {
		log.Fatalf("Failed to unmarshal config: %v", err)
	}

	log.Printf("Configuration loaded: Port=%s, MinIO URL=%s", config.Port, config.MinIOURL)
}

func initClients() {
	// Initialize OpenAI client
	if config.OpenAIKey != "" {
		openaiClient = openai.NewClient(config.OpenAIKey)
		log.Println("OpenAI client initialized")
	} else {
		log.Println("OpenAI API key not provided, chat functionality will be disabled")
	}

	// Initialize MinIO client
	if config.MinIOKey != "" && config.MinIOSecret != "" {
		var err error
		minioClient, err = minio.New(config.MinIOURL, &minio.Options{
			Creds:  credentials.NewStaticV4(config.MinIOKey, config.MinIOSecret, ""),
			Secure: false, // Set to true for HTTPS
		})
		if err != nil {
			log.Printf("Failed to initialize MinIO client: %v", err)
		} else {
			log.Println("MinIO client initialized")
		}
	} else {
		log.Println("MinIO credentials not provided, file upload functionality will be disabled")
	}
}

func main() {
	// Initialize configuration with Viper
	initConfig()

	// Initialize external clients
	initClients()

	// Create Chi router
	router := chi.NewMux()

	// Create Huma API
	api := humachi.New(router, huma.DefaultConfig("Test Renovate API", "1.0.0"))

	// Register API endpoints
	registerChatEndpoint(api)
	registerFileUploadEndpoint(api)
	registerHealthEndpoint(api)

	// Start server
	addr := fmt.Sprintf(":%s", config.Port)
	log.Printf("Starting server on %s", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}

func registerChatEndpoint(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "chat",
		Method:      http.MethodPost,
		Path:        "/chat",
		Summary:     "Send a message to OpenAI",
		Description: "Send a message to OpenAI and get a response using the configured API key",
	}, func(ctx context.Context, input *struct {
		Body ChatRequest
	}) (*struct {
		Body ChatResponse
	}, error) {
		if openaiClient == nil {
			return nil, huma.Error400BadRequest("OpenAI client not configured")
		}

		resp, err := openaiClient.CreateChatCompletion(
			ctx,
			openai.ChatCompletionRequest{
				Model: openai.GPT3Dot5Turbo,
				Messages: []openai.ChatCompletionMessage{
					{
						Role:    openai.ChatMessageRoleUser,
						Content: input.Body.Message,
					},
				},
			},
		)
		if err != nil {
			return nil, huma.Error500InternalServerError("Failed to get OpenAI response", err)
		}

		reply := "No response"
		if len(resp.Choices) > 0 {
			reply = resp.Choices[0].Message.Content
		}

		return &struct {
			Body ChatResponse
		}{
			Body: ChatResponse{Reply: reply},
		}, nil
	})
}

func registerFileUploadEndpoint(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "upload-file",
		Method:      http.MethodPost,
		Path:        "/upload",
		Summary:     "Upload a file to MinIO",
		Description: "Upload a text file to MinIO storage",
	}, func(ctx context.Context, input *struct {
		Body FileUploadRequest
	}) (*struct {
		Body FileUploadResponse
	}, error) {
		if minioClient == nil {
			return &struct {
				Body FileUploadResponse
			}{
				Body: FileUploadResponse{
					Success: false,
					Message: "MinIO client not configured",
				},
			}, nil
		}

		// Create bucket if it doesn't exist
		exists, err := minioClient.BucketExists(ctx, input.Body.BucketName)
		if err != nil {
			return &struct {
				Body FileUploadResponse
			}{
				Body: FileUploadResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to check bucket existence: %v", err),
				},
			}, nil
		}

		if !exists {
			err = minioClient.MakeBucket(ctx, input.Body.BucketName, minio.MakeBucketOptions{})
			if err != nil {
				return &struct {
					Body FileUploadResponse
				}{
					Body: FileUploadResponse{
						Success: false,
						Message: fmt.Sprintf("Failed to create bucket: %v", err),
					},
				}, nil
			}
		}

		// Upload file
		reader := strings.NewReader(input.Body.Content)
		_, err = minioClient.PutObject(ctx, input.Body.BucketName, input.Body.FileName, reader, int64(len(input.Body.Content)), minio.PutObjectOptions{
			ContentType: "text/plain",
		})
		if err != nil {
			return &struct {
				Body FileUploadResponse
			}{
				Body: FileUploadResponse{
					Success: false,
					Message: fmt.Sprintf("Failed to upload file: %v", err),
				},
			}, nil
		}

		return &struct {
			Body FileUploadResponse
		}{
			Body: FileUploadResponse{
				Success: true,
				Message: fmt.Sprintf("File %s uploaded successfully to bucket %s", input.Body.FileName, input.Body.BucketName),
			},
		}, nil
	})
}

func registerHealthEndpoint(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "health",
		Method:      http.MethodGet,
		Path:        "/health",
		Summary:     "Health check endpoint",
		Description: "Check the health status of the application and its dependencies",
	}, func(ctx context.Context, input *struct{}) (*struct {
		Body map[string]interface{}
	}, error) {
		status := map[string]interface{}{
			"status": "healthy",
			"services": map[string]bool{
				"openai": openaiClient != nil,
				"minio":  minioClient != nil,
			},
			"config": map[string]string{
				"port":      config.Port,
				"minio_url": config.MinIOURL,
			},
		}

		return &struct {
			Body map[string]interface{}
		}{
			Body: status,
		}, nil
	})
}
