#include "macos_native_bridge_darwin.h"

#include <ApplicationServices/ApplicationServices.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <math.h>
#include <stdlib.h>
#include <string.h>

static char* wmCopyCFString(CFStringRef value) {
	if (value == NULL) {
		return strdup("");
	}

	const char* direct = CFStringGetCStringPtr(value, kCFStringEncodingUTF8);
	if (direct != NULL) {
		return strdup(direct);
	}

	CFIndex capacity = CFStringGetMaximumSizeForEncoding(
		CFStringGetLength(value),
		kCFStringEncodingUTF8
	) + 1;
	char* buffer = malloc((size_t)capacity);
	if (buffer == NULL) {
		return strdup("");
	}
	if (!CFStringGetCString(value, buffer, capacity, kCFStringEncodingUTF8)) {
		free(buffer);
		return strdup("");
	}
	return buffer;
}

static int wmCFNumberToInt(CFTypeRef value) {
	if (value == NULL || CFGetTypeID(value) != CFNumberGetTypeID()) {
		return 0;
	}

	int result = 0;
	CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, &result);
	return result;
}

int wmAXIsProcessTrusted(void) {
	return AXIsProcessTrusted() ? 1 : 0;
}

int wmAXIsProcessTrustedWithPrompt(void) {
	const void* keys[] = {kAXTrustedCheckOptionPrompt};
	const void* values[] = {kCFBooleanTrue};
	CFDictionaryRef options = CFDictionaryCreate(
		kCFAllocatorDefault,
		keys,
		values,
		1,
		&kCFCopyStringDictionaryKeyCallBacks,
		&kCFTypeDictionaryValueCallBacks
	);
	if (options == NULL) {
		return AXIsProcessTrusted() ? 1 : 0;
	}
	Boolean trusted = AXIsProcessTrustedWithOptions(options);
	CFRelease(options);
	return trusted ? 1 : 0;
}

WMFrontmostWindowInfo wmGetFrontmostWindowInfo(void) {
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
		CFTypeRef item = CFArrayGetValueAtIndex(windowList, index);
		if (item == NULL || CFGetTypeID(item) != CFDictionaryGetTypeID()) {
			continue;
		}

		CFDictionaryRef window = (CFDictionaryRef)item;
		int layer = wmCFNumberToInt(CFDictionaryGetValue(window, kCGWindowLayer));
		int pid = wmCFNumberToInt(CFDictionaryGetValue(window, kCGWindowOwnerPID));
		if (layer != 0 || pid <= 0) {
			continue;
		}

		CFTypeRef appName = CFDictionaryGetValue(window, kCGWindowOwnerName);
		CFTypeRef windowTitle = CFDictionaryGetValue(window, kCGWindowName);
		free(result.app_name);
		free(result.window_title);
		result.pid = pid;
		result.app_name = appName != NULL && CFGetTypeID(appName) == CFStringGetTypeID()
			? wmCopyCFString((CFStringRef)appName)
			: strdup("");
		result.window_title = windowTitle != NULL && CFGetTypeID(windowTitle) == CFStringGetTypeID()
			? wmCopyCFString((CFStringRef)windowTitle)
			: strdup("");
		break;
	}

	CFRelease(windowList);
	return result;
}

WMFrontmostApplicationInfo wmGetFrontmostApplication(void) {
	WMFrontmostWindowInfo window = wmGetFrontmostWindowInfo();
	WMFrontmostApplicationInfo result;
	result.pid = window.pid;
	result.app_name = window.app_name;
	result.ok = window.ok;
	result.err = window.err;
	free(window.window_title);
	return result;
}

WMStringResult wmGetFocusedWindowTitle(int pid) {
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
			result.value = wmCopyCFString((CFStringRef)titleValue);
		} else {
			CFStringRef description = CFCopyDescription(titleValue);
			result.value = wmCopyCFString(description);
			if (description != NULL) {
				CFRelease(description);
			}
		}
	}

	if (titleValue != NULL) {
		CFRelease(titleValue);
	}
	CFRelease(focusedWindow);
	CFRelease(appElement);
	return result;
}

WMDoubleResult wmGetIdleSeconds(void) {
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
		result.err = strdup("CGEventSourceSecondsSinceLastEventType returned a non-finite value");
		return result;
	}
	if (seconds < 0) {
		seconds = 0;
	}

	result.value = (double)seconds;
	return result;
}

void wmFreeString(char* value) {
	if (value != NULL) {
		free(value);
	}
}
