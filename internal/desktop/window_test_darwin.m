#import "window_test_darwin.h"
#import "_cgo_export.h"

@implementation CVWindowTest {
    BOOL started;
    BOOL openedPicker;
    NSString *selectedFile;
    NSUInteger ticks;
    NSUInteger completedTicks;
}

- (void)fail:(NSString *)error {
    self.error = error;
    self.finished = YES;
}

- (void)tickWithState:(NSDictionary *)state choose:(NSButton *)choose codec:(NSPopUpButton *)codec format:(NSPopUpButton *)format
    convert:(NSButton *)convert progress:(NSProgressIndicator *)progress message:(NSTextView *)message window:(NSWindow *)window {
    ticks++;
    if (ticks > 150) {
        [self fail:@"La comprobación de conversión superó el tiempo de espera."];
        return;
    }
    if (!started) {
        if (![state[@"file"] length]) {
            if (convert.enabled) [self fail:@"El botón Convertir debe estar desactivado si no hay un video."];
            else [self fail:@"La comprobación necesita un video de prueba."];
            return;
        }
        if (!openedPicker) {
            if (![window.title isEqual:@"Conversor de Video"] ||
                ![choose.title isEqual:@"Seleccionar video"] || ![convert.title isEqual:@"Convertir"] ||
                ![codec.accessibilityLabel isEqual:@"Códec"] ||
                ![format.accessibilityLabel isEqual:@"Formato"]) {
                [self fail:@"Los controles de la ventana no están en español."];
                return;
            }
            NSString *original = [state[@"file"] pathExtension].lowercaseString;
            NSArray *allowed = state[@"options"][0][@"formats"];
            NSString *expected = [allowed containsObject:original] ? original : @"mp4";
            if (![codec.selectedItem.representedObject isEqual:@"h264"] ||
                ![format.selectedItem.representedObject isEqual:expected] ||
                ![state[@"format"] isEqual:expected]) {
                [self fail:@"Los controles no muestran el códec y el formato predeterminados."];
                return;
            }
            selectedFile = state[@"file"];
            openedPicker = YES;
            [choose performClick:nil];
            return;
        }
        if (window.attachedSheet) return;
        if (![state[@"file"] isEqual:selectedFile]) {
            [self fail:@"Cancelar el selector de archivos cambió el video seleccionado."];
            return;
        }
        if (!convert.enabled) {
            [self fail:@"Los controles no muestran el códec y el formato predeterminados."];
            return;
        }
        for (NSUInteger index = 0; index < codec.numberOfItems; index++) {
            [codec selectItemAtIndex:index];
            [NSApp sendAction:codec.action to:codec.target from:codec];
            NSString *identifier = codec.selectedItem.representedObject;
            NSDictionary *option = nil;
            for (NSDictionary *candidate in state[@"options"]) {
                if ([candidate[@"id"] isEqual:identifier]) option = candidate;
            }
            NSArray *actual = [format.itemArray valueForKey:@"representedObject"];
            if (![actual isEqual:option[@"formats"]]) {
                [self fail:@"El selector incluye un formato incompatible."];
                return;
            }
        }
        [codec selectItemAtIndex:0];
        [NSApp sendAction:codec.action to:codec.target from:codec];
        [format selectItemWithTitle:@"MP4"];
        [NSApp sendAction:format.action to:format.target from:format];
        [convert performClick:nil];
        if (convert.enabled || codec.enabled || format.enabled) {
            [self fail:@"Los controles deben estar desactivados durante la conversión."];
            return;
        }
        started = YES;
        return;
    }
    if ([state[@"status"] isEqual:@"failed"]) {
        [self fail:state[@"message"]];
        return;
    }
    if (![state[@"status"] isEqual:@"completed"]) return;
    if (progress.indeterminate || progress.doubleValue != 100 || !convert.enabled ||
        ![message.string containsString:state[@"output"]]) {
        [self fail:@"Los controles no muestran la conversión completada."];
        return;
    }
    if (++completedTicks < 4) return;
    NSString *imagePath = NSProcessInfo.processInfo.environment[@"CONVERTIDOR_TEST_IMAGE"];
    if (imagePath.length) {
        NSView *view = window.contentView;
        NSBitmapImageRep *image = [view bitmapImageRepForCachingDisplayInRect:view.bounds];
        [window.effectiveAppearance performAsCurrentDrawingAppearance:^{
            [view cacheDisplayInRect:view.bounds toBitmapImageRep:image];
        }];
        NSData *png = [image representationUsingType:NSBitmapImageFileTypePNG properties:@{}];
        NSError *error = nil;
        if (![png writeToFile:imagePath options:NSDataWritingAtomic error:&error]) {
            [self fail:[NSString stringWithFormat:@"No se pudo guardar la imagen de la ventana. Detalles técnicos: %@", error]];
            return;
        }
    }
    self.finished = YES;
}
@end
