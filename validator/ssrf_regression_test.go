package validator

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestPartialOptionsKeepSecureDefaults is the regression test for the option
// merge: overriding one field must not silently zero the rest. The dangerous
// case was ResolveHostTimeout, whose zero value meant "skip DNS resolution".
func TestPartialOptionsKeepSecureDefaults(t *testing.T) {
	// A caller narrowing the scheme list used to lose hostname resolution
	// entirely, so any name -- including one that resolves to link-local --
	// was accepted.
	opts := &URLOptions{AllowedSchemes: []string{"https"}}
	normalized := normalizeURLOptions(opts)

	if normalized.ResolveHostTimeout <= 0 {
		t.Errorf("ResolveHostTimeout = %v; a zero value must mean \"use the default\", not \"disable the check\"", normalized.ResolveHostTimeout)
	}
	if normalized.DisableHostResolution {
		t.Error("DisableHostResolution was set without the caller asking for it")
	}

	// Opting out is still possible, but has to be spelled out.
	off := normalizeURLOptions(&URLOptions{DisableHostResolution: true})
	if !off.DisableHostResolution {
		t.Error("DisableHostResolution was not honoured")
	}

	// An explicit timeout still wins.
	custom := normalizeURLOptions(&URLOptions{ResolveHostTimeout: 2 * time.Second})
	if custom.ResolveHostTimeout != 2*time.Second {
		t.Errorf("ResolveHostTimeout = %v, want 2s", custom.ResolveHostTimeout)
	}
}

func TestValidateURLBlocksInternalTargets(t *testing.T) {
	blocked := []string{
		"http://127.0.0.1/",
		"http://[::1]/",
		"http://169.254.169.254/latest/meta-data/",
		"http://10.0.0.1/",
		"http://192.168.1.1/",
		"http://100.64.0.1/",
		"http://0.0.0.0/",
		"http://255.255.255.255/",
		"http://192.0.2.1/",
		"http://240.0.0.1/",
		"http://localhost/",
		"http://localhost./",
		"http://foo.localhost/",
	}
	for _, u := range blocked {
		if err := ValidateURL(u, nil); err == nil {
			t.Errorf("ValidateURL(%q) = nil, want an error", u)
		}
	}
}

// TestSSRFDialControl covers the half of the protection ValidateURL cannot
// provide: the address actually being connected to, after the request's own
// DNS resolution, so a rebinding answer is still caught.
func TestSSRFDialControl(t *testing.T) {
	control := SSRFDialControl(nil)

	blocked := []string{
		"169.254.169.254:80",
		"127.0.0.1:8080",
		"10.1.2.3:443",
		"[::1]:80",
	}
	for _, addr := range blocked {
		if err := control("tcp", addr, nil); err == nil {
			t.Errorf("SSRFDialControl rejected nothing for %q", addr)
		}
	}

	if err := control("tcp", "93.184.216.34:443", nil); err != nil {
		t.Errorf("SSRFDialControl blocked a public address: %v", err)
	}
	if err := control("tcp", "not-an-address", nil); err == nil {
		t.Error("SSRFDialControl accepted an unparseable address")
	}

	// Honours the options it was built with.
	allowLocal := SSRFDialControl(&URLOptions{AllowLocalhost: true, AllowPrivateIP: true})
	if err := allowLocal("tcp", "127.0.0.1:8080", nil); err != nil {
		t.Errorf("AllowLocalhost was not honoured: %v", err)
	}
}

// TestValidatePathSymlinkedParentForNewFile is the regression test for the
// create-a-new-file escape: EvalSymlinks fails for a path that does not exist,
// and falling back to the unresolved name checked the policy against a name
// that pointed somewhere else entirely.
func TestValidatePathSymlinkedParentForNewFile(t *testing.T) {
	base := t.TempDir()
	allowed := filepath.Join(base, "allowed")
	outside := filepath.Join(base, "outside")
	if err := os.MkdirAll(allowed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	// /allowed/link -> /outside
	if err := os.Symlink(outside, filepath.Join(allowed, "link")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	opts := &PathOptions{AllowedDirs: []string{allowed}, CheckTraversal: true}

	// The file does not exist yet, which is exactly the create case.
	got, err := ValidatePath(filepath.Join(allowed, "link", "newfile.txt"), opts)
	if err == nil {
		t.Fatalf("ValidatePath returned %q for a path whose parent symlinks outside the allowed dir", got)
	}

	// A genuinely contained new file still validates.
	inside := filepath.Join(allowed, "sub", "newfile.txt")
	if _, err := ValidatePath(inside, opts); err != nil {
		t.Errorf("ValidatePath(%q) = %v, want success", inside, err)
	}
}

// TestTraversalCheckIsSegmentWise: ".." must only match a whole segment.
func TestTraversalCheckIsSegmentWise(t *testing.T) {
	base := t.TempDir()
	opts := &PathOptions{AllowedDirs: []string{base}, CheckTraversal: true}

	for _, name := range []string{"backup..2024.log", "..hidden", "v1..2.json"} {
		if _, err := ValidatePath(filepath.Join(base, name), opts); err != nil {
			t.Errorf("ValidatePath(%q) = %v; \"..\" is not a whole segment here", name, err)
		}
	}

	if _, err := ValidatePath(filepath.Join(base, "..", "escape"), opts); err == nil {
		t.Error("a real traversal was accepted")
	}
}
