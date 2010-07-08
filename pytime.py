#! /usr/bin/env python

# Text file format idea:
#  timestamp|idletime|window title
#  Update at some interval like 1 hz or so
#  possibly run the program super high rate to figure out the true switching
#  rate

# Python Imports
import os
import os.path
import sys
import csv
import ctypes
import logging
import traceback
from datetime import datetime
from optparse import OptionParser

# Library Imports
import Xlib.display
import Xlib.error

# Project Imports
import timeutil

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
        except Xlib.error.Errors, e:
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

    def _forceReset():
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

    def reset(self):
        self.idleTime.reset()
        self.currentWindow.reset()

def mainloop(dataWriter, options):
    # Create handles to the graphics system, and the time we refresh them
    resetTime = timeutil.time() + 5;
    usageInfo = UsageInfo()

    while 1:
        # Record the current time
        curTime = timeutil.time()

        # Recreate handle system
        if curTime >= resetTime:
            usageInfo.reset()
            resetTime = timeutil.time() + 5;

        # Gather our info
        winTitle,progName,timeIdle = usageInfo.getUsageInfo()

        # Log text
        dataWriter.writerow([winTitle,progName,timeIdle,curTime])

        # Wait until the next needed iteration
        wakeTime = curTime + (1.0 / options.rate)
        sleepTime = -1
        
        while sleepTime < 0:
            sleepTime = wakeTime - timeutil.time()
            wakeTime = wakeTime + (1.0 / options.rate)
            
        timeutil.sleep(sleepTime)

def getLogDir():
    # Retrieve the logging directory and make sure it exists
    logDir = os.path.join(os.environ['HOME'], '.workmuch')
    #logDir = '/home/jlisee/.workmuch'

    # Ensure the directory exists
    if not os.path.exists(logDir):
        os.mkdir(logDir)
    
    return logDir
    
def getLogFile():
    # Form the log path
    fileName = datetime.now().strftime("%Y-%m-%d.worklog")
    filePath = os.path.join(getLogDir(), fileName)

    # Open file for appending and return
    return open(filePath, 'a')

def main(argv = None):
    if argv is None:
        argv = sys.argv

    # Configure logging
    errorLogFileName = os.path.join(getLogDir(), 'error.log')
    logLevel = logging.DEBUG
    logFormat = "%(asctime)s %(levelname)s %(message)s"

    if sys.stdout.isatty():
        # If we are running in a terminal, log to it
        logging.basicConfig(level=logLevel, format=logFormat)
    else:
        # We are running in the background so log to file
        logging.basicConfig(level=logLevel, format=logFormat,
                            filename=errorLogFileName, filemode='a')   

    logging.info('Program started')

    # Define and parse arguments
    parser = OptionParser()
    parser.set_defaults(rate = 1.0)
    parser.add_option('-r', '--rate', type = "float", dest = 'rate',
                      help='samples per second')
    
    (options,args) = parser.parse_args(args = argv)

    logging.info('Recording at %fHz' % options.rate)

    # Do a little self check
    idleTime = IdleTime()
    currentWindow = CurrentWindowTitle()

    winTitle,progName = currentWindow.getCurrentWindowInfo()
    if winTitle is None:
        print "WARNING: your system does not supply window titles"
    if len(progName) == 0:
        print "WARNING: your system does not supply program names"

    # Release our resources
    idleTime.release()
    currentWindow.release()

    # Determine the filename based on the current date
    logFile = getLogFile()
    dataWriter = csv.writer(logFile, quoting=csv.QUOTE_MINIMAL)

    try:
        mainloop(dataWriter, options)
    except:
        logFile.close()
        raise

    logging.info('Program shutdown')

if __name__ == "__main__":
    retVal = 0
    try:
        retVal = main()
    except BaseException, e:
        logging.error(traceback.format_exc())

    sys.exit(retVal)
