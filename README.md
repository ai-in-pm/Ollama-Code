# Ollama Code

A terminal-based AI coding assistant powered by Ollama models. This tool provides a Claude Code-like experience but running fully locally using Ollama models. Optimized for Kali Linux with special features for security professionals.

- Official page and updates: https://ollama.com/influencepm/ollama-code

## Features

- **Code Generation**: Create code from descriptions
- **Code Explanation**: Get detailed explanations of existing code
- **Refactoring Assistance**: Receive suggestions for code improvements
- **Debugging Help**: Analyze code for bugs and potential fixes
- **Test Generation**: Create comprehensive test cases
- **Documentation**: Generate clear documentation for code
- **Project Awareness**: Understands file relationships and project structure
- **Syntax Highlighting**: Display code with proper formatting and coloring
- **Streaming Responses**: See responses as they're generated, not just after completion
- **History Management**: Track conversation history for context
- **Kali Linux Integration**: Special commands and features optimized for security tools

## Usage

### Interactive Mode

```bash
ollama-code
# Or use the shortcut command on Kali Linux
olc
```

### Direct Commands

```bash
# Generate code
ollama-code generate "Create a Python function that sorts a list of dictionaries by a specified key"

# Explain code
ollama-code explain path/to/file.py

# Refactor code
ollama-code refactor path/to/implementation.js

# Debug code
ollama-code debug path/to/file.py

# Generate tests
ollama-code test path/to/file.js

# Generate documentation
ollama-code doc path/to/file.go
```

### Kali Linux Security Tools Integration

When running on Kali Linux, additional commands are available:

```bash
# List detected security tools
ollama-code tools list

# Generate code that integrates with specific security tools
ollama-code tools generate nmap "Scan a network and output results in JSON format"
ollama-code tools generate metasploit "Create a script that connects to the Metasploit RPC API"
```

### Special Commands in Interactive Mode

```
/generate [description] - Generate code from description
/explain [file] - Explain code in file
/refactor [file] - Suggest refactoring for code
/debug [file] - Help debug code
/test [file] - Generate tests for code
/doc [file] - Generate documentation
/model [modelname] - Change the model
/temp [value] - Change temperature (0.0-1.0)
/help - Show help
```

## Configuration

Ollama Code uses a configuration file located at `~/.ollama-code/config.json`. You can modify it directly or use the commands in interactive mode.

## Security Focus

On Kali Linux, Ollama Code is optimized with:

- Security-focused prompt templates
- Integration with common security tools
- Awareness of penetration testing workflows
- Specialized commands for security code generation

## Installation

See the installation instructions in the INSTALL.md file.

For Kali Linux-specific installation instructions, see the "Quick Install for Kali Linux" section in INSTALL.md.

For latest announcements, tips, and updates, visit the official page:

- https://ollama.com/influencepm/ollama-code
