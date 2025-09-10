# Installation Guide for Kali Linux

## Prerequisites

- [Go](https://golang.org/dl/) 1.18 or higher
- [Ollama](https://ollama.com/download) installed and running
- The `qwen2.5-coder:1.5b` model or another code-focused model

## Quick Install for Kali Linux

### 1. Install Ollama (if not already installed)

```bash
curl -fsSL https://ollama.com/install.sh | sh
```

### 2. Pull the required model

```bash
ollama pull qwen2.5-coder:1.5b
```

### 3. Install Ollama Code

```bash
# Clone the repository
git clone https://github.com/ollama/ollama.git
cd ollama/ollama-code

# Build and install
go build -o ollama-code
sudo cp ollama-code /usr/local/bin/

# Create shortcut (recommended for Kali Linux)
sudo ln -sf /usr/local/bin/ollama-code /usr/bin/olc
```

### 4. Setup Kali Linux integration (optional but recommended)

To enhance integration with Kali Linux tools, run:

```bash
# Create configuration directory
mkdir -p ~/.ollama-code

# Create tool integration links
echo "Configuring Ollama Code for Kali Linux..."
cat > ~/.ollama-code/kali-tools.json << EOF
{
  "metasploit": "/usr/share/metasploit-framework",
  "exploitdb": "/usr/share/exploitdb",
  "wordlists": "/usr/share/wordlists",
  "nmap": "/usr/share/nmap"
}
EOF
```

## Usage in Kali Linux

```bash
# Start interactive mode
olc

# Generate a Python script using Metasploit framework
olc generate "Create a Python script that uses the Metasploit API to scan for vulnerable services"

# Explain a security tool script
olc explain /usr/share/exploitdb/exploits/linux/local/example.py

# Get help with a penetration testing script
olc debug ~/my-pentest-script.py
```

## Manual Configuration

The configuration file is stored at `~/.ollama-code/config.json`. You can modify it directly to change:

- Default model
- API URL
- Temperature and other generation parameters
- System prompts for different tasks

## Kali Linux-Specific Commands

When using Ollama Code on Kali Linux, the following additional commands are available:

```bash
# List detected security tools
olc tools list

# Generate code that uses specific security tools
olc tools generate <tool_name> "description"
```

## Uninstallation

```bash
# Remove the binary
sudo rm /usr/local/bin/ollama-code
sudo rm /usr/bin/olc

# Remove configuration files
rm -rf ~/.ollama-code
```

## Troubleshooting

### Model Not Found

If you see an error about the model not being found:

```bash
ollama pull qwen2.5-coder:1.5b
```

### API Connection Issues

Make sure Ollama is running:

```bash
# Check if Ollama is running
ps aux | grep ollama

# Start Ollama if needed
ollama serve
```

### UI Display Issues

The terminal UI requires a terminal with color support. If you're experiencing display issues, try:

```bash
# Run with simplified output
ollama-code --simple-output
```
