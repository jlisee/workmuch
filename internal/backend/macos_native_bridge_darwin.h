#ifndef WORKMUCH_MACOS_NATIVE_BRIDGE_DARWIN_H
#define WORKMUCH_MACOS_NATIVE_BRIDGE_DARWIN_H

typedef struct {
	int pid;
	char* app_name;
	char* window_title;
	int ok;
	char* err;
} WMFrontmostWindowInfo;

typedef struct {
	int pid;
	char* app_name;
	int ok;
	char* err;
} WMFrontmostApplicationInfo;

typedef struct {
	char* value;
	int ok;
	char* err;
} WMStringResult;

typedef struct {
	double value;
	int ok;
	char* err;
} WMDoubleResult;

int wmAXIsProcessTrusted(void);
int wmAXIsProcessTrustedWithPrompt(void);
WMFrontmostWindowInfo wmGetFrontmostWindowInfo(void);
WMFrontmostApplicationInfo wmGetFrontmostApplication(void);
WMStringResult wmGetFocusedWindowTitle(int pid);
WMDoubleResult wmGetIdleSeconds(void);
void wmFreeString(char* s);

#endif
