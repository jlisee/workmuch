import sys
from pathlib import Path

import pytest
import subprocess

sys.path.insert(0, str(Path(__file__).resolve().parents[1]))

from usageinfo_macos import UsageInfoMacOS


class _Proc(object):
    def __init__(self, returncode=0, stdout=""):
        self.returncode = returncode
        self.stdout = stdout


def test_macos_query_returns_tuple_without_crashing(monkeypatch):
    usage = UsageInfoMacOS()

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


def test_macos_query_handles_failures_without_crashing(monkeypatch):
    usage = UsageInfoMacOS()

    def failing_run(cmd, **kwargs):
        raise OSError("unavailable")

    monkeypatch.setattr(subprocess, "run", failing_run)

    win_title, prog_name, idle_time = usage.getUsageInfo()

    assert win_title == ""
    assert prog_name == ""
    assert idle_time == 0.0


@pytest.mark.skipif(sys.platform != "darwin", reason="macOS-only smoke test")
def test_macos_query_smoke():
    usage = UsageInfoMacOS()
    win_title, prog_name, idle_time = usage.getUsageInfo()

    assert isinstance(win_title, str)
    assert isinstance(prog_name, str)
    assert isinstance(idle_time, float)
    assert idle_time >= 0.0

    usage.reset()
    usage.release()
