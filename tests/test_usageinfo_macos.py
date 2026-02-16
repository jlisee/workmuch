import sys
from pathlib import Path

import pytest
import subprocess

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

import usageinfo_macos
from usageinfo_macos import UsageInfoMacOS, UsageInfoMacOSSubprocess


class _Proc(object):
    def __init__(self, returncode=0, stdout=""):
        self.returncode = returncode
        self.stdout = stdout


def test_macos_subprocess_query_returns_tuple_without_crashing(monkeypatch):
    usage = UsageInfoMacOSSubprocess()

    def fake_run(cmd, **kwargs):
        if cmd == usage._CMD_APP_NAME:
            return _Proc(returncode=0, stdout="Safari\n")
        if cmd == usage._CMD_WINDOW_TITLE:
            return _Proc(returncode=0, stdout="Welcome Page\n")
        if cmd == usage._CMD_IDLE_NS:
            return _Proc(returncode=0, stdout='"HIDIdleTime" = 2500000000\n')
        raise AssertionError("unexpected command")

    monkeypatch.setattr(subprocess, "run", fake_run)

    win_title, prog_name, idle_time = usage.getUsageInfo()

    assert win_title == "Welcome Page"
    assert prog_name == "Safari"
    assert idle_time == 2.5


def test_macos_subprocess_query_handles_failures_without_crashing(monkeypatch):
    usage = UsageInfoMacOSSubprocess()

    def failing_run(cmd, **kwargs):
        raise OSError("unavailable")

    monkeypatch.setattr(subprocess, "run", failing_run)

    win_title, prog_name, idle_time = usage.getUsageInfo()

    assert win_title == ""
    assert prog_name == ""
    assert idle_time == 0.0


def test_selector_uses_subprocess_backend(monkeypatch):
    usage = UsageInfoMacOS(backend="subprocess")
    assert isinstance(usage._impl, UsageInfoMacOSSubprocess)


def test_selector_uses_native_backend_when_available(monkeypatch):
    class _FakeNativeBackend(object):
        def getUsageInfo(self):
            return "win", "prog", 1.0

        def release(self):
            return None

        def reset(self):
            return None

    monkeypatch.setattr(usageinfo_macos, "UsageInfoMacOSNative", _FakeNativeBackend)
    usage = UsageInfoMacOS(backend="native")

    assert usage.getUsageInfo() == ("win", "prog", 1.0)


def test_selector_falls_back_to_subprocess_when_native_fails(monkeypatch):
    class _FailingNativeBackend(object):
        def __init__(self):
            raise RuntimeError("native unavailable")

    monkeypatch.setattr(usageinfo_macos, "UsageInfoMacOSNative", _FailingNativeBackend)
    usage = UsageInfoMacOS(backend="native")

    assert isinstance(usage._impl, UsageInfoMacOSSubprocess)


@pytest.mark.skipif(sys.platform != "darwin", reason="macOS-only smoke test")
def test_macos_selector_smoke():
    usage = UsageInfoMacOS(backend="native")
    win_title, prog_name, idle_time = usage.getUsageInfo()

    assert isinstance(win_title, str)
    assert isinstance(prog_name, str)
    assert isinstance(idle_time, float)
    assert idle_time >= 0.0

    usage.reset()
    usage.release()
