# Memofy Release & Update System - Quick Reference

## For End Users

### Checking for Updates

**Automatic Checking**:
- Memofy checks for updates every hour automatically
- You'll see a notification if an update is available

**Manual Checking**:
1. Click the Memofy menu bar icon
2. Select "Check for Updates"
3. Result appears in a dialog

### Installing Updates

**One-Click Update**:
1. Click Memofy menu bar icon
2. Select "Update Now"
3. Memofy downloads and installs the update
4. A notification asks you to restart

**Manual Update**:
```bash
./scripts/quick-install.sh
```

### Choosing Release Channel

**Stable Releases Only** (Default - Recommended):
- Your config is already set correctly
- You receive notifications about v0.2.0, v0.3.0, etc.
- You do NOT receive notifications about v0.2.0-rc1

**Include Pre-Releases** (For Testing New Features):
1. Edit `~/.config/memofy/detection-rules.json`
2. Find the line: `"allow_dev_updates": false`
3. Change to: `"allow_dev_updates": true`
4. Save the file
5. Restart Memofy
6. Now you'll get notifications about v0.2.0-rc1, v0.2.0-beta1, etc.

**Configuration File Location**:
```bash
~/.config/memofy/detection-rules.json
```

### Understanding Release Types

| Release | Example | When? | For Whom? |
|---------|---------|--------|-----------|
| **Stable** | v0.2.0 | Final, thoroughly tested | Everyone |
| **RC** | v0.2.0-rc1 | Ready for release, needs testing | Testers |
| **Beta** | v0.2.0-beta1 | Nearly complete, active development | Enthusiasts |
| **Alpha** | v0.2.0-alpha1 | Early development | Developers |

### Troubleshooting Updates

**Not seeing update notifications?**
1. Ensure `allow_dev_updates` matches your preference
2. Wait up to 1 hour (updates check hourly)
3. Click "Check for Updates" manually
4. Restart Memofy to reload configuration

**Update won't install?**
1. Check disk space: `df -h`
2. Verify folder is writable: `ls -la ~/.local/bin/`
3. Check error notification for details
4. Try manual install: `./scripts/quick-install.sh`

**Old version still running after update?**
- Restart Memofy completely
- Check which version is running: `memofy-core --version`

---

## For Maintainers / Developers

### Quick Release Process

**1. Create and publish a release**:

```bash
# Example: Release v0.2.0
git tag v0.2.0
git push origin main v0.2.0
```

**That's it!** GitHub Actions will:
- Automatically detect the tag
- Build for all 6 platforms
- Create a GitHub Release
- Upload all artifacts

**Workflow location**: `.github/workflows/release.yml`

### Release Types

**Stable Release**:
```bash
git tag v0.2.0
git push origin main v0.2.0
# Result: GitHub marks as "Latest Release"
# Users see: All stable users notified
```

**Pre-Release (Release Candidate)**:
```bash
git tag v0.2.0-rc1
git push origin main v0.2.0-rc1
# Result: GitHub marks as "Pre-release"
# Users see: Only those with allow_dev_updates: true are notified
```

**Pre-Release (Beta)**:
```bash
git tag v0.2.0-beta1
git push origin main v0.2.0-beta1
# Result: GitHub marks as "Pre-release"
```

**Pre-Release (Alpha)**:
```bash
git tag v0.2.0-alpha1
git push origin main v0.2.0-alpha1
# Result: GitHub marks as "Pre-release"
```

### Checking Release Status

**View all tags**:
```bash
git tag -l
```

**View releases on GitHub**:
```
https://github.com/tiroq/memofy/releases
```

**Check workflow status**:
```
https://github.com/tiroq/memofy/actions
```

### Testing Releases Locally

**Before pushing to GitHub**:

```bash
# Build locally (same process as GitHub Actions)
./scripts/build-release.sh

# Output: dist/memofy-*.tar.gz and memofy-*.zip
# Check file sizes exist (~50-100 MB each)
```

### Multi-Stage Release Process

**Typical workflow**:

```
1. Code development on main branch
2. Tag v0.2.0-alpha1 → Early testing
3. Fix issues found in testing
4. Tag v0.2.0-rc1 → Release candidate (public testing)
5. Fix RC issues
6. Tag v0.2.0 → Stable release (everyone notified)
```

### Semantic Versioning

```
vMAJOR.MINOR.PATCH

Examples:
v0.1.0       Stable
v0.1.1       Patch/bug fix
v0.2.0       New features
v1.0.0       Breaking changes / Major milestone
v0.2.0-rc1   Release candidate
v0.2.0-beta1 Beta version
```

### GitHub Actions Workflows

**Files**:
- `.github/workflows/release.yml` - Builds & releases on tag push
- `.github/workflows/ci.yml` - Tests on every push/PR

**Release Workflow Status**:
1. Create tag → Pushed to GitHub
2. GitHub detects tag matching `v*`
3. Workflow starts automatically
4. Builds 6 platform combinations
5. Creates GitHub Release
6. Users notified based on their channel preference

---

## System Architecture

### How It All Fits Together

```
Developer Workflow
  ↓
git tag v0.2.0 && git push
  ↓
GitHub Actions Detected (release.yml)
  ↓
Build All Platforms
  ↓
Create GitHub Release + Upload Assets
  ↓
Memofy Auto-Update System
  ↓
Hourly Check: GitHub API → GetLatestRelease()
  ↓
Compare with local version
  ↓
Filter based on release channel (ChannelStable vs ChannelPrerelease)
  ↓
User Notification
  ↓
One-Click Update → Download → Install → Done
```

### Code Organization

```
Memofy Project
├── .github/
│   └── workflows/
│       ├── release.yml          ← Automated releases
│       └── ci.yml               ← Testing pipeline
├── internal/
│   ├── autoupdate/
│   │   └── checker.go           ← Update checking + filtering
│   └── config/
│       └── detection_rules.go   ← Config with allow_dev_updates flag
├── pkg/macui/
│   └── statusbar.go             ← Menu bar + update notifications
├── scripts/
│   ├── quick-install.sh         ← One-command install (uses releases)
│   └── build-release.sh         ← Local build (for testing)
└── docs/
    ├── RELEASE_CHANNEL_CONFIGURATION.md     ← User guide
    ├── RELEASE_PROCESS_GUIDE.md             ← Maintainer guide
    ├── AUTO_UPDATE_SYSTEM.md                ← Technical details
    └── RELEASE_CHANNEL_IMPLEMENTATION.md    ← Implementation summary
```

### Release Channels in Code

```go
// ReleaseChannel - which releases to offer users
type ReleaseChannel string

const (
    ChannelStable     = "stable"      // v0.2.0
    ChannelPrerelease = "prerelease"  // v0.2.0-rc1, v0.2.0-beta1
    ChannelDev        = "dev"         // All releases (not used yet)
)
```

### Configuration

```json
{
    "allow_dev_updates": false  // false = ChannelStable
                                // true  = ChannelPrerelease
}
```

---

## Common Commands

### For Users

```bash
# Check current version
memofy-core --version

# Manually install/update
./scripts/quick-install.sh

# View configuration
cat ~/.config/memofy/detection-rules.json

# Edit configuration
open ~/.config/memofy/detection-rules.json

# View logs
tail -f ~/.cache/memofy/memofy-ui.log
```

### For Developers

```bash
# Build locally (all platforms)
./scripts/build-release.sh

# Create a release
git tag v0.2.0
git push origin main v0.2.0

# Check workflow status
open https://github.com/tiroq/memofy/actions

# View releases
open https://github.com/tiroq/memofy/releases

# Compare versions
git log v0.1.0..v0.2.0

# List all tags
git tag -l | sort -V
```

---

## GitHub Release Checklist

Before releasing, verify:

- [ ] All tests passing (CI workflow green)
- [ ] Code reviewed
- [ ] Changes committed to main branch
- [ ] Version tag created: `git tag vX.Y.Z`
- [ ] Tag follows pattern: v0.2.0 (not 0.2.0, not vX.Y.Z.rc)
- [ ] Tag pushed: `git push origin main vX.Y.Z`
- [ ] GitHub Actions build completed
- [ ] Release appears in [Release page](https://github.com/tiroq/memofy/releases)
- [ ] All platform artifacts present (6 files)
- [ ] Pre-release checkbox correct on GitHub
- [ ] Release notes updated (optional but recommended)

---

## FAQ

### Q: How do I get the latest features?
**A**: Set `allow_dev_updates: true` in your config, and you'll get pre-releases like v0.2.0-rc1 before the stable v0.2.0.

### Q: What if there's a bug in a pre-release I installed?
**A**: Report it on GitHub. Or revert by setting `allow_dev_updates: false` and waiting for the stable release.

### Q: How do I know what changed in a release?
**A**: Check the GitHub Releases page: https://github.com/tiroq/memofy/releases

### Q: Can I use older versions?
**A**: Yes, but you must download from GitHub's Releases page and install manually.

### Q: What if GitHub is down when update check runs?
**A**: Check fails silently, you'll see it the next hour when GitHub is back up.

### Q: Can I disable update checks?
**A**: Currently no, but you can block GitHub API in your firewall if needed.

### Q: How do I roll back to a previous version?
**A**: Download the previous version from GitHub Releases and run `quick-install.sh`.

### Q: What's the difference between rc, beta, and alpha?
**A**: Release candidate (rc) = nearly ready, beta = mostly ready, alpha = early dev. All are marked "pre-release" in GitHub.

---

## Documentation Index

**For Users**:
- [Release Channel Configuration](docs/RELEASE_CHANNEL_CONFIGURATION.md)
- [Auto-Update System](docs/AUTO_UPDATE_SYSTEM.md)

**For Maintainers**:
- [Release Process Guide](docs/RELEASE_PROCESS_GUIDE.md)
- [Release Channel Implementation](docs/RELEASE_CHANNEL_IMPLEMENTATION.md) (technical)

**Installation**:
- [Installation Guide](INSTALLATION_GUIDE.md)
- [Quick Install Implementation](QUICK_INSTALL_IMPLEMENTATION.md)

---

## Support

**Issues with auto-updates?**
- Check logs: `tail -f ~/.cache/memofy/*.log`
- Verify config: `cat ~/.config/memofy/detection-rules.json`
- Check GitHub status: https://www.githubstatus.com/

**Found a bug?**
- Report on GitHub: https://github.com/tiroq/memofy/issues

---

*Last Updated: 2024*
*Memofy v0.2.0 Release Channel System*
