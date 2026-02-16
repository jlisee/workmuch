# Copyright (c) 2026 Joseph Lisee <jlisee@gmail.com>
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.
#
# Author: Joseph Lisee <jlisee@gmail.com>
# File:  usageinfo_macos.py

# Python Imports
import logging
import os
import subprocess
import traceback

# Project Imports
from workmuch.usageinfo_base import UsageInfoBackend
from workmuch.usageinfo_macos_native import UsageInfoMacOSNative


class UsageInfoMacOSSubprocess(UsageInfoBackend):
    """Legacy subprocess backend for macOS."""

    _CMD_APP_NAME = [
        "osascript",
        "-e",
        'tell application "System Events" to get name of first application process whose frontmost is true',
    ]

    _CMD_WINDOW_TITLE = [
        "osascript",
        "-e",
        'tell application "System Events" to tell (first application process whose frontmost is true) to get value of attribute "AXTitle" of front window',
    ]

    _CMD_IDLE_NS = ["ioreg", "-c", "IOHIDSystem"]

    def getUsageInfo(self):
        try:
            return self._doGetUsageInfo()
        except Exception:
            # Keep parity with Linux backend behavior: never crash the main loop
            # from backend query failures.
            logging.error(traceback.format_exc())
            return "", "", 0.0

    def _doGetUsageInfo(self):
        progName = self._run_and_get_output(self._CMD_APP_NAME)
        winTitle = self._run_and_get_output(self._CMD_WINDOW_TITLE)
        timeIdle = self._get_idle_seconds()
        return winTitle, progName, timeIdle

    def _run_and_get_output(self, cmd):
        try:
            proc = subprocess.run(
                cmd,
                capture_output=True,
                text=True,
                check=False,
                timeout=1.0,
            )
        except (OSError, subprocess.TimeoutExpired):
            return ""

        if proc.returncode != 0:
            return ""
        return proc.stdout.strip()

    def _get_idle_seconds(self):
        try:
            proc = subprocess.run(
                self._CMD_IDLE_NS,
                capture_output=True,
                text=True,
                check=False,
                timeout=1.0,
            )
        except (OSError, subprocess.TimeoutExpired):
            return 0.0

        if proc.returncode != 0:
            return 0.0

        for line in proc.stdout.splitlines():
            if "HIDIdleTime" not in line:
                continue

            # Ex: '"HIDIdleTime" = 28600810750'
            value = line.split("=")[-1].strip()
            try:
                return float(value) / 1e9
            except ValueError:
                return 0.0

        return 0.0

    def release(self):
        # No persistent native handles with this backend.
        return None

    def reset(self):
        # No persistent native handles with this backend.
        return None


class UsageInfoMacOS(UsageInfoBackend):
    """Feature-flagged macOS backend selector."""

    _VALID_BACKENDS = ("native", "subprocess")

    def __init__(self, backend=None):
        requested_backend = backend
        if requested_backend is None:
            requested_backend = os.environ.get("WORKMUCH_MAC_BACKEND", "native")

        requested_backend = requested_backend.strip().lower()
        if requested_backend not in self._VALID_BACKENDS:
            logging.warning(
                "Unknown WORKMUCH_MAC_BACKEND=%s, defaulting to native.",
                requested_backend,
            )
            requested_backend = "native"

        self._backend_name = requested_backend
        self._impl = self._create_backend(requested_backend)

    def _create_backend(self, backend):
        if backend == "subprocess":
            return UsageInfoMacOSSubprocess()

        try:
            return UsageInfoMacOSNative()
        except Exception:
            logging.error(
                "Failed to initialize native macOS backend; falling back to subprocess."
            )
            logging.error(traceback.format_exc())
            self._backend_name = "subprocess"
            return UsageInfoMacOSSubprocess()

    def getUsageInfo(self):
        return self._impl.getUsageInfo()

    def release(self):
        return self._impl.release()

    def reset(self):
        return self._impl.reset()
