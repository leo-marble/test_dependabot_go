# test_renovate_go

A simple Go application demonstrating the usage of VIPER, HUMA, OpenAI client, and MinIO libraries. This application is designed for testing dependency management and upgrades.

## Features

This application includes:

- **Viper**: Configuration management with support for config files and environment variables
- **Huma v2**: Modern HTTP API framework with automatic OpenAPI documentation
- **OpenAI Client**: Integration with OpenAI's API for chat completions
- **MinIO**: Object storage client for file upload functionality

## Dependencies

The following packages are used in this project:

- `github.com/spf13/viper` - Configuration management
- `github.com/danielgtaylor/huma/v2` - HTTP API framework
- `github.com/sashabaranov/go-openai` - OpenAI client
- `github.com/minio/minio-go/v7` - MinIO client

## Configuration

The application can be configured using:

1. **Configuration file** (`config.yaml`):
   ```yaml
   port: "8080"
   openai_key: "your-openai-api-key-here"
   minio_url: "localhost:9000"
   minio_key: "your-minio-access-key"
   minio_secret: "your-minio-secret-key"
   ```

2. **Environment variables** (with `APP_` prefix):
   ```bash
   export APP_PORT=8080
   export APP_OPENAI_KEY=your-openai-api-key-here
   export APP_MINIO_URL=localhost:9000
   export APP_MINIO_KEY=your-minio-access-key
   export APP_MINIO_SECRET=your-minio-secret-key
   ```

## API Endpoints

The application exposes the following endpoints:

### GET /health
Health check endpoint that returns the status of all services.

### POST /chat
Send a message to OpenAI and receive a response.

**Request body:**
```json
{
  "message": "Hello, how are you?"
}
```

**Response:**
```json
{
  "reply": "I'm doing well, thank you for asking!"
}
```

### POST /upload
Upload a text file to MinIO storage.

**Request body:**
```json
{
  "bucket_name": "my-bucket",
  "file_name": "example.txt",
  "content": "This is the file content"
}
```

**Response:**
```json
{
  "success": true,
  "message": "File example.txt uploaded successfully to bucket my-bucket"
}
```

## Running the Application

1. **Install dependencies:**
   ```bash
   go mod tidy
   ```

2. **Build the application:**
   ```bash
   go build -o test-app .
   ```

3. **Configure the application** (optional, but recommended for full functionality):
   - Copy `.env.example` to `.env` and set your actual API keys and credentials
   - Or create a `config.yaml` file with your configuration

4. **Run the application:**
   ```bash
   ./test-app
   ```

The server will start on port 8080 by default. You can access the API documentation at `http://localhost:8080/docs`.

## Testing

You can test the endpoints using curl:

```bash
# Health check
curl http://localhost:8080/health

# Chat with OpenAI (requires API key)
curl -X POST http://localhost:8080/chat \
  -H "Content-Type: application/json" \
  -d '{"message": "Hello!"}'

# Upload file to MinIO (requires MinIO credentials)
curl -X POST http://localhost:8080/upload \
  -H "Content-Type: application/json" \
  -d '{"bucket_name": "test", "file_name": "hello.txt", "content": "Hello World!"}'
```

## Notes

- The application will start even if OpenAI or MinIO credentials are not provided, but those specific features will be disabled
- Environment variables take precedence over configuration file values
- The application uses Huma v2 which automatically generates OpenAPI documentation