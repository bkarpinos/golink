# GoLink - Personal URL Shortener
> Warning! This project was developed with the help of Claude 3.7 Sonnet

GoLink is a lightweight, personal URL shortening service that allows you to create and manage your own collection of short links. Instead of remembering long URLs, you can create intuitive aliases like `go/docs` or `go/meeting` that redirect to their respective destinations.

## 🚀 Features

- **Simple CLI**: Easily manage your links from the terminal
- **Fast Redirects**: Minimal latency for quick navigation
- **Categorization**: Organize links by categories
- **Local Storage**: All your links stored locally in a JSON file
- **Modern Web Interface**: Clean UI to browse and manage your links

## 📋 Installation

### Prerequisites
- Go 1.19 or higher

### Installation Steps

1. Clone the repository
   ```
   git clone https://github.com/yourusername/golink.git
   cd golink
   ```

2. Build the binary
   ```
   go build
   ```

3. Install the binary (optional)
   ```
   go install
   ```

## 🔧 Getting Started

### Starting the Server

Start the GoLink server:

```bash
# Start on default port (8080)
golink serve

# Start on a specific port
golink serve --port 8080

> Use port 80 to avoid adding a port to all of the following links

# Specify a URL to redirect to when links aren't found
golink serve --not-found https://google.com
```

### Managing Links

```bash
# Add a new link
golink add gh https://github.com/{username} --description "My GitHub Profile" --category "dev"

# List all links
golink list

# Delete a link
golink delete gh
```

### Accessing Links

Once the server is running, you can access your links in a web browser:

- Visit `http://localhost/` to see all your links
- Use `http://localhost/{alias}` to be redirected (e.g., `http://localhost/gh`)
- View service information at `http://localhost/info`

You can also open a link directly from the terminal:
```bash
# Open using go/alias
golink open gh

# Open using the direct url
golink open gh --direct
```

## 💻 Development Guide

### Project Structure

```
golink/
├── cmd/              # Command line interface definitions
├── internal/         # Internal packages
│   ├── link/         # Link data structure
│   ├── server/       # HTTP server implementation
│   └── storage/      # Storage implementations
├── main.go           # Application entry point
└── README.md         # Documentation
```

### Running in Development Mode

For development, you can run the application directly without building:

```bash
# Run server
go run main.go serve

# Execute commands
go run main.go add example https://example.com
go run main.go list
```


## Local Setup (Optional - for `go/{alias}` style URLs)

For a more streamlined local experience, you can configure your system to resolve URLs like `go/gh` to `http://localhost/gh`. This allows you to use short, convenient aliases for your local Go links server.

**Instructions (macOS):**

1.  **Edit the Hosts File:**
    * Add the following line to your `/etc/hosts` file: `127.0.0.1 go`

2.  **Usage:**
    * Now, you can access your Go links server using `http://go/{alias}` in your browser. For example, `http://go/gh` will resolve to `http://localhost/gh`.
    * **Important:** This method only maps the base hostname `go`. For full `go/{alias}` functionality, see the browser extension setup below.


## 🧩 Browser Extension Redirect Setup

For a true `go/alias` experience in your browser, you can use a redirect browser extension:

### Using Redirector Extension

1. **Install the Extension**:
    - [Redirector for Chrome](https://chrome.google.com/webstore/detail/redirector/ocgpenflpmgnfapjedencafcfakcekcd)
    - [Redirector for Firefox](https://addons.mozilla.org/en-US/firefox/addon/redirector/)

2. **Configure a Redirect Rule**:
    - Open extension settings
    - Add a new redirect with these settings:
      - Description: `GoLink Redirector`
      - Example URL: `http://go/docs`
      - Include pattern: `^http://go/(.*)$`
      - Redirect to: `http://go/$1` or if you didn't update the `/etc/hosts` file `http://localhost/$1`
      - Pattern type: `Regular Expression`

3. **Usage**:
    - Simply type `go/docs` (or any alias) in your browser's address bar
    - The extension will redirect to your local GoLink server
    - You must first open the link using `http://go/{alias}` for each link before the browser will recognize this as a valid path. (You can use the `open` command to do this quickly).


## 📊 Data Storage

Your links are by default stored in a JSON file at:
- `~/.config/golink/links.json`

You can back up this file to preserve your links.

## ⚙︎ Configuration Management

GoLink provides tools to manage your configuration through the command line.

### Default Configuration

By default, GoLink stores:
- Configuration file: `~/.config/golink/config.yaml`
- Link database: `~/.config/golink/links.json`

### Viewing Configuration

View your current configuration settings:

```bash
golink config view
```

This displays:
- Config directory location
- Storage directory location (where links are stored)
- Loaded config file
- All configuration settings

### Setting Custom Storage Location

You can store your links in a different directory:

```bash
# Set a custom storage directory
golink config storage-dir ~/Documents/my-links

# View the change (requires restart to take effect)
golink config view
```

### Using Environment Variables

GoLink supports environment variable configuration for all settings. Variables are automatically mapped from your config keys:

```bash
# Override storage directory temporarily
STORAGE_DIR=/tmp/links golink list
```

### Configuration Precedence

Settings are applied in the following order (highest priority first):
1. Environment variables
2. Configuration file
3. Default values

