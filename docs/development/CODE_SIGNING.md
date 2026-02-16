# Code Signing (macOS)

## Requirements

- Apple Developer account
- Valid signing certificate
- macOS 11.0+ 

## Sign Binaries

```bash
codesign --sign "Developer ID Application: Your Name" build/memofy-core
codesign --sign "Developer ID Application: Your Name" build/memofy-ui
```

## Verify

```bash
codesign -v build/memofy-core
codesign -d --entitlements - build/memofy-core
```

## Distribution

For GitHub releases, binaries should be:
1. Signed with Developer ID
2. Notarized by Apple
3. Stapled with notarization ticket

```bash
# Notarize
xcrun notarytool submit memofy.zip --wait
# Staple
xcrun stapler staple build/memofy-core
```

Not required for local development.
