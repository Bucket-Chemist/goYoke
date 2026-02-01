# GO Cobra CLI Conventions - Lisan al-Gaib

## Overview

Cobra is the standard CLI framework for GO. These conventions ensure professional-grade CLI applications with proper configuration, error handling, and user experience.

## Project Structure

```
myapp/
├── cmd/
│   └── myapp/
│       └── main.go           # Minimal entrypoint
├── internal/
│   └── cli/
│       ├── root.go           # Root command + global config
│       ├── serve/
│       │   └── command.go    # Subcommand factory
│       ├── config/
│       │   └── command.go
│       └── version/
│           └── command.go
├── go.mod
└── go.sum
```

## Main Entry Point

### Minimal main.go

```go
// cmd/myapp/main.go
package main

import (
    "os"
    "myapp/internal/cli"
)

func main() {
    if err := cli.Execute(); err != nil {
        os.Exit(1)
    }
}
```

## Root Command

### Standard Pattern

```go
// internal/cli/root.go
package cli

import (
    "fmt"
    "os"
    "path/filepath"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

var (
    cfgFile string
    verbose bool
)

var rootCmd = &cobra.Command{
    Use:   "myapp",
    Short: "A professional CLI tool",
    Long: `MyApp - A comprehensive tool for X.

Complete documentation at https://myapp.example.com`,
    
    // PersistentPreRunE runs before ANY subcommand
    PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
        return initConfig()
    },
}

func Execute() error {
    return rootCmd.Execute()
}

func init() {
    // Global flags (available to all subcommands)
    rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default $HOME/.myapp/config.toml)")
    rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
    
    // Bind to viper
    viper.BindPFlag("verbose", rootCmd.PersistentFlags().Lookup("verbose"))
    
    // Add subcommands
    rootCmd.AddCommand(serve.NewCommand())
    rootCmd.AddCommand(config.NewCommand())
    rootCmd.AddCommand(version.NewCommand())
}

func initConfig() error {
    if cfgFile != "" {
        viper.SetConfigFile(cfgFile)
    } else {
        home, err := os.UserHomeDir()
        if err != nil {
            return fmt.Errorf("find home directory: %w", err)
        }
        
        configDir := filepath.Join(home, ".myapp")
        viper.AddConfigPath(configDir)
        viper.SetConfigName("config")
        viper.SetConfigType("toml")
    }
    
    // Environment variables
    viper.SetEnvPrefix("MYAPP")
    viper.AutomaticEnv()
    
    // Read config (ignore if not found)
    if err := viper.ReadInConfig(); err != nil {
        if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
            return fmt.Errorf("read config: %w", err)
        }
    }
    
    return nil
}
```

## Subcommand Pattern

### Factory Function Pattern

```go
// internal/cli/serve/command.go
package serve

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"
    
    "github.com/spf13/cobra"
    "github.com/spf13/viper"
)

func NewCommand() *cobra.Command {
    cmd := &cobra.Command{
        Use:   "serve",
        Short: "Start the server",
        Long:  `Start the HTTP server on the specified port.`,
        Example: `  myapp serve --port 8080
  myapp serve --config /path/to/config.toml`,
        
        RunE: runServe,
    }
    
    // Local flags (only for this command)
    cmd.Flags().IntP("port", "p", 8080, "port to listen on")
    cmd.Flags().String("host", "localhost", "host to bind to")
    
    // Bind local flags to viper
    viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))
    viper.BindPFlag("server.host", cmd.Flags().Lookup("host"))
    
    return cmd
}

func runServe(cmd *cobra.Command, args []string) error {
    // CRITICAL: Silence usage on runtime errors
    cmd.SilenceUsage = true
    
    // Get values from viper (respects flag > env > config > default)
    port := viper.GetInt("server.port")
    host := viper.GetString("server.host")
    
    // Setup graceful shutdown
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    sigCh := make(chan os.Signal, 1)
    signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
    
    go func() {
        <-sigCh
        fmt.Println("\nShutting down...")
        cancel()
    }()
    
    // Start server
    return startServer(ctx, host, port)
}
```

## Viper Integration

### CRITICAL: Configuration Priority

```go
// Viper priority (highest to lowest):
// 1. Explicit flag value (--port 8080)
// 2. Environment variable (MYAPP_SERVER_PORT=8080)
// 3. Config file value
// 4. Default value

// WRONG: Using flag directly (ignores config file)
port, _ := cmd.Flags().GetInt("port")

// CORRECT: Using viper (respects full priority chain)
port := viper.GetInt("server.port")
```

### Binding Flags to Viper

```go
// In NewCommand():
cmd.Flags().Int("port", 8080, "port number")
viper.BindPFlag("server.port", cmd.Flags().Lookup("port"))

// In RunE:
port := viper.GetInt("server.port")

// Environment variable: MYAPP_SERVER_PORT (automatic with SetEnvPrefix)
```

### Config File Structure

```toml
# ~/.myapp/config.toml

[server]
port = 8080
host = "0.0.0.0"

[api]
key = "sk-..."
timeout = "30s"

[logging]
level = "info"
format = "json"
```

## Error Handling

### RunE vs Run

```go
// CORRECT: Use RunE for proper error propagation
RunE: func(cmd *cobra.Command, args []string) error {
    cmd.SilenceUsage = true  // Don't show usage on runtime errors
    
    if err := doWork(); err != nil {
        return fmt.Errorf("work failed: %w", err)
    }
    return nil
},

// WRONG: Using Run with os.Exit
Run: func(cmd *cobra.Command, args []string) {
    if err := doWork(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)  // Skips cleanup, bad practice
    }
},
```

### SilenceUsage Pattern

```go
RunE: func(cmd *cobra.Command, args []string) error {
    // Set this FIRST - prevents usage output on runtime errors
    cmd.SilenceUsage = true
    
    // Now do work...
    result, err := process(args)
    if err != nil {
        // Error shown, but NOT usage help
        return err
    }
    
    fmt.Println(result)
    return nil
},
```

## Argument Validation

### Built-in Validators

```go
cmd := &cobra.Command{
    Use:   "delete [id]",
    Args:  cobra.ExactArgs(1),  // Exactly 1 arg required
    RunE:  runDelete,
}

// Available validators:
// cobra.NoArgs              - No arguments allowed
// cobra.ExactArgs(n)        - Exactly n arguments
// cobra.MinimumNArgs(n)     - At least n arguments
// cobra.MaximumNArgs(n)     - At most n arguments
// cobra.RangeArgs(min, max) - Between min and max arguments
// cobra.OnlyValidArgs       - Must be in ValidArgs list
```

### Custom Validation

```go
cmd := &cobra.Command{
    Use:  "process [file]",
    Args: func(cmd *cobra.Command, args []string) error {
        if len(args) != 1 {
            return fmt.Errorf("requires exactly one file argument")
        }
        
        if _, err := os.Stat(args[0]); os.IsNotExist(err) {
            return fmt.Errorf("file %q does not exist", args[0])
        }
        
        return nil
    },
    RunE: runProcess,
}
```

## Completion Support

### Enable Shell Completion

```go
func init() {
    rootCmd.AddCommand(completionCmd)
}

var completionCmd = &cobra.Command{
    Use:   "completion [bash|zsh|fish|powershell]",
    Short: "Generate shell completion script",
    Long: `Generate shell completion script for the specified shell.

To load completions:

Bash:
  $ source <(myapp completion bash)
  # Or add to ~/.bashrc

Zsh:
  $ myapp completion zsh > "${fpath[1]}/_myapp"

Fish:
  $ myapp completion fish > ~/.config/fish/completions/myapp.fish

PowerShell:
  PS> myapp completion powershell | Out-String | Invoke-Expression
`,
    ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
    Args:      cobra.ExactValidArgs(1),
    RunE: func(cmd *cobra.Command, args []string) error {
        switch args[0] {
        case "bash":
            return rootCmd.GenBashCompletion(os.Stdout)
        case "zsh":
            return rootCmd.GenZshCompletion(os.Stdout)
        case "fish":
            return rootCmd.GenFishCompletion(os.Stdout, true)
        case "powershell":
            return rootCmd.GenPowerShellCompletionWithDesc(os.Stdout)
        default:
            return fmt.Errorf("unknown shell: %s", args[0])
        }
    },
}
```

### Dynamic Completion

```go
cmd := &cobra.Command{
    Use: "select [item]",
    ValidArgsFunction: func(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
        if len(args) != 0 {
            return nil, cobra.ShellCompDirectiveNoFileComp
        }
        
        // Return dynamic suggestions
        items := []string{"alpha", "beta", "gamma"}
        return items, cobra.ShellCompDirectiveNoFileComp
    },
}
```

## Output Formatting

### JSON Output Flag

```go
var outputFormat string

func init() {
    rootCmd.PersistentFlags().StringVarP(&outputFormat, "output", "o", "text", "output format (text|json)")
}

func printResult(result interface{}) error {
    switch outputFormat {
    case "json":
        enc := json.NewEncoder(os.Stdout)
        enc.SetIndent("", "  ")
        return enc.Encode(result)
    case "text":
        fmt.Printf("%+v\n", result)
        return nil
    default:
        return fmt.Errorf("unknown output format: %s", outputFormat)
    }
}
```

### Progress Output

```go
// Use stderr for progress, stdout for results
fmt.Fprintln(os.Stderr, "Processing...")

// Clear progress line
fmt.Fprint(os.Stderr, "\r                    \r")

// Final result to stdout
fmt.Fprintln(os.Stdout, result)
```

## Testing Commands

### Test Helper

```go
func executeCommand(root *cobra.Command, args ...string) (output string, err error) {
    buf := new(bytes.Buffer)
    root.SetOut(buf)
    root.SetErr(buf)
    root.SetArgs(args)
    
    err = root.Execute()
    return buf.String(), err
}

func TestServeCommand(t *testing.T) {
    output, err := executeCommand(rootCmd, "serve", "--port", "9090")
    require.NoError(t, err)
    assert.Contains(t, output, "Starting server")
}
```

## Version Command

### Standard Pattern

```go
// Set at build time with ldflags
var (
    version = "dev"
    commit  = "none"
    date    = "unknown"
)

var versionCmd = &cobra.Command{
    Use:   "version",
    Short: "Print version information",
    Run: func(cmd *cobra.Command, args []string) {
        fmt.Printf("myapp %s\n", version)
        fmt.Printf("  commit: %s\n", commit)
        fmt.Printf("  built:  %s\n", date)
    },
}

// Build with:
// go build -ldflags "-X main.version=1.0.0 -X main.commit=$(git rev-parse HEAD) -X main.date=$(date -u +%Y-%m-%dT%H:%M:%SZ)"
```

## Sharp Edges

### 1. Flag Binding Timing

```go
// WRONG: Binding in RunE (too late, viper already initialized)
RunE: func(cmd *cobra.Command, args []string) error {
    viper.BindPFlag("port", cmd.Flags().Lookup("port"))  // TOO LATE
    ...
}

// CORRECT: Binding in init() or NewCommand()
func NewCommand() *cobra.Command {
    cmd := &cobra.Command{...}
    cmd.Flags().Int("port", 8080, "port")
    viper.BindPFlag("port", cmd.Flags().Lookup("port"))  // CORRECT
    return cmd
}
```

### 2. Persistent vs Local Flags

```go
// Persistent: Available to this command AND all subcommands
rootCmd.PersistentFlags().Bool("verbose", false, "verbose output")

// Local: Only available to THIS command
serveCmd.Flags().Int("port", 8080, "port")
```

### 3. Required Flags

```go
cmd.Flags().String("api-key", "", "API key (required)")
cmd.MarkFlagRequired("api-key")

// Better UX: Custom validation with helpful message
RunE: func(cmd *cobra.Command, args []string) error {
    apiKey := viper.GetString("api-key")
    if apiKey == "" {
        return fmt.Errorf("API key required. Set via --api-key flag, MYAPP_API_KEY env var, or config file")
    }
    ...
}
```

### 4. Hidden Commands

```go
// For internal/debug commands
debugCmd := &cobra.Command{
    Use:    "debug",
    Hidden: true,  // Won't show in help
    ...
}
```
