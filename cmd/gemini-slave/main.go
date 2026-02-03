package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

const version = "3.0.0"

func main() {
	// Define flags
	outputFormat := flag.String("o", "", "Output format: text, json, stream-json")
	modelOverride := flag.String("m", "", "Override model selection")
	timeoutOverride := flag.Int("t", 0, "Override timeout in seconds")
	showVersion := flag.Bool("version", false, "Show version")
	showHelp := flag.Bool("h", false, "Show help")

	// Custom usage message
	flag.Usage = usage

	// Parse flags
	flag.Parse()

	if *showVersion {
		fmt.Printf("gemini-slave v%s\n", version)
		return
	}

	if *showHelp {
		usage()
		return
	}

	// CRITICAL: Capture stdin FIRST before any other operations
	// This must happen before flag parsing consumes any stdin data
	// Args after flags are file paths
	fileArgs := flag.Args()

	// Need at least protocol and instruction
	if len(fileArgs) < 2 {
		fmt.Fprintf(os.Stderr, "Error: Missing required arguments\n\n")
		usage()
		os.Exit(1)
	}

	protocolName := fileArgs[0]
	instruction := fileArgs[1]
	files := fileArgs[2:] // Remaining args are file paths

	// Load protocol configuration and content
	config, protocolContent, err := LoadProtocol(protocolName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n\n", err)
		fmt.Fprintf(os.Stderr, "Available protocols:\n")
		if protocols, err := ListProtocols(); err == nil {
			for _, p := range protocols {
				fmt.Fprintf(os.Stderr, "  %s\n", p)
			}
		}
		os.Exit(1)
	}

	// Apply overrides from flags or environment
	model := config.Model
	timeout := config.Timeout

	if *modelOverride != "" {
		model = *modelOverride
	} else if envModel := os.Getenv("GEMINI_MODEL"); envModel != "" {
		model = envModel
	}

	if *timeoutOverride > 0 {
		timeout = time.Duration(*timeoutOverride) * time.Second
	} else if envTimeout := os.Getenv("GEMINI_SLAVE_TIMEOUT"); envTimeout != "" {
		if t, err := strconv.Atoi(envTimeout); err == nil {
			timeout = time.Duration(t) * time.Second
		}
	}

	// Capture input (stdin or files)
	// CRITICAL: Must happen AFTER flag parsing but BEFORE any other I/O
	inputContext, err := CaptureInput(files)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Build full prompt: protocol + instruction + input context
	fullPrompt := buildPrompt(protocolContent, instruction, inputContext)

	// Create executor
	executor := NewGeminiExecutor(model, timeout)
	if *outputFormat != "" {
		executor.OutputFormat = *outputFormat
	}

	// Execute Gemini
	output, err := executor.Execute(fullPrompt)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}

	// Output result
	fmt.Print(output)
}

// buildPrompt assembles the full prompt from protocol, instruction, and input.
func buildPrompt(protocolContent, instruction, inputContext string) string {
	var sb strings.Builder

	// Protocol content
	sb.WriteString(protocolContent)
	sb.WriteString("\n\n---\n### SPECIFIC INSTRUCTION\n")
	sb.WriteString(instruction)
	sb.WriteString("\n\n---\n### INPUT CONTEXT\n")
	sb.WriteString(inputContext)

	return sb.String()
}

// usage prints the help message.
func usage() {
	fmt.Fprintf(os.Stderr, `gemini-slave - Gemini AI Protocol Executor

Usage: gemini-slave [OPTIONS] <protocol> "instruction" [files...]

Arguments:
  <protocol>      Protocol name (e.g., mapper, architect, debugger)
  "instruction"   Specific instruction for the protocol
  [files...]      Optional file paths to include as input

Options:
  -o <format>     Output format: text, json, stream-json (default: protocol-dependent)
  -m <model>      Override model selection (default: protocol-dependent)
  -t <seconds>    Override timeout in seconds (default: protocol-dependent)
  -h              Show this help message
  --version       Show version information

Environment Variables:
  GEMINI_MODEL           Override model for all protocols
  GEMINI_SLAVE_TIMEOUT   Default timeout in seconds

Examples:
  # Pipe code to mapper protocol
  find cmd/ -name "*.go" -exec cat {} \; | gemini-slave mapper "Map entry points"

  # Pass files as arguments
  gemini-slave deps "Extract dependencies" cmd/gemini-slave/*.go

  # Force JSON output
  gemini-slave architect "Review module" -o json < module.go

  # Override model
  gemini-slave -m gemini-3-pro-preview mapper "Analyze" < code.go

Protocol → Model Mapping:
  Flash (fast):    mapper, memory-drift, benchmark-score, deps, api-surface
  Pro (complex):   architect, debugger, memory-audit, benchmark-audit

`)
}
