# Python Imports
import os
import ctypes
import logging
import traceback

# Library Imports
import Xlib.display
import Xlib.error


# Load libXss with ctypes
class XScreenSaverInfo( ctypes.Structure):
    """ typedef struct { ... } XScreenSaverInfo; """
    _fields_ = [('window',      ctypes.c_ulong), # screen saver window
                ('state',       ctypes.c_int),   # off,on,disabled
                ('kind',        ctypes.c_int),   # blanked,internal,external
                ('since',       ctypes.c_ulong), # milliseconds
                ('idle',        ctypes.c_ulong), # milliseconds
                ('event_mask',  ctypes.c_ulong)] # events
xss = ctypes.cdll.LoadLibrary( 'libXss.so')
xss.XScreenSaverAllocInfo.restype = ctypes.POINTER(XScreenSaverInfo)
    
# Load libX11 with ctypes for use with libXss
xlib = ctypes.cdll.LoadLibrary( 'libX11.so')



class CurrentWindowTitle(object):
    """Class for efficiently querying the current window title and program"""
    
    def __init__(self):
        self._setup()

    def _setup(self):
        self.display = Xlib.display.Display()
        self.screen = self.display.screen()

    def makeTopLevelWindow(self, win):
        """Traces up the window tree until we get a top level window"""
    
        topLevelWin = win
        winData = topLevelWin.query_tree()

        while winData.parent != self.screen.root:
            newWin = winData.parent

            if type(newWin) is type(win):
                topLevelWin = newWin
                winData = topLevelWin.query_tree()
            else:
                break

            # Some X11 window servers don't have the main windows as direct
            # parents of root, so we go up until we get a name
            if topLevelWin.get_wm_name() is not None:
                break

        return topLevelWin

    def getCurrentWindowInfo(self):
        # Grab current window
        focus = self.display.get_input_focus()
        curWin = focus.focus

        # During early startup sometimes Xlib gives us back bogus objects
        if hasattr(curWin, 'query_tree'):
            # Find the top level window
            topLevelWin = self.makeTopLevelWindow(curWin)

            # Grab program name
            clsInfo = topLevelWin.get_wm_class()
            progName = ''
            if clsInfo is not None:
                progName = clsInfo[1]

            # Record window title bar            
            title = topLevelWin.get_wm_name()
 
            return title,progName
        else:
            return '',''

        # Windows syntax
        #  import win32gui
        #  w=win32gui
        #  w.GetWindowText (w.GetForegroundWindow())
    
    def release(self):
        if self.display is not None:
            self.display.close()
        self.display = None

    def reset(self):
        self.release()
        self._setup()

class IdleTime(object):
    """Class for efficiently querying the current x display idle time"""
    
    def __init__(self):
        self._setup()
        self.xss_info = xss.XScreenSaverAllocInfo()

    def _setup(self):
        self.dpy = xlib.XOpenDisplay(os.environ['DISPLAY'])
        self.root = xlib.XDefaultRootWindow(self.dpy)

    def getIdleTime(self):
        xss.XScreenSaverQueryInfo(self.dpy, self.root, self.xss_info)
        return float(self.xss_info.contents.idle) / 1000.0

    def release(self):
        if self.dpy is not None:
            xlib.XCloseDisplay(self.dpy)
        self.dpy = None

    def reset(self):
        self.release()
        self._setup()

class UsageInfo(object):
    """Gets the current window/program info and idle time"""

    def __init__(self):
        self.idleTime = IdleTime()
        self.currentWindow = CurrentWindowTitle()
        
    def getUsageInfo(self):
        try:
            return self._doGetUsageInfo()
        except Xlib.error.XError, e:
            # Log this error an try to handle it
            logging.error(traceback.format_exc())

            # Reset our interface, to work around any odd X bug
            self._forceReset()

            # Now attempt to grab the data again
            return self._doGetUsageInfo()

    def _doGetUsageInfo(self):
        """Use lower level API's to retrieve title,progname,idletime"""
        winTitle,progName = self.currentWindow.getCurrentWindowInfo()
        timeIdle = self.idleTime.getIdleTime()

        return winTitle,progName,timeIdle

    def _forceReset(self):
        # First the idle time
        try:
            self.idleTime.release()
        except BaseException, e:
            # Log an continue on
            logging.error(traceback.format_exc())

        self.idleTime.reset()

        # Now lets try to get currentWindow back
        try:
            self.currentWindow.release()
        except BaseException, e:
            # Log an continue on
            logging.error(traceback.format_exc())

        self.currentWindow.reset()

    def release(self):
        self.idleTime.release()
        self.currentWindow.release()

    def reset(self):
        self.idleTime.reset()
        self.currentWindow.reset()

