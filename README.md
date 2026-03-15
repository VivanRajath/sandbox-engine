<div align="center">
  <h1>Sandbox Engine</h1>
  <p><b>A fast, lightweight, and secure CLI tool for running Python projects in isolated environments.</b></p>
  <p><i>Built with Go</i></p>
</div>

---

## Overview

`sandbox-engine` allows developers to instantly spin up Python applications (like FastAPI, Django, Flask, Streamlit) without corrupting their global environment or modifying their local project files. 

It automatically detects your framework, analyzes your code to dynamically generate required dependencies, strictly isolates the execution environment, and cleans up the sandbox the moment you exit.

## Installation

Install globally using `pip`:

```bash
pip install sandbox-engine-cli
```

*(Note: The PyPI package is named `sandbox-engine-cli`, but the designated terminal command is `sandbox-engine`).*

## Core Commands

Navigate to any existing Python project directory and run:

### `scan`
Instantly analyzes the project structure.
- Discovers all Python files.
- Locates potential application entrypoints (e.g., `app.py`, `main.py`).
- Identifies existing virtual environments and dependency management files.

```bash
sandbox-engine scan
```

### `detect`
Intelligently infers your application architecture.
- Auto-detects the web framework (Flask, FastAPI, Django, Streamlit, Gradio).
- Identifies the exact port the application intends to use.
- **Deep AST Analysis:** Scans your code to find all third-party imports, filtering out over 150 standard library modules to determine exactly what your application requires to run.

```bash
sandbox-engine detect
```

### `run`
Standard execution with dependency injection.
- Scans your imports and cross-verifies them against your existing `requirements.txt`.
- Automatically appends missing dependencies.
- Sets up a standard virtual environment (`venv`) and executes the application.

```bash
sandbox-engine run
```

### `run --isolated` (Recommended)
**Zero-mutation execution.** The safest method to test an untrusted or disorganized project.
- Creates a temporary, hidden `.sandbox` environment.
- Dynamically generates a `requirements.txt` internal to the sandbox.
- Safely installs dependencies strictly within the isolated environment.
- Executes the application securely.
- **Graceful Teardown:** Terminating with `Ctrl+C` immediately kills the spawned processes and destroys the sandbox directory, returning your project cleanly to its original state.

```bash
sandbox-engine run --isolated
```

## Contributing
Contributions, issues, and feature requests are welcome. Please refer to the [issues page](https://github.com/VivanRajath/sandbox-engine/issues).

---

<div align="center">
  <p><b>Created by Vivan Rajath</b></p>
  <p>Contact: vivanrajath999@gmail.com</p>
</div>
