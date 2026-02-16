import sys
from pathlib import Path

import pytest

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from usageinfo_macos_native import UsageInfoMacOSNative, _MacOSNativeAPI


class _FakeAPI(object):
    def __init__(self, trusted=True, pid=123, prog_name="Safari", title="Window", idle=1.25):
        self._trusted = trusted
        self._pid = pid
        self._prog_name = prog_name
        self._title = title
        self._idle = idle
        self._frontmost_window_info = (0, "", "")
        self.closed = False
        self.reset_called = False
        self.title_queries = 0

    def is_accessibility_trusted(self):
        return self._trusted

    def get_frontmost_window_info(self):
        return self._frontmost_window_info

    def get_frontmost_application(self):
        return self._pid, self._prog_name

    def get_focused_window_title(self, pid):
        self.title_queries += 1
        return self._title

    def get_idle_seconds(self):
        return self._idle

    def close(self):
        self.closed = True

    def reset(self):
        self.reset_called = True


def test_native_backend_uses_api_values():
    api = _FakeAPI(trusted=True, pid=42, prog_name="Safari", title="Docs", idle=2.5)
    usage = UsageInfoMacOSNative(api=api)

    assert usage.getUsageInfo() == ("Docs", "Safari", 2.5)
    assert api.title_queries == 1


def test_native_backend_omits_window_title_without_ax_permission():
    api = _FakeAPI(trusted=False, pid=42, prog_name="Safari", title="Ignored", idle=2.5)
    usage = UsageInfoMacOSNative(api=api)

    assert usage.getUsageInfo() == ("", "Safari", 2.5)
    assert api.title_queries == 0


def test_native_backend_handles_exceptions_without_crashing():
    class _FailingAPI(_FakeAPI):
        def get_idle_seconds(self):
            raise RuntimeError("boom")

    usage = UsageInfoMacOSNative(api=_FailingAPI())
    assert usage.getUsageInfo() == ("", "", 0.0)


def test_native_backend_prefers_frontmost_window_info_when_available():
    api = _FakeAPI(trusted=True, pid=1, prog_name="Terminal", title="Shell", idle=1.0)
    api._frontmost_window_info = (42, "Safari", "Docs")
    usage = UsageInfoMacOSNative(api=api)

    assert usage.getUsageInfo() == ("Docs", "Safari", 1.0)
    assert api.title_queries == 0


def test_native_backend_release_and_reset_delegate_to_api():
    api = _FakeAPI()
    usage = UsageInfoMacOSNative(api=api)
    usage.reset()
    usage.release()

    assert api.reset_called is True
    assert api.closed is True


def test_native_api_frontmost_window_info_uses_top_layer_zero_window():
    class _FakeQuartz(object):
        kCGWindowListOptionOnScreenOnly = 1
        kCGWindowListExcludeDesktopElements = 2
        kCGNullWindowID = 0
        kCGWindowLayer = "kCGWindowLayer"
        kCGWindowOwnerPID = "kCGWindowOwnerPID"
        kCGWindowOwnerName = "kCGWindowOwnerName"
        kCGWindowName = "kCGWindowName"

        @staticmethod
        def CGWindowListCopyWindowInfo(options, window_id):
            assert options == 3
            assert window_id == 0
            return [
                {
                    "kCGWindowLayer": 25,
                    "kCGWindowOwnerPID": 999,
                    "kCGWindowOwnerName": "Overlay",
                    "kCGWindowName": "Ignore me",
                },
                {
                    "kCGWindowLayer": 0,
                    "kCGWindowOwnerPID": 321,
                    "kCGWindowOwnerName": "Safari",
                    "kCGWindowName": "Current Tab",
                },
            ]

    api = object.__new__(_MacOSNativeAPI)
    api._quartz = _FakeQuartz()

    assert api.get_frontmost_window_info() == (321, "Safari", "Current Tab")


@pytest.mark.skipif(sys.platform != "darwin", reason="macOS-only smoke test")
def test_native_backend_smoke():
    usage = UsageInfoMacOSNative()
    win_title, prog_name, idle_time = usage.getUsageInfo()

    assert isinstance(win_title, str)
    assert win_title != ''
    assert isinstance(prog_name, str)
    assert prog_name != ''
    assert isinstance(idle_time, float)
    assert idle_time >= 0.0
    usage.release()


@pytest.mark.skipif(sys.platform != "darwin", reason="macOS-only stability test")
def test_native_backend_stability_loop():
    usage = UsageInfoMacOSNative()
    for _ in range(250):
        win_title, prog_name, idle_time = usage.getUsageInfo()
        assert isinstance(win_title, str)
        assert isinstance(prog_name, str)
        assert isinstance(idle_time, float)
        assert idle_time >= 0.0
    usage.release()
