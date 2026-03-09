# ----------------------------------------------------------------------------------------
#   web_speedtest
#   -------------
#
#   Network speed test — server with web UI and CLI client for measuring ping,
#   download, and upload speeds.
#
#   (c) 2026 WaterJuice — Released under the Unlicense; see LICENSE.
#
#   Version History
#   ---------------
#   Mar 2026 - Created
# ----------------------------------------------------------------------------------------

# ----------------------------------------------------------------------------------------
#   Version
# ----------------------------------------------------------------------------------------

from .version import VERSION_STR

__version__ = VERSION_STR
__all__ = ["__version__"]
