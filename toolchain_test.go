package main

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// createMaliciousZip creates a zip file containing a file with a path traversal payload.
func createMaliciousZip(t *testing.T, targetPath string) string {
	zipPath := filepath.Join(t.TempDir(), "malicious.zip")
	f, err := os.Create(zipPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	// Malicious file name
	fWriter, err := w.Create("../../../../malicious_zip.txt")
	if err != nil {
		t.Fatal(err)
	}
	fWriter.Write([]byte("malicious content"))

	// Safe file name
	sWriter, err := w.Create("safe.txt")
	if err != nil {
		t.Fatal(err)
	}
	sWriter.Write([]byte("safe content"))

	w.Close()
	return zipPath
}

// createMaliciousTarGz creates a tar.gz file containing a file with a path traversal payload.
func createMaliciousTarGz(t *testing.T, targetPath string) string {
	tarPath := filepath.Join(t.TempDir(), "malicious.tar.gz")
	f, err := os.Create(tarPath)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	gzw := gzip.NewWriter(f)
	tw := tar.NewWriter(gzw)

	// Malicious file
	malContent := []byte("malicious content")
	err = tw.WriteHeader(&tar.Header{
		Name: "../../../../malicious_tar.txt",
		Mode: 0600,
		Size: int64(len(malContent)),
	})
	if err != nil {
		t.Fatal(err)
	}
	tw.Write(malContent)

	// Safe file
	safeContent := []byte("safe content")
	err = tw.WriteHeader(&tar.Header{
		Name: "safe.txt",
		Mode: 0600,
		Size: int64(len(safeContent)),
	})
	if err != nil {
		t.Fatal(err)
	}
	tw.Write(safeContent)

	tw.Close()
	gzw.Close()
	return tarPath
}

func TestUnzipZipSlip(t *testing.T) {
	tempDir := t.TempDir()
	extractDir := filepath.Join(tempDir, "extract")
	os.MkdirAll(extractDir, 0755)

	zipPath := createMaliciousZip(t, extractDir)

	err := unzip(zipPath, extractDir)
	if err != nil {
		t.Fatalf("unzip returned an error: %v", err)
	}

	// Check if malicious file was extracted outside
	maliciousPath := filepath.Join(extractDir, "../../../../malicious_zip.txt")
	if _, err := os.Stat(maliciousPath); err == nil {
		t.Errorf("ZipSlip vulnerability detected! File was extracted outside target directory.")
		os.Remove(maliciousPath) // Cleanup
	}

	// Check if safe file was extracted
	safePath := filepath.Join(extractDir, "safe.txt")
	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		t.Errorf("Safe file was not extracted.")
	}
}

func TestUntarTarSlip(t *testing.T) {
	tempDir := t.TempDir()
	extractDir := filepath.Join(tempDir, "extract")
	os.MkdirAll(extractDir, 0755)

	tarPath := createMaliciousTarGz(t, extractDir)

	err := untar(tarPath, extractDir)
	if err != nil {
		t.Fatalf("untar returned an error: %v", err)
	}

	// Check if malicious file was extracted outside
	maliciousPath := filepath.Join(extractDir, "../../../../malicious_tar.txt")
	if _, err := os.Stat(maliciousPath); err == nil {
		t.Errorf("TarSlip vulnerability detected! File was extracted outside target directory.")
		os.Remove(maliciousPath) // Cleanup
	}

	// Check if safe file was extracted
	safePath := filepath.Join(extractDir, "safe.txt")
	if _, err := os.Stat(safePath); os.IsNotExist(err) {
		t.Errorf("Safe file was not extracted.")
	}
}

func TestListRemoteToolchains_MockedAPI(t *testing.T) {
	// Mock JSON response matching go.dev/dl/ API structure
	mockJSON := `[
		{
			"version": "go1.22.1",
			"stable": true,
			"files": [{"filename": "go1.22.1.src.tar.gz"}]
		},
		{
			"version": "go1.22.0",
			"stable": true,
			"files": [{"filename": "go1.22.0.src.tar.gz"}]
		},
		{
			"version": "go1.23rc1",
			"stable": false,
			"files": [{"filename": "go1.23rc1.src.tar.gz"}]
		}
	]`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, mockJSON)
	}))
	defer ts.Close()

	// Backup original URLs and restore after test
	originalAPIURL := goDevAPIURL
	defer func() {
		goDevAPIURL = originalAPIURL
	}()

	// Override API URL with our mock server
	goDevAPIURL = ts.URL

	// Clear cache to bypass cache layer in test
	cachePath := filepath.Join(GetCraftHome(), string(CraftCacheDir), RemoteVersionsCacheFile)
	os.Remove(cachePath)
	defer os.Remove(cachePath)

	// We don't need to redirect stdout anymore because we are testing the headless fetcher.
	releases, err := FetchRemoteToolchains()
	if err != nil {
		t.Fatalf("FetchRemoteToolchains failed with mock data: %v", err)
	}

	if len(releases) == 0 {
		t.Fatal("Expected releases to be parsed, got none")
	}
	if releases[0].Version != "go1.22.1" {
		t.Errorf("Expected first version to be go1.22.1, got %v", releases[0].Version)
	}
}

func TestFetchRemoteToolchains_NoInternet(t *testing.T) {
	// Backup original URLs and restore after test
	originalAPIURL := goDevAPIURL
	defer func() { goDevAPIURL = originalAPIURL }()

	// Override API URL with a closed local port or invalid URL to simulate no internet
	goDevAPIURL = "http://127.0.0.1:0"

	// Clear cache to bypass cache layer in test
	cachePath := filepath.Join(GetCraftHome(), string(CraftCacheDir), RemoteVersionsCacheFile)
	os.Remove(cachePath)

	releases, err := FetchRemoteToolchains()
	if err == nil {
		t.Fatalf("FetchRemoteToolchains expected to fail with no internet, but succeeded. Releases: %v", releases)
	}
	if len(releases) != 0 {
		t.Errorf("Expected 0 releases on failure, got %v", len(releases))
	}
}

func TestCheckSystemGoVersion(t *testing.T) {
	// Let's assume the machine running the tests has Go installed,
	// but we'll check a fake version to ensure it returns false.
	fakeVersion := "go9.99.9"
	match, err := CheckSystemGoVersion(fakeVersion)
	if err != nil {
		t.Fatalf("CheckSystemGoVersion error: %v", err)
	}
	if match {
		t.Errorf("System should not match fake version %s", fakeVersion)
	}
}
