//go:build tray && darwin

// macOS menubar status item for the tiny-app window.
//
// This turns the tray build into a proper menubar app: an icon lives in the
// system status bar, a single click toggles the panel as a popover positioned
// under the icon, clicking away dismisses it, and a right-/ctrl-click menu
// quits. The app runs as an "accessory" (no Dock icon, no app-switcher entry) —
// the Wi-Fi-menu shape the panel was designed for.
//
// It reuses the SAME NSApplication and NSWindow that webview created (passed in
// as the NSWindow* from webview.Window()) rather than an NSPopover, so the
// WKWebView keeps focus and input, and there is one Cocoa run loop — webview's —
// not two competing ones. The window is just restyled chromeless and repointed
// under the icon on each show. Everything here runs on the main thread:
// runWindow calls installStatusItem on the main goroutine, which webview_go pins
// to the main OS thread, before webview's Run().
package main

/*
#cgo darwin CFLAGS: -x objective-c
#cgo darwin LDFLAGS: -framework Cocoa
#import <Cocoa/Cocoa.h>

@interface ValveTray : NSObject <NSWindowDelegate>
@property (strong) NSWindow *window;
@property (strong) NSStatusItem *item;
@property (strong) NSMenu *menu;
@property (strong) id monitor;
@end

@implementation ValveTray

// Place the window centered under the status-bar icon, just below the menubar,
// clamped to the icon's screen.
- (void)positionUnderIcon {
    NSStatusBarButton *b = self.item.button;
    NSRect br = b.window.frame; // status button's window, in screen coords
    NSSize ws = self.window.frame.size;
    NSScreen *scr = b.window.screen ?: NSScreen.mainScreen;
    NSRect vis = scr.visibleFrame;
    CGFloat x = NSMidX(br) - ws.width / 2.0;
    CGFloat maxX = NSMaxX(vis) - ws.width - 8;
    if (x > maxX) x = maxX;
    if (x < NSMinX(vis) + 8) x = NSMinX(vis) + 8;
    [self.window setFrameTopLeftPoint:NSMakePoint(x, NSMinY(br) - 6)];
}

- (void)showPopover {
    [self positionUnderIcon];
    [NSApp activateIgnoringOtherApps:YES];
    [self.window makeKeyAndOrderFront:nil];
}

- (void)hidePopover {
    [self.window orderOut:nil];
}

- (void)toggle {
    if (self.window.isVisible) [self hidePopover];
    else [self showPopover];
}

// Left click toggles the popover; right-/ctrl-click opens the menu (Quit lives
// there since there is no left-click menu anymore).
- (void)statusClicked:(id)sender {
    NSEvent *e = NSApp.currentEvent;
    BOOL secondary = e.type == NSEventTypeRightMouseUp ||
                     (e.modifierFlags & NSEventModifierFlagControl) != 0;
    if (secondary) {
        [self.menu popUpMenuPositioningItem:nil
                                 atLocation:NSZeroPoint
                                     inView:self.item.button];
    } else {
        [self toggle];
    }
}

- (void)open:(id)sender { [self showPopover]; }
- (void)quit:(id)sender { [NSApp terminate:nil]; }

// Cmd-W / any close request hides rather than tearing down the run loop, so the
// app keeps living in the menubar.
- (BOOL)windowShouldClose:(NSWindow *)sender {
    [sender orderOut:nil];
    return NO;
}
@end

// Strong global so ARC keeps the controller (menu targets, window delegate,
// event monitor) alive for the whole process; a local would be collected.
static ValveTray *gValveTray = nil;

void valveSetHealth(int kind); // defined below; install paints an initial dot

// valveHubImage draws the hub/route mark — a central node routing out to three
// others — as a template NSImage, so macOS tints it for the light/dark menubar.
// Drawn with NSBezierPath rather than shipping a PNG so it stays crisp at any
// backing scale. flipped:YES gives a top-left origin matching the 24-unit grid
// the mark is authored in (shared with cmd/valve-node-app/icon.svg).
static NSImage *valveHubImage(void) {
    const CGFloat S = 18.0;
    NSImage *img = [NSImage imageWithSize:NSMakeSize(S, S) flipped:YES
        drawingHandler:^BOOL(NSRect dst) {
            const CGFloat u = S / 24.0; // 24-grid unit → points
            [[NSColor blackColor] set];
            NSPoint c  = NSMakePoint(12 * u, 12 * u);
            NSPoint up = NSMakePoint(12 * u, 5 * u);
            NSPoint ll = NSMakePoint(6 * u, 18 * u);
            NSPoint lr = NSMakePoint(18 * u, 18 * u);
            NSBezierPath *spokes = [NSBezierPath bezierPath];
            spokes.lineWidth = 1.9 * u;
            spokes.lineCapStyle = NSLineCapStyleRound;
            [spokes moveToPoint:c]; [spokes lineToPoint:up];
            [spokes moveToPoint:c]; [spokes lineToPoint:ll];
            [spokes moveToPoint:c]; [spokes lineToPoint:lr];
            [spokes stroke];
            void (^node)(NSPoint, CGFloat) = ^(NSPoint p, CGFloat r) {
                [[NSBezierPath bezierPathWithOvalInRect:
                    NSMakeRect(p.x - r, p.y - r, 2 * r, 2 * r)] fill];
            };
            node(c, 3.0 * u);
            node(up, 2.2 * u);
            node(ll, 2.2 * u);
            node(lr, 2.2 * u);
            return YES;
        }];
    img.template = YES;
    img.accessibilityDescription = @"Jumpgate";
    return img;
}

void valveInstallStatusItem(void *nswindow) {
    NSApplication *app = [NSApplication sharedApplication];
    // Menubar app: live in the status bar, not the Dock or app switcher.
    [app setActivationPolicy:NSApplicationActivationPolicyAccessory];

    ValveTray *tray = [[ValveTray alloc] init];
    tray.window = (__bridge NSWindow *)nswindow;

    // Restyle webview's titled window into a chromeless popover panel. It stays
    // titled (so it can still become key for text input) but the title bar is
    // transparent, the title hidden, and the traffic-light buttons removed.
    NSWindow *win = tray.window;
    win.styleMask |= NSWindowStyleMaskFullSizeContentView;
    win.titlebarAppearsTransparent = YES;
    win.titleVisibility = NSWindowTitleHidden;
    [win standardWindowButton:NSWindowCloseButton].hidden = YES;
    [win standardWindowButton:NSWindowMiniaturizeButton].hidden = YES;
    [win standardWindowButton:NSWindowZoomButton].hidden = YES;
    win.level = NSFloatingWindowLevel; // hover above ordinary windows
    win.delegate = tray;

    NSStatusItem *item = [[NSStatusBar systemStatusBar]
        statusItemWithLength:NSVariableStatusItemLength];
    item.button.image = valveHubImage(); // custom hub glyph (template)
    // One click = our action, not a dropdown; also deliver right-clicks so the
    // menu can open there.
    item.button.target = tray;
    item.button.action = @selector(statusClicked:);
    [item.button sendActionOn:(NSEventMaskLeftMouseUp | NSEventMaskRightMouseUp)];
    item.button.imagePosition = NSImageLeft; // glyph, then the health dot
    tray.item = item;

    NSMenu *menu = [[NSMenu alloc] init];
    NSMenuItem *open = [[NSMenuItem alloc] initWithTitle:@"Open Jumpgate"
                                                  action:@selector(open:)
                                           keyEquivalent:@""];
    open.target = tray;
    [menu addItem:open];
    [menu addItem:[NSMenuItem separatorItem]];
    NSMenuItem *quit = [[NSMenuItem alloc] initWithTitle:@"Quit Jumpgate"
                                                  action:@selector(quit:)
                                           keyEquivalent:@"q"];
    quit.target = tray;
    [menu addItem:quit];
    tray.menu = menu;

    // Dismiss when the user clicks another app or the desktop. Clicks on our
    // own status button are in-process, so this global monitor never sees them —
    // that is what keeps a click on the icon a clean single toggle instead of
    // hide-then-reopen.
    tray.monitor = [NSEvent
        addGlobalMonitorForEventsMatchingMask:(NSEventMaskLeftMouseDown | NSEventMaskRightMouseDown)
                                      handler:^(NSEvent *ev) {
                                          if (tray.window.isVisible) [tray hidePopover];
                                      }];

    gValveTray = tray;

    valveSetHealth(0); // neutral dot until the first health poll lands

    // Start hidden and open under the icon on the next run-loop tick (once the
    // status button has a real frame to anchor to), so launch shows a popover
    // under the icon rather than a centered window flash.
    [win orderOut:nil];
    dispatch_async(dispatch_get_main_queue(), ^{ [tray showPopover]; });
}

// valveSetHealth repaints the status dot beside the glyph from a healthKind
// (0 off/grey, 1 ok/green, 2 warn/amber, 3 down/red). The base glyph stays a
// template image (system tints it for the light/dark menubar); the dot is a
// small colored "●" set as the button's title, so its color survives. Must run
// on the main thread — the poller marshals via webview.Dispatch.
void valveSetHealth(int kind) {
    if (gValveTray == nil) return;
    NSColor *color;
    NSString *tip;
    switch (kind) {
        case 1: color = NSColor.systemGreenColor;  tip = @"Jumpgate — serving"; break;
        case 2: color = NSColor.systemOrangeColor; tip = @"Jumpgate — degraded"; break;
        case 3: color = NSColor.systemRedColor;    tip = @"Jumpgate — a gateway is unavailable"; break;
        default: color = NSColor.tertiaryLabelColor; tip = @"Jumpgate — idle"; break;
    }
    NSDictionary *attrs = @{
        NSForegroundColorAttributeName: color,
        NSFontAttributeName: [NSFont systemFontOfSize:9],
    };
    gValveTray.item.button.attributedTitle =
        [[NSAttributedString alloc] initWithString:@" ●" attributes:attrs];
    gValveTray.item.button.toolTip = tip;
}
*/
import "C"

import "unsafe"

// installStatusItem adds the menubar status item bound to the webview window.
func installStatusItem(nswindow unsafe.Pointer) {
	C.valveInstallStatusItem(nswindow)
}

// setHealth repaints the menubar status dot. Must be called on the main thread
// (the poller marshals via webview.Dispatch).
func setHealth(k healthKind) {
	C.valveSetHealth(C.int(k))
}
