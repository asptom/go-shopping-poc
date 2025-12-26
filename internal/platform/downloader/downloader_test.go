package downloader

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestNewHTTPDownloader(t *testing.T) {
	tempDir := t.TempDir()

	downloader, err := NewHTTPDownloader(tempDir)
	if err != nil {
		t.Fatalf("Failed to create downloader: %v", err)
	}

	if downloader == nil {
		t.Fatal("Downloader is nil")
	}

	// Test with options
	downloader2, err := NewHTTPDownloader(tempDir,
		WithTimeout(10*time.Second),
		WithUserAgent("test-agent"),
	)
	if err != nil {
		t.Fatalf("Failed to create downloader with options: %v", err)
	}

	if downloader2 == nil {
		t.Fatal("Downloader with options is nil")
	}
}

func TestGetCachePath(t *testing.T) {
	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	url := "https://example.com/image.jpg"
	cachePath := downloader.GetCachePath(url)

	if cachePath == "" {
		t.Error("Cache path is empty")
	}

	if !strings.HasPrefix(cachePath, tempDir) {
		t.Errorf("Cache path %s does not start with temp dir %s", cachePath, tempDir)
	}

	// Test that same URL produces same cache path
	cachePath2 := downloader.GetCachePath(url)
	if cachePath != cachePath2 {
		t.Errorf("Same URL produced different cache paths: %s vs %s", cachePath, cachePath2)
	}

	// Test different URLs produce different paths
	url2 := "https://example.com/image2.jpg"
	cachePath3 := downloader.GetCachePath(url2)
	if cachePath == cachePath3 {
		t.Error("Different URLs produced same cache path")
	}
}

func TestExtractExtension(t *testing.T) {
	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com/image.jpg", ".jpg"},
		{"https://example.com/image.png", ".png"},
		{"https://example.com/image.jpeg", ".jpeg"},
		{"https://example.com/image.gif", ".gif"},
		{"https://example.com/image.webp", ".webp"},
		{"https://example.com/image.JPG", ".jpg"}, // case insensitive
		{"https://example.com/image.jpg?param=value", ".jpg"},
		{"https://example.com/image.jpg#fragment", ".jpg"},
		{"https://example.com/image", ""},                          // no extension
		{"https://example.com/image.", ""},                         // empty extension
		{"https://example.com/image.very-long-extension-name", ""}, // too long
	}

	for _, test := range tests {
		result := downloader.(*HTTPDownloaderImpl).extractExtension(test.url)
		if result != test.expected {
			t.Errorf("extractExtension(%s) = %s, expected %s", test.url, result, test.expected)
		}
	}
}

func TestDownload(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "image/jpeg")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fake image data"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	ctx := context.Background()
	cachePath, err := downloader.Download(ctx, server.URL+"/image.jpg")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	// Check that file exists
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Error("Downloaded file does not exist")
	}

	// Check that content is correct
	content, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "fake image data" {
		t.Errorf("Downloaded content = %s, expected 'fake image data'", string(content))
	}

	// Test caching - second download should use cache
	cachePath2, err := downloader.Download(ctx, server.URL+"/image.jpg")
	if err != nil {
		t.Fatalf("Second download failed: %v", err)
	}

	if cachePath != cachePath2 {
		t.Error("Second download did not use cache")
	}
}

func TestIsCached(t *testing.T) {
	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	url := "https://example.com/image.jpg"

	// Initially not cached
	if downloader.IsCached(url) {
		t.Error("URL should not be cached initially")
	}

	// Create cache file manually
	cachePath := downloader.GetCachePath(url)
	if err := os.WriteFile(cachePath, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create cache file: %v", err)
	}

	// Now should be cached
	if !downloader.IsCached(url) {
		t.Error("URL should be cached after file creation")
	}
}

func TestClearCache(t *testing.T) {
	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	// Create some test files
	files := []string{"file1.txt", "file2.jpg", "file3.png"}
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if err := os.WriteFile(path, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Verify files exist
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Errorf("Test file %s does not exist", file)
		}
	}

	// Clear cache
	err := downloader.ClearCache()
	if err != nil {
		t.Fatalf("ClearCache failed: %v", err)
	}

	// Verify files are gone
	for _, file := range files {
		path := filepath.Join(tempDir, file)
		if _, err := os.Stat(path); !os.IsNotExist(err) {
			t.Errorf("Test file %s still exists after clear", file)
		}
	}
}

func TestGetCacheStats(t *testing.T) {
	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	// Empty cache
	stats, err := downloader.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}

	if stats.TotalFiles != 0 {
		t.Errorf("Expected 0 files, got %d", stats.TotalFiles)
	}

	if stats.TotalSize != 0 {
		t.Errorf("Expected 0 size, got %d", stats.TotalSize)
	}

	// Create some test files
	testData := []string{"short", "longer content here", "even longer content with more data"}
	var expectedSize int64
	oldestTime := time.Now()
	newestTime := time.Time{}

	for i, content := range testData {
		filename := filepath.Join(tempDir, "file"+string(rune('0'+i))+".txt")
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
		expectedSize += int64(len(content))

		// Set different modification times
		modTime := oldestTime.Add(time.Duration(i) * time.Hour)
		if err := os.Chtimes(filename, modTime, modTime); err != nil {
			t.Fatalf("Failed to set file time: %v", err)
		}

		if newestTime.IsZero() || modTime.After(newestTime) {
			newestTime = modTime
		}
	}

	// Get stats again
	stats, err = downloader.GetCacheStats()
	if err != nil {
		t.Fatalf("GetCacheStats failed: %v", err)
	}

	if stats.TotalFiles != len(testData) {
		t.Errorf("Expected %d files, got %d", len(testData), stats.TotalFiles)
	}

	if stats.TotalSize != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, stats.TotalSize)
	}

	if stats.CacheDirectory != tempDir {
		t.Errorf("Expected cache dir %s, got %s", tempDir, stats.CacheDirectory)
	}

	// Check timestamps (with some tolerance)
	timeTolerance := time.Second
	if stats.OldestFile.Sub(oldestTime) > timeTolerance {
		t.Errorf("Oldest file time %v doesn't match expected %v", stats.OldestFile, oldestTime)
	}

	if newestTime.Sub(stats.NewestFile) > timeTolerance {
		t.Errorf("Newest file time %v doesn't match expected %v", stats.NewestFile, newestTime)
	}
}

func TestCachePolicy(t *testing.T) {
	tempDir := t.TempDir()
	policy := CachePolicy{
		MaxAge:  time.Hour,
		MaxSize: 100,
	}

	downloader, _ := NewHTTPDownloader(tempDir, WithCachePolicy(policy))

	// Test setting policy
	newPolicy := CachePolicy{MaxAge: 2 * time.Hour}
	downloader.SetCachePolicy(newPolicy)

	// We can't easily test the internal policy without exposing it,
	// but we can verify the method exists and doesn't panic
}

func TestDownloadHTTPError(t *testing.T) {
	// Create test server that returns 404
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	downloader, _ := NewHTTPDownloader(tempDir)

	ctx := context.Background()
	_, err := downloader.Download(ctx, server.URL+"/notfound.jpg")
	if err == nil {
		t.Error("Expected download to fail with 404")
	}

	if !strings.Contains(err.Error(), "HTTP 404") {
		t.Errorf("Expected error to contain 'HTTP 404', got: %v", err)
	}
}

func TestDownloadWithHeaders(t *testing.T) {
	headersReceived := make(map[string]string)

	// Create test server that captures headers
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		headersReceived["User-Agent"] = r.Header.Get("User-Agent")
		headersReceived["X-Custom"] = r.Header.Get("X-Custom")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	headers := map[string]string{
		"X-Custom": "test-value",
	}

	downloader, _ := NewHTTPDownloader(tempDir,
		WithUserAgent("custom-agent"),
		WithHeaders(headers),
	)

	ctx := context.Background()
	_, err := downloader.Download(ctx, server.URL+"/test")
	if err != nil {
		t.Fatalf("Download failed: %v", err)
	}

	if headersReceived["User-Agent"] != "custom-agent" {
		t.Errorf("Expected User-Agent 'custom-agent', got '%s'", headersReceived["User-Agent"])
	}

	if headersReceived["X-Custom"] != "test-value" {
		t.Errorf("Expected X-Custom 'test-value', got '%s'", headersReceived["X-Custom"])
	}
}
