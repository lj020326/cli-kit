package validator

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// PathOptions configures path validation behavior
type PathOptions struct {
	// AllowRelative allows relative paths (default: true)
	AllowRelative bool
	// AllowedDirs restricts paths to specific directories (default: empty, no restriction)
	AllowedDirs []string
	// CheckTraversal checks for path traversal attacks (default: true)
	CheckTraversal bool
}

// defaultPathOptions returns default path validation options
func defaultPathOptions() *PathOptions {
	return &PathOptions{
		AllowRelative:  true,
		AllowedDirs:    nil,
		CheckTraversal: true,
	}
}

// ValidatePath validates a file path to prevent path traversal attacks
//
// This function validates file paths, including:
// - Path traversal detection (..), including after normalization to resist bypasses
// - Absolute vs relative path handling
// - Optional directory restrictions
//
// Parameters:
//   - path: File path to validate
//   - opts: Optional validation options (nil uses defaults)
//
// Returns:
//   - string: Normalized absolute path
//   - error: Returns error if path is invalid or has security risks; otherwise returns nil
//
// NOTE: the returned path is a name, not an open handle, so the usual
// time-of-check/time-of-use gap applies -- a component can be replaced with a
// symlink between this call and the open. Where that matters, open first and
// validate the opened file (os.File.Name plus a Stat comparison), or hold the
// directory open across both operations.
func ValidatePath(path string, opts *PathOptions) (string, error) {
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Use default options if not provided
	if opts == nil {
		opts = defaultPathOptions()
	}

	// Check for path traversal in original path before converting to absolute.
	// ".." only counts as a whole path segment: "backup..2024.log" and
	// "..hidden" are legal names and were being rejected by a substring match.
	if opts.CheckTraversal {
		if containsTraversalSegment(path) {
			return "", fmt.Errorf("path cannot contain path traversal characters (..)")
		}
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("unable to parse path: %w", err)
	}

	// Security: after normalization, ensure no ".." segment remains (guards against
	// encoding/unicode bypasses or platform quirks that might bypass the string check)
	if opts.CheckTraversal {
		cleaned := filepath.Clean(absPath)
		if containsTraversalSegment(cleaned) {
			return "", fmt.Errorf("path cannot contain path traversal characters (..)")
		}
		absPath = cleaned
	}

	// Resolve existing symlinks so policy checks apply to the real on-disk target.
	// For non-existent paths, keep the cleaned absolute path as-is.
	resolvedPath, err := resolvePathForPolicy(absPath)
	if err != nil {
		return "", err
	}

	// Check directory restrictions: path must be exactly allowedDir or under it (no prefix bypass)
	if len(opts.AllowedDirs) > 0 {
		allowed := false
		for _, allowedDir := range opts.AllowedDirs {
			allowedAbsDir, err := filepath.Abs(allowedDir)
			if err != nil {
				continue
			}
			allowedAbsDir = filepath.Clean(allowedAbsDir)

			if isPathWithinBase(resolvedPath, allowedAbsDir) {
				allowed = true
				break
			}

			// Resolve the allowed base the SAME way the requested path was
			// resolved: partially, down to its deepest existing ancestor.
			//
			// EvalSymlinks alone fails outright when the allowed directory
			// does not exist yet. With `alias -> /real` and
			// AllowedDirs: ["alias/future"], the requested "alias/future/file"
			// canonicalizes to "/real/future/file" while the base stayed the
			// lexical "alias/future", so a genuinely contained creation was
			// rejected.
			allowedResolvedDir, err := resolvePathForPolicy(allowedAbsDir)
			if err == nil && isPathWithinBase(resolvedPath, allowedResolvedDir) {
				allowed = true
				break
			}
		}
		if !allowed {
			// Do not include AllowedDirs in error to avoid leaking allowed paths to callers (e.g. API responses)
			return "", fmt.Errorf("path is not under allowed directories")
		}
	}

	return resolvedPath, nil
}

// containsTraversalSegment returns true if path contains ".." as a path segment.
func containsTraversalSegment(path string) bool {
	// The drive prefix comes off first. A Windows drive-RELATIVE path fuses
	// its first segment to the drive letter: "C:..\secret" is "C:../secret"
	// after ToSlash, whose leading segment is "C:..", not "..". That slipped
	// through here, and filepath.Abs then resolved and cleaned the traversal
	// away before the containment check below could see it -- so
	// CheckTraversal accepted an input the substring test it replaced had
	// rejected.
	path = path[windowsVolumeLen(path):]

	// Split on BOTH separators, on every platform. filepath.ToSlash is a
	// no-op outside Windows, so a backslash-separated path validated on Linux
	// was one long segment and no traversal in it was visible -- and the same
	// reasoning as windowsVolumeLen applies: the string need not have been
	// written on the machine validating it, and a path this validator blesses
	// may well be used on one where "\" separates. Empty segments are dropped;
	// none of them is a parent reference.
	for _, part := range strings.FieldsFunc(path, func(r rune) bool {
		return r == '/' || r == '\\'
	}) {
		if isParentSegment(part) {
			return true
		}
	}
	return false
}

// isParentSegment reports whether a path segment refers to the parent
// directory once the platform is done with it.
//
// Not just `part == ".."`. Win32 strips trailing spaces and periods from a
// path component, so ".. " is opened as ".." and traverses -- an exact
// comparison accepted `safe\.. \secret` while the substring check this
// replaced had rejected it.
//
// Any segment made only of periods and spaces with at least two periods
// counts. That is wider than the one spelling: the exact order in which Win32
// strips a trailing run of periods and spaces is not something a security
// check should depend on, and the segments this over-rejects -- "...",
// ".. ." -- cannot name a file on Windows at all, since normalization leaves
// them empty. On POSIX "..." IS a legal name, so this refuses one legal
// spelling; under an explicitly requested traversal check that is the right
// side to err on, and the same reasoning as windowsVolumeLen below: a path
// string reaching this validator need not have been written on the machine
// validating it.
func isParentSegment(part string) bool {
	dots := 0
	for i := 0; i < len(part); i++ {
		switch part[i] {
		case '.':
			dots++
		case ' ':
		default:
			return false
		}
	}
	return dots >= 2
}

// windowsVolumeLen returns the length of a leading Windows drive prefix
// ("C:"), or 0 when there is none.
//
// filepath.VolumeName is not used because it only recognises one when GOOS is
// windows, and a path string reaching this validator need not have been
// written on the machine validating it. For a traversal check, recognising one
// too eagerly only rejects more.
func windowsVolumeLen(path string) int {
	if len(path) >= 2 && path[1] == ':' {
		if c := path[0]; ('a' <= c && c <= 'z') || ('A' <= c && c <= 'Z') {
			return 2
		}
	}
	return 0
}

// resolvePathForPolicy resolves symlinks so policy checks apply to the real
// on-disk target.
//
// For a path that does not exist yet -- creating a file, say -- EvalSymlinks
// fails outright, and simply falling back to the cleaned path checked the
// policy against the *unresolved* name. That let a symlinked parent escape:
// with /allowed/link a symlink to /etc, validating /allowed/link/newfile
// looked like it was under /allowed while the write actually landed in
// /etc/newfile. Reads of existing files were protected; creating new ones was
// not.
//
// So resolve the deepest ancestor that does exist, then re-attach the
// remaining components to the resolved prefix.
func resolvePathForPolicy(path string) (string, error) {
	resolved, err := filepath.EvalSymlinks(path)
	if err == nil {
		return filepath.Clean(resolved), nil
	}
	if !os.IsNotExist(err) {
		return "", fmt.Errorf("unable to resolve path symlinks: %w", err)
	}

	cleaned := filepath.Clean(path)
	var trailing []string
	current := cleaned

	for {
		parent := filepath.Dir(current)
		if parent == current {
			// Reached the root without finding anything that exists.
			return cleaned, nil
		}
		trailing = append([]string{filepath.Base(current)}, trailing...)
		current = parent

		resolvedParent, err := filepath.EvalSymlinks(current)
		if err == nil {
			return filepath.Clean(filepath.Join(append([]string{resolvedParent}, trailing...)...)), nil
		}
		if !os.IsNotExist(err) {
			return "", fmt.Errorf("unable to resolve path symlinks: %w", err)
		}
	}
}

// isPathWithinBase returns true when path == base or path is under base.
func isPathWithinBase(path, base string) bool {
	rel, err := filepath.Rel(base, path)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// ErrFileNotFound is returned when a file does not exist
var ErrFileNotFound = fmt.Errorf("file not found")

// ErrNotAFile is returned when the path is not a regular file
var ErrNotAFile = fmt.Errorf("path is not a file")

// ErrDirNotFound is returned when a directory does not exist
var ErrDirNotFound = fmt.Errorf("directory not found")

// ErrNotADirectory is returned when the path is not a directory
var ErrNotADirectory = fmt.Errorf("path is not a directory")

// ErrFileNotReadable is returned when a file cannot be read
var ErrFileNotReadable = fmt.Errorf("file is not readable")

// ErrDirNotWritable is returned when a directory is not writable
var ErrDirNotWritable = fmt.Errorf("directory is not writable")

// ValidateFileExists validates that a file exists at the given path
//
// Parameters:
//   - path: The file path to validate
//
// Returns:
//   - error: Returns ErrFileNotFound if the file doesn't exist, ErrNotAFile if the path is a directory, nil otherwise
func ValidateFileExists(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrFileNotFound, path)
		}
		return fmt.Errorf("unable to access path: %w", err)
	}

	if info.IsDir() {
		return fmt.Errorf("%w: %s is a directory", ErrNotAFile, path)
	}

	return nil
}

// ValidateFileReadable validates that a file exists and is readable
//
// Parameters:
//   - path: The file path to validate
//
// Returns:
//   - error: Returns error if file doesn't exist or can't be read, nil otherwise
func ValidateFileReadable(path string) error {
	// First check if file exists
	if err := ValidateFileExists(path); err != nil {
		return err
	}

	// Try to open the file for reading
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("%w: %s", ErrFileNotReadable, path)
	}
	_ = f.Close()

	return nil
}

// ValidateDirExists validates that a directory exists at the given path
//
// Parameters:
//   - path: The directory path to validate
//
// Returns:
//   - error: Returns ErrDirNotFound if the directory doesn't exist, ErrNotADirectory if the path is a file, nil otherwise
func ValidateDirExists(path string) error {
	if path == "" {
		return fmt.Errorf("path cannot be empty")
	}

	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("%w: %s", ErrDirNotFound, path)
		}
		return fmt.Errorf("unable to access path: %w", err)
	}

	if !info.IsDir() {
		return fmt.Errorf("%w: %s is a file", ErrNotADirectory, path)
	}

	return nil
}

// ValidateDirWritable validates that a directory exists and is writable
// It creates a temporary file to verify write permissions
//
// Parameters:
//   - path: The directory path to validate
//
// Returns:
//   - error: Returns error if directory doesn't exist or is not writable, nil otherwise
func ValidateDirWritable(path string) error {
	// First check if directory exists
	if err := ValidateDirExists(path); err != nil {
		return err
	}

	// Try to create a temporary file to verify write permissions
	f, err := os.CreateTemp(path, ".write_test_"+randomSuffix()+"_*")
	if err != nil {
		return fmt.Errorf("%w: %s", ErrDirNotWritable, path)
	}
	testFile := f.Name()
	_ = f.Close()
	_ = os.Remove(testFile)

	return nil
}

// randomSuffix generates a random suffix for write-test filenames to avoid predictability and races.
func randomSuffix() string {
	b := make([]byte, 4)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}
