# Release Process

## Quick Release

```bash
task release-{major|minor|patch|alpha|beta|rc|stable}
```

Auto-runs: tests, lint, git check, tag creation, CI verification

## Manual

```bash
make test && make lint
git tag v0.2.0
git push origin main v0.2.0
gh run watch
```

## Versioning

- **v0.2.0** - Stable
- **v0.2.0-rc1** - Release candidate
- **v0.2.0-beta1** - Beta
- **v0.2.0-alpha1** - Alpha

**Breaking**: Bump major | **Features**: Bump minor | **Fixes**: Bump patch

## Build Matrix

macOS/Linux/Windows × amd64/arm64 → `.tar.gz` / `.zip`

## Checklist

- [ ] Tests pass
- [ ] Linter clean
- [ ] Git clean
- [ ] On main branch
