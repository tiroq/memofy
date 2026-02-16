# Memofy Documentation

Complete documentation for Memofy - an automatic meeting recorder for macOS.

## 📚 Documentation Structure

### 🚀 [Guides](guides/)
User-facing documentation and how-to guides:
- **[Release Pipeline](guides/RELEASE_PIPELINE.md)** - Quick guide for creating releases
- **[Installation](guides/installation.md)** - Installation instructions
- **[Taskfile Guide](guides/taskfile-guide.md)** - Complete Taskfile task reference
- **[Quick Reference](guides/quick-reference.md)** - Quick command reference
- **[Release Process](guides/release-process.md)** - Detailed release documentation

### 🔧 [Development](development/)
Development guides and processes:
- **[Development Guide](development/development.md)** - Development workflow and setup
- **[Code Signing](development/CODE_SIGNING.md)** - macOS code signing setup
- **[Logging](development/logging.md)** - Logging implementation and usage
- **[Testing Plan](development/TESTING_PLAN.md)** - Testing strategy
- **[Implementation Roadmap](development/GO_IMPLEMENTATION_ROADMAP.md)** - Go implementation details
- **[Handoff](development/HANDOFF.md)** - Project handoff documentation
- **[Duplicate Prevention](development/IMPLEMENTATION_DUPLICATE_PREVENTION.md)** - Implementation details

### 🏗️ [Architecture](architecture/)
System architecture and design:
- **[Process Lifecycle](architecture/PROCESS_LIFECYCLE.md)** - Process management
- **[Startup Sequence](architecture/STARTUP_SEQUENCE.md)** - Application startup flow
- **[OBS Integration](architecture/OBS_INTEGRATION.md)** - OBS WebSocket integration
- **[Duplicate Instance Prevention](architecture/duplicate-instance-prevention.md)** - Single instance mechanism

### 🔍 [Reference](reference/)
Technical reference documentation:
- **[Index](reference/INDEX.md)** - Documentation index
- **[Tasks](reference/tasks.md)** - Available Taskfile tasks
- **[TODOs](reference/TODOs.md)** - Project TODOs and roadmap

### 🐛 [Troubleshooting](troubleshooting/)
Problem-solving guides:
- **[Troubleshooting](troubleshooting/TROUBLESHOOTING.md)** - Common issues and solutions
- **[PID File Issues](troubleshooting/PID_FILE_ISSUES.md)** - PID file related problems

### 📦 [Phase 6](phase6/)
Phase 6 implementation documentation (archived):
- **[Specification](phase6/PHASE6_SPECIFICATION.md)** - Phase 6 requirements
- **[Implementation Report](phase6/PHASE6_IMPLEMENTATION_REPORT.md)** - Implementation details
- **[Completion Summary](phase6/PHASE6_COMPLETION_SUMMARY.md)** - Phase completion summary
- **[Quick Reference](phase6/PHASE6_QUICK_REFERENCE.md)** - Phase 6 quick reference
- **[Status](phase6/PHASE6_STATUS.md)** - Phase status tracking
- **[Session Complete](phase6/PHASE6_SESSION_COMPLETE.md)** - Session completion notes

## 🎯 Quick Links

| I want to... | Go to... |
|--------------|----------|
| **Create a release** | [Release Pipeline Guide](guides/RELEASE_PIPELINE.md) |
| **Install Memofy** | [Installation Guide](guides/installation.md) |
| **Run tasks** | [Taskfile Guide](guides/taskfile-guide.md) |
| **Develop features** | [Development Guide](development/development.md) |
| **Understand architecture** | [Architecture Docs](architecture/) |
| **Fix issues** | [Troubleshooting](troubleshooting/TROUBLESHOOTING.md) |
| **See available commands** | [Tasks Reference](reference/tasks.md) |

## 🚀 Getting Started

1. **Installation**: Start with the [Installation Guide](guides/installation.md)
2. **Development**: Set up your environment using the [Development Guide](development/development.md)
3. **Build & Test**: Use [Taskfile Guide](guides/taskfile-guide.md) for common tasks
4. **Release**: Follow the [Release Pipeline](guides/RELEASE_PIPELINE.md) for releases

## 📖 Core Concepts

### Application Structure
- **memofy-core**: Background daemon that monitors meetings
- **memofy-ui**: Menu bar application for user interaction
- **OBS Integration**: WebSocket-based recording control

### Key Features
- Automatic meeting detection (Zoom, Teams, Google Meet)
- OBS integration for recording
- LaunchAgent-based auto-start
- PID-based single instance enforcement
- Menu bar controls

## 🔗 External Resources

- **Repository**: https://github.com/tiroq/memofy
- **Issues**: https://github.com/tiroq/memofy/issues
- **Releases**: https://github.com/tiroq/memofy/releases

## 🤝 Contributing

See [Development Guide](development/development.md) for development workflow and guidelines.

## 📝 License

See [LICENSE](../LICENSE) in the root directory.

---

**Last Updated**: February 2026  
**Version**: See latest git tag
