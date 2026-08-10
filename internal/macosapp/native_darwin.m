#include "native_darwin.h"

#import <AppKit/AppKit.h>
#import <ServiceManagement/ServiceManagement.h>
#include <stdlib.h>
#include <string.h>

static char* wmCopyNSError(NSError* error) {
	if (error == nil) {
		return strdup("");
	}
	const char* message = [[error localizedDescription] UTF8String];
	return strdup(message == NULL ? "" : message);
}

WMIntResult wmMainAppServiceStatus(void) {
	WMIntResult result = {.value = -1, .ok = 1, .err = NULL};
	@autoreleasepool {
		if (@available(macOS 13.0, *)) {
			switch ([[SMAppService mainAppService] status]) {
			case SMAppServiceStatusNotRegistered:
				result.value = 0;
				break;
			case SMAppServiceStatusEnabled:
				result.value = 1;
				break;
			case SMAppServiceStatusRequiresApproval:
				result.value = 2;
				break;
			case SMAppServiceStatusNotFound:
				result.value = 3;
				break;
			}
		}
	}
	return result;
}

WMErrorResult wmRegisterMainAppService(void) {
	WMErrorResult result = {.ok = 1, .err = NULL};
	@autoreleasepool {
		if (@available(macOS 13.0, *)) {
			NSError* error = nil;
			if (![[SMAppService mainAppService] registerAndReturnError:&error]) {
				result.ok = 0;
				result.err = wmCopyNSError(error);
			}
			return result;
		}
		result.ok = 0;
		result.err = strdup("SMAppService.mainAppService requires macOS 13 or later");
	}
	return result;
}

WMErrorResult wmShowMoveToApplicationsDialog(void) {
	WMErrorResult result = {.ok = 1, .err = NULL};
	@autoreleasepool {
		[NSApplication sharedApplication];
		[NSApp activateIgnoringOtherApps:YES];
		NSAlert* alert = [[NSAlert alloc] init];
		[alert setAlertStyle:NSAlertStyleInformational];
		[alert setMessageText:@"Move WorkMuch to Applications"];
		[alert setInformativeText:@"Drag WorkMuch.app to Applications, then open it there. WorkMuch will not run from the disk image or another folder."];
		[alert addButtonWithTitle:@"Quit"];
		[alert runModal];
		[alert release];
	}
	return result;
}

void wmMacOSAppFreeString(char* value) {
	if (value != NULL) {
		free(value);
	}
}
