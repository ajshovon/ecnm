package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const windowsPathPrefix = `\\?\`

func normalizePath(path string) string {
	if runtime.GOOS == "windows" {
		if !strings.HasPrefix(path, windowsPathPrefix) && len(path) > 260 {
			return windowsPathPrefix + path
		}
	}
	return path
}

func retry(attempts int, sleep time.Duration, fn func() error) error {
	var lastErr error
	for i := 0; i < attempts; i++ {
		if err := fn(); err == nil {
			return nil
		} else {
			lastErr = err
			if i < attempts-1 {
				time.Sleep(sleep)
			}
		}
	}
	return lastErr
}

func removeNodeModules(dir string) error {
	return filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			normalizedPath := normalizePath(path)
			if normalizedPath != path {
				if _, err := os.Stat(normalizedPath); err == nil {
					path = normalizedPath
				} else {
					fmt.Printf("Warning: Cannot access path %q: %v\n", path, err)
					return nil
				}
			}
		}

		if d != nil && d.IsDir() && d.Name() == "node_modules" {
			fmt.Printf("Attempting to remove: %s\n", path)

			err := retry(3, time.Second*2, func() error {
				normalizedPath := normalizePath(path)
				return os.RemoveAll(normalizedPath)
			})

			if err != nil {
				fmt.Printf("Failed to remove %q: %v\n", path, err)
				subItems, readErr := os.ReadDir(normalizePath(path))
				if readErr == nil {
					for _, item := range subItems {
						subPath := filepath.Join(path, item.Name())
						normalizedSubPath := normalizePath(subPath)
						if removeErr := os.RemoveAll(normalizedSubPath); removeErr != nil {
							fmt.Printf("Failed to remove %q: %v\n", subPath, removeErr)
						}
					}
				}
				return nil
			}

			fmt.Printf("Successfully removed: %s\n", path)
			return filepath.SkipDir
		}
		return nil
	})
}

func main() {
	help := flag.Bool("h", false, "Show help information")
	flag.BoolVar(help, "help", false, "Show help information")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [OPTIONS] DIRECTORY\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "A tool to recursively remove node_modules directories.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n  %s /path/to/project\n", os.Args[0])
	}

	flag.Parse()
	args := flag.Args()

	if *help {
		flag.Usage()
		os.Exit(0)
	}

	if len(args) < 1 {
		fmt.Println("Error: Please provide a directory path to clean")
		fmt.Println("Run with -h or --help for usage information")
		os.Exit(1)
	}

	dir := args[0]
	fmt.Printf("Cleaning all 'node_modules' folders in: %s\n", dir)

	if err := removeNodeModules(dir); err != nil {
		fmt.Printf("Error during cleanup: %v\n", err)
		os.Exit(1)
	}
}
