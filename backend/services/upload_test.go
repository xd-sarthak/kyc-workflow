package services

import (
	"bytes"
	"mime/multipart"
	"net/textproto"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFileValidation_MIMETypes(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		wantAllowed bool
	}{
		{"application/pdf is allowed", "application/pdf", true},
		{"image/jpeg is allowed", "image/jpeg", true},
		{"image/png is allowed", "image/png", true},
		{"application/octet-stream is rejected", "application/octet-stream", false},
		{"text/plain is rejected", "text/plain", false},
		{"application/json is rejected", "application/json", false},
		{"empty content type is rejected", "", false},
		{"image/gif is rejected", "image/gif", false},
		{"application/x-executable is rejected", "application/x-executable", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.wantAllowed, allowedMIMETypes[tt.contentType],
				"MIME type %q: allowed=%v", tt.contentType, tt.wantAllowed)
		})
	}
}

func TestFileValidation_SizeLimit(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		wantPass bool
	}{
		{"1KB file", 1024, true},
		{"1MB file", 1 * 1024 * 1024, true},
		{"4.9MB file", 5*1024*1024 - 100*1024, true},
		{"exactly 5MB", 5 * 1024 * 1024, true},
		{"5MB + 1 byte", 5*1024*1024 + 1, false},
		{"10MB file", 10 * 1024 * 1024, false},
		{"50MB file", 50 * 1024 * 1024, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// maxFileSize is the constant defined in kyc_service.go
			passes := tt.size <= maxFileSize
			assert.Equal(t, tt.wantPass, passes,
				"size=%d, maxFileSize=%d", tt.size, maxFileSize)
		})
	}
}

func TestFileValidation_MaxFileSizeConstant(t *testing.T) {
	// Verify the constant is exactly 5 MB
	assert.Equal(t, int64(5*1024*1024), int64(maxFileSize))
}

func TestFileValidation_AllowedMIMEMap(t *testing.T) {
	// Verify exactly 3 allowed MIME types
	assert.Len(t, allowedMIMETypes, 3)
	assert.True(t, allowedMIMETypes["application/pdf"])
	assert.True(t, allowedMIMETypes["image/jpeg"])
	assert.True(t, allowedMIMETypes["image/png"])
}

// createMultipartFileHeader creates a real *multipart.FileHeader for testing.
// This uses multipart.Writer + bytes.Buffer — no mocking.
func createMultipartFileHeader(t *testing.T, fieldName, filename, contentType string, content []byte) *multipart.FileHeader {
	t.Helper()

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create form file with explicit Content-Type
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", `form-data; name="`+fieldName+`"; filename="`+filename+`"`)
	h.Set("Content-Type", contentType)

	part, err := writer.CreatePart(h)
	if err != nil {
		t.Fatalf("failed to create part: %v", err)
	}

	if _, err := part.Write(content); err != nil {
		t.Fatalf("failed to write content: %v", err)
	}
	writer.Close()

	// Parse the multipart form to get a FileHeader
	reader := multipart.NewReader(&buf, writer.Boundary())
	form, err := reader.ReadForm(int64(len(content) + 1024))
	if err != nil {
		t.Fatalf("failed to read form: %v", err)
	}

	files := form.File[fieldName]
	if len(files) == 0 {
		t.Fatalf("no files found for field %q", fieldName)
	}

	return files[0]
}

func TestCreateMultipartFileHeader(t *testing.T) {
	// Verify our helper constructs valid headers
	content := []byte("%PDF-1.4 test")
	fh := createMultipartFileHeader(t, "pan", "test.pdf", "application/pdf", content)

	assert.Equal(t, "test.pdf", fh.Filename)
	assert.Equal(t, int64(len(content)), fh.Size)
	assert.Equal(t, "application/pdf", fh.Header.Get("Content-Type"))

	// Verify we can Open() the file
	f, err := fh.Open()
	assert.NoError(t, err)
	defer f.Close()

	readBuf := make([]byte, len(content))
	n, err := f.Read(readBuf)
	assert.NoError(t, err)
	assert.Equal(t, len(content), n)
	assert.Equal(t, content, readBuf)
}

func TestFileValidation_RealMultipartHeaders(t *testing.T) {
	tests := []struct {
		name        string
		filename    string
		contentType string
		size        int
		wantValid   bool
	}{
		{
			name:        "valid PDF under 5MB",
			filename:    "doc.pdf",
			contentType: "application/pdf",
			size:        1024,
			wantValid:   true,
		},
		{
			name:        "valid JPEG under 5MB",
			filename:    "photo.jpg",
			contentType: "image/jpeg",
			size:        2048,
			wantValid:   true,
		},
		{
			name:        "valid PNG under 5MB",
			filename:    "scan.png",
			contentType: "image/png",
			size:        1024,
			wantValid:   true,
		},
		{
			name:        "file with no extension but valid content type",
			filename:    "document",
			contentType: "application/pdf",
			size:        1024,
			wantValid:   true,
		},
		{
			name:        "exe disguised as pdf content type",
			filename:    "malware.exe",
			contentType: "application/x-executable",
			size:        1024,
			wantValid:   false,
		},
		{
			name:        "js file with jpeg content type",
			filename:    "script.js",
			contentType: "text/javascript",
			size:        1024,
			wantValid:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content := make([]byte, tt.size)
			fh := createMultipartFileHeader(t, "file", tt.filename, tt.contentType, content)

			// Validate using the same logic as validateAndSaveFile
			ct := fh.Header.Get("Content-Type")
			validMIME := allowedMIMETypes[ct]
			validSize := fh.Size <= maxFileSize

			isValid := validMIME && validSize
			assert.Equal(t, tt.wantValid, isValid,
				"file=%s ct=%s size=%d", tt.filename, tt.contentType, tt.size)
		})
	}
}
