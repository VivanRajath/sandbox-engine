import os
import sys
import subprocess
import platform

def run():
    # Determine the name of the binary
    bin_name = "sandbox-engine.exe" if platform.system() == "Windows" else "sandbox-engine"
    
    # Path to the binary which was downloaded/built during setup
    bin_path = os.path.join(os.path.dirname(os.path.abspath(__file__)), "bin", bin_name)
    
    if not os.path.exists(bin_path):
        print(f"Error: Could not find sandbox-engine executable at {bin_path}")
        print("Please ensure it was built or downloaded correctly during installation.")
        sys.exit(1)
        
    # Forward all command line arguments to the Go binary
    args = [bin_path] + sys.argv[1:]
    
    try:
        # Run the binary and replace the current process (on Unix) or just wait for it (on Windows)
        if platform.system() == "Windows":
            sys.exit(subprocess.call(args))
        else:
            os.execv(bin_path, args)
    except Exception as e:
        print(f"Error executing sandbox-engine: {e}")
        sys.exit(1)
        
if __name__ == "__main__":
    run()
