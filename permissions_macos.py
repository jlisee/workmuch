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
# File:  permissions_macos.py

import sys

def _can_prompt_user():
    return sys.stdin.isatty() and sys.stdout.isatty()

def _ask_yes_no(prompt_text):
    while True:
        sys.stdout.write("%s [y/n]: " % prompt_text)
        sys.stdout.flush()
        try:
            answer = input().strip().lower()
        except EOFError:
            return False

        if answer in ('y', 'yes'):
            return True
        if answer in ('n', 'no'):
            return False

        print("Please enter 'y' or 'n'.")

def _create_macos_accessibility_api():
    if sys.platform != 'darwin':
        return None

    try:
        from usageinfo_macos_native import _MacOSNativeAPI
        return _MacOSNativeAPI()
    except Exception:
        logging.error('Unable to initialize macOS accessibility APIs')
        logging.error(traceback.format_exc())
        return None

def _is_accessibility_trusted(api, prompt = False):
    try:
        return bool(api.is_accessibility_trusted_with_prompt(prompt = prompt))
    except Exception:
        logging.error('Failed to check macOS accessibility trust state')
        logging.error(traceback.format_exc())
        return False

def wait_for_accessibility(api, timeout_seconds = 30.0, poll_interval_seconds = 0.5):
    # Prompt once, then poll without prompting.
    if _is_accessibility_trusted(api, prompt = True):
        return True

    timeoutAt = timeutil.time() + timeout_seconds
    while timeutil.time() < timeoutAt:
        timeutil.sleep(poll_interval_seconds)
        if _is_accessibility_trusted(api):
            return True

    return False


def have_accessibility_perms():
    axApi = _create_macos_accessibility_api()
    if axApi is None:
        logging.error(
            'Accessibility permission is required on macOS, but native APIs '
            'could not be loaded. Install PyObjC and retry.'
        )
        return False

    try:
        if not _is_accessibility_trusted(axApi):
            if not _can_prompt_user():
                logging.error(
                    'Accessibility permission is required on macOS. '
                    'Grant access to iTerm2 in System Settings -> '
                    'Privacy & Security -> Accessibility, then retry.'
                )
                return False

            print(
                'workmuch requires macOS Accessibility permission to '
                'capture window titles.'
            )
            print(
                'Grant access to iTerm2 in System Settings -> Privacy & '
                'Security -> Accessibility.'
            )
            if not _ask_yes_no('Request permission now'):
                logging.error('Accessibility permission denied by user')
                return False

            if not _wait_for_accessibility(
                axApi,
                timeout_seconds = 30.0,
                poll_interval_seconds = 0.5,
            ):
                logging.error(
                    'Accessibility permission was not granted within '
                    '30 seconds'
                )
                return False
    finally:
        axApi.close()
    return True
