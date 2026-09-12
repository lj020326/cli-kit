package validator

import (
	"net"
	"os"
	"path/filepath"
	"strings"
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

// --- Codex review round 2 (PR #6) ---

// TestGloballyReachableAnycastIsNotPrivate: 192.0.0.0/24 is mostly IETF
// protocol assignments, but it carries documented globally reachable
// exceptions -- 192.0.0.9 (PCP anycast, RFC 7723) and 192.0.0.10 (TURN
// anycast, RFC 8155). Blocking the whole /24 forced callers validating URLs
// for those services to enable ALL private addresses to reach a public
// endpoint.
func TestGloballyReachableAnycastIsNotPrivate(t *testing.T) {
	for _, ip := range []string{"192.0.0.9", "192.0.0.10"} {
		if isPrivateIP(net.ParseIP(ip)) {
			t.Errorf("%s classified private; it is a documented globally reachable anycast address", ip)
		}
	}
	// The rest of the block, and TEST-NET-1, stay blocked.
	for _, ip := range []string{"192.0.0.1", "192.0.0.8", "192.0.0.11", "192.0.2.1"} {
		if !isPrivateIP(net.ParseIP(ip)) {
			t.Errorf("%s is not a routable destination and must stay blocked", ip)
		}
	}
}

// TestDialControlAcceptsScopedIPv6: SplitHostPort returns "fe80::1%eth0" for
// "[fe80::1%eth0]:443", which net.ParseIP cannot parse -- so a scoped
// link-local address was rejected as "not an IP" even when link-local was
// explicitly allowed, and a zone is normally required for such an address to
// be usable at all.
func TestDialControlAcceptsScopedIPv6(t *testing.T) {
	control := SSRFDialControl(&URLOptions{AllowPrivateIP: true, DisableHostResolution: true})

	if err := control("tcp6", "[fe80::1%eth0]:443", nil); err != nil {
		t.Errorf("scoped link-local dial rejected: %v", err)
	}
	// Without AllowPrivateIP it is still refused -- but as a private address,
	// not as an unparseable one.
	strict := SSRFDialControl(&URLOptions{DisableHostResolution: true})
	err := strict("tcp6", "[fe80::1%eth0]:443", nil)
	if err == nil {
		t.Error("scoped link-local dial allowed without AllowPrivateIP")
	} else if strings.Contains(err.Error(), "not an IP") {
		t.Errorf("scoped address reported as unparseable: %v", err)
	}
}

// TestAllowedDirsResolvePartially is the regression test for resolving the
// requested path partially while leaving each allowed base lexical. With
// `alias -> /real` and AllowedDirs: ["alias/future"], the requested
// "alias/future/file" canonicalizes to "/real/future/file" while the base
// stayed "alias/future", so a genuinely contained creation was rejected.
func TestAllowedDirsResolvePartially(t *testing.T) {
	root := t.TempDir()
	real := filepath.Join(root, "real")
	if err := os.MkdirAll(real, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(real, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	opts := &PathOptions{AllowedDirs: []string{filepath.Join(alias, "future")}}

	// "future" does not exist yet: this is the create-new-file case.
	target := filepath.Join(alias, "future", "file.txt")
	if _, err := ValidatePath(target, opts); err != nil {
		t.Errorf("ValidatePath(%q) = %v, want it allowed: the base resolves to the same place", target, err)
	}

	// Containment is still enforced.
	outside := filepath.Join(alias, "elsewhere", "file.txt")
	if _, err := ValidatePath(outside, opts); err == nil {
		t.Error("a path outside the allowed directory was accepted")
	}
}

// --- Codex review round 6 (PR #6) ---

// TestTraversalRejectsWindowsNormalizedParents is the regression test for the
// exact ".." segment comparison.
//
// Win32 strips trailing spaces and periods from a path component, so `.. ` is
// opened as `..` and traverses to the parent. Comparing the segment exactly
// accepted `safe\.. \secret`, which the substring check it replaced had
// rejected -- so relaxing the check to allow legal names like "..hidden"
// reopened a traversal on Windows.
func TestTraversalRejectsWindowsNormalizedParents(t *testing.T) {
	opts := &PathOptions{CheckTraversal: true}

	for _, path := range []string{
		// The reported case, in both separators: the validator has to treat a
		// path string as untrusted regardless of the host it is running on.
		`safe\.. \secret`,
		"safe/.. /secret",
		"safe/..  /secret",

		// Spellings whose exact Win32 normalization is not worth depending on.
		"safe/.../secret",
		"safe/.. ./secret",
		"safe/ ../secret",

		// The plain form must keep being rejected.
		"safe/../secret",
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := ValidatePath(path, opts); err == nil {
				t.Errorf("ValidatePath(%q) was accepted; it reaches the parent directory once the platform normalizes it", path)
			}
		})
	}
}

// TestTraversalKeepsLegalDottedNames guards the other direction: relaxing the
// substring check was itself a fix, and these names must stay valid.
func TestTraversalKeepsLegalDottedNames(t *testing.T) {
	opts := &PathOptions{CheckTraversal: true}

	for _, path := range []string{
		"safe/..hidden/file",
		"safe/backup..2024.log",
		"safe/file..txt",
		"safe/.config/file",
		"safe/a..b/file",
	} {
		t.Run(path, func(t *testing.T) {
			if _, err := ValidatePath(path, opts); err != nil {
				t.Errorf("ValidatePath(%q) = %v, want it accepted: no segment is a parent reference", path, err)
			}
		})
	}
}
