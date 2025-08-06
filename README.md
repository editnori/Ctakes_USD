# cTAKES TUI

Terminal interface for Apache cTAKES clinical text analysis.

## Quick Start

```bash
git clone https://github.com/yourusername/ctakes-tui
cd ctakes-tui
make build
./ctakes-tui
```

## What It Does

• Process clinical documents in batch
• Analyze medical text in real-time
• Configure cTAKES processing pipelines
• Export analysis results in multiple formats
• Manage medical dictionaries and vocabularies

## Requirements

• Go 1.18+
• Apache cTAKES (pending integration)
• 256-color terminal

## Navigation

• `↑/↓` or `j/k` - Move through items
• `Enter` or `Space` - Select
• `Tab` - Switch panels
• `Esc` - Back
• `q` - Quit

## Project Structure

```
ctakes-tui/
├── main.go                 # Entry point
├── internal/
│   ├── config/            # Configuration management
│   ├── ctakes/            # cTAKES integration
│   ├── theme/             # UI theming
│   └── utils/             # Shared utilities
├── views/
│   ├── dashboard/         # Main dashboard view
│   ├── document.go        # Document processing
│   ├── analyze.go         # Text analysis
│   └── pipeline.go        # Pipeline configuration
└── docs/                  # Documentation
```

## Build Commands

```bash
make build      # Build the binary
make run        # Build and run
make test       # Run tests
make clean      # Clean build artifacts
make release    # Build for all platforms
```

## Current Status

Working:
• File browser with preview
• System monitor
• Document selection
• Basic UI navigation

In Progress:
• cTAKES integration
• Actual text analysis
• Pipeline configuration

## Contributing

Fork, branch, code, test, PR.

Keep it simple. Follow existing patterns.

## License

Apache 2.0