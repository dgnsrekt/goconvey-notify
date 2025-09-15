# GoConvey-Notify

**Enhanced GoConvey with Audio & Push Notifications**

[![Build Status](https://app.travis-ci.com/smartystreets/goconvey.svg?branch=master)](https://app.travis-ci.com/smartystreets/goconvey)
[![GoDoc](https://godoc.org/github.com/smartystreets/goconvey?status.svg)](http://godoc.org/github.com/smartystreets/goconvey)

A fork of [GoConvey](https://github.com/smartystreets/goconvey) that adds intelligent audio alerts and push notifications to enhance your Go testing workflow.

## What's New in GoConvey-Notify

- **Dual Sound Alerts**: Play different sounds for test success vs failure
- **NTFY Push Notifications**: Real-time notifications to your mobile/desktop via [NTFY](https://ntfy.sh)
- **Smart Auto-Detection**: Automatically detects valid configurations without explicit enable flags
- **Backward Compatible**: Works seamlessly with existing GoConvey setups
- **JSON Configuration**: Simple file-based configuration with flexible options

## Quick Start

### Installation

```bash
go install github.com/dgnsrekt/goconvey-notify@latest
```

### Basic Setup

1. **Create a configuration file** (`goconvey-notifications.json`):

```json
{
    "sound": {
        "success_file_path": "success.mp3",
        "failure_file_path": "failure.mp3"
    },
    "ntfy": {
        "server": "https://ntfy.sh",
        "topic": "my-go-tests",
        "timeout": 30
    }
}
```

2. **Run GoConvey with notifications**:

```bash
goconvey-notify --config goconvey-notifications.json
```

3. **Enable notifications in the web UI** by toggling the Sound and NTFY switches in the settings panel.

That's it! Now you'll get:
- Different sounds for passing vs failing tests
- Push notifications on your devices via NTFY

## Configuration Options

### Sound Configuration

```json
{
    "sound": {
        "success_file_path": "path/to/success.mp3",
        "failure_file_path": "path/to/failure.mp3"
    }
}
```

**Supported formats**: `.mp3`, `.wav`, `.ogg`, `.m4a`, `.webm`

**Smart behavior**:
- Success sound plays when all tests pass
- Failure sound plays when any test fails, panics, or build fails
- Falls back to single sound if only one file is configured

### NTFY Configuration

```json
{
    "ntfy": {
        "server": "https://ntfy.sh",
        "topic": "your-unique-topic",
        "timeout": 30,
        "auth_header": "Bearer your-token"
    }
}
```

**Options**:
- `server`: NTFY server URL (default: `https://ntfy.sh`)
- `topic`: Unique topic name (letters, numbers, `_`, `-` only)
- `timeout`: Request timeout in seconds (default: 30)
- `auth_header`: Optional authentication header

### Legacy Support

For backward compatibility, you can still use the old format:

```json
{
    "sound": {
        "file_path": "notification.mp3"
    }
}
```

This single file will be used for both success and failure.

## Usage Examples

### Minimal Setup (Sound Only)

```json
{
    "sound": {
        "success_file_path": "ding.mp3",
        "failure_file_path": "buzz.mp3"
    }
}
```

### NTFY with Custom Server

```json
{
    "ntfy": {
        "server": "https://your-ntfy-server.com",
        "topic": "ci-alerts",
        "timeout": 15,
        "auth_header": "Bearer sk-1234567890"
    }
}
```

### Full Configuration

```json
{
    "sound": {
        "success_file_path": "assets/success.wav",
        "failure_file_path": "assets/failure.wav"
    },
    "ntfy": {
        "server": "https://ntfy.sh",
        "topic": "goconvey-dgnsrekt",
        "timeout": 30
    }
}
```

### Different Config File

```bash
goconvey-notify --config my-custom-config.json --port 9000
```

## Core GoConvey Features

This fork maintains full compatibility with GoConvey's core testing features. For comprehensive documentation on:

- **Writing tests with GoConvey syntax**
- **Web UI features and navigation**
- **Integration with `go test`**
- **Test coverage and reporting**

Please refer to the [original GoConvey documentation](https://github.com/smartystreets/goconvey/wiki).

## API Reference

GoConvey-Notify adds several new endpoints:

### Configuration Status
```
GET /config-status
```
Returns JSON indicating which features are configured:
```json
{
    "soundConfigured": true,
    "successSoundConfigured": true,
    "failureSoundConfigured": true,
    "ntfyConfigured": true
}
```

### Sound File Endpoints
```
GET /sound-file/success  # Serves success sound file
GET /sound-file/failure  # Serves failure sound file
GET /sound-file          # Legacy single sound file
```

### NTFY Notifications
```
POST /ntfy
Content-Type: application/x-www-form-urlencoded

title=Test Results&body=5 passed, 2 failed
```

## Command Line Options

All original GoConvey options are supported, plus:

```bash
goconvey-notify [options]

Notification Options:
  --config string    Path to notification configuration file (default "goconvey-notifications.json")

Standard GoConvey Options:
  --port int         Port for web server (default 8080)
  --host string      Host for web server (default "127.0.0.1")
  --cover            Enable test coverage (default true)
  --depth int        Directory scanning depth (default -1)
  --packages int     Parallel package testing (default 10)
  --poll duration    File system polling interval (default 250ms)
```

## 🎯 Use Cases

- **🔊 Audio Feedback**: Get immediate audio cues while coding without watching the screen
- **📱 Remote Monitoring**: Receive push notifications on your phone for CI/CD pipelines
- **👥 Team Alerts**: Share test results with team members via NTFY topics
- **🎮 Gamification**: Use fun sounds to make testing more engaging
- **♿ Accessibility**: Audio cues for developers with visual impairments

## 🤝 Contributing

This project builds on the excellent foundation of [GoConvey by SmartyStreets](https://github.com/smartystreets/goconvey).

For issues specific to the notification features, please use this repository's issue tracker. For core GoConvey functionality, consider contributing to the [original project](https://github.com/smartystreets/goconvey).

## 📄 License

This project maintains the same license as the original GoConvey project. See [LICENSE.md](LICENSE.md) for details.

---

**Credits**: GoConvey-Notify is built on [GoConvey](https://github.com/smartystreets/goconvey) by [SmartyStreets](https://github.com/smartystreets) and [contributors](https://github.com/smartystreets/goconvey/graphs/contributors). Notification features added by [@dgnsrekt](https://github.com/dgnsrekt).