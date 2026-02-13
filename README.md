# Memofy - Automatic Meeting Recorder

![Memofy Logo](./docs/memofy.png)

Memofy is a macOS menu bar app that automatically detects and records Zoom, Microsoft Teams, and Google Meet meetings using OBS Studio.

---

## Features

- **Auto-Detection**: Detects meetings via process/window monitoring
- **Smart Recording**: 3/6 debounce prevents false starts
- **Menu Bar Control**: Native macOS UI with notifications
- **OBS Integration**: WebSocket control + auto-configuration
- **Auto-Update**: One-click updates from menu bar
- **Auto-Start**: LaunchAgent ensures daemon runs at login

---

## Quick Start

### Install (One Command)

```bash
curl -fsSL https://raw.githubusercontent.com/tiroq/memofy/main/scripts/quick-install.sh | bash
```

### Setup OBS

1. Install OBS: `brew install --cask obs`
2. Enable **Tools → WebSocket Server Settings** (port 4455)
3. Memofy auto-creates audio/display sources on first run

### Use

1. Grant macOS permissions (Screen Recording + Accessibility)
2. Start a Zoom/Teams/Google Meet meeting
3. Memofy auto-records and stops when meeting ends

---

## Requirements

- macOS 11.0+
- OBS Studio 28.0+
- 50MB disk space

---

## Documentation

- 📘 [Installation Guide](docs/installation.md) - Setup & troubleshooting
- 👨‍💻 [Development Guide](docs/development.md) - Build from source
- 🚀 [Release Process](docs/release-process.md) - For maintainers
- ⚙️ [Taskfile Guide](docs/taskfile-guide.md) - Task commands

---

## Contributing

Pull requests welcome! See [Development Guide](docs/development.md).

1. Fork the repo
2. Create feature branch: `git checkout -b feature/my-feature`
3. Make changes with tests
4. Run: `make test lint`
5. Submit PR

---

## License

