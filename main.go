package main

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/ai-in-pm/Ollama-Code/api"
	"github.com/ai-in-pm/Ollama-Code/ui"
	"github.com/spf13/cobra"
)

// Configuration structure for Ollama Code
type OllamaCodeConfig struct {
	Model           string            `json:"model"`
	ApiURL          string            `json:"api_url"`
	ContextSize     int               `json:"context_size"`
	Temperature     float64           `json:"temperature"`
	TopP            float64           `json:"top_p"`
	MaxTokens       int               `json:"max_tokens"`
	SystemPrompts   map[string]string `json:"system_prompts"`
	HistoryFilePath string            `json:"history_file_path"`
	KaliTools       []string          `json:"kali_tools,omitempty"`
}

// Global configuration
var config OllamaCodeConfig

// isKaliLinux checks if the current OS is Kali Linux
func isKaliLinux() bool {
	// Check /etc/os-release for Kali Linux
	if _, err := os.Stat("/etc/os-release"); err == nil {
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			return strings.Contains(string(data), "ID=kali")
		}
	}
	return false
}

// getAvailableKaliTools returns a list of installed Kali tools
func getAvailableKaliTools() []string {
	var tools []string

	// Common Kali tools to check for
	commonTools := []string{
		"nmap", "metasploit", "burpsuite", "wireshark", "aircrack-ng",
		"hydra", "john", "sqlmap", "dirbuster", "nikto",
	}

	for _, tool := range commonTools {
		cmd := exec.Command("which", tool)
		if err := cmd.Run(); err == nil {
			tools = append(tools, tool)
		}
	}

	return tools
}

// Initialize configuration with default values
func initConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		os.Exit(1)
	}

	// Default configuration
	config = OllamaCodeConfig{
		Model:           "qwen2.5-coder:1.5b",
		ApiURL:          "http://localhost:11434",
		ContextSize:     8192,
		Temperature:     0.2,
		TopP:            0.95,
		MaxTokens:       2048,
		HistoryFilePath: filepath.Join(homeDir, ".ollama-code", "history.json"),
		SystemPrompts: map[string]string{
			"generate": "You are an expert code generator optimized for Kali Linux environments. Create clean, efficient, and well-commented code based on the user's requirements. Focus on security tools integration when relevant.",
			"explain":  "You are a code explanation expert with knowledge of Kali Linux and security tooling. Analyze the provided code and explain how it works in clear, concise terms. Focus on security implications when relevant.",
			"refactor": "You are a code refactoring specialist with Kali Linux expertise. Analyze the provided code and suggest improvements to make it more efficient, readable, and maintainable without changing its core functionality. Consider security best practices.",
			"debug":    "You are a debugging expert familiar with Kali Linux environments. Analyze the code and error messages to identify issues. Provide clear explanations of the bugs and suggest fixes with improved code.",
			"test":     "You are a testing specialist for security-focused applications. Create comprehensive test cases for the provided code, covering edge cases, security vulnerabilities, and typical usage patterns.",
			"doc":      "You are a documentation expert familiar with Kali Linux tools and conventions. Generate clear, concise documentation for the provided code, including function descriptions, parameters, return values, and usage examples.",
		},
	}

	// Add Kali Linux specific configuration
	if isKaliLinux() {
		config.KaliTools = getAvailableKaliTools()
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Join(homeDir, ".ollama-code")
	if _, err := os.Stat(configDir); os.IsNotExist(err) {
		if err := os.MkdirAll(configDir, 0755); err != nil {
			fmt.Println("Error creating config directory:", err)
		}
	}

	// Check if config file exists and load it
	configPath := filepath.Join(configDir, "config.json")
	if _, err := os.Stat(configPath); err == nil {
		data, err := os.ReadFile(configPath)
		if err == nil {
			_ = json.Unmarshal(data, &config)
		}
	} else {
		// Save default config
		data, _ := json.MarshalIndent(config, "", "  ")
		_ = os.WriteFile(configPath, data, 0644)
	}
}

// BuildPrompt constructs a specialized prompt based on the task
func buildPrompt(task string, language string, context string, userPrompt string) string {
	systemMsg, ok := config.SystemPrompts[task]
	if !ok {
		systemMsg = "You are a helpful AI coding assistant."
	}

	return fmt.Sprintf(
		"System: %s\nLanguage: %s\nContext:\n```\n%s\n```\n\nUser request: %s",
		systemMsg,
		language,
		context,
		userPrompt,
	)
}

// readFileContent reads and returns the content of a file
func readFileContent(filePath string) (string, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// detectLanguage tries to determine the programming language from the file extension
func detectLanguage(filePath string) string {
	ext := strings.ToLower(filepath.Ext(filePath))

	languageMap := map[string]string{
		".py":    "Python",
		".js":    "JavaScript",
		".ts":    "TypeScript",
		".html":  "HTML",
		".css":   "CSS",
		".java":  "Java",
		".c":     "C",
		".cpp":   "C++",
		".cs":    "C#",
		".go":    "Go",
		".rb":    "Ruby",
		".php":   "PHP",
		".swift": "Swift",
		".kt":    "Kotlin",
		".rs":    "Rust",
		".sh":    "Shell",
		".bash":  "Bash",
		".sql":   "SQL",
	}

	if lang, ok := languageMap[ext]; ok {
		return lang
	}
	return "Unknown"
}

// Main function for handling interactive session
func interactiveSession() {
	fmt.Println("Starting Ollama Code interactive session...")
	fmt.Println("Model:", config.Model)
	fmt.Println("Type 'exit' or 'quit' to end the session")
	fmt.Println("Type '/help' for available commands")

	client := api.NewClient(config.ApiURL, config.Model)

	// Check if model exists
	models, err := client.ListModels(context.Background())
	if err != nil {
		fmt.Printf("Warning: Could not verify model availability: %v\n", err)
	} else {
		modelExists := false
		for _, model := range models {
			if model == config.Model {
				modelExists = true
				break
			}
		}

		if !modelExists {
			fmt.Printf("Warning: Model '%s' not found. Available models: %v\n", config.Model, models)
			fmt.Println("Using model anyway. If it doesn't exist, Ollama will attempt to download it.")
		}
	}

	// Create terminal UI
	terminal := ui.NewTerminalUI()

	// Start UI in background
	go func() {
		err := terminal.Start()
		if err != nil {
			fmt.Printf("Error starting terminal UI: %v\n", err)
			os.Exit(1)
		}
	}()

	scanner := bufio.NewScanner(os.Stdin)

	for {
		fmt.Print("\nollama-code> ")
		if !scanner.Scan() {
			break
		}

		input := scanner.Text()
		if input == "exit" || input == "quit" {
			break
		}

		if strings.HasPrefix(input, "/") {
			handleCommand(client, terminal, input)
			continue
		}

		// Regular prompt
		handlePrompt(client, terminal, input, "")
	}
}

// Handle special commands
func handleCommand(client *api.OllamaClient, terminal *ui.TerminalUI, input string) {
	cmd := strings.TrimSpace(strings.TrimPrefix(input, "/"))
	parts := strings.Fields(cmd)

	if len(parts) == 0 {
		return
	}

	switch parts[0] {
	case "help":
		terminal.AddMessage("system", "Available commands:\n"+
			"  /generate <description> - Generate code from description\n"+
			"  /explain <file> - Explain code in file\n"+
			"  /refactor <file> - Suggest refactoring for code\n"+
			"  /debug <file> - Help debug code\n"+
			"  /test <file> - Generate tests for code\n"+
			"  /doc <file> - Generate documentation\n"+
			"  /model <modelname> - Change the model\n"+
			"  /temp <value> - Change temperature (0.0-1.0)\n"+
			"  /help - Show this help")

	case "generate", "explain", "refactor", "debug", "test", "doc":
		if len(parts) < 2 {
			terminal.AddMessage("system", fmt.Sprintf("/%s requires additional arguments", parts[0]))
			return
		}

		task := parts[0]
		arg := strings.Join(parts[1:], " ")

		// Check if argument is a file path
		fileInfo, err := os.Stat(arg)
		if err == nil && !fileInfo.IsDir() {
			content, err := readFileContent(arg)
			if err != nil {
				terminal.AddMessage("system", "Error reading file: "+err.Error())
				return
			}
			language := detectLanguage(arg)
			handlePrompt(client, terminal, "", buildPrompt(task, language, content, ""))
		} else {
			// Treat as direct prompt
			handlePrompt(client, terminal, "", buildPrompt(task, "Unknown", "", arg))
		}

	case "model":
		if len(parts) < 2 {
			terminal.AddMessage("system", "Current model: "+config.Model)
			return
		}
		config.Model = parts[1]
		client.DefaultModel = parts[1]
		terminal.AddMessage("system", "Model changed to: "+config.Model)

		// Save config
		saveConfig()

	case "temp":
		if len(parts) < 2 {
			terminal.AddMessage("system", fmt.Sprintf("Current temperature: %.2f", config.Temperature))
			return
		}

		// Parse temperature value
		var temp float64
		_, _ = fmt.Sscanf(parts[1], "%f", &temp)
		if temp < 0 {
			temp = 0
		} else if temp > 1 {
			temp = 1
		}

		config.Temperature = temp
		terminal.AddMessage("system", fmt.Sprintf("Temperature changed to: %.2f", config.Temperature))

		// Save config
		saveConfig()

	default:
		terminal.AddMessage("system", "Unknown command. Type /help for available commands")
	}
}

// Save configuration to file
func saveConfig() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		fmt.Println("Error getting home directory:", err)
		return
	}

	configPath := filepath.Join(homeDir, ".ollama-code", "config.json")
	data, _ := json.MarshalIndent(config, "", "  ")
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		fmt.Println("Error writing config:", err)
	}
}

// Handle a user prompt
func handlePrompt(client *api.OllamaClient, terminal *ui.TerminalUI, userInput string, formattedPrompt string) {
	ctx := context.Background()

	prompt := formattedPrompt
	if prompt == "" {
		prompt = userInput
	}

	terminal.AddMessage("user", userInput)

	// Start spinning indicator
	terminal.SetLoading(true, "Thinking...")

	options := map[string]interface{}{
		"temperature": config.Temperature,
		"top_p":       config.TopP,
		"max_tokens":  config.MaxTokens,
	}

	// Stream response from model
	err := client.GenerateStream(ctx, &api.GenerateRequest{
		Model:   config.Model,
		Prompt:  prompt,
		Options: options,
	}, func(resp interface{}) {
		if genResp, ok := resp.(*api.GenerateResponse); ok {
			terminal.StreamOutput(genResp.Response)
		}
	})

	if err != nil {
		terminal.SetLoading(false, "")
		terminal.AddMessage("system", "Error: "+err.Error())
	} else {
		terminal.SetLoading(false, "")
	}
}

func main() {
	// Initialize configuration
	initConfig()

	// Create root command
	rootCmd := &cobra.Command{
		Use:   "ollama-code",
		Short: "AI coding assistant powered by Ollama",
		Long:  `A terminal-based AI coding assistant that leverages Ollama's models for code generation, explanation, and more.`,
		Run: func(cmd *cobra.Command, args []string) {
			// If no arguments provided, start interactive session
			if len(args) == 0 {
				interactiveSession()
				return
			}

			// Otherwise, treat arguments as a prompt
			prompt := strings.Join(args, " ")

			// Create client
			client := api.NewClient(config.ApiURL, config.Model)

			// Create terminal UI
			terminal := ui.NewTerminalUI()

			handlePrompt(client, terminal, prompt, "")
		},
	}

	// Define command line flags
	rootCmd.PersistentFlags().StringVarP(&config.Model, "model", "m", config.Model, "Specify the Ollama model to use")
	rootCmd.PersistentFlags().Float64VarP(&config.Temperature, "temperature", "t", config.Temperature, "Set the temperature for model responses")
	rootCmd.PersistentFlags().StringVarP(&config.ApiURL, "api", "a", config.ApiURL, "Ollama API URL")

	// Add subcommands
	generateCmd := &cobra.Command{
		Use:   "generate [description]",
		Short: "Generate code from description",
		Args:  cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			prompt := strings.Join(args, " ")
			client := api.NewClient(config.ApiURL, config.Model)
			terminal := ui.NewTerminalUI()
			handlePrompt(client, terminal, "", buildPrompt("generate", "Unknown", "", prompt))
		},
	}

	explainCmd := &cobra.Command{
		Use:   "explain [file]",
		Short: "Explain code in file",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			content, err := readFileContent(filePath)
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}
			language := detectLanguage(filePath)
			client := api.NewClient(config.ApiURL, config.Model)
			terminal := ui.NewTerminalUI()
			handlePrompt(client, terminal, "", buildPrompt("explain", language, content, ""))
		},
	}

	refactorCmd := &cobra.Command{
		Use:   "refactor [file]",
		Short: "Suggest refactoring for code",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			content, err := readFileContent(filePath)
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}
			language := detectLanguage(filePath)
			client := api.NewClient(config.ApiURL, config.Model)
			terminal := ui.NewTerminalUI()
			handlePrompt(client, terminal, "", buildPrompt("refactor", language, content, ""))
		},
	}

	debugCmd := &cobra.Command{
		Use:   "debug [file]",
		Short: "Help debug code",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			content, err := readFileContent(filePath)
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}
			language := detectLanguage(filePath)
			client := api.NewClient(config.ApiURL, config.Model)
			terminal := ui.NewTerminalUI()
			handlePrompt(client, terminal, "", buildPrompt("debug", language, content, ""))
		},
	}

	testCmd := &cobra.Command{
		Use:   "test [file]",
		Short: "Generate tests for code",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			content, err := readFileContent(filePath)
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}
			language := detectLanguage(filePath)
			client := api.NewClient(config.ApiURL, config.Model)
			terminal := ui.NewTerminalUI()
			handlePrompt(client, terminal, "", buildPrompt("test", language, content, ""))
		},
	}

	docCmd := &cobra.Command{
		Use:   "doc [file]",
		Short: "Generate documentation",
		Args:  cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			filePath := args[0]
			content, err := readFileContent(filePath)
			if err != nil {
				fmt.Println("Error reading file:", err)
				os.Exit(1)
			}
			language := detectLanguage(filePath)
			client := api.NewClient(config.ApiURL, config.Model)
			terminal := ui.NewTerminalUI()
			handlePrompt(client, terminal, "", buildPrompt("doc", language, content, ""))
		},
	}

	rootCmd.AddCommand(generateCmd, explainCmd, refactorCmd, debugCmd, testCmd, docCmd)

	// Add Kali Linux specific commands
	if isKaliLinux() {
		toolsCmd := &cobra.Command{
			Use:   "tools",
			Short: "Work with Kali Linux security tools",
			Long:  `Commands for working with Kali Linux security tools and generating code that integrates with them.`,
		}

		listToolsCmd := &cobra.Command{
			Use:   "list",
			Short: "List available security tools",
			Run: func(cmd *cobra.Command, args []string) {
				fmt.Println("Detected Kali Linux Security Tools:")
				fmt.Println("===================================")

				if len(config.KaliTools) == 0 {
					fmt.Println("No tools detected. Run 'ollama-code' once to auto-detect tools.")
					return
				}

				for _, tool := range config.KaliTools {
					fmt.Printf("- %s\n", tool)
				}

				fmt.Println("\nTool paths:")
				toolPaths := map[string]string{
					"metasploit": "/usr/share/metasploit-framework",
					"exploitdb":  "/usr/share/exploitdb",
					"wordlists":  "/usr/share/wordlists",
					"nmap":       "/usr/share/nmap",
				}

				for name, path := range toolPaths {
					if _, err := os.Stat(path); err == nil {
						fmt.Printf("- %s: %s [✓]\n", name, path)
					} else {
						fmt.Printf("- %s: %s [✗]\n", name, path)
					}
				}
			},
		}

		toolsGenerateCmd := &cobra.Command{
			Use:   "generate [tool] [description]",
			Short: "Generate code that integrates with a specific Kali Linux tool",
			Args:  cobra.ExactArgs(2),
			Run: func(cmd *cobra.Command, args []string) {
				toolName := args[0]
				description := args[1]

				// Build a specialized prompt for the security tool
				prompt := fmt.Sprintf(
					"Generate code that uses the Kali Linux security tool '%s'. The code should: %s\n\n"+
						"Please include detailed comments explaining how the code works with %s and any prerequisites needed.",
					toolName, description, toolName,
				)

				// Call the API
				client := api.NewClient(config.ApiURL, config.Model)
				terminal := ui.NewTerminalUI()
				handlePrompt(client, terminal, "", buildPrompt("generate", "Security", "", prompt))
			},
		}

		toolsCmd.AddCommand(listToolsCmd, toolsGenerateCmd)
		rootCmd.AddCommand(toolsCmd)
	}

	// Execute the command
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
