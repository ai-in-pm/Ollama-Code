package context_manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ContextManager handles file and project context for more intelligent responses
type ContextManager struct {
	rootPath      string
	fileCache     map[string]string
	fileMtimes    map[string]int64
	mutex         sync.RWMutex
	ignoreDirs    []string
	ignoreFiles   []string
	maxContextLen int
	isKaliLinux   bool
}

// NewContextManager creates a new context manager for the given root directory
func NewContextManager(rootPath string) *ContextManager {
	// Check if running on Kali Linux
	isKali := false
	if _, err := os.Stat("/etc/os-release"); err == nil {
		data, err := os.ReadFile("/etc/os-release")
		if err == nil {
			isKali = strings.Contains(string(data), "ID=kali")
		}
	}

	// Default dirs to ignore
	ignoreDirs := []string{
		".git", "node_modules", "__pycache__", "venv", ".env", ".venv",
	}

	// Add Kali-specific directories to ignore list
	if isKali {
		kaliSpecificDirs := []string{
			".msf4", "metasploit-framework", "wordlists", "exploitdb",
		}
		ignoreDirs = append(ignoreDirs, kaliSpecificDirs...)
	}

	return &ContextManager{
		rootPath:      rootPath,
		fileCache:     make(map[string]string),
		fileMtimes:    make(map[string]int64),
		ignoreDirs:    ignoreDirs,
		ignoreFiles:   []string{".DS_Store", "*.pyc", "*.o", "*.out", "*.log"},
		maxContextLen: 16384, // Default max context size
		isKaliLinux:   isKali,
	}
}

// SetMaxContextLength sets the maximum number of characters to include in context
func (cm *ContextManager) SetMaxContextLength(maxLen int) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()
	cm.maxContextLen = maxLen
}

// AddIgnorePattern adds a pattern to the ignore list
func (cm *ContextManager) AddIgnorePattern(pattern string) {
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Determine if it's a directory or file pattern
	if strings.HasSuffix(pattern, "/") {
		cm.ignoreDirs = append(cm.ignoreDirs, strings.TrimSuffix(pattern, "/"))
	} else {
		cm.ignoreFiles = append(cm.ignoreFiles, pattern)
	}
}

// GetFileContent reads a file and caches its content
func (cm *ContextManager) GetFileContent(filePath string) (string, error) {
	cm.mutex.RLock()

	// Check if we have this file cached and it hasn't changed
	if content, ok := cm.fileCache[filePath]; ok {
		if info, err := os.Stat(filePath); err == nil {
			if mtime, ok := cm.fileMtimes[filePath]; ok && mtime == info.ModTime().UnixNano() {
				cm.mutex.RUnlock()
				return content, nil
			}
		}
	}
	cm.mutex.RUnlock()

	// Read the file content
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", err
	}

	content := string(data)

	// Cache the content and modification time
	info, err := os.Stat(filePath)
	if err == nil {
		cm.mutex.Lock()
		cm.fileCache[filePath] = content
		cm.fileMtimes[filePath] = info.ModTime().UnixNano()
		cm.mutex.Unlock()
	}

	return content, nil
}

// ShouldIgnore checks if a file or directory should be ignored
func (cm *ContextManager) ShouldIgnore(path string) bool {
	// Get the base name of the path
	base := filepath.Base(path)

	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if info.IsDir() {
		for _, ignoreDir := range cm.ignoreDirs {
			if base == ignoreDir {
				return true
			}
		}
	} else {
		for _, ignorePattern := range cm.ignoreFiles {
			matched, err := filepath.Match(ignorePattern, base)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// GetProjectStructure returns a tree representation of the project structure
func (cm *ContextManager) GetProjectStructure() (string, error) {
	var result strings.Builder
	result.WriteString("Project Structure:\n")

	err := filepath.Walk(cm.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip ignored files and directories
		if path != cm.rootPath && cm.ShouldIgnore(path) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Calculate the depth for indentation
		relPath, err := filepath.Rel(cm.rootPath, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		depth := len(strings.Split(relPath, string(os.PathSeparator))) - 1
		indent := strings.Repeat("  ", depth)

		// Add the file or directory to the result
		if info.IsDir() {
			result.WriteString(fmt.Sprintf("%s%s/\n", indent, info.Name()))
		} else {
			result.WriteString(fmt.Sprintf("%s%s\n", indent, info.Name()))
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return result.String(), nil
}

// GetRelevantFiles finds files that may be relevant to the given query
func (cm *ContextManager) GetRelevantFiles(query string, maxFiles int) ([]string, error) {
	var relevantFiles []string

	err := filepath.Walk(cm.rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip files we can't access
		}

		// Skip directories
		if info.IsDir() {
			if cm.ShouldIgnore(path) {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip ignored files
		if cm.ShouldIgnore(path) {
			return nil
		}

		// For now, use simple substring matching to find relevant files
		// This could be replaced with more sophisticated techniques like TF-IDF or embeddings
		relPath, _ := filepath.Rel(cm.rootPath, path)
		if strings.Contains(strings.ToLower(relPath), strings.ToLower(query)) {
			relevantFiles = append(relevantFiles, path)
		}

		// Limit the number of files
		if len(relevantFiles) >= maxFiles {
			return filepath.SkipDir
		}

		return nil
	})

	return relevantFiles, err
}

// GetFileContext returns context information about a file and its related files
func (cm *ContextManager) GetFileContext(filePath string) (string, error) {
	var context strings.Builder

	// Get the content of the main file
	content, err := cm.GetFileContent(filePath)
	if err != nil {
		return "", err
	}

	// Add the file content to context
	relPath, _ := filepath.Rel(cm.rootPath, filePath)
	context.WriteString(fmt.Sprintf("File: %s\n\n```\n%s\n```\n\n", relPath, content))

	// Get imports or dependencies from this file
	relatedFiles := cm.findRelatedFiles(filePath, content)

	// Add related files to context
	totalLength := context.Len()
	for _, relatedFile := range relatedFiles {
		relatedContent, err := cm.GetFileContent(relatedFile)
		if err != nil {
			continue
		}

		relPath, _ = filepath.Rel(cm.rootPath, relatedFile)
		fileContext := fmt.Sprintf("Related file: %s\n\n```\n%s\n```\n\n", relPath, relatedContent)

		// Check if adding this would exceed the context limit
		if totalLength+len(fileContext) > cm.maxContextLen {
			context.WriteString(fmt.Sprintf("(Additional related file %s not included due to context length limits)\n", relPath))
			continue
		}

		context.WriteString(fileContext)
		totalLength += len(fileContext)
	}

	return context.String(), nil
}

// findRelatedFiles attempts to find files related to the given file
func (cm *ContextManager) findRelatedFiles(filePath, content string) []string {
	var relatedFiles []string
	ext := filepath.Ext(filePath)
	dir := filepath.Dir(filePath)

	// Different strategies based on language
	switch ext {
	case ".go":
		// Find imports in Go files
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import ") || (strings.HasPrefix(line, "import(") && !strings.HasSuffix(line, ")")) {
				if strings.HasPrefix(line, "import (") {
					continue
				}

				importPath := ""
				if strings.HasPrefix(line, "import \"") {
					// Single line import
					importPath = strings.Trim(strings.TrimPrefix(line, "import "), "\"")

					// For local imports
					if strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../") {
						resolvedPath := filepath.Join(dir, importPath)
						if info, err := os.Stat(resolvedPath); err == nil && !info.IsDir() {
							relatedFiles = append(relatedFiles, resolvedPath)
						} else if err == nil && info.IsDir() {
							// It's a directory, look for .go files
							entries, err := os.ReadDir(resolvedPath)
							if err == nil {
								for _, entry := range entries {
									if !entry.IsDir() && filepath.Ext(entry.Name()) == ".go" {
										relatedFiles = append(relatedFiles, filepath.Join(resolvedPath, entry.Name()))
									}
								}
							}
						}
					}
				}
			}
		}

	case ".js", ".ts":
		// Find imports in JS/TS files
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "require(") {
				var importPath string

				if strings.HasPrefix(line, "import ") {
					parts := strings.Split(line, "from")
					if len(parts) > 1 {
						importPath = strings.Trim(strings.TrimSpace(parts[1]), "'\";\r\n")
					}
				} else if strings.HasPrefix(line, "require(") {
					importPath = strings.Trim(strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(line, "require("), ");")), "'\"")
				}

				// For local imports
				if importPath != "" && (strings.HasPrefix(importPath, "./") || strings.HasPrefix(importPath, "../")) {
					resolvedPath := filepath.Join(dir, importPath)

					// Check if the path exists as is
					if info, err := os.Stat(resolvedPath); err == nil {
						if !info.IsDir() {
							relatedFiles = append(relatedFiles, resolvedPath)
						}
					} else {
						// Try with extensions
						for _, ext := range []string{".js", ".ts", ".jsx", ".tsx"} {
							pathWithExt := resolvedPath + ext
							if info, err := os.Stat(pathWithExt); err == nil && !info.IsDir() {
								relatedFiles = append(relatedFiles, pathWithExt)
								break
							}
						}
					}
				}
			}
		}

	case ".py":
		// Find imports in Python files
		for _, line := range strings.Split(content, "\n") {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "import ") || strings.HasPrefix(line, "from ") {
				if strings.HasPrefix(line, "from ") {
					parts := strings.Split(line, "import")
					if len(parts) > 0 {
						module := strings.TrimSpace(strings.TrimPrefix(parts[0], "from "))

						// For local imports
						if strings.HasPrefix(module, ".") {
							// Convert relative import to path
							modulePath := module
							if modulePath == "." {
								modulePath = ""
							} else {
								modulePath = strings.ReplaceAll(modulePath, ".", "/")
							}

							resolvedDir := filepath.Join(dir, modulePath)

							// Check for Python files in the resolved directory
							entries, err := os.ReadDir(resolvedDir)
							if err == nil {
								for _, entry := range entries {
									if !entry.IsDir() && filepath.Ext(entry.Name()) == ".py" {
										relatedFiles = append(relatedFiles, filepath.Join(resolvedDir, entry.Name()))
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Also look for files with the same base name but different extensions
	baseName := filepath.Base(filePath)
	baseWithoutExt := strings.TrimSuffix(baseName, ext)

	entries, err := os.ReadDir(dir)
	if err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			nameWithoutExt := strings.TrimSuffix(name, filepath.Ext(name))

			if nameWithoutExt == baseWithoutExt && name != baseName {
				relatedFiles = append(relatedFiles, filepath.Join(dir, name))
			}
		}
	}

	return relatedFiles
}
