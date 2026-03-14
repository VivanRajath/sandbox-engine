import os
import sys
import platform
import urllib.request
import zipfile
import tarfile
import subprocess
from setuptools import setup, find_packages
from setuptools.command.build_py import build_py
from setuptools.command.install import install

# --- CONFIGURATION ---
PACKAGE_NAME = "sandbox-engine"
VERSION = "1.0.0"
GITHUB_REPO = "VivanRajath/sandbox-engine"  # Just the username/repo format
# ---------------------

def get_binary_name():
    return f"{PACKAGE_NAME}.exe" if platform.system() == "Windows" else PACKAGE_NAME

def get_download_url():
    """Generates the GitHub Release URL matching the current OS and architecture."""
    system = platform.system().lower()
    machine = platform.machine().lower()
    
    # Normalize architecture names
    if machine in ["x86_64", "amd64"]:
        arch = "amd64"
    elif machine in ["arm64", "aarch64"]:
        arch = "arm64"
    else:
        raise Exception(f"Unsupported architecture: {machine}")
        
    if system == "windows":
        filename = f"{PACKAGE_NAME}-windows-{arch}.exe"
    elif system == "darwin":
        filename = f"{PACKAGE_NAME}-darwin-{arch}"
    elif system == "linux":
        filename = f"{PACKAGE_NAME}-linux-{arch}"
    else:
        raise Exception(f"Unsupported OS: {system}")
        
    return f"https://github.com/{GITHUB_REPO}/releases/download/v{VERSION}/{filename}"

def build_or_download_binary():
    """Tries to build from source using Go. If Go is missing, downloads precompiled binary."""
    bin_name = get_binary_name()
    bin_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "sandbox_engine", "bin")
    
    os.makedirs(bin_path, exist_ok=True)
    target_bin = os.path.join(bin_path, bin_name)
    
    # Check if we already built/downloaded it
    if os.path.exists(target_bin):
        return target_bin
        
    print("Attempting to compile Go source...")
    try:
        # Try to build locally
        subprocess.check_call(["go", "build", "-o", target_bin, "./cmd/sandbox-engine"])
        print("Successfully compiled Go binary.")
        return target_bin
    except (subprocess.CalledProcessError, FileNotFoundError):
        print("Go build failed or Go is not installed. Falling back to pre-compiled binary download...")
        
    # Download fallback
    try:
        url = get_download_url()
        print(f"Downloading from {url}...")
        urllib.request.urlretrieve(url, target_bin)
        
        # Make executable on Unix
        if platform.system() != "Windows":
            os.chmod(target_bin, 0o755)
            
        print("Successfully downloaded pre-compiled binary.")
        return target_bin
    except Exception as e:
        print(f"Failed to download binary: {e}")
        print("\nPlease install Go (https://golang.org/dl/) to compile from source, or check your internet connection.")
        sys.exit(1)

class CustomBuildPy(build_py):
    def run(self):
        # Build or download binary BEFORE python package build
        build_or_download_binary()
        super().run()

class CustomInstall(install):
    def run(self):
        super().run()

setup(
    name=PACKAGE_NAME,
    version=VERSION,
    description="A CLI tool to run Python projects in sandboxed environments.",
    long_description=open("README.md", "r", encoding="utf-8").read() if os.path.exists("README.md") else "",
    long_description_content_type="text/markdown",
    author="Vivan Rajath",
    author_email="your.email@example.com",  # REPLACE THIS
    url=f"https://github.com/{GITHUB_REPO}",
    packages=["sandbox_engine"],
    package_data={
        "sandbox_engine": ["bin/*", "bin/*.exe"],
    },
    include_package_data=True,
    # This exposes the binary directly to the python Scripts/bin path
    entry_points={
        "console_scripts": [
            f"{PACKAGE_NAME}=sandbox_engine.cli:run",
        ],
    },
    cmdclass={
        "build_py": CustomBuildPy,
        "install": CustomInstall,
    },
    python_requires=">=3.8",
)
