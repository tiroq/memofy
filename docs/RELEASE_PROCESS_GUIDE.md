# Release Process Guide for Maintainers

This guide explains how to create and publish releases for Memofy using GitHub Actions.

## Quick Start: Publishing a Release

### Using Task (Recommended)

The simplest way to create releases is using [Task](https://taskfile.dev/):

```bash
# Stable release (v0.2.0)
task release-stable VERSION=0.2.0

# Auto-bumped stable releases (no args)
task release-major
task release-minor
task release-patch

# Release candidate (v0.2.0-rc1)
task release-rc VERSION=0.2.0 RC=1

# Beta release (v0.2.0-beta1)
task release-beta VERSION=0.2.0 BETA=1

# Alpha release (v0.2.0-alpha1)
task release-alpha VERSION=0.2.0 ALPHA=1

# Auto alpha/beta dev releases (no args)
task release-alpha-auto
task release-beta-auto
```

The task automatically:
- ✅ Runs all tests
- ✅ Runs linter
- ✅ Verifies git is clean
- ✅ Creates the tag
- ✅ Pushes to GitHub
- ✅ Triggers GitHub Actions

See [Taskfile Guide](TASKFILE_GUIDE.md) for complete details.

### Manual Method

If you prefer manual control:

1. **Prepare the Release**

```bash
# Update version in code (if needed)
# Edit any relevant version files

# Commit final changes
git add .
git commit -m "Release: v0.2.0"

# Create version tag
git tag v0.2.0

# Push to GitHub
git push origin main v0.2.0
```

GitHub Actions automatically detects the tag and:
- Builds for all platforms (macOS, Linux, Windows)
- Creates pre-compiled binaries for each platform
- Creates a GitHub Release with all artifacts attached

2. **Verify Release**

1. Go to [GitHub Releases](https://github.com/tiroq/memofy/releases)
2. Confirm the release appears with all platform artifacts
3. Users will be notified automatically

## Release Types

### Stable Release

**Format**: `v0.2.0`

```bash
git tag v0.2.0
git push origin main v0.2.0
```

- Marked as "Latest Release" on GitHub
- All users are notified via auto-update system
- Shows in stable release channel

### Pre-Release (Release Candidate)

**Format**: `v0.2.0-rc1`

```bash
git tag v0.2.0-rc1
git push origin main v0.2.0-rc1
```

- Marked as "Pre-release" on GitHub
- Only notifies users with `allow_dev_updates: true`
- Useful for public testing before stable release

### Pre-Release (Beta)

**Format**: `v0.2.0-beta1`

```bash
git tag v0.2.0-beta1
git push origin main v0.2.0-beta1
```

- Marked as "Pre-release" on GitHub
- Early access for testers
- Similar to RC but indicates earlier development stage

### Pre-Release (Alpha)

**Format**: `v0.2.0-alpha1`

```bash
git tag v0.2.0-alpha1
git push origin main v0.2.0-alpha1
```

- Marked as "Pre-release" on GitHub
- Earliest development milestone
- For internal development team

**Auto alpha/beta behavior**:
- `task release-alpha-auto` and `task release-beta-auto` derive the base version from the latest stable tag and bump the patch by 1 (e.g., latest stable `v1.4.2` → `v1.4.3-alpha1`).
- If a tag for that base already exists, the numeric suffix is incremented (e.g., `alpha2`, `beta3`).

## Using Task for Releases

[Task](https://taskfile.dev/) provides convenient commands for the entire release workflow.

### Release Workflow with Task

**1. Check if ready:**
```bash
task release-check
```
Automatically verifies:
- ✅ All tests pass
- ✅ Linter passes  
- ✅ Git working directory is clean
- ✅ On main branch

**2. Create release:**
```bash
# Stable release
task release-stable VERSION=0.2.0

# Or pre-release
task release-rc VERSION=0.2.0 RC=1
```

**3. Monitor:**
```bash
task release-status
```

### All Task Release Commands

```bash
# Stable releases
task release-stable VERSION=0.2.0
task release-major
task release-minor
task release-patch

# Pre-releases
task release-rc VERSION=0.2.0 RC=1      # Release candidate
task release-beta VERSION=0.2.0 BETA=1  # Beta version
task release-alpha VERSION=0.2.0 ALPHA=1 # Alpha version
task release-alpha-auto                 # Auto alpha based on latest stable
task release-beta-auto                  # Auto beta based on latest stable

# Utilities
task release-check           # Verify ready for release
task release-list            # List all tags
task release-status          # Check GitHub Actions status
task release-delete VERSION=v0.2.0  # Delete a tag

# Local testing
task release-local           # Build all platforms locally
```

See [Taskfile Guide](TASKFILE_GUIDE.md) for complete documentation.

## Semantic Versioning

Memofy follows [Semantic Versioning](https://semver.org/):

- **Major.Minor.Patch** (e.g., `0.2.0`)
- **Major**: Breaking changes
- **Minor**: New features, backward compatible
- **Patch**: Bug fixes

Examples:
- `v0.1.0` → `v0.2.0` (new features)
- `v0.2.0` → `v0.2.1` (bug fix)
- `v0.2.0` → `v1.0.0` (breaking changes)

## GitHub Actions Workflow Details

### File Location

`.github/workflows/release.yml` - Automated release workflow

### Trigger

Workflow runs when:
- A tag matching `v*` pattern is pushed
- Example: `v0.2.0`, `v0.2.0-rc1`

### Build Matrix

Builds for all combinations:

| OS | Architecture |
|------|-------------|
| macOS | arm64 (Apple Silicon) |
| macOS | amd64 (Intel) |
| Linux | amd64 |
| Linux | arm64 |
| Windows | amd64 |
| Windows | arm64 |

### Build Output

Each build produces:

**macOS & Linux**:
- `memofy-core-{os}-{arch}` - Core binary
- `memofy-ui-{os}-{arch}` - UI binary
- Combined into `.tar.gz` archive

**Windows**:
- `memofy-core-windows-{arch}.exe` - Core binary
- `memofy-ui-windows-{arch}.exe` - UI binary
- Combined into `.zip` archive

### Pre-Release Detection

Workflow automatically detects pre-releases by checking tag content:

```
if tag contains: rc, beta, or alpha
  → Mark as "Pre-release" on GitHub
else
  → Mark as "Latest Release" on GitHub
```

## Testing a Release Locally

Before pushing a tag, test the release process locally:

```bash
# Use the local release build script
./scripts/build-release.sh

# This creates the same output as GitHub Actions
# Outputs: ./dist/memofy-*.tar.gz and memofy-*.zip
```

Check:
1. All binaries are created
2. File sizes are reasonable (indicates successful compilation)
3. Archives contain both core and ui binaries

## Version Bumping Workflow

### Typical Development Cycle

```
v0.1.0 (current stable)
  ↓
v0.2.0-alpha1 (first development build)
  ↓
v0.2.0-rc1 (release candidate)
  ↓
v0.2.0 (stable release)
```

### Steps for Next Release

1. **Feature branch development** (on main or feature branch)
   ```bash
   git checkout -b feature/my-feature
   # ... make changes ...
   git commit -m "Add my feature"
   ```

2. **Create pre-release for testing**
   ```bash
   git tag v0.2.0-rc1
   git push origin main v0.2.0-rc1
   # Users with allow_dev_updates: true are notified
   ```

3. **Testing phase**
   - Users test the RC version
   - Bugs are fixed on main branch
   - Commits are cherry-picked or merged to main

4. **Create stable release**
   ```bash
   git tag v0.2.0
   git push origin main v0.2.0
   # All users are notified
   ```

## Handling Release Failures

### If GitHub Actions build fails

1. Check the [Actions tab](https://github.com/tiroq/memofy/actions)
2. View the failed workflow log
3. Fix the issue in code
4. Delete the failed tag:
   ```bash
   git tag -d v0.2.0
   git push origin :v0.2.0
   ```
5. Commit fixes and retry:
   ```bash
   # Fix the issue
   git commit -m "Fix build issue"
   git tag v0.2.0
   git push origin main v0.2.0
   ```

### If release is published but has issues

1. Delete the release on GitHub (keep the tag)
2. Fix the issue in code
3. Amend master branch:
   ```bash
   git commit --amend
   git push origin main -f  # Force push (use carefully!)
   ```
4. Delete and recreate the tag:
   ```bash
   git tag -d v0.2.0
   git push origin :v0.2.0
   git tag v0.2.0
   git push origin main v0.2.0
   ```

## User Notification

### How users are notified

1. **Stable releases** (e.g., `v0.2.0`):
   - All users notified
   - Memofy checks GitHub every hour for updates
   - Desktop notification appears

2. **Pre-releases** (e.g., `v0.2.0-rc1`):
   - Only users with `allow_dev_updates: true` are notified
   - Useful for controlled rollout

3. **Auto-update process**:
   - User sees notification: "New version available: v0.2.0"
   - User can click "Update Now" in menu bar
   - Binary is downloaded and installed
   - Memofy suggests restart

## Monitoring Releases

### Check Release Status

```bash
# View all releases
git tag -l

# View latest release
git describe --tags --abbrev=0

# View history
git log --oneline --graph --tags
```

### GitHub Release Page

Visit: https://github.com/tiroq/memofy/releases

Shows:
- All releases with download counts
- Which is "Latest Release"
- Which are marked "Pre-release"
- Release notes and assets

## Release Checklist

Before publishing a release:

- [ ] All tests passing (CI workflow green)
- [ ] Code reviewed
- [ ] Version bumped in relevant files (if needed)
- [ ] Changes committed to main branch
- [ ] Release notes documented (in GitHub Release)
- [ ] Tag created: `git tag vX.Y.Z`
- [ ] Tag pushed: `git push origin main vX.Y.Z`
- [ ] GitHub Actions workflow completed
- [ ] Release appears on GitHub Releases page
- [ ] All platform artifacts are present
- [ ] Test on at least one platform (download and run)

## Release Notes Template

When creating a GitHub Release, include:

```markdown
# v0.2.0 - Release Name/Date

## What's New

- Feature 1 description
- Feature 2 description
- Improvement 1 description

## Bug Fixes

- Bug 1 fixed
- Bug 2 fixed

## Breaking Changes

None (or describe if applicable)

## Installation

Download the binary for your platform:
- macOS Intel: `memofy-darwin-amd64.tar.gz`
- macOS Apple Silicon: `memofy-darwin-arm64.tar.gz`
- Linux: `memofy-linux-amd64.tar.gz`
- Windows: `memofy-windows-amd64.zip`

See [Installation Guide](https://github.com/tiroq/memofy/blob/main/INSTALLATION_GUIDE.md).

## Known Issues

- Known issue 1
- Known issue 2

## Contributors

- @contributor1
- @contributor2

---

**Auto-update**: Memofy will notify users of this release automatically. Users with `allow_dev_updates: false` (default) will see stable releases only.
```

## Troubleshooting Release Issues

### Build fails for specific platform

1. Check the workflow log for the failing build
2. Common issues:
   - Missing Go version
   - Missing dependencies
   - CGO issues on macOS
   - ARM64 build issues on Linux

### Release is marked as pre-release when it shouldn't be

Ensure tag doesn't contain `rc`, `beta`, or `alpha`:
- ✅ `v0.2.0` - Stable
- ❌ `v0.2.0.rc1` - Will be detected as pre-release (use dash, not dot)
- ✅ `v0.2.0-rc1` - Pre-release

### Users not seeing the update notification

1. Check GitHub Release was created successfully
2. Verify release is marked "Latest Release" (for stable)
3. Users need to restart Memofy to check for updates
4. Update check runs every hour

## Advanced: Manual Release (Without GitHub Actions)

If needed, create releases manually:

```bash
# Build locally
./scripts/build-release.sh

# Create GitHub Release manually
gh release create v0.2.0 dist/* --draft

# Or visit: https://github.com/tiroq/memofy/releases/new
# Upload files and publish
```

But GitHub Actions automation is recommended for consistency.

## Questions?

- Review GitHub Actions workflow: `.github/workflows/release.yml`
- Check Memofy documentation: [Installation Guide](../INSTALLATION_GUIDE.md)
- See release channel documentation: [Release Channel Configuration](../docs/RELEASE_CHANNEL_CONFIGURATION.md)
