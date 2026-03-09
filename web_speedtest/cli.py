# ----------------------------------------------------------------------------------------
#   cli.py
#   ------
#
#   CLI argument parsing and subcommand dispatch. Provides two modes:
#
#     server — start the speed test HTTP server with web UI
#     client — run a speed test against a server from the command line
#
#   (c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
#
#   Version History
#   ---------------
#   Mar 2026 - Created
# ----------------------------------------------------------------------------------------

# ----------------------------------------------------------------------------------------
#   Imports
# ----------------------------------------------------------------------------------------

import sys
import traceback
from collections.abc import Callable
from .argbuilder import ArgsParser
from .argbuilder import Namespace
from .client import run_client
from .server import run_server
from .version import VERSION_STR

# ----------------------------------------------------------------------------------------
#   Constants
# ----------------------------------------------------------------------------------------

# Type alias for subcommand handler functions.
_CommandHandler = Callable[[Namespace], int]

_LICENCE_TEXT = """\
web-speedtest — Released under the Unlicense (public domain)

This is free and unencumbered software released into the public domain.

Anyone is free to copy, modify, publish, use, compile, sell, or
distribute this software, either in source code form or as a compiled
binary, for any purpose, commercial or non-commercial, and by any
means.

For more information, please refer to <https://unlicense.org/>
"""

# ----------------------------------------------------------------------------------------
#   Argument Parser
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def _create_parser() -> ArgsParser:
    """Build the argument parser with subcommands."""
    parser = ArgsParser(
        prog="web-speedtest",
        description=(
            "Network speed test — server with web UI and CLI client.\n"
            "If no command is given, 'client' is assumed."
        ),
        version=f"web-speedtest: {VERSION_STR}\npython: {sys.version.split()[0]}",
        default_command="client",
    )

    # Top-level options -------------------------------------------------------
    parser.add_argument(
        "--license",
        action="store_true",
        dest="license",
        help="Show license information and exit",
    )

    # server -----------------------------------------------------------------
    server_cmd = parser.add_command(
        "server",
        help="Start the speed test HTTP server",
    )
    server_cmd.add_argument(
        "--host",
        "-H",
        default="0.0.0.0",
        metavar="HOST",
        help="Address to bind to (default: 0.0.0.0)",
    )
    server_cmd.add_argument(
        "--port",
        "-p",
        type=int,
        default=8080,
        metavar="PORT",
        help="Port to listen on (default: 8080)",
    )
    server_cmd.add_argument(
        "--name",
        "-n",
        default="Speed Test",
        metavar="NAME",
        help="Server display name shown in the web UI (default: Speed Test)",
    )
    server_cmd.add_argument(
        "--quiet",
        "-q",
        action="store_true",
        help="Suppress request logging",
    )

    # client -----------------------------------------------------------------
    client_cmd = parser.add_command(
        "client",
        help="Run a speed test against a server (default)",
    )
    client_cmd.add_argument(
        "server_url",
        metavar="SERVER",
        help="Server URL or host:port to test against (e.g. http://localhost:8080)",
    )
    client_cmd.add_argument(
        "--compact",
        "-c",
        action="store_true",
        help="Compact one-line output",
    )
    client_cmd.add_argument(
        "--json",
        "-j",
        action="store_true",
        dest="json_output",
        help="Output results as JSON",
    )

    return parser


# ----------------------------------------------------------------------------------------
#   Subcommand Handlers
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def _cmd_server(args: Namespace) -> int:
    """Start the speed test server."""
    host: str = args.host
    port: int = args.port
    name: str = args.name
    quiet: bool = args.quiet
    return run_server(host, port, verbose=not quiet, name=name)


# ----------------------------------------------------------------------------------------
def _cmd_client(args: Namespace) -> int:
    """Run a speed test against a server."""
    server_url: str = args.server_url
    compact: bool = args.compact
    json_output: bool = args.json_output
    if json_output:
        return run_client(server_url, output="json")
    if compact:
        return run_client(server_url, output="compact")
    return run_client(server_url, output="normal")


# ----------------------------------------------------------------------------------------
#   Main Entry Point
# ----------------------------------------------------------------------------------------


# ----------------------------------------------------------------------------------------
def main() -> int:
    """Entry point: parse arguments and dispatch to subcommand."""
    try:
        return _main_inner()
    except KeyboardInterrupt:
        return 0
    except SystemExit:
        raise
    except BaseException as e:
        t = "-------------------------------------------------------------------\n"
        t += "UNHANDLED EXCEPTION OCCURRED!!\n"
        t += "\n"
        t += traceback.format_exc()
        t += "\n"
        t += f"EXCEPTION: {type(e)} {e}\n"
        t += "-------------------------------------------------------------------\n"
        print(t, file=sys.stderr)
        return 1


# ----------------------------------------------------------------------------------------
def _main_inner() -> int:
    """Inner main function that does the actual work."""
    # Handle --license before parsing (no command needed).
    if "--license" in sys.argv:
        print(_LICENCE_TEXT)
        return 0

    parser = _create_parser()
    args: Namespace = parser.parse()

    commands: dict[str, _CommandHandler] = {
        "server": _cmd_server,
        "client": _cmd_client,
    }

    return commands[args.command](args)
