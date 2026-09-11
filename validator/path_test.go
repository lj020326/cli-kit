package validator

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Get current working directory for tests
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	tests := []struct {
		name      string
		path      string
		opts      *PathOptions
		wantErr   bool
		errSubstr string
	}{
		// Default options
		{"valid relative path", "test.txt", nil, false, ""},
		{"valid absolute path", cwd, nil, false, ""},
		{"empty path", "", nil, true, "empty"},
		{"path with ..", "../test.txt", nil, true, "traversal"},
		{"path with multiple ..", "../../etc/passwd", nil, true, "traversal"},

		// With directory restrictions
		{"path in allowed dir", cwd, &PathOptions{AllowedDirs: []string{cwd}}, false, ""},
		{"path outside allowed dir", "/tmp", &PathOptions{AllowedDirs: []string{cwd}}, true, "allowed"},
		{"path in one of allowed dirs", cwd, &PathOptions{AllowedDirs: []string{"/tmp", cwd}}, false, ""},
		{"path prefix bypass rejected", "/tmpfoo/bar", &PathOptions{AllowedDirs: []string{"/tmp"}}, true, "allowed"},

		// With traversal check disabled
		{"traversal check disabled", "../test.txt", &PathOptions{CheckTraversal: false}, false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ValidatePath(tt.path, tt.opts)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidatePath(%q, %+v) error = %v, wantErr %v", tt.path, tt.opts, err, tt.wantErr)
				return
			}
			if tt.wantErr {
				if tt.errSubstr != "" && err != nil && !contains(err.Error(), tt.errSubstr) {
					t.Errorf("ValidatePath(%q, %+v) error = %v, want error containing %q", tt.path, tt.opts, err, tt.errSubstr)
				}
				return
			}
			// Check that returned path is absolute
			if !filepath.IsAbs(got) {
				t.Errorf("ValidatePath(%q, %+v) = %q, want absolute path", tt.path, tt.opts, got)
			}
		})
	}
}

func TestContainsTraversalSegment(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/a/b/c", false},
		{"/a/../b", true},
		{"..", true},
		{"a/..", true},
		{"", false},
		{string(filepath.Separator) + "..", true},

		// Windows drive-RELATIVE paths: the parent segment is fused to the
		// drive letter, so splitting on separators alone never sees "..".
		// filepath.Abs then resolved the traversal away before the
		// containment check could catch it.
		{"C:../secret", true},
		{"c:../secret", true},
		{"C:..", true},
		{"C:/absolute/ok", false},

		// A leading "X:" that is not a drive letter, and ordinary names that
		// merely start with "..", stay allowed.
		{"1:../secret", false},
		{"..hidden", false},
		{"backup..2024.log", false},
	}
	for _, tt := range tests {
		got := containsTraversalSegment(tt.path)
		if got != tt.want {
			t.Errorf("containsTraversalSegment(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsPathWithinBase(t *testing.T) {
	base := filepath.Clean("/tmp/base")
	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/base", true},
		{"/tmp/base/file.txt", true},
		{"/tmp/base/sub/dir", true},
		{"/tmp/base2/file.txt", false},
		{"/tmp/other", false},
	}

	for _, tt := range tests {
		got := isPathWithinBase(filepath.Clean(tt.path), base)
		if got != tt.want {
			t.Errorf("isPathWithinBase(%q, %q) = %v, want %v", tt.path, base, got, tt.want)
		}
	}
}

func TestValidatePath_SymlinkEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink test is platform/permission dependent on Windows")
	}

	root := t.TempDir()
	allowedDir := filepath.Join(root, "allowed")
	outsideDir := filepath.Join(root, "outside")
	if err := os.MkdirAll(allowedDir, 0o755); err != nil {
		t.Fatalf("Failed to create allowed dir: %v", err)
	}
	if err := os.MkdirAll(outsideDir, 0o755); err != nil {
		t.Fatalf("Failed to create outside dir: %v", err)
	}

	secretFile := filepath.Join(outsideDir, "secret.txt")
	if err := os.WriteFile(secretFile, []byte("secret"), 0o600); err != nil {
		t.Fatalf("Failed to write secret file: %v", err)
	}

	linkDir := filepath.Join(allowedDir, "escape")
	if err := os.Symlink(outsideDir, linkDir); err != nil {
		t.Skipf("Skipping symlink test due to platform restrictions: %v", err)
	}

	_, err := ValidatePath(filepath.Join(linkDir, "secret.txt"), &PathOptions{
		AllowedDirs: []string{allowedDir},
	})
	if err == nil {
		t.Fatal("ValidatePath() should reject symlink escape outside allowed directories")
	}
	if !contains(err.Error(), "allowed") {
		t.Errorf("ValidatePath() error = %v, want error containing %q", err, "allowed")
	}
}

func TestValidatePath_EdgeCases(t *testing.T) {
	// Test with non-existent path (should still work, just normalize)
	path, err := ValidatePath("./nonexistent.txt", nil)
	if err != nil {
		t.Errorf("ValidatePath() with non-existent path error = %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ValidatePath() = %q, want absolute path", path)
	}

	// Test with allowed dir that fails to convert to absolute (should skip it)
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get current directory: %v", err)
	}

	// This test case covers the error handling in AllowedDirs processing
	// where filepath.Abs might fail for an allowed dir
	path, err = ValidatePath(cwd, &PathOptions{
		AllowedDirs: []string{cwd, "/nonexistent/dir/that/might/fail"},
	})
	if err != nil {
		t.Errorf("ValidatePath() with potentially invalid allowed dir error = %v", err)
	}
	if !filepath.IsAbs(path) {
		t.Errorf("ValidatePath() = %q, want absolute path", path)
	}
}

func TestValidateFileExists(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_file_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFilePath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFilePath) }()

	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test_dir_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{"existing file", tmpFilePath, false, nil},
		{"non-existent file", "/nonexistent/path/to/file.txt", true, ErrFileNotFound},
		{"directory instead of file", tmpDir, true, ErrNotAFile},
		{"empty path", "", true, nil},
	}

	// Path that causes non-IsNotExist error (e.g. NUL in path on Unix)
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			wantErr bool
			errType error
		}{"invalid path NUL", "\x00", true, nil})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileExists(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileReadable(t *testing.T) {
	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_readable_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFilePath := tmpFile.Name()
	_, _ = tmpFile.WriteString("test content")
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFilePath) }()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"readable file", tmpFilePath, false},
		{"non-existent file", "/nonexistent/path/to/file.txt", true},
		{"empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileReadable(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileReadable(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}

	// File exists but not readable (no read permission) - covers os.Open failure path.
	//
	// Permission bits are not enforced for uid 0, so this case alone cannot be
	// simulated in a root container -- the default for many CI images. The
	// table-driven cases above do not depend on permission bits and keep
	// running there.
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		noReadFile, err := os.CreateTemp("", "test_noread_*")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		_, _ = noReadFile.WriteString("x")
		_ = noReadFile.Close()
		path := noReadFile.Name()
		defer func() {
			_ = os.Chmod(path, 0o644)
			_ = os.Remove(path)
		}()
		if err := os.Chmod(path, 0o000); err != nil {
			t.Skipf("Cannot chmod 0 for test: %v", err)
		}
		err = ValidateFileReadable(path)
		if err == nil {
			t.Error("ValidateFileReadable() on non-readable file want error, got nil")
		}
	}
}

func TestValidateDirExists(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test_dir_exists_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Create a temporary file for testing
	tmpFile, err := os.CreateTemp("", "test_file_*")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	tmpFilePath := tmpFile.Name()
	_ = tmpFile.Close()
	defer func() { _ = os.Remove(tmpFilePath) }()

	tests := []struct {
		name    string
		path    string
		wantErr bool
		errType error
	}{
		{"existing directory", tmpDir, false, nil},
		{"non-existent directory", "/nonexistent/path/to/dir", true, ErrDirNotFound},
		{"file instead of directory", tmpFilePath, true, ErrNotADirectory},
		{"empty path", "", true, nil},
	}
	if runtime.GOOS != "windows" {
		tests = append(tests, struct {
			name    string
			path    string
			wantErr bool
			errType error
		}{"invalid path NUL", "\x00", true, nil})
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirExists(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirExists(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}
}

func TestValidateDirWritable(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "test_dir_writable_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"writable directory", tmpDir, false},
		{"non-existent directory", "/nonexistent/path/to/dir", true},
		{"empty path", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDirWritable(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDirWritable(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
			}
		})
	}

	// Read-only directory: Create should fail (covers ErrDirNotWritable path).
	// Skipped for uid 0, which is not subject to the permission bits; the
	// table-driven cases above still run there.
	if runtime.GOOS != "windows" && os.Geteuid() != 0 {
		readOnlyDir, err := os.MkdirTemp("", "test_readonly_*")
		if err != nil {
			t.Fatalf("Failed to create temp dir: %v", err)
		}
		defer func() { _ = os.RemoveAll(readOnlyDir) }()
		if err := os.Chmod(readOnlyDir, 0o555); err != nil {
			t.Skipf("Cannot chmod read-only for test: %v", err)
		}
		err = ValidateDirWritable(readOnlyDir)
		if err == nil {
			t.Error("ValidateDirWritable() on read-only dir want error, got nil")
		}
		_ = os.Chmod(readOnlyDir, 0o755)
	}
}
