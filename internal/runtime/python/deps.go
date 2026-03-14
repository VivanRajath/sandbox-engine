package runtime

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"sandbox-engine/internal/types"
)

// Python standard library modules — these should NOT appear in requirements.txt
var stdlibModules = map[string]bool{
	"abc": true, "aifc": true, "argparse": true, "array": true, "ast": true,
	"asyncio": true, "atexit": true, "base64": true, "binascii": true, "binhex": true,
	"bisect": true, "builtins": true, "bz2": true, "calendar": true, "cgi": true,
	"cgitb": true, "chunk": true, "cmath": true, "cmd": true, "code": true,
	"codecs": true, "codeop": true, "collections": true, "colorsys": true,
	"compileall": true, "concurrent": true, "configparser": true, "contextlib": true,
	"contextvars": true, "copy": true, "copyreg": true, "cProfile": true, "crypt": true,
	"csv": true, "ctypes": true, "curses": true, "dataclasses": true, "datetime": true,
	"dbm": true, "decimal": true, "difflib": true, "dis": true, "distutils": true,
	"doctest": true, "email": true, "encodings": true, "enum": true, "errno": true,
	"faulthandler": true, "fcntl": true, "filecmp": true, "fileinput": true,
	"fnmatch": true, "fractions": true, "ftplib": true, "functools": true, "gc": true,
	"getopt": true, "getpass": true, "gettext": true, "glob": true, "grp": true,
	"gzip": true, "hashlib": true, "heapq": true, "hmac": true, "html": true,
	"http": true, "idlelib": true, "imaplib": true, "imghdr": true, "imp": true,
	"importlib": true, "inspect": true, "io": true, "ipaddress": true, "itertools": true,
	"json": true, "keyword": true, "lib2to3": true, "linecache": true, "locale": true,
	"logging": true, "lzma": true, "mailbox": true, "mailcap": true, "marshal": true,
	"math": true, "mimetypes": true, "mmap": true, "modulefinder": true,
	"multiprocessing": true, "netrc": true, "nis": true, "nntplib": true, "numbers": true,
	"operator": true, "optparse": true, "os": true, "ossaudiodev": true,
	"pathlib": true, "pdb": true, "pickle": true, "pickletools": true, "pipes": true,
	"pkgutil": true, "platform": true, "plistlib": true, "poplib": true, "posix": true,
	"posixpath": true, "pprint": true, "profile": true, "pstats": true, "pty": true,
	"pwd": true, "py_compile": true, "pyclbr": true, "pydoc": true, "queue": true,
	"quopri": true, "random": true, "re": true, "readline": true, "reprlib": true,
	"resource": true, "rlcompleter": true, "runpy": true, "sched": true, "secrets": true,
	"select": true, "selectors": true, "shelve": true, "shlex": true, "shutil": true,
	"signal": true, "site": true, "smtpd": true, "smtplib": true, "sndhdr": true,
	"socket": true, "socketserver": true, "spwd": true, "sqlite3": true, "sre_compile": true,
	"sre_constants": true, "sre_parse": true, "ssl": true, "stat": true,
	"statistics": true, "string": true, "stringprep": true, "struct": true,
	"subprocess": true, "sunau": true, "symtable": true, "sys": true, "sysconfig": true,
	"syslog": true, "tabnanny": true, "tarfile": true, "telnetlib": true, "tempfile": true,
	"termios": true, "test": true, "textwrap": true, "threading": true, "time": true,
	"timeit": true, "tkinter": true, "token": true, "tokenize": true, "tomllib": true,
	"trace": true, "traceback": true, "tracemalloc": true, "tty": true, "turtle": true,
	"turtledemo": true, "types": true, "typing": true, "unicodedata": true,
	"unittest": true, "urllib": true, "uu": true, "uuid": true, "venv": true,
	"warnings": true, "wave": true, "weakref": true, "webbrowser": true, "winreg": true,
	"winsound": true, "wsgiref": true, "xdrlib": true, "xml": true, "xmlrpc": true,
	"zipapp": true, "zipfile": true, "zipimport": true, "zlib": true, "_thread": true,
}

func GenerateRequirements(scan *types.ScanResult) ([]string, error) {

	deps := map[string]bool{}

	importRegex := regexp.MustCompile(`(?m)^\s*(?:import|from)\s+([a-zA-Z0-9_]+)`)

	for _, file := range scan.PythonFiles {

		f, err := os.Open(file)
		if err != nil {
			continue
		}

		scanner := bufio.NewScanner(f)

		for scanner.Scan() {

			line := scanner.Text()

			match := importRegex.FindStringSubmatch(line)

			if len(match) > 1 {

				pkg := match[1]

				if !stdlibModules[pkg] {
					deps[pkg] = true
				}
			}
		}

		f.Close()
	}

	var list []string

	for d := range deps {
		list = append(list, d)
	}

	sort.Strings(list)

	return list, nil
}

// ReadRequirements reads an existing requirements.txt and returns a list of package names (without versions)
func ReadRequirements(path string) ([]string, error) {

	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var pkgs []string
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// strip version specifiers: flask>=2.0 -> flask
		for _, sep := range []string{">=", "<=", "==", "!=", "~=", ">", "<"} {
			if idx := strings.Index(line, sep); idx > 0 {
				line = line[:idx]
				break
			}
		}

		pkgs = append(pkgs, strings.TrimSpace(strings.ToLower(line)))
	}

	return pkgs, nil
}

// CrossVerifyRequirements compares detected imports against existing requirements.txt
// returns the list of missing packages that need to be added
func CrossVerifyRequirements(reqPath string, detectedDeps []string) ([]string, error) {

	existing, err := ReadRequirements(reqPath)
	if err != nil {
		return nil, err
	}

	existingSet := map[string]bool{}
	for _, pkg := range existing {
		existingSet[pkg] = true
	}

	var missing []string
	for _, dep := range detectedDeps {
		if !existingSet[strings.ToLower(dep)] {
			missing = append(missing, dep)
		}
	}

	return missing, nil
}

// AppendRequirements appends missing packages to an existing requirements.txt
func AppendRequirements(path string, deps []string) error {

	f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return err
	}
	defer f.Close()

	for _, d := range deps {
		_, err := f.WriteString(d + "\n")
		if err != nil {
			return err
		}
	}

	fmt.Printf("Added %d missing packages to %s\n", len(deps), path)
	return nil
}

func WriteRequirements(deps []string) error {
	return WriteRequirementsTo("requirements.txt", deps)
}

// WriteRequirementsTo writes dependencies to a requirements file at the given path
func WriteRequirementsTo(path string, deps []string) error {

	file, err := os.Create(path)
	if err != nil {
		return err
	}

	defer file.Close()

	for _, d := range deps {

		_, err := file.WriteString(d + "\n")
		if err != nil {
			return err
		}
	}

	fmt.Printf("Generated %s\n", path)

	return nil
}