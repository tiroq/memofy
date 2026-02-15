# Code Signing for macOS Binaries

## Problem

macOS enforces code signing validation for all executables. Unsigned or improperly signed binaries will be killed by the kernel with `SIGKILL` (signal 9), producing errors like:

```
CODE SIGNING: cs_invalid_page(...): denying page sending SIGKILL
CODE SIGNING: process [...]: rejecting invalid page (tainted:1)
```

This happens when:
- Binaries are built but not signed
- Binaries are modified after signing (corrupting the signature)
- Binaries are copied/downloaded without preserving signatures

## Solution

All macOS binaries must be signed with at least an **ad-hoc signature** using the `codesign` command.

### Ad-hoc Signing

Ad-hoc signing doesn't require a Developer ID certificate from Apple, making it suitable for:
- Local development builds
- Open-source projects without paid developer accounts
- CI/CD pipelines on GitHub Actions

```bash
codesign --force --sign - /path/to/binary
```

The `-` (dash) indicates ad-hoc signing with no specific identity.

## Implementation

### 1. Makefile (Development Builds)

The `Makefile` automatically signs binaries after building:

```makefile
build:
	mkdir -p bin
	$(GO) build -o $(BINARY_CORE) cmd/memofy-core/main.go
	$(GO) build -o $(BINARY_UI) cmd/memofy-ui/main.go
	@codesign --force --sign - $(BINARY_CORE)
	@codesign --force --sign - $(BINARY_UI)
```

### 2. Release Script (Local Releases)

The `scripts/build-release.sh` signs binaries before packaging:

```bash
# Sign macOS binaries with ad-hoc signature
echo "Signing binaries..."
codesign --force --deep --sign - "$output_dir/memofy-core"
codesign --force --deep --sign - "$output_dir/memofy-ui"

# Verify signatures
codesign --verify --verbose "$output_dir/memofy-core"
codesign --verify --verbose "$output_dir/memofy-ui"
```

### 3. GitHub Actions (CI/CD)

The `.github/workflows/release.yml` includes a signing step:

```yaml
- name: Sign binaries (macOS)
  if: matrix.goos == 'darwin'
  run: |
    echo "Signing binaries with ad-hoc signature..."
    codesign --force --deep --sign - "bin/memofy-core-${{ matrix.artifact_name }}"
    codesign --force --deep --sign - "bin/memofy-ui-${{ matrix.artifact_name }}"
    
    echo "Verifying signatures..."
    codesign --verify --verbose "bin/memofy-core-${{ matrix.artifact_name }}"
    codesign --verify --verbose "bin/memofy-ui-${{ matrix.artifact_name }}"
```

### 4. Installation Script

The `scripts/quick-install.sh` signs binaries after copying to installation directory:

```bash
# Sign binaries on macOS to prevent code signing issues
if [[ "$OS" == "Darwin" ]]; then
    print_info "Signing installed binaries..."
    codesign --force --sign - "$INSTALL_DIR/memofy-core"
    codesign --force --sign - "$INSTALL_DIR/memofy-ui"
fi
```

## Verification

To verify a binary is properly signed:

```bash
# Check if binary has a valid signature
codesign --verify --verbose /path/to/binary

# Expected output:
# /path/to/binary: valid on disk
# /path/to/binary: satisfies its Designated Requirement

# View signature details
codesign -dv /path/to/binary
```

## Troubleshooting

### Binary Gets Killed Immediately

**Symptom:**
```bash
$ memofy-core
zsh: killed     memofy-core
```

**Check System Logs:**
```bash
log show --predicate 'process == "kernel" AND eventMessage CONTAINS "CODE SIGNING"' --last 5m
```

**Solution:**
Re-sign the binary:
```bash
codesign --force --sign - /path/to/memofy-core
```

### Invalid Signature After Build

**Symptom:**
```bash
$ codesign --verify bin/memofy-core
code object is not signed at all
```

**Solution:**
Ensure the build process includes signing:
```bash
make clean && make build
```

## Why Ad-hoc Signing?

| Feature | Ad-hoc Signing | Apple Developer ID |
|---------|----------------|-------------------|
| Cost | Free | $99/year |
| Gatekeeper | ❌ Blocks | ✅ Allows |
| Code Signing | ✅ Valid | ✅ Valid |
| Notarization | ❌ Not possible | ✅ Possible |
| Binary Execution | ✅ Works locally | ✅ Works everywhere |

For open-source projects and local development, ad-hoc signing is sufficient. Users may need to:
1. Right-click → Open (first run only), or
2. Run: `xattr -d com.apple.quarantine /path/to/binary`

## Future: Developer ID Signing

For wider distribution, consider:
1. Purchase Apple Developer Program membership ($99/year)
2. Create signing certificate
3. Update workflows to use real signature:
   ```bash
   codesign --force --sign "Developer ID Application: Your Name" binary
   ```
4. Notarize binaries with Apple
5. Staple notarization ticket:
   ```bash
   xcrun stapler staple binary
   ```

## References

- [Apple Code Signing Guide](https://developer.apple.com/library/archive/documentation/Security/Conceptual/CodeSigningGuide/)
- [codesign man page](x-man-page://codesign)
- [Notarization Guide](https://developer.apple.com/documentation/security/notarizing_macos_software_before_distribution)
