package scanner

import (
	"os"
	"path/filepath"
	
	"sandbox-engine/internal/types"
)

var ignoreDirs = map[string]bool{
	".git": true,
	"venv": true,
	".venv": true,
	".sandbox": true,
	"node_modules": true,
	"__pycache__": true,
	"dist": true,
	"build": true,
}

var entryCandidates = map[string]bool{
	"main.py": true,
	"app.py": true,
	"server.py": true,
	"run.py": true,
	"manage.py": true,
}

var dependencyFiles = map[string]bool{
	"requirements.txt": true,
	"pyproject.toml": true,
	"Pipfile": true,
	"setup.py": true,
}

func ScanRepo(root string) (*types.ScanResult, error) {

	result := &types.ScanResult{
		RepoRoot: root,
	}

	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

		if err != nil {
			return err
		}

		name := info.Name()

		if info.IsDir() {

			// Detect existing venv
			if name == "venv" || name == ".venv" {
				result.HasVenv = true
				result.VenvPath = path
				return filepath.SkipDir
			}

			if ignoreDirs[name] {
				return filepath.SkipDir
			}

			return nil
		}

		result.FileTree = append(result.FileTree, path)

		if filepath.Ext(name) == ".py" {
			result.PythonFiles = append(result.PythonFiles, path)

			if entryCandidates[name] {
				result.Entrypoints = append(result.Entrypoints, path)
			}
		}

		if dependencyFiles[name] {
			result.DependencyFiles = append(result.DependencyFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}