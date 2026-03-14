package runtime

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"

	"sandbox-engine/internal/types"
)

// pythonBin returns the path to the python binary inside a venv
func pythonBin(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "python.exe")
	}
	return filepath.Join(venvDir, "bin", "python")
}

// pipBin returns the path to the pip binary inside a venv
func pipBin(venvDir string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", "pip.exe")
	}
	return filepath.Join(venvDir, "bin", "pip")
}

// frameworkBin returns the path to a framework CLI tool inside a venv (e.g. streamlit, uvicorn)
func frameworkBin(venvDir string, name string) string {
	if runtime.GOOS == "windows" {
		return filepath.Join(venvDir, "Scripts", name+".exe")
	}
	return filepath.Join(venvDir, "bin", name)
}

// createVenv creates a virtual environment at the given path
func createVenv(venvDir string) error {
	fmt.Printf("Creating virtual environment at %s...\n", venvDir)

	cmd := exec.Command("python", "-m", "venv", venvDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// installFromRequirements installs packages from a requirements file using pip
func installFromRequirements(venvDir string, reqPath string) error {
	if _, err := os.Stat(reqPath); os.IsNotExist(err) {
		fmt.Println("No requirements.txt found, skipping dependency install")
		return nil
	}

	fmt.Printf("Installing dependencies from %s...\n", reqPath)

	cmd := exec.Command(
		pipBin(venvDir),
		"install",
		"-r",
		reqPath,
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// buildRunCommand creates the exec.Cmd to run the application based on framework
func buildRunCommand(project *types.Project, venvDir string) *exec.Cmd {

	python := pythonBin(venvDir)

	switch project.Framework {

	case "FastAPI":
		// Extract module name from entrypoint: "app.py" -> "app:app"
		entry := strings.TrimSuffix(filepath.Base(project.Entrypoint), ".py") + ":app"
		return exec.Command(
			python,
			"-m",
			"uvicorn",
			entry,
			"--reload",
			"--port", fmt.Sprintf("%d", project.Port),
		)

	case "Flask":
		return exec.Command(python, project.Entrypoint)

	case "Django":
		return exec.Command(
			python,
			"manage.py",
			"runserver",
			fmt.Sprintf("0.0.0.0:%d", project.Port),
		)

	case "Streamlit":
		return exec.Command(
			frameworkBin(venvDir, "streamlit"),
			"run",
			project.Entrypoint,
			"--server.port", fmt.Sprintf("%d", project.Port),
		)

	case "Gradio":
		return exec.Command(python, project.Entrypoint)

	default:
		return exec.Command(python, project.Entrypoint)
	}
}

// Run runs the Python application.
// If isolated is true: creates a temporary .sandbox venv, runs from there, destroys on Ctrl+C.
// If isolated is false: creates/reuses venv at project root, manages requirements.txt, runs normally.
func Run(project *types.Project, scan *types.ScanResult, isolated bool) error {

	if isolated {
		return runIsolated(project, scan)
	}
	return runNormal(project, scan)
}

// runNormal creates a venv at the project root and manages requirements
func runNormal(project *types.Project, scan *types.ScanResult) error {

	venvDir := "venv"

	fmt.Println("Preparing environment...")

	// Create venv if it doesn't exist
	if _, err := os.Stat(venvDir); os.IsNotExist(err) {
		err := createVenv(venvDir)
		if err != nil {
			return fmt.Errorf("failed to create virtual environment: %w", err)
		}
	} else {
		fmt.Println("Using existing virtual environment")
	}

	// Handle requirements.txt
	if _, err := os.Stat("requirements.txt"); os.IsNotExist(err) {
		// No requirements.txt — generate one from detected imports
		if len(project.Dependencies) > 0 {
			fmt.Println("No requirements.txt found, generating from detected imports...")
			err := WriteRequirements(project.Dependencies)
			if err != nil {
				return fmt.Errorf("failed to write requirements.txt: %w", err)
			}
		}
	} else {
		// requirements.txt exists — cross-verify with detected imports
		if len(project.Dependencies) > 0 {
			missing, err := CrossVerifyRequirements("requirements.txt", project.Dependencies)
			if err != nil {
				fmt.Println("Warning: could not cross-verify requirements:", err)
			} else if len(missing) > 0 {
				fmt.Printf("Found %d missing packages: %s\n", len(missing), strings.Join(missing, ", "))
				err := AppendRequirements("requirements.txt", missing)
				if err != nil {
					return fmt.Errorf("failed to update requirements.txt: %w", err)
				}
			} else {
				fmt.Println("requirements.txt is up to date")
			}
		}
	}

	// Install dependencies
	err := installFromRequirements(venvDir, "requirements.txt")
	if err != nil {
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Run the application
	fmt.Printf("Starting %s application on port %d...\n", project.Framework, project.Port)

	cmd := buildRunCommand(project, venvDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

// runIsolated creates a temporary .sandbox venv and destroys it on exit
func runIsolated(project *types.Project, scan *types.ScanResult) error {

	sandboxDir := ".sandbox"

	fmt.Println("Preparing isolated sandbox...")

	// Create the sandbox venv
	if _, err := os.Stat(sandboxDir); os.IsNotExist(err) {
		err := createVenv(sandboxDir)
		if err != nil {
			return fmt.Errorf("failed to create sandbox: %w", err)
		}
	}

	// Set up Ctrl+C handler to clean up the sandbox
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	cleanup := func() {
		fmt.Println("\nCleaning up sandbox...")
		err := os.RemoveAll(sandboxDir)
		if err != nil {
			fmt.Println("Warning: failed to remove sandbox:", err)
		} else {
			fmt.Println("Sandbox destroyed")
		}
	}

	// Build requirements inside sandbox — never write to project root
	sandboxReqPath := filepath.Join(sandboxDir, "requirements.txt")

	if _, err := os.Stat("requirements.txt"); err == nil {
		// Project has a requirements.txt — copy it into sandbox and cross-verify
		existingData, err := os.ReadFile("requirements.txt")
		if err != nil {
			cleanup()
			return fmt.Errorf("failed to read requirements.txt: %w", err)
		}
		err = os.WriteFile(sandboxReqPath, existingData, 0644)
		if err != nil {
			cleanup()
			return fmt.Errorf("failed to copy requirements.txt to sandbox: %w", err)
		}

		// Cross-verify and append missing deps into the sandbox copy
		if len(project.Dependencies) > 0 {
			missing, err := CrossVerifyRequirements(sandboxReqPath, project.Dependencies)
			if err != nil {
				fmt.Println("Warning: could not cross-verify requirements:", err)
			} else if len(missing) > 0 {
				fmt.Printf("Found %d missing packages: %s\n", len(missing), strings.Join(missing, ", "))
				err := AppendRequirements(sandboxReqPath, missing)
				if err != nil {
					cleanup()
					return fmt.Errorf("failed to update sandbox requirements: %w", err)
				}
			}
		}
	} else {
		// No requirements.txt — generate one inside sandbox from detected imports
		if len(project.Dependencies) > 0 {
			fmt.Println("No requirements.txt found, generating from detected imports...")
			err := WriteRequirementsTo(sandboxReqPath, project.Dependencies)
			if err != nil {
				cleanup()
				return fmt.Errorf("failed to write requirements to sandbox: %w", err)
			}
		}
	}

	// Install dependencies in sandbox
	err := installFromRequirements(sandboxDir, sandboxReqPath)
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to install dependencies: %w", err)
	}

	// Run the application
	fmt.Printf("Starting %s application in sandbox on port %d...\n", project.Framework, project.Port)
	fmt.Println("Press Ctrl+C to stop and destroy the sandbox")

	cmd := buildRunCommand(project, sandboxDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	// Start the process
	err = cmd.Start()
	if err != nil {
		cleanup()
		return fmt.Errorf("failed to start application: %w", err)
	}

	// Wait for either the process to finish or Ctrl+C
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-sigChan:
		// Ctrl+C received — kill the process and clean up
		fmt.Println("\nReceived interrupt signal...")
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		cleanup()
		return nil

	case err := <-done:
		// Process finished on its own
		cleanup()
		if err != nil {
			return fmt.Errorf("application exited with error: %w", err)
		}
		return nil
	}
}