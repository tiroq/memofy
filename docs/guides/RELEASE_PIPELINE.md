# Release Pipeline Guide

Quick reference for creating releases using the automated Taskfile workflow.

## 🚀 Quick Start

All release tasks are **automated** — just run the task, no version numbers needed!

```bash
# Patch release (0.2.0 → 0.2.1)
task release-patch

# Minor release (0.2.0 → 0.3.0)
task release-minor

# Major release (0.2.0 → 1.0.0)
task release-major
```

## 📋 Release Workflow

### Standard Release Cycle

1. **Development Phase**
   - Make changes and commit to `main`
   - Run tests: `task test`
   - Verify build: `task build`

2. **Pre-release Testing** (Optional)
   ```bash
   # Create alpha release for early testing
   task release-alpha    # Creates v0.3.0-alpha1, v0.3.0-alpha2, etc.
   
   # Create beta release for broader testing
   task release-beta     # Creates v0.3.0-beta1, v0.3.0-beta2, etc.
   
   # Create release candidate
   task release-rc       # Creates v0.3.0-rc1, v0.3.0-rc2, etc.
   ```

3. **Stable Release**
   ```bash
   # Option A: Auto-promote latest pre-release to stable
   task release-stable   # v0.3.0-rc2 → v0.3.0
   
   # Option B: Bump version directly
   task release-patch    # Patch: v0.2.0 → v0.2.1
   task release-minor    # Minor: v0.2.1 → v0.3.0
   task release-major    # Major: v0.3.0 → v1.0.0
   ```

4. **Automated Process**
   - Task runs `release-check` (tests, lint, git status)
   - Creates and pushes git tag
   - Triggers GitHub Actions workflow
   - Automatically verifies workflow completion
   - Builds and publishes release artifacts

## 🎯 Release Types

### Semantic Versioning

Format: `vMAJOR.MINOR.PATCH[-PRERELEASE]`

| Type | Command | Example | When to Use |
|------|---------|---------|-------------|
| **Major** | `task release-major` | v1.0.0 → v2.0.0 | Breaking changes |
| **Minor** | `task release-minor` | v0.2.0 → v0.3.0 | New features |
| **Patch** | `task release-patch` | v0.2.0 → v0.2.1 | Bug fixes |
| **Alpha** | `task release-alpha` | v0.3.0-alpha1 | Early testing |
| **Beta** | `task release-beta` | v0.3.0-beta1 | Broader testing |
| **RC** | `task release-rc` | v0.3.0-rc1 | Release candidate |
| **Stable** | `task release-stable` | v0.3.0-rc2 → v0.3.0 | Promote to stable |

### Pre-release Progression

Typical flow: `alpha` → `beta` → `rc` → `stable`

```bash
task release-alpha    # v0.3.0-alpha1
# ... test and fix ...
task release-alpha    # v0.3.0-alpha2 (auto-increments)

task release-beta     # v0.3.0-beta1
# ... broader testing ...
task release-beta     # v0.3.0-beta2 (auto-increments)

task release-rc       # v0.3.0-rc1
# ... final validation ...

task release-stable   # v0.3.0 (promotes rc to stable)
```

## 🔧 Utility Tasks

```bash
# Check if ready to release
task release-check

# Verify a release workflow (auto-detects latest)
task release-verify
# Or verify specific version
task release-verify VERSION=v0.2.0

# Check GitHub Actions status
task release-status

# List all releases
task release-list

# Delete a release (manual)
task release-delete VERSION=v0.2.0

# Build locally for testing (no publish)
task release-local          # macOS only
task release-local-all      # All platforms
```

## ⚙️ How It Works

### Automated Version Detection

All tasks automatically determine the next version:

- **release-patch/minor/major**: Reads latest stable tag, increments appropriately
- **release-alpha/beta/rc**: Finds latest tag for that type, increments counter
- **release-stable**: Promotes latest pre-release or bumps patch if none exists

### Pre-release Checks

Before creating a tag, `release-check` validates:
- ✅ All tests pass
- ✅ Code passes linting
- ✅ Git working directory is clean
- ✅ Currently on `main` branch (warns if not)

### GitHub Actions Integration

When a tag is pushed:
1. GitHub Actions workflow triggers automatically
2. Builds binaries for all platforms (macOS, Linux, Windows)
3. Creates GitHub Release with artifacts
4. Task monitors and reports workflow status

## 📝 Examples

### Quick Patch Release
```bash
task release-patch
# ✓ Runs tests and lint
# ✓ Creates v0.2.1 (auto-detected)
# ✓ Pushes to GitHub
# ✓ Verifies workflow completion
```

### Pre-release Workflow
```bash
# Start with alpha
task release-alpha
# → Creates v0.3.0-alpha1

# After fixes
task release-alpha
# → Creates v0.3.0-alpha2

# Move to beta
task release-beta
# → Creates v0.3.0-beta1

# Final release
task release-stable
# → Creates v0.3.0
```

### Emergency Hotfix
```bash
# Quick patch release
task release-patch

# Or manually specify version
task release-tag VERSION=v0.2.2
```

## 🚨 Troubleshooting

**Release failed validation**
```bash
# Check what's wrong
task release-check

# Common fixes:
git add . && git commit -m "Fix"
task fmt
task lint
```

**Wrong version created**
```bash
# Delete the tag
task release-delete VERSION=v0.2.1

# Create correct one
task release-tag VERSION=v0.2.2
```

**Workflow not starting**
```bash
# Check GitHub Actions
task release-status

# Manual verification
task release-verify VERSION=v0.2.0

# Or visit GitHub Actions page
open https://github.com/tiroq/memofy/actions
```

## 📚 Related Documentation

- [Taskfile Guide](taskfile-guide.md) - Complete Taskfile reference
- [Release Process](release-process.md) - Detailed release documentation
- [Development Guide](../development/development.md) - Development workflow

## 🔗 Requirements

- **Git**: Clean working directory, on `main` branch
- **Go**: For tests and builds
- **GitHub CLI (gh)**: For workflow verification (optional)
- **golangci-lint**: For code quality checks (install with `task lint-install`)

---

**Pro Tip**: Always run `task release-check` before releasing to catch issues early!
