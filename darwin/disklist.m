#import <Foundation/Foundation.h>
#import <DiskArbitration/DiskArbitration.h>
#import "disklist.h"

static void freeDriveInfo(DriveInfo info) {
    free(info.device);
    free(info.displayName);
    free(info.description);
    free(info.raw);
    
    for (int i = 0; i < info.mountpointsCount; i++) {
        free(info.mountpoints[i].path);
    }
    
    free(info.mountpoints);
}

void FreeDriveList(DriveList list) {
    for (int i = 0; i < list.count; i++) {
        freeDriveInfo(list.drives[i]);
    }
    
    free(list.drives);
}

static char* strdup_safe(const char* s) {
    if (s == NULL) {
        return strdup("");
    }
    return strdup(s);
}

static bool isDiskMountable(DADiskRef disk) {
    CFBooleanRef boolRef = DADiskCopyWholeDiskHasFixedContent(disk);
    if (boolRef == NULL) {
        return true;
    }
    
    bool hasFixedContent = CFBooleanGetValue(boolRef);
    CFRelease(boolRef);
    
    return !hasFixedContent;
}

static bool isSystemDisk(DADiskRef disk) {
    CFURLRef url = CFURLCreateWithFileSystemPath(kCFAllocatorDefault, CFSTR("/"), kCFURLPOSIXPathStyle, true);
    CFBooleanRef result = NULL;
    
    if (url != NULL) {
        CFDictionaryRef mountInfo = DADiskCopyDescription(disk);
        if (mountInfo != NULL) {
            CFURLRef mountURL = CFDictionaryGetValue(mountInfo, kDADiskDescriptionVolumePathKey);
            if (mountURL != NULL) {
                result = CFURLEqual(url, mountURL) ? kCFBooleanTrue : kCFBooleanFalse;
            }
            CFRelease(mountInfo);
        }
        CFRelease(url);
    }
    
    return result == kCFBooleanTrue;
}

static bool isProtectedDisk(DADiskRef disk) {
    CFDictionaryRef description = DADiskCopyDescription(disk);
    if (description == NULL) {
        return false;
    }
    
    CFBooleanRef writable = CFDictionaryGetValue(description, kDADiskDescriptionMediaWritableKey);
    bool isProtected = (writable == NULL) || (writable == kCFBooleanFalse);
    
    CFRelease(description);
    return isProtected;
}

static NSArray* getDiskMountPoints(DADiskRef disk) {
    NSMutableArray* mountpoints = [NSMutableArray array];
    
    CFDictionaryRef description = DADiskCopyDescription(disk);
    if (description != NULL) {
        CFURLRef volumePath = CFDictionaryGetValue(description, kDADiskDescriptionVolumePathKey);
        if (volumePath != NULL) {
            NSString* path = (__bridge NSString*)CFURLCopyFileSystemPath(volumePath, kCFURLPOSIXPathStyle);
            [mountpoints addObject:path];
            [path release];
        }
        CFRelease(description);
    }
    
    return mountpoints;
}

static uint64_t getDiskSize(DADiskRef disk) {
    CFDictionaryRef description = DADiskCopyDescription(disk);
    if (description == NULL) {
        return 0;
    }
    
    CFNumberRef size = CFDictionaryGetValue(description, kDADiskDescriptionMediaSizeKey);
    uint64_t bytes = 0;
    
    if (size != NULL) {
        CFNumberGetValue(size, kCFNumberLongLongType, &bytes);
    }
    
    CFRelease(description);
    return bytes;
}

static NSString* getDiskDescription(DADiskRef disk) {
    CFDictionaryRef description = DADiskCopyDescription(disk);
    if (description == NULL) {
        return @"";
    }
    
    CFStringRef model = CFDictionaryGetValue(description, kDADiskDescriptionMediaModelKey);
    NSString* modelStr = model != NULL ? (__bridge NSString*)model : @"";
    
    CFRelease(description);
    return modelStr;
}

static NSString* getDiskName(DADiskRef disk) {
    CFDictionaryRef description = DADiskCopyDescription(disk);
    if (description == NULL) {
        return @"";
    }
    
    CFStringRef name = CFDictionaryGetValue(description, kDADiskDescriptionMediaNameKey);
    NSString* nameStr = name != NULL ? (__bridge NSString*)name : @"";
    
    CFRelease(description);
    return nameStr;
}

DriveList GetDriveList(void) {
    DriveList result = {NULL, 0};
    
    DASessionRef session = DASessionCreate(kCFAllocatorDefault);
    if (session == NULL) {
        return result;
    }
    
    NSMutableArray* drives = [NSMutableArray array];
    
    // Get list of disks from IO Registry
    CFMutableDictionaryRef matchingDict = IOServiceMatching("IOMedia");
    CFDictionaryAddValue(matchingDict, CFSTR(kIOMediaWholeKey), kCFBooleanTrue);
    
    io_iterator_t iter;
    kern_return_t kr = IOServiceGetMatchingServices(kIOMasterPortDefault, matchingDict, &iter);
    if (kr != KERN_SUCCESS) {
        CFRelease(session);
        return result;
    }
    
    io_service_t service;
    while ((service = IOIteratorNext(iter)) != IO_OBJECT_NULL) {
        DADiskRef disk = DADiskCreateFromIOMedia(kCFAllocatorDefault, session, service);
        if (disk != NULL) {
            // Get device name (e.g. /dev/disk0)
            const char* bsdName = DADiskGetBSDName(disk);
            if (bsdName != NULL) {
                // Check if this is a mountable disk (skip CD/DVD and other special media)
                if (isDiskMountable(disk)) {
                    NSString* devicePath = [NSString stringWithFormat:@"/dev/%s", bsdName];
                    NSString* rawPath = [NSString stringWithFormat:@"/dev/r%s", bsdName];
                    NSString* description = getDiskDescription(disk);
                    NSString* displayName = getDiskName(disk);
                    if ([displayName length] == 0) {
                        displayName = devicePath;
                    }
                    
                    NSArray* mountpoints = getDiskMountPoints(disk);
                    uint64_t size = getDiskSize(disk);
                    bool isProtected = isProtectedDisk(disk);
                    bool isSystem = isSystemDisk(disk);
                    
                    NSDictionary* driveInfo = @{
                        @"device": devicePath,
                        @"displayName": displayName,
                        @"description": description,
                        @"size": @(size),
                        @"mountpoints": mountpoints,
                        @"raw": rawPath,
                        @"protected": @(isProtected),
                        @"system": @(isSystem)
                    };
                    
                    [drives addObject:driveInfo];
                }
            }
            CFRelease(disk);
        }
        IOObjectRelease(service);
    }
    
    IOObjectRelease(iter);
    CFRelease(session);
    
    // Convert NSArray to C array
    result.count = (int)[drives count];
    result.drives = (DriveInfo*)malloc(sizeof(DriveInfo) * result.count);
    
    for (int i = 0; i < result.count; i++) {
        NSDictionary* drive = drives[i];
        
        result.drives[i].device = strdup_safe([[drive objectForKey:@"device"] UTF8String]);
        result.drives[i].displayName = strdup_safe([[drive objectForKey:@"displayName"] UTF8String]);
        result.drives[i].description = strdup_safe([[drive objectForKey:@"description"] UTF8String]);
        result.drives[i].size = [[drive objectForKey:@"size"] unsignedLongLongValue];
        result.drives[i].raw = strdup_safe([[drive objectForKey:@"raw"] UTF8String]);
        result.drives[i].is_protected = [[drive objectForKey:@"protected"] boolValue];
        result.drives[i].system = [[drive objectForKey:@"system"] boolValue];
        
        NSArray* mountpoints = [drive objectForKey:@"mountpoints"];
        result.drives[i].mountpointsCount = (int)[mountpoints count];
        result.drives[i].mountpoints = (Mountpoint*)malloc(sizeof(Mountpoint) * result.drives[i].mountpointsCount);
        
        for (int j = 0; j < result.drives[i].mountpointsCount; j++) {
            result.drives[i].mountpoints[j].path = strdup_safe([[mountpoints objectAtIndex:j] UTF8String]);
        }
    }
    
    return result;
} 