# Taskfile Usage Guide

This project uses [Task](https://taskfile.dev/) as a modern task runner and build tool. Task is an alternative to Make with better cross-platform support and simpler syntax.

## Installation

### macOS
```bash
brew install go-task/tap/go-task
```

### Linux
```bash
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b ~/.local/bin
```

### Windows
```bash
choco install go-task
```

Or download from [GitHub Releases](https://github.com/go-task/task/releases).

## Quick Start

```bash
# List all available tasks
task

# Build both binaries
task build

# Install to ~/.local/bin
task install

# Run tests
task test
```

## Common Tasks

### Development

```bash
# Build core daemon
task build-core

# Build menu bar UI
task build-ui

# Run daemon in foreground (for debugging)
task dev-daemon

# Run UI in foreground (for debugging)
task dev-ui

# Format code
task fmt

# Run linter
task lint

# Run all tests
task test

# Run integration tests only
task test-integration

# Generate coverage report
task test-coverage
```

### Installation & Management

```bash
# Install binaries and LaunchAgent
task install

# Uninstall
task uninstall

# Check status
task status

# Restart daemon
task restart

# Stop daemon
task stop

# Start daemon
task start

# View logs
task logs          # Daemon output logs
task logs-ui       # UI logs
task logs-error    # Daemon error logs
```

### Release Management

#### Quick Release Commands

```bash
# Create stable release (v0.2.0)
task release-stable VERSION=0.2.0

# Create release candidate (v0.2.0-rc1)
task release-rc VERSION=0.2.0 RC=1

# Create beta release (v0.2.0-beta1)
task release-beta VERSION=0.2.0 BETA=1

# Create alpha release (v0.2.0-alpha1)
task release-alpha VERSION=0.2.0 ALPHA=1
```

#### Release Workflow

**1. Check if ready for release:**
```bash
task release-check
```
This verifies:
- ✅ All tests pass
- ✅ Linter passes
- ✅ Git working directory is clean
- ✅ On main branch (recommended)

**2. Create and push release tag:**

Option A - Stable release:
```bash
task release-stable VERSION=0.2.0
```

Option B - Pre-release:
```bash
task release-rc VERSION=0.2.0 RC=1
```

**3. GitHub Actions automatically:**
- Detects tag push
- Builds all 6 platforms
- Creates GitHub Release
- Uploads artifacts

**4. Monitor release:**
```bash
task release-status
```

Or visit: https://github.com/tiroq/memofy/actions

#### Advanced Release Tasks

```bash
# Manual tag creation (with any format)
task release-tag VERSION=v0.2.0

# Build releases locally (for testing)
task release-local

# List all tags
task release-list

# Delete a tag (local and remote)
task release-delete VERSION=v0.2.0
```

### Dependencies

```bash
# Download dependencies
task deps

# Update all dependencies
task deps-update

# Vendor dependencies
task deps-vendor
```

### Cleanup

```bash
# Clean build artifacts
task clean
```

### CI/CD Simulation

```bash
# Simulate CI test workflow
task ci-test

# Simulate CI build workflow
task ci-build
```

## Release Examples

### Example 1: Stable Release

```bash
# Ensure everything is ready
task release-check

# Create and push v0.2.0 stable release
task release-stable VERSION=0.2.0

# Output:
# → Running tests...
# → Running linter...
# → Checking git status...
# → Creating tag v0.2.0...
# → Pushing tag to GitHub...
# ✓ Tag v0.2.0 created and pushed
# ✓ GitHub Actions will now build and publish the release
# → View workflow: https://github.com/tiroq/memofy/actions
```

### Example 2: Release Candidate

```bash
# Create release candidate v0.2.0-rc1
task release-rc VERSION=0.2.0 RC=1

# This creates and pushes tag: v0.2.0-rc1
# GitHub marks it as "Pre-release"
# Only users with allow_dev_updates: true are notified
```

### Example 3: Beta Testing

```bash
# First beta
task release-beta VERSION=0.2.0 BETA=1

# Second beta (after fixes)
task release-beta VERSION=0.2.0 BETA=2

# Release candidate
task release-rc VERSION=0.2.0 RC=1

# Final stable
task release-stable VERSION=0.2.0
```

### Example 4: Fix Bad Release

```bash
# Delete the bad tag
task release-delete VERSION=v0.2.0

# Fix the code issue
git commit -m "Fix release issue"

# Re-create the release
task release-stable VERSION=0.2.0
```

## Task Features

### Automatic Dependency Resolution

Tasks automatically run dependencies when needed:

```bash
# This runs release-check first
task release-tag VERSION=v0.2.0

# This builds binaries first
task install
```

### Incremental Builds

Task tracks file changes and only rebuilds when sources change:

```bash
# First build
task build-core  # Compiles

# No changes
task build-core  # Skipped (up to date)

# After editing internal/detector/zoom.go
task build-core  # Recompiles
```

### Variables

You can override variables:

```bash
# Custom install directory
INSTALL_DIR=/usr/local/bin task install

# Custom version
VERSION=v0.3.0-custom task build
```

## Comparison with Make

| Feature | Task | Make |
|---------|------|------|
| Cross-platform | ✅ Native | ⚠️ Requires GNU Make on Windows |
| Syntax | YAML (simple) | Makefile syntax (complex) |
| Incremental builds | ✅ Built-in | Manual implementation |
| Dependencies | ✅ Automatic | Manual |
| Variables | ✅ Simple | Complex |
| Preconditions | ✅ Built-in | Manual |
| Error messages | ✅ Clear | Cryptic |

## Troubleshooting

### Task not found

```bash
# Check installation
which task

# Install if missing
brew install go-task/tap/go-task  # macOS
```

### Permission denied

```bash
# Make sure Task is executable
chmod +x $(which task)
```

### Version mismatch

```bash
# Check Task version (requires v3+)
task --version

# Update Task
brew upgrade go-task  # macOS
```

## Advanced Usage

### Parallel Execution

Some tasks can run in parallel:

```bash
# Run tests and linter in parallel
task test & task lint & wait
```

### Silent Mode

```bash
# Run without output
task --silent build
```

### Dry Run

```bash
# See what would be executed
task --dry build
```

### List All Tasks

```bash
# Show all tasks with descriptions
task --list

# Shorter version
task -l
```

## Integration with IDEs

### VS Code

Install the [Task extension](https://marketplace.visualstudio.com/items?itemName=task.vscode-task):

```bash
code --install-extension task.vscode-task
```

Then run tasks from Command Palette: `Tasks: Run Task`

### Other IDEs

Most IDEs support running shell commands, so you can create run configurations for common tasks:

- **Build**: `task build`
- **Test**: `task test`
- **Install**: `task install`

## More Information

- [Task Documentation](https://taskfile.dev/)
- [Task GitHub Repository](https://github.com/go-task/task)
- [Taskfile Schema](https://taskfile.dev/api/)

## Common Workflows

### Daily Development

```bash
# Morning: Update dependencies
task deps-update

# Work on code...
# (edit files)

# Test changes
task test

# Lint code
task lint

# Build and install
task build
task install

# Check it works
task status
task logs
```

### Release Day

```bash
# 1. Final checks
task release-check

# 2. Create release
task release-stable VERSION=0.2.0

# 3. Monitor GitHub Actions
task release-status

# Or visit:
open https://github.com/tiroq/memofy/actions

# 4. Verify release published
open https://github.com/tiroq/memofy/releases
```

### Debugging

```bash
# Run daemon in foreground
task dev-daemon

# In another terminal, watch logs
task logs

# In another terminal, run UI
task dev-ui

# Make changes, rebuild
task build

# Restart to test
task restart
```
