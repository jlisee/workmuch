# Copyright (C) 2008 Maryland Robotics Club
# Copyright (c) 2008 Joseph Lisee <jlisee@gmail.com>
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
# nanosleep code: Copyright (c) 2006 Lovely Systems and Contributors. (ZPL)
#
# Author: Joseph Lisee <jlisee@gmail.com>
# File:  timeutil.py

"""
This module contains class wrappers for sleeps and getting the current time.
"""

# STD Imports
import os
import ctypes
import math
import time as _time

# Begin nanosleep code 
# Under the ZPL, Please see the full text: http://www.zope.org/Resources/ZPL
# Changes: by Joseph Lisee on Jan 20, 2008

try:    
    # Linux
    try:
        _libc = ctypes.CDLL("libc.so.6")
    except OSError:
        _libc = None
    if _libc is None:
        # MAC OS-X
        try:
            _libc = ctypes.CDLL("libc.dylib", ctypes.RTLD_GLOBAL)
        except OSError:
            raise ImportError

    # Define the timespec structure in python
    class _TIMESPEC(ctypes.Structure):
        _fields_ = [('secs', ctypes.c_long),
                    ('nsecs', ctypes.c_long),
                   ]

    _libc.nanosleep.argtypes = \
            [ctypes.POINTER(_TIMESPEC), ctypes.POINTER(_TIMESPEC)]


    def nanosleep(sec, nsec):
        sleeptime = _TIMESPEC()
        sleeptime.secs = sec
        sleeptime.nsecs = nsec
        remaining = _TIMESPEC()
        _libc.nanosleep(sleeptime, remaining)
        return (remaining.secs, remaining.nsecs)

except ImportError:
    # if ctypes is not available or no reasonable library is found we provide
    # a dummy which uses time.sleep

    def nanosleep(sec, nsec):
        _time.sleep(sec + (nsec * 0.000000001))
        
# End nanosleep code

def sleep(seconds):
    """
    Sleeps the current thread the given number of seconds useing nanosleep
    
    @type  seconds: float 
    @param seconds: The number of seconds to sleep
    """
    
    # Round down to our seconds
    secs = math.floor(float(seconds))
    # Convert the remainder to nano seconds
    nsecs = (seconds - secs) * 1e9;
    
    nanosleep(long(secs), long(nsecs))

def time():
    """
    Returns the time since program start
    
    Due to some odd platform differences different time module functions 
    have different accuracies, on different platforms.  The function takes
    that into account.
    
    @rtype:  double
    @return: Seconds since program start
    """
    # This is most accuracte on Linux and Mac
    if 'posix' == os.name:
        return _time.time()
    # This on on Windows
    else:
        return _time.clock()

