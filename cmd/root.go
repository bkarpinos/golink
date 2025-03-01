package cmd

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"

	"github.com/bkarpinos/golink/internal/link"
	"github.com/bkarpinos/golink/internal/server"
	"github.com/bkarpinos/golink/internal/storage"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	configDir  string // Directory containing config files
	storageDir string // Directory to store links (configurable)
	store      *storage.JSONStorage
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "golink",
	Short: "A terminal UI for managing go/links",
	Long: `A terminal user interface application for creating and managing
go/links for quick navigation to frequently used URLs.. For example:

go/meeting -> http://zoom.us/...
go/drive -> https://docs.google.com/...
go/gh -> https://github.com/...`,
}

// Add command
var addCmd = &cobra.Command{
	Use:   "add [alias] [url]",
	Short: "Add a new go link",
	Args:  cobra.ExactArgs(2),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]
		url := args[1]
		description, _ := cmd.Flags().GetString("description")
		category, _ := cmd.Flags().GetString("category")

		l := link.NewLink(alias, url, description, category)
		if err := store.Create(l); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Printf("Created go link: %s -> %s\n", alias, url)
	},
}

// List command
var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all go links",
	Run: func(cmd *cobra.Command, args []string) {
		links := store.List()
		if len(links) == 0 {
			fmt.Println("No links found.")
			return
		}

		fmt.Println("Go Links:")
		fmt.Println("=========")
		for _, link := range links {
			fmt.Printf("%-15s -> URL: %s\n", link.Alias, link.URL)
			if link.Description != "" {
				fmt.Printf("%18s Description: %s\n", "", link.Description)
			}
			if link.Category != "" {
				fmt.Printf("%18s Category: %s\n", "", link.Category)
			}
			fmt.Println()
		}
	},
}

// Open command
var openCmd = &cobra.Command{
	Use:   "open [alias]",
	Short: "Open a go link in the default browser",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]
		link, err := store.Get(alias)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}

		useDirectURL, _ := cmd.Flags().GetBool("direct")

		var urlToOpen string
		if useDirectURL {
			urlToOpen = link.URL
			fmt.Printf("Opening %s (%s) in browser\n", alias, urlToOpen)
		} else {
			// Create golink URL format
			urlToOpen = fmt.Sprintf("http://go/%s", link.Alias)
			fmt.Printf("Opening %s (%s) in browser\n", alias, urlToOpen)
		}

		// Open URL in the default browser
		var openCmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			openCmd = exec.Command("open", urlToOpen)
		case "linux":
			openCmd = exec.Command("xdg-open", urlToOpen)
		case "windows":
			openCmd = exec.Command("cmd", "/c", "start", urlToOpen)
		default:
			fmt.Fprintf(os.Stderr, "Unsupported operating system: %s\n", runtime.GOOS)
			return
		}

		err = openCmd.Run()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error opening URL: %v\n", err)
		}
	},
}

// Delete command
var deleteCmd = &cobra.Command{
	Use:   "delete [alias]",
	Short: "Delete a go link",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		alias := args[0]
		if err := store.Delete(alias); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			return
		}
		fmt.Printf("Deleted go link: %s\n", alias)
	},
}

// Serve command
var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the go links HTTP server",
	Run: func(cmd *cobra.Command, args []string) {
		port, _ := cmd.Flags().GetInt("port")
		notFoundURL, _ := cmd.Flags().GetString("not-found")

		// Create the server
		srv := server.NewServer(store, port, notFoundURL)

		// Handle graceful shutdown
		stop := make(chan os.Signal, 1)
		signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

		// Start the server in a goroutine
		go func() {
			if err := srv.Start(); err != nil && err != http.ErrServerClosed {
				log.Fatalf("Server error: %v", err)
			}
		}()

		// Wait for interrupt signal
		<-stop

		// Shutdown gracefully with timeout
		fmt.Println("\nShutting down server...")
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := srv.Shutdown(ctx); err != nil {
			fmt.Printf("Error during server shutdown: %v\n", err)
		}

		fmt.Println("Server stopped")
	},
}

// Config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage golink configuration",
}

// Set storage directory command
var setStorageDirCmd = &cobra.Command{
	Use:   "storage-dir [path]",
	Short: "Set the directory to store links",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		path := args[0]
		// Convert to absolute path if needed
		if !filepath.IsAbs(path) {
			absPath, err := filepath.Abs(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error converting to absolute path: %v\n", err)
				return
			}
			path = absPath
		}

		// Set in config
		viper.Set("storage_dir", path)

		// Write config
		if err := viper.WriteConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				// Config file doesn't exist yet
				if err := viper.SafeWriteConfigAs(filepath.Join(configDir, "config.yaml")); err != nil {
					fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
					return
				}
			} else {
				fmt.Fprintf(os.Stderr, "Error writing config: %v\n", err)
				return
			}
		}

		fmt.Printf("Storage directory set to: %s\n", path)
		fmt.Println("Restart the application for changes to take effect.")
	},
}

// View config command
var viewConfigCmd = &cobra.Command{
	Use:   "view",
	Short: "View current configuration",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("Config directory: %s\n", configDir)
		fmt.Printf("Storage directory: %s\n", storageDir)
		if viper.ConfigFileUsed() != "" {
			fmt.Printf("Config file: %s\n", viper.ConfigFileUsed())
		} else {
			fmt.Printf("Config file: not found (using defaults)\n")
		}

		fmt.Println("\nAll settings:")
		settings := viper.AllSettings()
		for k, v := range settings {
			fmt.Printf("  %s: %v\n", k, v)
		}
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig() {
	// Ensure config directory exists
	if err := os.MkdirAll(configDir, 0755); err != nil {
		log.Fatalf("Failed to create config directory: %v", err)
	}

	// Set up Viper to use a config file in configDir
	viper.SetConfigType("yaml")
	viper.SetConfigName("config")
	viper.AddConfigPath(configDir)

	// Read in environment variables that match
	viper.AutomaticEnv()

	// If config file exists, read it in
	if err := viper.ReadInConfig(); err != nil {
		// Only display error if it's not simply "file not found"
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			fmt.Fprintf(os.Stderr, "Error reading config file: %v\n", err)
		}
	}

	// Get storageDir from config or set default
	if viper.IsSet("storage_dir") {
		storageDir = viper.GetString("storage_dir")
	} else {
		// Default to config directory
		storageDir = configDir
		// Save default to config
		viper.Set("storage_dir", storageDir)
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storageDir, 0755); err != nil {
		log.Fatalf("Failed to create storage directory: %v", err)
	}

	// Initialize storage with the correct directory
	var err error
	store, err = storage.NewJSONStorage(filepath.Join(storageDir, "links.json"))
	if err != nil {
		log.Fatalf("Failed to initialize storage: %v", err)
	}
}

func init() {
	// Define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// Determine config directory
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}
	configDir = filepath.Join(homeDir, ".config", "golink")

	// Initialize config before executing commands
	cobra.OnInitialize(initConfig)

	// // Create storage
	// store, err = storage.NewJSONStorage(filepath.Join(configDir, "links.json"))
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	addCmd.Flags().StringP("description", "d", "", "Description of the link")
	addCmd.Flags().StringP("category", "c", "", "Category for the link")

	// Add flags for the serve command
	serveCmd.Flags().IntP("port", "p", 80, "Port to serve on")
	serveCmd.Flags().String("not-found", "", "URL to redirect to when a go link is not found (optional)")

	// Add direct flag to open command
	openCmd.Flags().BoolP("direct", "d", false, "Open the direct URL instead of the go/link format")

	// Add commands to root
	rootCmd.AddCommand(addCmd, listCmd, openCmd, deleteCmd, serveCmd)

	// Add config command and subcommands
	configCmd.AddCommand(setStorageDirCmd, viewConfigCmd)
	rootCmd.AddCommand(configCmd)
}
