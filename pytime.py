#! /usr/bin/env python

# Copyright (c) 2010 Joseph Lisee <jlisee@gmail.com>
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
# File:  pytime.py

# Python Imports
import os
import os.path
import sys
import csv
import logging
import traceback
from datetime import datetime
from optparse import OptionParser

# Project Imports
import timeutil
from usageinfo import UsageInfo

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
    parser.set_defaults(rate = 1.0, delay = 0.0)
    parser.add_option('-r', '--rate', type = "float", dest = 'rate',
                      help='samples per second')
    parser.add_option('-d', '--start-delay', type = "float", dest = 'delay',
                      help='time between program startup and logging start')
    
    (options,args) = parser.parse_args(args = argv)

    # Report startup options
    logging.info('Recording at %fHz' % options.rate)
    logging.info('Waiting %f seconds before starting logging' % options.delay)

    # wait to start our logging
    if options.delay > 0:
        timeutil.sleep(options.delay)
        logging.info('Delay complete, logging commensing')

    # Do a little self check
    usageInfo = UsageInfo()
    winTitle,progName,timeIdle = usageInfo.getUsageInfo()

    if winTitle is None:
        logging.warn("your system does not supply window titles")
    if len(progName) == 0:
        logging.warn("your system does not supply program names")

    # Release our resources
    usageInfo.release()
    
    # Determine the filename based on the current date
    logFile = getLogFile()
    dataWriter = csv.writer(logFile, quoting=csv.QUOTE_MINIMAL)

    # Start the logging, and make sure close the file no matter the exit method
    try:
        mainloop(dataWriter, options)
    except (KeyboardInterrupt):
        logging.info('Ctrl+C shutdown')
    except (SystemExit):
        logging.info('Forced shutdown')
    finally:
        logFile.close()

    logging.info('Program shutdown complete')

if __name__ == "__main__":
    retVal = 0
    try:
        retVal = main()
    except BaseException, e:
        logging.error(traceback.format_exc())

    sys.exit(retVal)
