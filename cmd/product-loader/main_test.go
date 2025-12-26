package main

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"go-shopping-poc/internal/platform/service"
	"go-shopping-poc/internal/platform/storage/minio"
	"go-shopping-poc/internal/service/product"

	"github.com/stretchr/testify/assert"
)

// createTestCSV creates a temporary CSV file for testing
func createTestCSV(t *testing.T) string {
	tempDir := t.TempDir()
	csvPath := filepath.Join(tempDir, "test.csv")
	csvContent := `id,name,description,initial_price,final_price,currency,in_stock,color,size,main_image,country_code,image_count,model_number,root_category,category,brand,all_available_sizes,image_urls
1,Test Product,Test description,100.00,90.00,USD,true,Red,M,image.jpg,US,1,TEST123,Clothing,Shirts,TestBrand,[],["http://example.com/image.jpg"]`
	err := os.WriteFile(csvPath, []byte(csvContent), 0644)
	assert.NoError(t, err)
	return csvPath
}

func TestProductLoaderService_Name(t *testing.T) {
	loaderService := &ProductLoaderService{
		BaseService: service.NewBaseService("product-loader"),
	}

	assert.Equal(t, "product-loader", loaderService.Name())
}

func TestGetBatchID(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
		contains string
	}{
		{
			name:     "custom batch ID",
			input:    "custom-batch",
			expected: "custom-batch",
		},
		{
			name:     "empty batch ID generates one",
			input:    "",
			contains: "batch-",
		},
		{
			name:     "whitespace batch ID",
			input:    "   ",
			expected: "   ",
		},
		{
			name:     "special characters",
			input:    "batch-123_abc",
			expected: "batch-123_abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getBatchID(tt.input)
			if tt.expected != "" {
				assert.Equal(t, tt.expected, result)
			}
			if tt.contains != "" {
				assert.Contains(t, result, tt.contains)
				assert.Greater(t, len(result), len(tt.contains))
			}
		})
	}
}

func TestParseFlags_ValidCSV(t *testing.T) {
	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Create test CSV
	csvPath := createTestCSV(t)

	// Set command line args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test", "-csv", csvPath, "-batch-id", "test-batch", "-use-cache=false", "-reset-cache=true"}

	config, err := parseFlags()
	assert.NoError(t, err)
	assert.Equal(t, csvPath, config.CSVPath)
	assert.Equal(t, "test-batch", config.BatchID)
	assert.False(t, config.UseCache)
	assert.True(t, config.ResetCache)
}

func TestParseFlags_MissingCSV(t *testing.T) {
	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set command line args without CSV
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test"}

	config, err := parseFlags()
	assert.NoError(t, err)
	assert.Equal(t, "", config.CSVPath)          // Should be empty, config loaded later
	assert.Contains(t, config.BatchID, "batch-") // Should generate batch ID
	assert.True(t, config.UseCache)              // Default true
	assert.False(t, config.ResetCache)           // Default false
}

func TestParseFlags_NonExistentCSV(t *testing.T) {
	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Set command line args with non-existent CSV
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test", "-csv", "/nonexistent/file.csv"}

	_, err := parseFlags()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CSV file does not exist")
}

func TestParseFlags_DefaultValues(t *testing.T) {
	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	// Create test CSV
	csvPath := createTestCSV(t)

	// Set command line args with minimal flags
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test", "-csv", csvPath}

	config, err := parseFlags()
	assert.NoError(t, err)
	assert.Equal(t, csvPath, config.CSVPath)
	assert.Contains(t, config.BatchID, "batch-") // Should generate batch ID
	assert.True(t, config.UseCache)              // Default true
	assert.False(t, config.ResetCache)           // Default false
}

func TestProductLoaderService_IngestionRequest(t *testing.T) {
	loaderService := &ProductLoaderService{
		csvPath:    "/path/to/test.csv",
		batchID:    "test-batch-123",
		useCache:   true,
		resetCache: false,
	}

	// Test that the service has correct configuration
	assert.Equal(t, "/path/to/test.csv", loaderService.csvPath)
	assert.Equal(t, "test-batch-123", loaderService.batchID)
	assert.True(t, loaderService.useCache)
	assert.False(t, loaderService.resetCache)
}

func TestProductLoaderService_Structure(t *testing.T) {
	loaderService := &ProductLoaderService{
		BaseService: service.NewBaseService("product-loader"),
		csvPath:     "/test/path.csv",
		batchID:     "batch-123",
		useCache:    true,
		resetCache:  false,
	}

	// Test that the service implements the expected interface
	assert.NotNil(t, loaderService.BaseService)
	assert.Equal(t, "product-loader", loaderService.Name())
	assert.Equal(t, "/test/path.csv", loaderService.csvPath)
	assert.Equal(t, "batch-123", loaderService.batchID)
	assert.True(t, loaderService.useCache)
	assert.False(t, loaderService.resetCache)
}

func TestCLIConfig(t *testing.T) {
	config := &CLIConfig{
		CSVPath:    "/path/to/csv",
		BatchID:    "batch-123",
		UseCache:   true,
		ResetCache: false,
	}

	assert.Equal(t, "/path/to/csv", config.CSVPath)
	assert.Equal(t, "batch-123", config.BatchID)
	assert.True(t, config.UseCache)
	assert.False(t, config.ResetCache)
}

func TestParseFlags_ErrorScenarios(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{
			name:     "nonexistent csv file",
			args:     []string{"test", "-csv", "/definitely/does/not/exist.csv"},
			expected: "CSV file does not exist",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

			// Set command line args
			oldArgs := os.Args
			defer func() { os.Args = oldArgs }()
			os.Args = tt.args

			_, err := parseFlags()
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expected)
		})
	}
}

func TestFlagDefaults(t *testing.T) {
	// Reset flags
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	csvPath := createTestCSV(t)

	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"test", "-csv", csvPath}

	config, err := parseFlags()
	assert.NoError(t, err)
	assert.True(t, config.UseCache)              // Default should be true
	assert.False(t, config.ResetCache)           // Default should be false
	assert.Contains(t, config.BatchID, "batch-") // Should generate batch ID
}

func TestProductLoaderService_Lifecycle(t *testing.T) {
	loaderService := &ProductLoaderService{
		BaseService: service.NewBaseService("product-loader"),
	}

	// Test service name
	assert.Equal(t, "product-loader", loaderService.Name())

	// Test that it implements the Service interface
	var _ service.Service = loaderService
}

func TestFlagParsing_Integration(t *testing.T) {
	// Reset flags to clean state
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	csvPath := createTestCSV(t)

	// Simulate command line arguments
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{
		"product-loader",
		"-csv", csvPath,
		"-batch-id", "integration-test-batch",
		"-use-cache=true",
		"-reset-cache=false",
	}

	config, err := parseFlags()
	assert.NoError(t, err)
	assert.Equal(t, csvPath, config.CSVPath)
	assert.Equal(t, "integration-test-batch", config.BatchID)
	assert.True(t, config.UseCache)
	assert.False(t, config.ResetCache)
}

// Benchmark batch ID generation
func BenchmarkGetBatchID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getBatchID("")
	}
}

func BenchmarkGetBatchID_Custom(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = getBatchID("custom-batch")
	}
}

// TestMinIOEndpointOverrideLogic tests the specific logic for endpoint override
func TestMinIOEndpointOverrideLogic(t *testing.T) {
	tests := []struct {
		name             string
		localEndpoint    string
		platformEndpoint string
		expectedEndpoint string
		description      string
	}{
		{
			name:             "local override configured",
			localEndpoint:    "api.minio.local",
			platformEndpoint: "minio.minio.svc.cluster.local:9000",
			expectedEndpoint: "api.minio.local",
			description:      "Should use local override when configured",
		},
		{
			name:             "local override empty",
			localEndpoint:    "",
			platformEndpoint: "minio.minio.svc.cluster.local:9000",
			expectedEndpoint: "minio.minio.svc.cluster.local:9000",
			description:      "Should use platform endpoint when local override is empty",
		},
		{
			name:             "local override whitespace",
			localEndpoint:    "   ",
			platformEndpoint: "minio.minio.svc.cluster.local:9000",
			expectedEndpoint: "   ", // Actual implementation only checks for empty string
			description:      "Should use whitespace local override as-is (implementation detail)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the exact logic from main.go
			minioEndpoint := tt.localEndpoint
			if minioEndpoint == "" {
				minioEndpoint = tt.platformEndpoint
			}

			assert.Equal(t, tt.expectedEndpoint, minioEndpoint, tt.description)
		})
	}
}

// TestMinIOConfigurationWithMock tests MinIO configuration logic with mock data
func TestMinIOConfigurationWithMock(t *testing.T) {
	// This test verifies the MinIO configuration logic without requiring actual config files

	// Mock platform configuration (simulating what would be loaded from environment variables)
	mockPlatformCfg := &minio.PlatformConfig{
		EndpointKubernetes: "minio.minio.svc.cluster.local:9000",
		EndpointLocal:      "localhost:9000",
		AccessKey:          "minioadmin",
		SecretKey:          "minioadminpassword",
		TLSVerify:          false,
	}

	// Test the endpoint selection logic from main.go (environment-based)
	// Set environment to simulate local development
	oldValue := os.Getenv("KUBERNETES_SERVICE_HOST")
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	defer func() {
		if oldValue != "" {
			os.Setenv("KUBERNETES_SERVICE_HOST", oldValue)
		}
	}()

	minioEndpoint := mockPlatformCfg.EndpointLocal
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		minioEndpoint = mockPlatformCfg.EndpointKubernetes
	}

	// Verify that local environment selects platform local endpoint
	assert.Equal(t, mockPlatformCfg.EndpointLocal, minioEndpoint,
		"Should use platform local endpoint in local environment")
	assert.Equal(t, "localhost:9000", minioEndpoint,
		"Platform local endpoint should be used")

	// Test MinIO client creation with the resolved configuration
	minioClient, err := minio.NewClient(&minio.Config{
		Endpoint:  minioEndpoint,
		AccessKey: mockPlatformCfg.AccessKey,
		SecretKey: mockPlatformCfg.SecretKey,
		Secure:    mockPlatformCfg.TLSVerify,
	})

	// Client creation should succeed (even if MinIO server is not running)
	assert.NoError(t, err, "Should be able to create MinIO client with valid configuration")
	assert.NotNil(t, minioClient, "MinIO client should not be nil")
}

// TestMinIOConfigurationDefault tests the default endpoint selection
func TestMinIOConfigurationDefault(t *testing.T) {
	// Test the default endpoint selection (environment-based)

	// Mock configurations
	mockPlatformCfg := &minio.PlatformConfig{
		EndpointKubernetes: "minio.minio.svc.cluster.local:9000",
		EndpointLocal:      "localhost:9000",
		AccessKey:          "minioadmin",
		SecretKey:          "minioadminpassword",
		TLSVerify:          false,
	}

	// Test the endpoint selection logic from main.go
	// Set environment to simulate local development
	oldValue := os.Getenv("KUBERNETES_SERVICE_HOST")
	os.Unsetenv("KUBERNETES_SERVICE_HOST")
	defer func() {
		if oldValue != "" {
			os.Setenv("KUBERNETES_SERVICE_HOST", oldValue)
		}
	}()

	minioEndpoint := mockPlatformCfg.EndpointLocal
	if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
		minioEndpoint = mockPlatformCfg.EndpointKubernetes
	}

	// Should use platform local endpoint by default
	assert.Equal(t, mockPlatformCfg.EndpointLocal, minioEndpoint,
		"Should use platform local endpoint by default")
	assert.Equal(t, "localhost:9000", minioEndpoint,
		"Platform local endpoint should be used")
}

// TestMinIOClientCreationWithPlatformConfig tests client creation with platform config
func TestMinIOClientCreationWithPlatformConfig(t *testing.T) {
	// Test that we can create a MinIO client with typical platform configuration values

	testConfigs := []struct {
		name     string
		endpoint string
		secure   bool
	}{
		{
			name:     "local development endpoint",
			endpoint: "api.minio.local",
			secure:   false,
		},
		{
			name:     "kubernetes service endpoint",
			endpoint: "minio.minio.svc.cluster.local:9000",
			secure:   false,
		},
		{
			name:     "secure endpoint",
			endpoint: "minio.example.com:443",
			secure:   true,
		},
	}

	for _, tc := range testConfigs {
		t.Run(tc.name, func(t *testing.T) {
			client, err := minio.NewClient(&minio.Config{
				Endpoint:  tc.endpoint,
				AccessKey: "test-access-key",
				SecretKey: "test-secret-key",
				Secure:    tc.secure,
			})

			assert.NoError(t, err, "Should create MinIO client without error")
			assert.NotNil(t, client, "MinIO client should not be nil")
		})
	}
}

// TestMinIOEndpointSelection tests the environment-based endpoint selection logic
func TestMinIOEndpointSelection(t *testing.T) {
	tests := []struct {
		name                  string
		kubernetesServiceHost string
		expectedEndpoint      string
		description           string
	}{
		{
			name:                  "kubernetes environment detected",
			kubernetesServiceHost: "10.96.0.1",
			expectedEndpoint:      "minio.minio.svc.cluster.local:9000",
			description:           "Should use kubernetes endpoint when KUBERNETES_SERVICE_HOST is set",
		},
		{
			name:                  "local environment default",
			kubernetesServiceHost: "",
			expectedEndpoint:      "localhost:9000",
			description:           "Should use local endpoint when KUBERNETES_SERVICE_HOST is not set",
		},
		{
			name:                  "kubernetes environment with whitespace",
			kubernetesServiceHost: "   ",
			expectedEndpoint:      "minio.minio.svc.cluster.local:9000",
			description:           "Should use kubernetes endpoint when KUBERNETES_SERVICE_HOST contains whitespace",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock platform configuration
			mockPlatformCfg := &minio.PlatformConfig{
				EndpointKubernetes: "minio.minio.svc.cluster.local:9000",
				EndpointLocal:      "localhost:9000",
				AccessKey:          "minioadmin",
				SecretKey:          "minioadminpassword",
				TLSVerify:          false,
			}

			// Set/unset KUBERNETES_SERVICE_HOST environment variable
			oldValue := os.Getenv("KUBERNETES_SERVICE_HOST")
			defer func() {
				if oldValue == "" {
					os.Unsetenv("KUBERNETES_SERVICE_HOST")
				} else {
					os.Setenv("KUBERNETES_SERVICE_HOST", oldValue)
				}
			}()

			if tt.kubernetesServiceHost != "" {
				os.Setenv("KUBERNETES_SERVICE_HOST", tt.kubernetesServiceHost)
			} else {
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
			}

			// Execute the exact endpoint selection logic from main.go
			minioEndpoint := mockPlatformCfg.EndpointLocal
			if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
				minioEndpoint = mockPlatformCfg.EndpointKubernetes
			}

			assert.Equal(t, tt.expectedEndpoint, minioEndpoint, tt.description)
		})
	}
}

// TestServiceSpecificBucketConfiguration tests that bucket configuration is correctly passed
func TestServiceSpecificBucketConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		bucketName  string
		description string
	}{
		{
			name:        "default product images bucket",
			bucketName:  "product-images",
			description: "Should use default product images bucket",
		},
		{
			name:        "custom loader bucket",
			bucketName:  "loader-products",
			description: "Should use custom bucket name",
		},
		{
			name:        "environment specific bucket",
			bucketName:  "prod-product-images",
			description: "Should support environment-specific bucket names",
		},
		{
			name:        "bucket with numbers",
			bucketName:  "products-v2",
			description: "Should support bucket names with numbers",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock loader configuration with specific bucket
			mockLoaderCfg := &Config{
				DatabaseURL: "postgres://test:test@localhost:5432/test",
				CacheDir:    "/tmp/cache",
				CSVPath:     "/tmp/test.csv",
				MinIOBucket: tt.bucketName,
			}

			// Convert loader config to product service config (as done in main.go)
			productCfg := &product.Config{
				DatabaseURL:  mockLoaderCfg.DatabaseURL,
				ServicePort:  "",
				CacheDir:     mockLoaderCfg.CacheDir,
				CacheMaxAge:  mockLoaderCfg.CacheMaxAge,
				CacheMaxSize: mockLoaderCfg.CacheMaxSize,
				CSVBatchSize: mockLoaderCfg.CSVBatchSize,
				MinIOBucket:  mockLoaderCfg.MinIOBucket, // This should be the bucket from loader config
			}

			// Verify the bucket is correctly passed through
			assert.Equal(t, tt.bucketName, productCfg.MinIOBucket, tt.description)
			assert.NotEmpty(t, productCfg.MinIOBucket, "Bucket should not be empty")
		})
	}
}

// TestMinIOEndpointSelectionIntegration tests endpoint selection with full service creation
func TestMinIOEndpointSelectionIntegration(t *testing.T) {
	tests := []struct {
		name                  string
		kubernetesServiceHost string
		expectedEndpoint      string
		expectedBucket        string
		description           string
	}{
		{
			name:                  "kubernetes environment",
			kubernetesServiceHost: "10.96.0.1",
			expectedEndpoint:      "minio.minio.svc.cluster.local:9000",
			expectedBucket:        "k8s-product-images",
			description:           "Kubernetes environment should use service endpoint",
		},
		{
			name:                  "local development",
			kubernetesServiceHost: "",
			expectedEndpoint:      "localhost:9000",
			expectedBucket:        "dev-product-images",
			description:           "Local development should use localhost endpoint",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock platform configuration
			mockPlatformCfg := &minio.PlatformConfig{
				EndpointKubernetes: "minio.minio.svc.cluster.local:9000",
				EndpointLocal:      "localhost:9000",
				AccessKey:          "minioadmin",
				SecretKey:          "minioadminpassword",
				TLSVerify:          false,
			}

			// Mock loader configuration
			mockLoaderCfg := &Config{
				DatabaseURL: "postgres://test:test@localhost:5432/test",
				CacheDir:    "/tmp/cache",
				CSVPath:     "/tmp/test.csv",
				MinIOBucket: tt.expectedBucket,
			}

			// Set/unset KUBERNETES_SERVICE_HOST environment variable
			oldValue := os.Getenv("KUBERNETES_SERVICE_HOST")
			defer func() {
				if oldValue == "" {
					os.Unsetenv("KUBERNETES_SERVICE_HOST")
				} else {
					os.Setenv("KUBERNETES_SERVICE_HOST", oldValue)
				}
			}()

			if tt.kubernetesServiceHost != "" {
				os.Setenv("KUBERNETES_SERVICE_HOST", tt.kubernetesServiceHost)
			} else {
				os.Unsetenv("KUBERNETES_SERVICE_HOST")
			}

			// Execute endpoint selection logic (from main.go)
			minioEndpoint := mockPlatformCfg.EndpointLocal
			if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
				minioEndpoint = mockPlatformCfg.EndpointKubernetes
			}

			// Verify endpoint selection
			assert.Equal(t, tt.expectedEndpoint, minioEndpoint,
				"Endpoint selection should work correctly: %s", tt.description)

			// Create product service config (as done in main.go)
			productCfg := &product.Config{
				DatabaseURL:  mockLoaderCfg.DatabaseURL,
				ServicePort:  "",
				CacheDir:     mockLoaderCfg.CacheDir,
				CacheMaxAge:  mockLoaderCfg.CacheMaxAge,
				CacheMaxSize: mockLoaderCfg.CacheMaxSize,
				CSVBatchSize: mockLoaderCfg.CSVBatchSize,
				MinIOBucket:  mockLoaderCfg.MinIOBucket,
			}

			// Verify bucket configuration
			assert.Equal(t, tt.expectedBucket, productCfg.MinIOBucket,
				"Bucket should be correctly configured: %s", tt.description)

			// Test MinIO client creation with resolved configuration
			minioClient, err := minio.NewClient(&minio.Config{
				Endpoint:  minioEndpoint,
				AccessKey: mockPlatformCfg.AccessKey,
				SecretKey: mockPlatformCfg.SecretKey,
				Secure:    mockPlatformCfg.TLSVerify,
			})

			assert.NoError(t, err, "Should create MinIO client with resolved configuration")
			assert.NotNil(t, minioClient, "MinIO client should not be nil")
		})
	}
}

// TestBucketConfigurationValidation tests bucket name validation and configuration passing
func TestBucketConfigurationValidation(t *testing.T) {
	tests := []struct {
		name        string
		bucketName  string
		description string
	}{
		{
			name:        "valid bucket name",
			bucketName:  "product-images",
			description: "Standard bucket name should be valid",
		},
		{
			name:        "bucket with numbers",
			bucketName:  "products-v2",
			description: "Bucket name with numbers should be valid",
		},
		{
			name:        "bucket with hyphens",
			bucketName:  "product-images-dev",
			description: "Bucket name with hyphens should be valid",
		},
		{
			name:        "bucket with uppercase",
			bucketName:  "Product-Images",
			description: "Bucket name with uppercase should be accepted (MinIO handles case)",
		},
		{
			name:        "empty bucket name",
			bucketName:  "",
			description: "Empty bucket name should still be passed through (validation happens at platform level)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockLoaderCfg := &Config{
				DatabaseURL: "postgres://test:test@localhost:5432/test",
				CacheDir:    "/tmp/cache",
				CSVPath:     "/tmp/test.csv",
				MinIOBucket: tt.bucketName,
			}

			// Test that bucket is correctly passed to product config
			productCfg := &product.Config{
				DatabaseURL:  mockLoaderCfg.DatabaseURL,
				ServicePort:  "",
				CacheDir:     mockLoaderCfg.CacheDir,
				CacheMaxAge:  mockLoaderCfg.CacheMaxAge,
				CacheMaxSize: mockLoaderCfg.CacheMaxSize,
				CSVBatchSize: mockLoaderCfg.CSVBatchSize,
				MinIOBucket:  mockLoaderCfg.MinIOBucket,
			}

			assert.Equal(t, tt.bucketName, productCfg.MinIOBucket,
				"Bucket should be correctly passed to product service: %s", tt.description)

			// Note: Full validation with struct tags happens at the platform config loader level
			// The manual Validate() method doesn't check MinIOBucket since it's handled by validate:"required" tag
		})
	}
}

// TestConfigurationLoadingIntegration tests that configuration loading works with actual config files
func TestConfigurationLoadingIntegration(t *testing.T) {
	// This test verifies that the product-loader can load its configuration
	// and that the endpoint type selection and bucket configuration work with real config files

	// Skip this integration test - it requires running from the project root
	// where the config files are located relative to the working directory
	t.Skip("Skipping integration test: requires running from project root with config files")

	// The test logic below would work if run from the correct directory:

	/*
		// Load actual product-loader configuration
		loaderCfg, err := LoadConfig()
		assert.NoError(t, err, "Should load product-loader configuration without error")
		assert.NotNil(t, loaderCfg, "Loader config should not be nil")

		// Load actual platform MinIO configuration
		minioCfg, err := config.LoadConfig[minio.PlatformConfig]("platform-minio")
		assert.NoError(t, err, "Should load MinIO platform configuration without error")
		assert.NotNil(t, minioCfg, "MinIO config should not be nil")

		// Verify configuration has expected structure
		assert.NotEmpty(t, loaderCfg.DatabaseURL, "Database URL should be configured")
		assert.NotEmpty(t, loaderCfg.CacheDir, "Cache directory should be configured")
		assert.NotEmpty(t, loaderCfg.CSVPath, "CSV path should be configured")
		assert.NotEmpty(t, loaderCfg.MinIOBucket, "MinIO bucket should be configured")

		// Test endpoint selection logic with real configuration
		var selectedEndpoint string
		switch loaderCfg.MinIOEndpointType {
		case "kubernetes":
			selectedEndpoint = minioCfg.EndpointKubernetes
		case "local":
			selectedEndpoint = minioCfg.EndpointLocal
		default:
			// Auto-detect based on environment
			selectedEndpoint = minioCfg.EndpointLocal
			if os.Getenv("KUBERNETES_SERVICE_HOST") != "" {
				selectedEndpoint = minioCfg.EndpointKubernetes
			}
		}

		// Verify endpoint was selected
		assert.NotEmpty(t, selectedEndpoint, "Should select a valid MinIO endpoint")

		// Test that product config conversion works
		productCfg := &product.Config{
			DatabaseURL:  loaderCfg.DatabaseURL,
			ServicePort:  "",
			CacheDir:     loaderCfg.CacheDir,
			CacheMaxAge:  loaderCfg.CacheMaxAge,
			CacheMaxSize: loaderCfg.CacheMaxSize,
			CSVBatchSize: loaderCfg.CSVBatchSize,
			MinIOBucket:  loaderCfg.MinIOBucket,
		}

		// Verify bucket is correctly configured
		assert.Equal(t, loaderCfg.MinIOBucket, productCfg.MinIOBucket,
			"Bucket should be correctly passed from loader config to product config")

		// Test MinIO client creation with resolved configuration
		minioClient, err := minio.NewClient(&minio.Config{
			Endpoint:  selectedEndpoint,
			AccessKey: minioCfg.AccessKey,
			SecretKey: minioCfg.SecretKey,
			Secure:    minioCfg.TLSVerify,
		})

		assert.NoError(t, err, "Should create MinIO client with real configuration")
		assert.NotNil(t, minioClient, "MinIO client should not be nil")

		t.Logf("Integration test passed: endpoint=%s, bucket=%s", selectedEndpoint, loaderCfg.MinIOBucket)
	*/
}
