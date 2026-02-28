#include "macos_native_bridge_darwin.h"

#include <ApplicationServices/ApplicationServices.h>
#include <AppKit/AppKit.h>
#include <CoreGraphics/CoreGraphics.h>
#include <Foundation/Foundation.h>
#include <math.h>
#include <stdlib.h>
#include <string.h>

static char* wmCopyCString(const char* value) {
	if (value == NULL) {
		return strdup("");
	}
	return strdup(value);
}

static char* wmCopyNSString(NSString* value) {
	if (value == nil) {
		return strdup("");
	}

	const char* utf8 = [value UTF8String];
	if (utf8 == NULL) {
		return strdup("");
	}
	return strdup(utf8);
}

static char* wmCopyError(NSString* message) {
	return wmCopyNSString(message);
}

int wmAXIsProcessTrusted(void) {
	@autoreleasepool {
		return AXIsProcessTrusted() ? 1 : 0;
	}
}

WMFrontmostApplicationInfo wmGetFrontmostApplication(void) {
	@autoreleasepool {
		WMFrontmostApplicationInfo result;
		result.pid = 0;
		result.app_name = strdup("");
		result.ok = 1;
		result.err = NULL;

		NSRunningApplication* app = [[NSWorkspace sharedWorkspace] frontmostApplication];
		if (app == nil) {
			return result;
		}

		free(result.app_name);
		result.pid = (int)[app processIdentifier];
		result.app_name = wmCopyNSString([app localizedName]);
		return result;
	}
}

WMFrontmostWindowInfo wmGetFrontmostWindowInfo(void) {
	@autoreleasepool {
		WMFrontmostWindowInfo result;
		result.pid = 0;
		result.app_name = strdup("");
		result.window_title = strdup("");
		result.ok = 1;
		result.err = NULL;

		CGWindowListOption options = kCGWindowListOptionOnScreenOnly | kCGWindowListExcludeDesktopElements;
		CFArrayRef windowList = CGWindowListCopyWindowInfo(options, kCGNullWindowID);
		if (windowList == NULL) {
			return result;
		}

		CFIndex count = CFArrayGetCount(windowList);
		for (CFIndex index = 0; index < count; index++) {
			NSDictionary* window = (NSDictionary*)CFArrayGetValueAtIndex(windowList, index);
			if (window == nil) {
				continue;
			}

			NSNumber* layerNumber = [window objectForKey:(id)kCGWindowLayer];
			NSNumber* pidNumber = [window objectForKey:(id)kCGWindowOwnerPID];
			int layer = [layerNumber respondsToSelector:@selector(intValue)] ? [layerNumber intValue] : 0;
			int pid = [pidNumber respondsToSelector:@selector(intValue)] ? [pidNumber intValue] : 0;
			if (layer != 0 || pid <= 0) {
				continue;
			}

			NSString* appName = [window objectForKey:(id)kCGWindowOwnerName];
			NSString* windowTitle = [window objectForKey:(id)kCGWindowName];
			free(result.app_name);
			free(result.window_title);
			result.pid = pid;
			result.app_name = wmCopyNSString(appName);
			result.window_title = wmCopyNSString(windowTitle);
			break;
		}

		CFRelease(windowList);
		return result;
	}
}

WMStringResult wmGetFocusedWindowTitle(int pid) {
	@autoreleasepool {
		WMStringResult result;
		result.value = strdup("");
		result.ok = 1;
		result.err = NULL;

		if (pid <= 0) {
			return result;
		}

		AXUIElementRef appElement = AXUIElementCreateApplication(pid);
		if (appElement == NULL) {
			return result;
		}

		CFTypeRef focusedWindow = NULL;
		AXError focusedError = AXUIElementCopyAttributeValue(appElement, kAXFocusedWindowAttribute, &focusedWindow);
		if (focusedError != kAXErrorSuccess || focusedWindow == NULL) {
			if (focusedWindow != NULL) {
				CFRelease(focusedWindow);
			}
			CFRelease(appElement);
			return result;
		}

		CFTypeRef titleValue = NULL;
		AXError titleError = AXUIElementCopyAttributeValue((AXUIElementRef)focusedWindow, kAXTitleAttribute, &titleValue);
		if (titleError == kAXErrorSuccess && titleValue != NULL) {
			free(result.value);
			if (CFGetTypeID(titleValue) == CFStringGetTypeID()) {
				result.value = wmCopyNSString((NSString*)titleValue);
			} else {
				result.value = wmCopyNSString([(id)titleValue description]);
			}
		}

		if (titleValue != NULL) {
			CFRelease(titleValue);
		}
		CFRelease(focusedWindow);
		CFRelease(appElement);
		return result;
	}
}

WMDoubleResult wmGetIdleSeconds(void) {
	@autoreleasepool {
		WMDoubleResult result;
		result.value = 0.0;
		result.ok = 1;
		result.err = NULL;

		CGFloat seconds = CGEventSourceSecondsSinceLastEventType(
			kCGEventSourceStateCombinedSessionState,
			kCGAnyInputEventType
		);

		if (!isfinite(seconds)) {
			result.ok = 0;
			result.err = wmCopyError(@"CGEventSourceSecondsSinceLastEventType returned a non-finite value");
			return result;
		}

		if (seconds < 0) {
			seconds = 0;
		}

		result.value = (double)seconds;
		return result;
	}
}

void wmFreeString(char* s) {
	if (s != NULL) {
		free(s);
	}
}
