//go:build tray && darwin

// macOS menubar status item for the tiny-app window.
//
// This turns the tray build into a proper menubar app: an icon lives in the
// system status bar, its menu opens/quits the panel window, and the app runs as
// an "accessory" (no Dock icon, no app-switcher entry) — the Wi-Fi-menu shape
// the panel was designed for. Closing the window's red button hides it rather
// than quitting, so the menubar icon stays put and reopens it.
//
// It hooks the SAME NSApplication and NSWindow that webview created (passed in
// as the NSWindow* from webview.Window()), so there is one Cocoa run loop —
// webview's — not two competing ones. Everything here must run on the main
// thread; runWindow calls installStatusItem on the main goroutine, which
// webview_go pins to the main OS thread, before webview's Run().
package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

@interface ValveTray : NSObject <NSWindowDelegate>
@property (strong) NSWindow *window;
@property (strong) NSStatusItem *item;
@end

@implementation ValveTray
- (void)open:(id)sender {
    [NSApp activateIgnoringOtherApps:YES];
    [self.window makeKeyAndOrderFront:nil];
}
- (void)quit:(id)sender {
    [NSApp terminate:nil];
}
// Red-button close hides the window instead of tearing down the run loop, so
// the app keeps living in the menubar. Reopen from the status-item menu.
- (BOOL)windowShouldClose:(NSWindow *)sender {
    [sender orderOut:nil];
    return NO;
}
@end

// Strong global so ARC keeps the controller (its menu targets and window
// delegate) alive for the whole process; a local would be collected.
static ValveTray *gValveTray = nil;

void valveInstallStatusItem(void *nswindow) {
    NSApplication *app = [NSApplication sharedApplication];
    // Menubar app: live in the status bar, not the Dock or app switcher.
    [app setActivationPolicy:NSApplicationActivationPolicyAccessory];

    ValveTray *tray = [[ValveTray alloc] init];
    tray.window = (__bridge NSWindow *)nswindow;
    tray.window.delegate = tray;

    NSStatusItem *item = [[NSStatusBar systemStatusBar]
        statusItemWithLength:NSVariableStatusItemLength];
    NSImage *img = nil;
    if (@available(macOS 11.0, *)) {
        img = [NSImage imageWithSystemSymbolName:@"network"
                        accessibilityDescription:@"Valve"];
    }
    if (img) {
        img.template = YES; // adapt to light/dark menubar
        item.button.image = img;
    } else {
        item.button.title = @"◈"; // ◈ fallback for pre-11 menubars
    }

    NSMenu *menu = [[NSMenu alloc] init];
    NSMenuItem *open = [[NSMenuItem alloc] initWithTitle:@"Open Valve Panel"
                                                  action:@selector(open:)
                                           keyEquivalent:@""];
    open.target = tray;
    [menu addItem:open];
    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quit = [[NSMenuItem alloc] initWithTitle:@"Quit Valve"
                                                  action:@selector(quit:)
                                           keyEquivalent:@"q"];
    quit.target = tray;
    [menu addItem:quit];
    item.menu = menu;

    tray.item = item;
    gValveTray = tray;
}
*/
import "C"

import "unsafe"

// installStatusItem adds the menubar status item bound to the webview window.
func installStatusItem(nswindow unsafe.Pointer) {
	C.valveInstallStatusItem(nswindow)
}
