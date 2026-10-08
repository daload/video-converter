#import <Cocoa/Cocoa.h>
#import "window_darwin.h"
#import "_cgo_export.h"
#import "window_test_darwin.h"

@interface CVWindow : NSObject <NSApplicationDelegate, NSWindowDelegate>
@property NSWindow *window;
@property NSTextField *file;
@property NSPopUpButton *codec;
@property NSPopUpButton *format;
@property NSTextField *help;
@property NSButton *choose;
@property NSButton *convert;
@property NSProgressIndicator *progress;
@property NSTextView *message;
@property NSTimer *timer;
@property NSDictionary *state;
@property CVWindowTest *test;
@property NSString *error;
@property BOOL stopping;
- (void)stop;
@end

static __weak CVWindow *activeWindow;

static NSTextField *label(NSString *text, CGFloat size) {
    NSTextField *field = [NSTextField wrappingLabelWithString:text];
    field.font = [NSFont systemFontOfSize:size];
    field.selectable = YES;
    return field;
}

static NSStackView *stack(NSArray<NSView *> *views, NSUserInterfaceLayoutOrientation orientation) {
    NSStackView *view = [NSStackView stackViewWithViews:views];
    view.orientation = orientation;
    view.alignment = orientation == NSUserInterfaceLayoutOrientationVertical ? NSLayoutAttributeLeading : NSLayoutAttributeCenterY;
    view.spacing = 10;
    return view;
}

@implementation CVWindow
- (void)build {
    self.window = [[NSWindow alloc] initWithContentRect:NSMakeRect(0, 0, 660, 520)
        styleMask:NSWindowStyleMaskTitled | NSWindowStyleMaskClosable | NSWindowStyleMaskMiniaturizable | NSWindowStyleMaskResizable
        backing:NSBackingStoreBuffered defer:NO];
    self.window.title = @"Conversor de Video";
    self.window.minSize = NSMakeSize(640, 520);
    self.window.delegate = self;
    self.window.releasedWhenClosed = NO;
    [self.window center];
    NSTextField *title = label(@"Convierte tu video", 25);
    title.font = [NSFont boldSystemFontOfSize:25];
    NSTextField *intro = label(@"Selecciona un video, un códec y un formato.", 13);
    intro.textColor = NSColor.secondaryLabelColor;
    self.choose = [NSButton buttonWithTitle:@"Seleccionar video" target:self action:@selector(chooseVideo:)];
    self.choose.bezelStyle = NSBezelStyleRounded;
    self.file = label(@"No se ha seleccionado un video", 13);
    [self.file.heightAnchor constraintGreaterThanOrEqualToConstant:40].active = YES;
    NSStackView *fileRow = stack(@[self.choose, self.file], NSUserInterfaceLayoutOrientationHorizontal);
    [self.choose.widthAnchor constraintEqualToConstant:160].active = YES;

    self.codec = [[NSPopUpButton alloc] initWithFrame:NSZeroRect pullsDown:NO];
    self.codec.target = self;
    self.codec.action = @selector(codecChanged:);
    [self.codec setAccessibilityLabel:@"Códec"];
    self.format = [[NSPopUpButton alloc] initWithFrame:NSZeroRect pullsDown:NO];
    self.format.target = self;
    self.format.action = @selector(formatChanged:);
    [self.format setAccessibilityLabel:@"Formato"];
    NSStackView *codecColumn = stack(@[label(@"Códec", 13), self.codec], NSUserInterfaceLayoutOrientationVertical);
    NSStackView *formatColumn = stack(@[label(@"Formato", 13), self.format], NSUserInterfaceLayoutOrientationVertical);
    [self.codec.widthAnchor constraintEqualToAnchor:codecColumn.widthAnchor].active = YES;
    [self.format.widthAnchor constraintEqualToAnchor:formatColumn.widthAnchor].active = YES;
    NSStackView *choices = stack(@[codecColumn, formatColumn], NSUserInterfaceLayoutOrientationHorizontal);
    [codecColumn.widthAnchor constraintEqualToAnchor:formatColumn.widthAnchor multiplier:2].active = YES;

    self.help = label(@"", 13);
    self.help.textColor = NSColor.secondaryLabelColor;
    self.convert = [NSButton buttonWithTitle:@"Convertir" target:self action:@selector(convertVideo:)];
    self.convert.bezelStyle = NSBezelStyleRounded;
    self.convert.keyEquivalent = @"\r";
    [self.convert.widthAnchor constraintEqualToConstant:125].active = YES;
    self.progress = [[NSProgressIndicator alloc] initWithFrame:NSZeroRect];
    self.progress.style = NSProgressIndicatorStyleBar;
    self.progress.minValue = 0;
    self.progress.maxValue = 100;

    NSScrollView *scroll = [[NSScrollView alloc] initWithFrame:NSZeroRect];
    scroll.hasVerticalScroller = YES;
    scroll.drawsBackground = NO;
    self.message = [[NSTextView alloc] initWithFrame:NSMakeRect(0, 0, 580, 110)];
    self.message.editable = NO;
    self.message.selectable = YES;
    self.message.drawsBackground = NO;
    self.message.font = [NSFont systemFontOfSize:13];
    self.message.verticallyResizable = YES;
    self.message.horizontallyResizable = NO;
    self.message.autoresizingMask = NSViewWidthSizable;
    self.message.textContainer.widthTracksTextView = YES;
    [self.message setAccessibilityLabel:@"Estado de la conversión"];
    scroll.documentView = self.message;
    [scroll.heightAnchor constraintGreaterThanOrEqualToConstant:110].active = YES;

    NSStackView *body = stack(@[title, intro, fileRow, choices, self.help, self.convert, self.progress, scroll],
        NSUserInterfaceLayoutOrientationVertical);
    body.spacing = 16;
    body.translatesAutoresizingMaskIntoConstraints = NO;
    [self.window.contentView addSubview:body];
    [NSLayoutConstraint activateConstraints:@[
        [body.leadingAnchor constraintEqualToAnchor:self.window.contentView.leadingAnchor constant:28],
        [body.trailingAnchor constraintEqualToAnchor:self.window.contentView.trailingAnchor constant:-28],
        [body.topAnchor constraintEqualToAnchor:self.window.contentView.topAnchor constant:24],
        [body.bottomAnchor constraintEqualToAnchor:self.window.contentView.bottomAnchor constant:-24]
    ]];
    for (NSView *view in @[title, intro, fileRow, choices, self.help, self.progress, scroll]) {
        [view.widthAnchor constraintEqualToAnchor:body.widthAnchor].active = YES;
    }
    self.window.initialFirstResponder = self.choose;
    NSMenu *menu = [[NSMenu alloc] initWithTitle:@"Conversor de Video"];
    NSMenuItem *appItem = [[NSMenuItem alloc] initWithTitle:@"Conversor de Video" action:NULL keyEquivalent:@""];
    NSMenu *appMenu = [[NSMenu alloc] initWithTitle:@"Conversor de Video"];
    [appMenu addItemWithTitle:@"Salir de Conversor de Video" action:@selector(terminate:) keyEquivalent:@"q"];
    appItem.submenu = appMenu;
    [menu addItem:appItem];
    NSApp.mainMenu = menu;
    cv_tick();
    [self.window makeKeyAndOrderFront:nil];
    [NSApp activateIgnoringOtherApps:YES];
    self.timer = [NSTimer scheduledTimerWithTimeInterval:0.2 target:self selector:@selector(tick:) userInfo:nil repeats:YES];
}

- (void)chooseVideo:(id)sender {
    NSOpenPanel *panel = [NSOpenPanel openPanel];
    panel.title = @"Seleccionar un video";
    panel.prompt = @"Seleccionar";
    panel.canChooseFiles = YES;
    panel.canChooseDirectories = NO;
    panel.allowsMultipleSelection = NO;
    [panel beginSheetModalForWindow:self.window completionHandler:^(NSModalResponse result) {
        if (result == NSModalResponseOK) {
            cv_action(0, (char *)panel.URL.path.UTF8String);
            cv_tick();
        }
    }];
    if (self.test) {
        dispatch_after(dispatch_time(DISPATCH_TIME_NOW, NSEC_PER_SEC / 5), dispatch_get_main_queue(), ^{
            [panel cancel:nil];
        });
    }
}

- (void)codecChanged:(id)sender {
    cv_action(1, (char *)[self.codec.selectedItem.representedObject UTF8String]);
    cv_tick();
}

- (void)formatChanged:(id)sender {
    cv_action(2, (char *)[self.format.selectedItem.representedObject UTF8String]);
    cv_tick();
}

- (void)convertVideo:(id)sender {
    cv_action(3, NULL);
    cv_tick();
}

- (void)tick:(NSTimer *)timer {
    cv_tick();
    [self.test tickWithState:self.state choose:self.choose codec:self.codec format:self.format
        convert:self.convert progress:self.progress message:self.message window:self.window];
    if (self.test.finished) [self stop];
}

- (void)application:(NSApplication *)application openFiles:(NSArray<NSString *> *)files {
    if (files.count == 1) cv_action(0, (char *)files.firstObject.UTF8String);
    else cv_action(4, NULL);
    cv_tick();
    [application replyToOpenOrPrint:NSApplicationDelegateReplySuccess];
}

- (BOOL)canClose {
    if (self.stopping) return YES;
    if (![self.state[@"busy"] boolValue]) return YES;
    NSAlert *alert = [[NSAlert alloc] init];
    alert.messageText = @"¿Detener la conversión y cerrar?";
    alert.informativeText = @"Se eliminará el archivo incompleto. El video original no cambiará.";
    [alert addButtonWithTitle:@"Seguir convirtiendo"];
    [alert addButtonWithTitle:@"Detener y cerrar"];
    return [alert runModal] == NSAlertSecondButtonReturn;
}

- (BOOL)windowShouldClose:(NSWindow *)sender {
    if (![self canClose]) return NO;
    [self stop];
    return YES;
}

- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)application {
    if ([self canClose]) {
        [self.window orderOut:nil];
        [self stop];
    }
    return NSTerminateCancel;
}

- (void)stop {
    self.stopping = YES;
    [self.timer invalidate];
    [NSApp stop:nil];
    NSEvent *event = [NSEvent otherEventWithType:NSEventTypeApplicationDefined location:NSZeroPoint
        modifierFlags:0 timestamp:0 windowNumber:0 context:nil subtype:0 data1:0 data2:0];
    [NSApp postEvent:event atStart:NO];
}
@end

void cv_render(const char *json) {
    CVWindow *view = activeWindow;
    NSError *error = nil;
    NSData *data = [[NSString stringWithUTF8String:json] dataUsingEncoding:NSUTF8StringEncoding];
    NSDictionary *state = [NSJSONSerialization JSONObjectWithData:data options:0 error:&error];
    if (error || ![state isKindOfClass:NSDictionary.class]) {
        view.error = [NSString stringWithFormat:@"No se pudo leer el estado de la ventana. Detalles técnicos: %@", error];
        [view stop];
        return;
    }
    view.state = state;
    NSArray *options = state[@"options"];
    if (view.codec.numberOfItems != options.count) {
        [view.codec removeAllItems];
        for (NSDictionary *option in options) {
            [view.codec addItemWithTitle:option[@"label"]];
            view.codec.lastItem.representedObject = option[@"id"];
        }
    }
    for (NSDictionary *option in options) {
        if ([option[@"id"] isEqual:state[@"codec"]]) {
            [view.codec selectItemAtIndex:[options indexOfObject:option]];
            view.help.stringValue = option[@"help"];
        }
    }
    NSArray *formats = state[@"formats"];
    NSArray *titles = [formats valueForKey:@"uppercaseString"];
    if (![view.format.itemTitles isEqual:titles]) {
        [view.format removeAllItems];
        for (NSString *format in formats) {
            [view.format addItemWithTitle:format.uppercaseString];
            view.format.lastItem.representedObject = format;
        }
    }
    [view.format selectItemWithTitle:[state[@"format"] uppercaseString]];
    BOOL busy = [state[@"busy"] boolValue];
    BOOL selected = [state[@"file"] length] > 0;
    view.file.stringValue = selected ? state[@"file"] : @"No se ha seleccionado un video";
    view.file.toolTip = state[@"file"];
    view.choose.enabled = !busy;
    view.codec.enabled = !busy;
    view.format.enabled = !busy;
    view.convert.enabled = !busy && selected;
    id progress = state[@"progress"];
    view.progress.indeterminate = busy && progress == NSNull.null;
    if (view.progress.indeterminate) [view.progress startAnimation:nil];
    else {
        [view.progress stopAnimation:nil];
        view.progress.doubleValue = progress == NSNull.null ? 0 : [progress doubleValue];
    }
    if (![view.message.string isEqual:state[@"message"]]) view.message.string = state[@"message"];
    view.message.textColor = [state[@"status"] isEqual:@"failed"] ? NSColor.systemRedColor : NSColor.labelColor;
}

char *cv_run(int test) {
    @autoreleasepool {
        [NSApplication sharedApplication];
        [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
        CVWindow *view = [[CVWindow alloc] init];
        activeWindow = view;
        if (test) view.test = [[CVWindowTest alloc] init];
        NSApp.delegate = view;
        [view build];
        [NSApp run];
        [view.timer invalidate];
        [view.window close];
        NSApp.delegate = nil;
        activeWindow = nil;
        NSString *error = view.error ?: view.test.error;
        return error ? strdup(error.UTF8String) : NULL;
    }
}

void cv_stop(void) {
    dispatch_async(dispatch_get_main_queue(), ^{
        [activeWindow.window.attachedSheet close];
        [activeWindow.window orderOut:nil];
        [activeWindow stop];
    });
}
