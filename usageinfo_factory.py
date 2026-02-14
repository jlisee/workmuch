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
# File:  usageinfo_factory.py

# STD Imports
import sys

def createUsageInfo(platform = None):
    """
    Create and return the currently supported usage info backend.

    @type  platform: str
    @param platform: Optional override for testing
    """
    if platform is None:
        platform = sys.platform

    # Current implementation remains X11/Linux-only.  The factory exists so
    # additional platform backends (macOS/Windows) can be introduced cleanly.
    if platform.startswith('linux'):
        from usageinfo import UsageInfo
        return UsageInfo()

    raise NotImplementedError('Unsupported platform: %s' % platform)
