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
# File:  usageinfo_macos_native.py

# Python Imports
import logging
import sys
import traceback

# Project Imports
from workmuch.usageinfo_base import UsageInfoBackend


class _MacOSNativeAPI(object):
    """Native macOS calls backed by PyObjC framework bindings."""

    TARGET_QUERY_OVERHEAD_MS = 2.0

    def __init__(self):
        if sys.platform != "darwin":
            raise RuntimeError("Native macOS backend only supported on macOS")

        try:
            import ApplicationServices as app_services
            import Quartz as quartz
            from AppKit import NSWorkspace
        except Exception as exc:
            raise RuntimeError(
                "PyObjC frameworks unavailable for native macOS backend"
            ) from exc

        self._app_services = app_services
        self._quartz = quartz
        self._workspace = NSWorkspace.sharedWorkspace()

    def _to_text(self, value):
        if value is None:
            return ""
        return str(value)

    def is_accessibility_trusted(self):
        return bool(self._app_services.AXIsProcessTrusted())

    def _build_ax_prompt_options(self, prompt):
        option_key = getattr(
            self._app_services,
            "kAXTrustedCheckOptionPrompt",
            "AXTrustedCheckOptionPrompt",
        )
        return {option_key: bool(prompt)}

    def is_accessibility_trusted_with_prompt(self, prompt=False):
        trust_checker = getattr(
            self._app_services,
            "AXIsProcessTrustedWithOptions",
            None,
        )
        if trust_checker is None:
            return self.is_accessibility_trusted()
        return bool(trust_checker(self._build_ax_prompt_options(prompt)))

    def get_frontmost_application(self):
        app = self._workspace.frontmostApplication()
        if app is None:
            return 0, ""

        pid = int(app.processIdentifier())
        prog_name = self._to_text(app.localizedName())
        return pid, prog_name

    def get_frontmost_window_info(self):
        """Return (pid, app_name, window_title) for the top visible window."""
        options = (
            self._quartz.kCGWindowListOptionOnScreenOnly
            | self._quartz.kCGWindowListExcludeDesktopElements
        )
        window_list = self._quartz.CGWindowListCopyWindowInfo(
            options,
            self._quartz.kCGNullWindowID,
        )
        if not window_list:
            return 0, "", ""

        for window in window_list:
            try:
                layer = int(window.get(self._quartz.kCGWindowLayer, 0))
                pid = int(window.get(self._quartz.kCGWindowOwnerPID, 0))
            except (TypeError, ValueError):
                continue

            if layer != 0 or pid <= 0:
                continue

            app_name = self._to_text(window.get(self._quartz.kCGWindowOwnerName))
            window_title = self._to_text(window.get(self._quartz.kCGWindowName))
            return pid, app_name, window_title

        return 0, "", ""

    def get_focused_window_title(self, pid):
        if pid <= 0:
            return ""

        app_element = self._app_services.AXUIElementCreateApplication(pid)
        if app_element is None:
            return ""

        error, focused_window = self._app_services.AXUIElementCopyAttributeValue(
            app_element,
            self._app_services.kAXFocusedWindowAttribute,
            None,
        )
        if (
            error != self._app_services.kAXErrorSuccess
            or focused_window is None
        ):
            return ""

        title_error, title_value = self._app_services.AXUIElementCopyAttributeValue(
            focused_window,
            self._app_services.kAXTitleAttribute,
            None,
        )
        if title_error != self._app_services.kAXErrorSuccess or title_value is None:
            return ""

        return self._to_text(title_value)

    def get_idle_seconds(self):
        seconds = self._quartz.CGEventSourceSecondsSinceLastEventType(
            self._quartz.kCGEventSourceStateCombinedSessionState,
            self._quartz.kCGAnyInputEventType,
        )
        if seconds < 0:
            return 0.0
        return float(seconds)

    def reset(self):
        return None

    def close(self):
        self._workspace = None
        return None


class UsageInfoMacOSNative(UsageInfoBackend):
    """Native macOS backend powered by PyObjC framework bindings."""

    def __init__(self, api=None):
        self._api = api if api is not None else _MacOSNativeAPI()
        self._ax_trusted = bool(self._api.is_accessibility_trusted())
        self._ax_permission_logged = False
        if not self._ax_trusted:
            self._log_ax_permission_once()

    def _log_ax_permission_once(self):
        if self._ax_permission_logged:
            return
        self._ax_permission_logged = True
        logging.warning(
            "Accessibility permission not granted; window titles will be empty."
        )

    def getUsageInfo(self):
        try:
            return self._doGetUsageInfo()
        except Exception:
            logging.error(traceback.format_exc())
            return "", "", 0.0

    def _doGetUsageInfo(self):
        pid, prog_name, win_title = self._api.get_frontmost_window_info()

        if pid <= 0:
            pid, prog_name = self._api.get_frontmost_application()

        # Refresh AX trust to pick up permission changes during runtime.
        if not self._ax_trusted:
            self._ax_trusted = bool(self._api.is_accessibility_trusted())

        if self._ax_trusted:
            if not win_title:
                win_title = self._api.get_focused_window_title(pid)
        else:
            self._log_ax_permission_once()

        idle_seconds = self._api.get_idle_seconds()
        return win_title, prog_name, idle_seconds

    def release(self):
        self._api.close()
        return None

    def reset(self):
        self._api.reset()
        return None
