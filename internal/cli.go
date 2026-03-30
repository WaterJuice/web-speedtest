// ---------------------------------------------------------------------------------------
//
//	cli.go
//	------
//
//	CLI argument parsing, help text, and subcommand dispatch. Provides two modes:
//
//	  server — start the speed test HTTP server with web UI
//	  client — run a speed test against a server from the command line
//
//	(c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
//
//	Version History
//	---------------
//	Mar 2026 - Created (Python)
//	Mar 2026 - Rewritten in Go
//
// ---------------------------------------------------------------------------------------
package internal

// ---------------------------------------------------------------------------------------
//
//	Imports
//
// ---------------------------------------------------------------------------------------

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ---------------------------------------------------------------------------------------
//
//	Constants
//
// ---------------------------------------------------------------------------------------

const licenceText = `web-speedtest — Released under the Unlicense (public domain)

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

For more information, please refer to <https://unlicense.org/>
`

// ---------------------------------------------------------------------------------------
//
//	Run
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// Run is the main entry point called from main.go.
func Run(version string) {
	args := os.Args[1:]

	if len(args) == 0 {
		printUsage()
		os.Exit(0)
	}

	switch args[0] {
	case "--help", "-h":
		printUsage()
	case "--version":
		fmt.Printf("web-speedtest: %s\n", version)
	case "--license":
		fmt.Print(licenceText)
	case "server":
		runServerCmd(version, args[1:])
	case "client":
		runClientCmd(args[1:])
	default:
		// If first arg doesn't look like a flag, assume client mode
		if !strings.HasPrefix(args[0], "-") {
			runClientCmd(args)
			return
		}
		fmt.Fprintf(os.Stderr, "Unknown option: %s\n", args[0])
		fmt.Fprintln(os.Stderr, "Run with --help for usage")
		os.Exit(1)
	}
}

// ---------------------------------------------------------------------------------------
//
//	Help Text
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// printUsage prints the main help text showing available subcommands and global options.
// Output is coloured with ANSI codes when stdout is a TTY.
func printUsage() {
	tty := isStdoutTerminal()
	if tty {
		h := "\033[1;34m" // heading: bold blue
		p := "\033[1;35m" // program: bold magenta
		s := "\033[32m"   // short flag: green
		l := "\033[36m"   // long flag: cyan
		m := "\033[33m"   // metavar: yellow
		S := "\033[1;32m" // short flag bold: bold green
		L := "\033[1;36m" // long flag bold: bold cyan
		r := "\033[0m"    // reset

		fmt.Printf("%susage: %s%sweb-speedtest%s [%s-h%s] [%s--version%s] [%s--license%s] {%sserver%s,%sclient%s} ...\n",
			h, r, p, r, s, r, l, r, l, r, m, r, m, r)
		fmt.Println()
		fmt.Println("Network speed test — server with web UI and CLI client.")
		fmt.Println("If no command is given, 'client' is assumed.")
		fmt.Println()
		fmt.Printf("%scommands:%s\n", h, r)
		fmt.Printf("  %sserver%s              Start the speed test HTTP server\n", L, r)
		fmt.Printf("  %sclient%s              Run a speed test against a server (default)\n", L, r)
		fmt.Println()
		fmt.Printf("%soptions:%s\n", h, r)
		fmt.Printf("  %s-h%s, %s--help%s          show this help message and exit\n", S, r, L, r)
		fmt.Printf("  %s--version%s           show version and exit\n", L, r)
		fmt.Printf("  %s--license%s           show licence information and exit\n", L, r)
	} else {
		fmt.Println("usage: web-speedtest [-h] [--version] [--license] {server,client} ...")
		fmt.Println()
		fmt.Println("Network speed test — server with web UI and CLI client.")
		fmt.Println("If no command is given, 'client' is assumed.")
		fmt.Println()
		fmt.Println("commands:")
		fmt.Println("  server              Start the speed test HTTP server")
		fmt.Println("  client              Run a speed test against a server (default)")
		fmt.Println()
		fmt.Println("options:")
		fmt.Println("  -h, --help          show this help message and exit")
		fmt.Println("  --version           show version and exit")
		fmt.Println("  --license           show licence information and exit")
	}
}

// ---------------------------------------------------------------------------------------
// printServerUsage prints the help text for the "server" subcommand, listing all
// available options (--host, --port, --name, --duration, --quiet).
func printServerUsage() {
	tty := isStdoutTerminal()
	if tty {
		h := "\033[1;34m"
		p := "\033[1;35m"
		s := "\033[32m"
		m := "\033[33m"
		S := "\033[1;32m"
		L := "\033[1;36m"
		M := "\033[1;33m"
		r := "\033[0m"

		fmt.Printf("%susage: %s%sweb-speedtest server%s [%s-H%s %sHOST%s] [%s-p%s %sPORT%s] [%s-n%s %sNAME%s] [%s-d%s %sSECS%s] [%s-q%s]\n",
			h, r, p, r, s, r, m, r, s, r, m, r, s, r, m, r, s, r, m, r, s, r)
		fmt.Println()
		fmt.Println("Start the speed test HTTP server.")
		fmt.Println()
		fmt.Printf("%soptions:%s\n", h, r)
		fmt.Printf("  %s-h%s, %s--help%s                show this help message and exit\n", S, r, L, r)
		fmt.Printf("  %s-H%s, %s--host%s %sHOST%s           address to bind to (default: 0.0.0.0)\n", S, r, L, r, M, r)
		fmt.Printf("  %s-p%s, %s--port%s %sPORT%s           port to listen on (default: 8080)\n", S, r, L, r, M, r)
		fmt.Printf("  %s-n%s, %s--name%s %sNAME%s           server display name (default: Speed Test)\n", S, r, L, r, M, r)
		fmt.Printf("  %s-d%s, %s--duration%s %sSECONDS%s    duration of each test phase (default: 8)\n", S, r, L, r, M, r)
		fmt.Printf("  %s-q%s, %s--quiet%s               suppress request logging\n", S, r, L, r)
	} else {
		fmt.Println("usage: web-speedtest server [-H HOST] [-p PORT] [-n NAME] [-d SECS] [-q]")
		fmt.Println()
		fmt.Println("Start the speed test HTTP server.")
		fmt.Println()
		fmt.Println("options:")
		fmt.Println("  -h, --help                show this help message and exit")
		fmt.Println("  -H, --host HOST           address to bind to (default: 0.0.0.0)")
		fmt.Println("  -p, --port PORT           port to listen on (default: 8080)")
		fmt.Println("  -n, --name NAME           server display name (default: Speed Test)")
		fmt.Println("  -d, --duration SECONDS    duration of each test phase (default: 8)")
		fmt.Println("  -q, --quiet               suppress request logging")
	}
}

// ---------------------------------------------------------------------------------------
// printClientUsage prints the help text for the "client" subcommand, listing the
// required SERVER argument and available options (--compact, --json).
func printClientUsage() {
	tty := isStdoutTerminal()
	if tty {
		h := "\033[1;34m"
		p := "\033[1;35m"
		s := "\033[32m"
		m := "\033[33m"
		S := "\033[1;32m"
		L := "\033[1;36m"
		M := "\033[1;33m"
		r := "\033[0m"

		fmt.Printf("%susage: %s%sweb-speedtest client%s [%s-c%s] [%s-j%s] %sSERVER%s\n",
			h, r, p, r, s, r, s, r, m, r)
		fmt.Println()
		fmt.Println("Run a speed test against a server.")
		fmt.Println()
		fmt.Printf("%sarguments:%s\n", h, r)
		fmt.Printf("  %sSERVER%s                server URL or host:port\n", M, r)
		fmt.Println()
		fmt.Printf("%soptions:%s\n", h, r)
		fmt.Printf("  %s-h%s, %s--help%s            show this help message and exit\n", S, r, L, r)
		fmt.Printf("  %s-c%s, %s--compact%s          compact one-line output\n", S, r, L, r)
		fmt.Printf("  %s-j%s, %s--json%s             output results as JSON\n", S, r, L, r)
	} else {
		fmt.Println("usage: web-speedtest client [-c] [-j] SERVER")
		fmt.Println()
		fmt.Println("Run a speed test against a server.")
		fmt.Println()
		fmt.Println("arguments:")
		fmt.Println("  SERVER                server URL or host:port")
		fmt.Println()
		fmt.Println("options:")
		fmt.Println("  -h, --help            show this help message and exit")
		fmt.Println("  -c, --compact         compact one-line output")
		fmt.Println("  -j, --json            output results as JSON")
	}
}

// ---------------------------------------------------------------------------------------
//
//	Subcommand Handlers
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// runServerCmd parses the "server" subcommand arguments and starts the HTTP server.
// Calls os.Exit with the server's return code.
func runServerCmd(version string, args []string) {
	host := "0.0.0.0"
	port := 8080
	name := "Speed Test"
	duration := defaultTestDuration
	quiet := false

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--help":
			printServerUsage()
			os.Exit(0)
		case "-H", "--host":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --host requires a value")
				os.Exit(1)
			}
			host = args[i]
		case "-p", "--port":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --port requires a value")
				os.Exit(1)
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: --port must be a number")
				os.Exit(1)
			}
			port = n
		case "-n", "--name":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --name requires a value")
				os.Exit(1)
			}
			name = args[i]
		case "-d", "--duration":
			i++
			if i >= len(args) {
				fmt.Fprintln(os.Stderr, "Error: --duration requires a value")
				os.Exit(1)
			}
			n, err := strconv.Atoi(args[i])
			if err != nil {
				fmt.Fprintln(os.Stderr, "Error: --duration must be a number")
				os.Exit(1)
			}
			duration = n
		case "-q", "--quiet":
			quiet = true
		default:
			fmt.Fprintf(os.Stderr, "Unknown option: %s\n", args[i])
			fmt.Fprintln(os.Stderr, "Run 'web-speedtest server --help' for usage")
			os.Exit(1)
		}
		i++
	}

	os.Exit(startServer(version, host, port, name, duration, !quiet))
}

// ---------------------------------------------------------------------------------------
// runClientCmd parses the "client" subcommand arguments (SERVER, --compact, --json)
// and runs the speed test. Calls os.Exit with the client's return code.
func runClientCmd(args []string) {
	var serverURL string
	compact := false
	jsonOutput := false

	i := 0
	for i < len(args) {
		switch args[i] {
		case "-h", "--help":
			printClientUsage()
			os.Exit(0)
		case "-c", "--compact":
			compact = true
		case "-j", "--json":
			jsonOutput = true
		default:
			if strings.HasPrefix(args[i], "-") {
				fmt.Fprintf(os.Stderr, "Unknown option: %s\n", args[i])
				fmt.Fprintln(os.Stderr, "Run 'web-speedtest client --help' for usage")
				os.Exit(1)
			}
			if serverURL != "" {
				fmt.Fprintln(os.Stderr, "Error: multiple server URLs given")
				os.Exit(1)
			}
			serverURL = args[i]
		}
		i++
	}

	if serverURL == "" {
		fmt.Fprintln(os.Stderr, "Error: server URL is required")
		fmt.Fprintln(os.Stderr, "Run 'web-speedtest client --help' for usage")
		os.Exit(1)
	}

	os.Exit(runClient(serverURL, compact, jsonOutput))
}

// ---------------------------------------------------------------------------------------
//
//	Terminal / Colour Helpers
//
// ---------------------------------------------------------------------------------------

// ---------------------------------------------------------------------------------------
// isStdoutTerminal reports whether stdout is connected to a terminal (TTY).
func isStdoutTerminal() bool {
	info, err := os.Stdout.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

// ---------------------------------------------------------------------------------------
// colorBold wraps s in ANSI bold codes if tty is true, otherwise returns s unchanged.
func colorBold(s string, tty bool) string {
	if !tty {
		return s
	}
	return "\033[1m" + s + "\033[0m"
}

// ---------------------------------------------------------------------------------------
// colorDim wraps s in ANSI dim codes if tty is true, otherwise returns s unchanged.
func colorDim(s string, tty bool) string {
	if !tty {
		return s
	}
	return "\033[2m" + s + "\033[0m"
}

// ---------------------------------------------------------------------------------------
// colorGreen wraps s in ANSI green codes if tty is true, otherwise returns s unchanged.
func colorGreen(s string, tty bool) string {
	if !tty {
		return s
	}
	return "\033[32m" + s + "\033[0m"
}

// ---------------------------------------------------------------------------------------
// colorCyan wraps s in ANSI cyan codes if tty is true, otherwise returns s unchanged.
func colorCyan(s string, tty bool) string {
	if !tty {
		return s
	}
	return "\033[36m" + s + "\033[0m"
}

// ---------------------------------------------------------------------------------------
// colorYellow wraps s in ANSI yellow codes if tty is true, otherwise returns s unchanged.
func colorYellow(s string, tty bool) string {
	if !tty {
		return s
	}
	return "\033[33m" + s + "\033[0m"
}

// ---------------------------------------------------------------------------------------
// colorRed wraps s in ANSI red codes if tty is true, otherwise returns s unchanged.
func colorRed(s string, tty bool) string {
	if !tty {
		return s
	}
	return "\033[31m" + s + "\033[0m"
}
