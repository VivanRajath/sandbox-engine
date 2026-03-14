package detector

import (
	"os"
	"strings"

	python "sandbox-engine/internal/runtime/python"
	"sandbox-engine/internal/types"
)

func DetectFramework(scan *types.ScanResult) (*types.Project, error) {

	project := &types.Project{}

	for _, file := range scan.PythonFiles {

		content, err := os.ReadFile(file)
		if err != nil {
			continue
		}

		text := strings.ToLower(string(content))

		if strings.Contains(text, "fastapi") {
			project.Framework = "FastAPI"
			project.Port = 8000
		}

		if strings.Contains(text, "flask") {
			project.Framework = "Flask"
			project.Port = 5000
		}

		if strings.Contains(text, "django") {
			project.Framework = "Django"
			project.Port = 8000
		}

		if strings.Contains(text, "streamlit") {
			project.Framework = "Streamlit"
			project.Port = 8501
		}

		if strings.Contains(text, "gradio") {
			project.Framework = "Gradio"
			project.Port = 7860
		}

	}

	if len(scan.Entrypoints) > 0 {
		project.Entrypoint = scan.Entrypoints[0]
	}

	// Detect code-level imports
	deps, err := python.GenerateRequirements(scan)
	if err == nil {
		project.Dependencies = deps
		scan.Imports = deps
	}

	return project, nil
}