# cTAKES TUI

A Terminal User Interface for Apache cTAKES (clinical Text Analysis and Knowledge Extraction System), built with Go and Charm libraries.

## Features

- 📄 **Document Processing** - Process clinical documents in batch
- 🔍 **Text Analysis** - Analyze clinical text in real-time
- ⚙️ **Pipeline Configuration** - Configure cTAKES processing pipeline components
- 📊 **Results Viewer** - View and export analysis results
- 🗂️ **Dictionary Management** - Manage medical dictionaries and vocabularies
- 🎨 **Beautiful TUI** - Modern terminal interface powered by Charm's Bubble Tea

## Prerequisites

- Go 1.18 or higher
- Apache cTAKES (to be integrated)

## Installation

```bash
git clone https://github.com/yourusername/ctakes-tui
cd ctakes-tui
go build
```

## Usage

```bash
./ctakes-tui
```

### Navigation

- `↑/↓` or `j/k` - Navigate menu items
- `Enter` or `Space` - Select item
- `Esc` - Go back to main menu
- `q` or `Ctrl+C` - Quit application

## Architecture

The application is built with:
- **Bubble Tea** - Terminal UI framework
- **Lipgloss** - Styling library
- **Bubbles** - TUI components

## Project Structure

```
ctakes-tui/
├── main.go           # Main application entry point
├── views/            # UI views
│   ├── document.go   # Document processing view
│   ├── analyze.go    # Text analysis view
│   └── pipeline.go   # Pipeline configuration view
├── go.mod            # Go module dependencies
└── README.md         # This file
```

## cTAKES Integration (Planned)

The integration with Apache cTAKES will include:
- Java process management for cTAKES runtime
- REST API client for cTAKES services
- File-based processing pipeline
- Real-time text analysis
- Results caching and export

## Development

To contribute to this project:

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Submit a pull request

## License

[To be determined]

## Acknowledgments

- [Apache cTAKES](https://ctakes.apache.org/) - Clinical NLP system
- [Charm](https://charm.sh/) - Terminal UI libraries
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) - TUI framework