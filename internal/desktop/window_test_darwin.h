#import <Cocoa/Cocoa.h>

@interface CVWindowTest : NSObject
@property NSString *error;
@property BOOL finished;
- (void)tickWithState:(NSDictionary *)state choose:(NSButton *)choose codec:(NSPopUpButton *)codec format:(NSPopUpButton *)format
    convert:(NSButton *)convert progress:(NSProgressIndicator *)progress message:(NSTextView *)message window:(NSWindow *)window;
@end
