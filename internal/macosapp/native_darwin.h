#ifndef WORKMUCH_MACOS_APP_NATIVE_DARWIN_H
#define WORKMUCH_MACOS_APP_NATIVE_DARWIN_H

typedef struct {
	int value;
	int ok;
	char* err;
} WMIntResult;

typedef struct {
	int ok;
	char* err;
} WMErrorResult;

WMIntResult wmMainAppServiceStatus(void);
WMErrorResult wmRegisterMainAppService(void);
WMErrorResult wmShowMoveToApplicationsDialog(void);
void wmMacOSAppFreeString(char* value);

#endif
