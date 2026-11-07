// si Studio for macOS: windows, menus and WebKit around the web Studio.
// Go (main.go) starts and stops the si --web processes; this file is the UI.

#import <Cocoa/Cocoa.h>
#import <WebKit/WebKit.h>

extern char *goOpenFolder(char *path);
extern void goCloseURL(char *url);
extern void goShutdown(void);
extern char *goStatus(void);

static NSString *const kLastFolder = @"lastFolder";
static NSString *const kRecent = @"recentFolders"; // newest first, at most ten

@interface SIWindow : NSObject <WKUIDelegate, WKNavigationDelegate, NSWindowDelegate>
@property(strong) NSWindow *window;
@property(strong) WKWebView *web;
@property(copy) NSString *studioURL; // set on a folder's main window: closing it stops its si
@end

@interface SIApp : NSObject <NSApplicationDelegate, NSMenuDelegate>
@end

static NSMutableArray<SIWindow *> *windows;
static NSDictionary *siStatus; // goStatus: the si in use, and a newer release if there is one

// bannerJS: a small "update available" note in the Studio window; its link opens the
// release page in the browser (an external page, see isExternal).
static NSString *bannerJS(void) {
	NSString *tag = siStatus[@"update"], *url = siStatus[@"url"];
	if (tag.length == 0 || url.length == 0) return nil;
	NSData *d = [NSJSONSerialization dataWithJSONObject:@[tag, url] options:0 error:nil];
	NSString *args = [[NSString alloc] initWithData:d encoding:NSUTF8StringEncoding];
	return [NSString stringWithFormat:@"(function(a){if(document.getElementById('si-update'))return;var b=document.createElement('div');b.id='si-update';"
	        "b.setAttribute('role','status');b.style.cssText='position:fixed;right:14px;bottom:14px;z-index:99999;font:500 13px system-ui;background:#1f2937;color:#fff;"
	        "padding:9px 12px;border-radius:10px;box-shadow:0 8px 24px rgba(0,0,0,.25);display:flex;gap:10px;align-items:center';"
	        "var l=document.createElement('a');l.href=a[1];l.textContent='Download';l.style.color='#93c5fd';var x=document.createElement('button');x.setAttribute('aria-label','Dismiss');x.textContent='×';x.style.cssText='all:unset;cursor:pointer;opacity:.7';"
	        "b.textContent='si '+a[0].replace(/^v/,'')+' is available';b.appendChild(l);b.appendChild(x);"
	        "x.onclick=function(){b.remove()};document.body.appendChild(b)})(%@);", args];
}

// isExternal: a web page of another site, which opens in the default browser. Everything
// else stays in the app: the Studio (loopback), and windows with no address yet, like
// the teleprompter the deck opens with window.open('') and then writes itself.
static BOOL isExternal(NSURL *u) {
	NSString *scheme = u.scheme.lowercaseString, *h = u.host.lowercaseString;
	if (![scheme isEqualToString:@"http"] && ![scheme isEqualToString:@"https"]) return NO;
	return !([h isEqualToString:@"127.0.0.1"] || [h isEqualToString:@"localhost"] || [h isEqualToString:@"[::1]"]);
}

@implementation SIWindow

+ (SIWindow *)windowWithConfiguration:(WKWebViewConfiguration *)cfg size:(NSSize)size title:(NSString *)title {
	SIWindow *w = [SIWindow new];
	NSRect frame = NSMakeRect(0, 0, size.width, size.height);
	w.window = [[NSWindow alloc] initWithContentRect:frame
	                                       styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable |
	                                                 NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable
	                                         backing:NSBackingStoreBuffered
	                                           defer:NO];
	w.window.title = title;
	w.window.releasedWhenClosed = NO;
	w.window.delegate = w;
	w.window.collectionBehavior = NSWindowCollectionBehaviorFullScreenPrimary;
	w.window.tabbingMode = NSWindowTabbingModeDisallowed;
	if (@available(macOS 12.3, *)) cfg.preferences.elementFullscreenEnabled = YES; // Present: Full screen
	w.web = [[WKWebView alloc] initWithFrame:frame configuration:cfg];
	w.web.UIDelegate = w;
	w.web.navigationDelegate = w;
	w.web.allowsBackForwardNavigationGestures = NO;
	if (@available(macOS 13.3, *)) w.web.inspectable = YES; // Safari ▸ Develop, for the curious
	w.window.contentView = w.web;
	[w.window center];
	[w.window makeKeyAndOrderFront:nil];
	[windows addObject:w];
	return w;
}

// window.open and target=_blank (the teleprompter, Play, Open in a new tab): a new window
- (WKWebView *)webView:(WKWebView *)webView createWebViewWithConfiguration:(WKWebViewConfiguration *)cfg
       forNavigationAction:(WKNavigationAction *)action
            windowFeatures:(WKWindowFeatures *)features {
	if (isExternal(action.request.URL)) {
		[[NSWorkspace sharedWorkspace] openURL:action.request.URL];
		return nil;
	}
	NSSize size = NSMakeSize(features.width ? features.width.doubleValue : 1280, features.height ? features.height.doubleValue : 800);
	SIWindow *w = [SIWindow windowWithConfiguration:cfg size:size title:@"si Studio"];
	return w.web; // WebKit loads the request into it
}

- (void)webViewDidClose:(WKWebView *)webView {
	[self.window close];
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationAction:(WKNavigationAction *)action
    decisionHandler:(void (^)(WKNavigationActionPolicy))decide {
	NSURL *u = action.request.URL;
	if (isExternal(u)) {
		[[NSWorkspace sharedWorkspace] openURL:u];
		decide(WKNavigationActionPolicyCancel);
		return;
	}
	if ([u.scheme isEqualToString:@"mailto"] || [u.scheme isEqualToString:@"tel"]) {
		[[NSWorkspace sharedWorkspace] openURL:u];
		decide(WKNavigationActionPolicyCancel);
		return;
	}
	decide(WKNavigationActionPolicyAllow);
}

- (void)webView:(WKWebView *)webView didFinishNavigation:(WKNavigation *)navigation {
	if (webView.title.length && !self.studioURL) self.window.title = webView.title;
}

// alert, confirm and prompt from the page
- (void)webView:(WKWebView *)webView runJavaScriptAlertPanelWithMessage:(NSString *)message
    initiatedByFrame:(WKFrameInfo *)frame completionHandler:(void (^)(void))done {
	NSAlert *a = [NSAlert new];
	a.messageText = message;
	[a beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse r) { done(); }];
}

- (void)webView:(WKWebView *)webView runJavaScriptConfirmPanelWithMessage:(NSString *)message
    initiatedByFrame:(WKFrameInfo *)frame completionHandler:(void (^)(BOOL))done {
	NSAlert *a = [NSAlert new];
	a.messageText = message;
	[a addButtonWithTitle:@"OK"];
	[a addButtonWithTitle:@"Cancel"];
	[a beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse r) { done(r == NSAlertFirstButtonReturn); }];
}

- (void)webView:(WKWebView *)webView runJavaScriptTextInputPanelWithPrompt:(NSString *)prompt
    defaultText:(NSString *)text initiatedByFrame:(WKFrameInfo *)frame completionHandler:(void (^)(NSString *))done {
	NSAlert *a = [NSAlert new];
	a.messageText = prompt;
	NSTextField *f = [[NSTextField alloc] initWithFrame:NSMakeRect(0, 0, 260, 24)];
	f.stringValue = text ?: @"";
	a.accessoryView = f;
	[a addButtonWithTitle:@"OK"];
	[a addButtonWithTitle:@"Cancel"];
	[a beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse r) { done(r == NSAlertFirstButtonReturn ? f.stringValue : nil); }];
}

- (void)windowWillClose:(NSNotification *)n {
	if (self.studioURL) goCloseURL((char *)self.studioURL.UTF8String);
	self.web.UIDelegate = nil;
	self.web.navigationDelegate = nil;
	dispatch_async(dispatch_get_main_queue(), ^{ [windows removeObject:self]; });
}

@end

static NSMenuItem *item(NSString *title, SEL action, NSString *key) {
	return [[NSMenuItem alloc] initWithTitle:title action:action keyEquivalent:key];
}

static NSMenu *submenu(NSMenu *bar, NSString *title) {
	NSMenuItem *top = [NSMenuItem new];
	NSMenu *m = [[NSMenu alloc] initWithTitle:title];
	top.submenu = m;
	[bar addItem:top];
	return m;
}

@implementation SIApp

- (void)applicationWillFinishLaunching:(NSNotification *)n {
	windows = [NSMutableArray new]; // a folder dropped on the icon arrives before didFinishLaunching
	[self buildMenu];
}

- (void)applicationDidFinishLaunching:(NSNotification *)n {
	if (windows.count) { [NSApp activateIgnoringOtherApps:YES]; return; }
	NSArray<NSString *> *args = NSProcessInfo.processInfo.arguments;
	NSString *dir = nil;
	for (NSUInteger i = 1; i < args.count; i++) {
		BOOL isDir = NO;
		if (![args[i] hasPrefix:@"-"] && [[NSFileManager defaultManager] fileExistsAtPath:args[i] isDirectory:&isDir] && isDir) dir = args[i];
	}
	if (!dir) {
		NSString *last = [[NSUserDefaults standardUserDefaults] stringForKey:kLastFolder];
		BOOL isDir = NO;
		if (last && [[NSFileManager defaultManager] fileExistsAtPath:last isDirectory:&isDir] && isDir) dir = last;
	}
	if (dir) [self openPath:dir];
	else if (windows.count == 0) [self openFolder:nil];
	[NSApp activateIgnoringOtherApps:YES];
	[self checkStatus];
}

// checkStatus asks Go which si runs and whether a newer release is out (it may
// wait up to 3 s for GitHub, so off the main thread), then updates menu and windows.
- (void)checkStatus {
	dispatch_async(dispatch_get_global_queue(QOS_CLASS_UTILITY, 0), ^{
		char *r = goStatus();
		NSData *d = [NSData dataWithBytes:r length:strlen(r)];
		free(r);
		NSDictionary *s = [NSJSONSerialization JSONObjectWithData:d options:0 error:nil];
		dispatch_async(dispatch_get_main_queue(), ^{
			siStatus = s;
			NSString *tag = s[@"update"];
			if (tag.length) {
				NSMenu *app = NSApp.mainMenu.itemArray.firstObject.submenu;
				NSMenuItem *it = item([NSString stringWithFormat:@"Update Available — %@", tag], @selector(openUpdate:), @"");
				it.target = self;
				[app insertItem:it atIndex:1];
				NSString *js = bannerJS();
				for (SIWindow *w in windows)
					if (w.studioURL) [w.web evaluateJavaScript:js completionHandler:nil];
			}
		});
	});
}

- (void)openUpdate:(id)sender {
	NSString *url = siStatus[@"url"];
	if (url.length) [[NSWorkspace sharedWorkspace] openURL:[NSURL URLWithString:url]];
}

- (void)showAbout:(id)sender {
	NSString *v = siStatus[@"si"] ?: @"", *note = siStatus[@"note"] ?: @"";
	NSString *text = [NSString stringWithFormat:@"Runs si %@%@", v, note.length ? [@"\n" stringByAppendingString:note] : @""];
	[NSApp orderFrontStandardAboutPanelWithOptions:@{NSAboutPanelOptionCredits: [[NSAttributedString alloc] initWithString:text]}];
}

// a folder dropped on the Dock icon, or opened with `open -a "si Studio" folder`
- (BOOL)application:(NSApplication *)app openFile:(NSString *)path {
	BOOL isDir = NO;
	if (![[NSFileManager defaultManager] fileExistsAtPath:path isDirectory:&isDir]) return NO;
	[self openPath:isDir ? path : path.stringByDeletingLastPathComponent];
	return YES;
}

- (void)openFolder:(id)sender {
	NSOpenPanel *p = [NSOpenPanel openPanel];
	p.canChooseDirectories = YES;
	p.canChooseFiles = NO;
	p.allowsMultipleSelection = NO;
	p.message = @"Choose a folder of decks, documents or a site";
	p.prompt = @"Open";
	if ([p runModal] == NSModalResponseOK) [self openPath:p.URL.path];
	else if (windows.count == 0 && !sender) [NSApp terminate:nil];
}

- (void)openPath:(NSString *)path {
	char *r = goOpenFolder((char *)path.UTF8String);
	NSString *res = [NSString stringWithUTF8String:r];
	free(r);
	if ([res hasPrefix:@"error:"]) {
		NSAlert *a = [NSAlert new];
		a.messageText = [NSString stringWithFormat:@"si Studio cannot open %@", path.lastPathComponent];
		a.informativeText = [res substringFromIndex:6];
		[a runModal];
		return;
	}
	[[NSUserDefaults standardUserDefaults] setObject:path forKey:kLastFolder];
	[[NSDocumentController sharedDocumentController] noteNewRecentDocumentURL:[NSURL fileURLWithPath:path]];
	NSMutableArray *recent = [([[NSUserDefaults standardUserDefaults] arrayForKey:kRecent] ?: @[]) mutableCopy];
	[recent removeObject:path];
	[recent insertObject:path atIndex:0];
	while (recent.count > 10) [recent removeLastObject];
	[[NSUserDefaults standardUserDefaults] setObject:recent forKey:kRecent];
	WKWebViewConfiguration *cfg = [WKWebViewConfiguration new];
	cfg.websiteDataStore = [WKWebsiteDataStore defaultDataStore];
	NSString *js = bannerJS();
	if (js) [cfg.userContentController addUserScript:[[WKUserScript alloc] initWithSource:js injectionTime:WKUserScriptInjectionTimeAtDocumentEnd forMainFrameOnly:YES]];
	NSSize size = NSScreen.mainScreen ? NSMakeSize(NSScreen.mainScreen.visibleFrame.size.width * 0.9, NSScreen.mainScreen.visibleFrame.size.height * 0.9)
	                                  : NSMakeSize(1440, 900);
	SIWindow *w = [SIWindow windowWithConfiguration:cfg size:size title:[NSString stringWithFormat:@"si Studio — %@", path.lastPathComponent]];
	w.studioURL = res;
	w.window.representedURL = [NSURL fileURLWithPath:path];
	[w.web loadRequest:[NSURLRequest requestWithURL:[NSURL URLWithString:res]]];
}

- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)app {
	return YES;
}

- (void)applicationWillTerminate:(NSNotification *)n {
	goShutdown();
}

// Open Recent: the folders opened last, newest first; missing ones are skipped.
- (void)menuNeedsUpdate:(NSMenu *)menu {
	[menu removeAllItems];
	NSArray<NSString *> *recent = [[NSUserDefaults standardUserDefaults] arrayForKey:kRecent] ?: @[];
	for (NSString *path in recent) {
		BOOL isDir = NO;
		if (![[NSFileManager defaultManager] fileExistsAtPath:path isDirectory:&isDir] || !isDir) continue;
		NSMenuItem *it = item(path.lastPathComponent, @selector(openRecent:), @"");
		it.target = self;
		it.representedObject = path;
		it.toolTip = path;
		it.image = [[NSWorkspace sharedWorkspace] iconForFile:path];
		it.image.size = NSMakeSize(16, 16);
		[menu addItem:it];
	}
	if (menu.numberOfItems == 0) {
		NSMenuItem *none = item(@"No Recent Folders", nil, @"");
		none.enabled = NO;
		[menu addItem:none];
		return;
	}
	[menu addItem:[NSMenuItem separatorItem]];
	NSMenuItem *clear = item(@"Clear Menu", @selector(clearRecent:), @"");
	clear.target = self;
	[menu addItem:clear];
}

- (void)openRecent:(NSMenuItem *)sender {
	[self openPath:sender.representedObject];
}

- (void)clearRecent:(id)sender {
	[[NSUserDefaults standardUserDefaults] removeObjectForKey:kRecent];
}

- (void)buildMenu {
	NSMenu *bar = [NSMenu new];
	NSMenu *app = submenu(bar, @"si Studio");
	NSMenuItem *about = item(@"About si Studio", @selector(showAbout:), @"");
	about.target = self;
	[app addItem:about];
	[app addItem:[NSMenuItem separatorItem]];
	[app addItem:item(@"Hide si Studio", @selector(hide:), @"h")];
	NSMenuItem *others = item(@"Hide Others", @selector(hideOtherApplications:), @"h");
	others.keyEquivalentModifierMask = NSEventModifierFlagCommand | NSEventModifierFlagOption;
	[app addItem:others];
	[app addItem:item(@"Show All", @selector(unhideAllApplications:), @"")];
	[app addItem:[NSMenuItem separatorItem]];
	[app addItem:item(@"Quit si Studio", @selector(terminate:), @"q")];

	NSMenu *file = submenu(bar, @"File");
	NSMenuItem *open = item(@"Open Folder…", @selector(openFolder:), @"o");
	open.target = self;
	[file addItem:open];
	NSMenuItem *recentItem = item(@"Open Recent", nil, @"");
	NSMenu *recent = [[NSMenu alloc] initWithTitle:@"Open Recent"];
	recent.delegate = self; // filled from the saved list each time it opens
	recentItem.submenu = recent;
	[file addItem:recentItem];
	[file addItem:[NSMenuItem separatorItem]];
	[file addItem:item(@"Close Window", @selector(performClose:), @"w")];

	// the web editor needs these for ⌘Z ⌘X ⌘C ⌘V ⌘A to reach it
	NSMenu *edit = submenu(bar, @"Edit");
	[edit addItem:item(@"Undo", @selector(undo:), @"z")];
	NSMenuItem *redo = item(@"Redo", @selector(redo:), @"z");
	redo.keyEquivalentModifierMask = NSEventModifierFlagCommand | NSEventModifierFlagShift;
	[edit addItem:redo];
	[edit addItem:[NSMenuItem separatorItem]];
	[edit addItem:item(@"Cut", @selector(cut:), @"x")];
	[edit addItem:item(@"Copy", @selector(copy:), @"c")];
	[edit addItem:item(@"Paste", @selector(paste:), @"v")];
	[edit addItem:item(@"Select All", @selector(selectAll:), @"a")];

	NSMenu *view = submenu(bar, @"View");
	[view addItem:item(@"Reload", @selector(reload:), @"r")];
	NSMenuItem *fs = item(@"Enter Full Screen", @selector(toggleFullScreen:), @"f");
	fs.keyEquivalentModifierMask = NSEventModifierFlagCommand | NSEventModifierFlagControl;
	[view addItem:fs];

	NSMenu *win = submenu(bar, @"Window");
	[win addItem:item(@"Minimize", @selector(performMiniaturize:), @"m")];
	[win addItem:item(@"Zoom", @selector(performZoom:), @"")];
	NSApp.windowsMenu = win;

	NSApp.mainMenu = bar;
}

@end

void siRun(void) {
	@autoreleasepool {
		NSApplication *app = [NSApplication sharedApplication];
		app.activationPolicy = NSApplicationActivationPolicyRegular;
		SIApp *delegate = [SIApp new];
		app.delegate = delegate;
		[app run];
	}
}
