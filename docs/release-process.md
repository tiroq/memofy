# Release Process (Maintainers)

## Quick Release Commands

### Using Task

```bash
# Auto-bump stable releases
task release-major    # 0.1.0 → 1.0.0
task release-minor    # 0.1.0 → 0.2.0  
task release-patch    # 0.1.0 → 0.1.1

# Auto pre-releases (from latest stable + 1 patch)
task release-alpha-auto    # v0.1.0 → v0.1.1-alpha1
task release-beta-auto     # v0.1.0 → v0.1.1-beta1

# Specific versions
task release-stable VERSION=0.2.0
task release-alpha VERSION=0.2.0 ALPHA=1
task release-beta VERSION=0.2.0 BETA=1
task release-rc VERSION=0.2.0 RC=1
```

All release tasks automatically:
- ✅ Run tests + linter
- ✅ Verify clean git tree
- ✅ Create and push tag
- ✅ Trigger GitHub Actions
- ✅ Wait for CI to complete
- ✅ Exit with error if build fails

---

## Manual Release

If not using Task:

```bash
# 1. Verify ready
make test
make lint
git status  # ensure clean

# 2. Create tag
git tag v0.2.0

# 3. Push to GitHub
git push origin main v0.2.0

# 4. Monitor build
gh run watch
```

---

## Version Format

Follow [Semantic Versioning](https://semver.org/):

- **Stable**: `v0.2.0`
- **Release Candidate**: `v0.2.0-rc1`
- **Beta**: `v0.2.0-beta1`
- **Alpha**: `v0.2.0-alpha1`

**Breaking changes**: Bump major (0.x.x → 1.0.0)
**New features**: Bump minor (0.1.x → 0.2.0)
**Bug fixes**: Bump patch (0.1.0 → 0.1.1)

---

## CI Build Matrix

GitHub Actions builds for all platforms:

| OS | Arch |
|---------|--------|
| macOS | amd64, arm64 |
| Linux | amd64, arm64 |
| Windows | amd64, arm64 |

**Artifacts**:
- macOS/Linux: `.tar.gz` with both binaries
- Windows: `.zip` with `.exe` files

---

## Release Checklist

Before releasing:

- [ ] All tests pass
- [ ] Linter passes
- [ ] Version bumped if needed
- [ ] CHANGELOG/notes prepared
- [ ] Git tree clean
- [ ] On main branch

After pushing tag:

- [ ] CI builds complete successfully
- [ ] All platform artifacts present
- [ ] GitHub Release created
- [ ] Test download on one platform
- [ ] Update release notes

---

## Handling Failed Releases

If CI build fails:

```bash
# 1. Delete tag locally and remotely
git tag -d v0.2.0
git push origin :v0.2.0

# 2. Fix issue
git commit -m "fix: build issue"

# 3. Retry release
git tag v0.2.0
git push origin main v0.2.0
```

---

## User Notifications

**Stable releases (`v0.2.0`)**:
- All users notified via auto-update
- Marked as "Latest Release"

**Pre-releases (`v0.2.0-alpha1`)**:
- Only users with `allow_dev_updates: true`
- Marked as "Pre-release"
- Use `--pre-release` install flag

---

## Testing Releases

### Download and Test

```bash
# Test latest stable
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash

# Test latest pre-release
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --pre-release

# Test specific version
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash -s -- --release 0.2.0
```

### Local Build Test

```bash
# Build all platforms locally (requires Docker)
task release-local

# Outputs to dist/ directory
ls -lh dist/
```

---

## Utilities

```bash
# List all releases
task release-list

# Check GitHub Actions status
task release-status

# Delete a release (use carefully!)
task release-delete VERSION=v0.2.0

# Verify release
task release-verify VERSION=v0.2.0
```

---

## Workflow File

See `.github/workflows/release.yml` for CI configuration.

Triggered on any `v*` tag push.

---

**Related**: [Taskfile Guide](taskfile-guide.md) | [Development](development.md)
