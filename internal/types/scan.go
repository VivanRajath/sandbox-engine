package types

type ScanResult struct {
	RepoRoot string

	PythonFiles []string

	Entrypoints []string

	DependencyFiles []string

	FileTree []string

	HasVenv  bool
	VenvPath string

	Imports []string
}