#import <Foundation/Foundation.h>
#import <DiskArbitration/DiskArbitration.h>
#include "disklist.h"
#include <string.h>

static bool IsDiskPartition(NSString *disk) {
    NSPredicate *partitionRegEx = [NSPredicate predicateWithFormat:@"SELF MATCHES %@", @"disk\\d+s\\d+"];
    return [partitionRegEx evaluateWithObject:disk];
}

static NSNumber *DictionaryGetNumber(CFDictionaryRef dict, const void *key) {
    return (NSNumber*)CFDictionaryGetValue(dict, key);
}

static DriveInfo CreateDriveInfo(NSString *diskBsdName, CFDictionaryRef diskDescription) {
    DriveInfo info = {0};
    
    NSString *devicePath = [NSString stringWithFormat:@"/dev/%@", diskBsdName];
    info.device = strdup([devicePath UTF8String]);
    info.displayName = strdup([devicePath UTF8String]);
    
    NSString *mediaContent = (NSString*)CFDictionaryGetValue(diskDescription, kDADiskDescriptionMediaContentKey);
    NSString *mediaName = (NSString*)CFDictionaryGetValue(diskDescription, kDADiskDescriptionMediaNameKey);
    NSString *description = nil;
    if (mediaContent) {
        if (mediaName) {
            description = [NSString stringWithFormat:@"%@ %@", mediaContent, mediaName];
        } else {
            description = mediaContent;
        }
    } else if (mediaName) {
        description = mediaName;
    }
    info.description = strdup(description ? [description UTF8String] : "");
    
    info.size = [DictionaryGetNumber(diskDescription, kDADiskDescriptionMediaSizeKey) unsignedLongValue];
    
    NSString *rawPath = [NSString stringWithFormat:@"/dev/r%@", diskBsdName];
    info.raw = strdup([rawPath UTF8String]);
    
    info.protected = ![DictionaryGetNumber(diskDescription, kDADiskDescriptionMediaWritableKey) boolValue];
    
    bool isInternal = [DictionaryGetNumber(diskDescription, kDADiskDescriptionDeviceInternalKey) boolValue];
    bool isRemovable = [DictionaryGetNumber(diskDescription, kDADiskDescriptionMediaRemovableKey) boolValue];
    info.system = isInternal && !isRemovable;
    
    return info;
}

DriveList* GetDriveList(void) {
    DriveList* result = (DriveList*)calloc(1, sizeof(DriveList));
    NSMutableArray *drives = [NSMutableArray array];
    
    DASessionRef session = DASessionCreate(kCFAllocatorDefault);
    if (session == nil) {
        return result;
    }
    
    NSArray *volumeKeys = @[NSURLVolumeNameKey, NSURLVolumeLocalizedNameKey];
    NSArray *volumePaths = [[NSFileManager defaultManager] mountedVolumeURLsIncludingResourceValuesForKeys:volumeKeys options:0];
    
    NSFileManager *fileManager = [NSFileManager defaultManager];
    NSArray *paths = [fileManager contentsOfDirectoryAtPath:@"/dev" error:nil];
    
    for (NSString *path in paths) {
        if (![path hasPrefix:@"disk"] || IsDiskPartition(path)) {
            continue;
        }
        
        DADiskRef disk = DADiskCreateFromBSDName(kCFAllocatorDefault, session, [path UTF8String]);
        if (disk == nil) {
            continue;
        }
        
        CFDictionaryRef diskDescription = DADiskCopyDescription(disk);
        if (diskDescription == nil) {
            CFRelease(disk);
            continue;
        }
        
        DriveInfo info = CreateDriveInfo(path, diskDescription);
        
        // Get mountpoints
        NSMutableArray *mountpoints = [NSMutableArray array];
        for (NSURL *volumePath in volumePaths) {
            DADiskRef volumeDisk = DADiskCreateFromVolumePath(kCFAllocatorDefault, session, (__bridge CFURLRef)volumePath);
            if (volumeDisk == nil) {
                continue;
            }
            
            const char *bsdnameChar = DADiskGetBSDName(volumeDisk);
            if (bsdnameChar == nil) {
                CFRelease(volumeDisk);
                continue;
            }
            
            NSString *partitionBsdName = [NSString stringWithUTF8String:bsdnameChar];
            NSString *diskBsdName = [partitionBsdName substringWithRange:NSMakeRange(0, [partitionBsdName rangeOfString:@"s" options:0 range:NSMakeRange(5, [partitionBsdName length] - 5)].location)];
            
            if ([diskBsdName isEqualToString:path]) {
                [mountpoints addObject:[volumePath path]];
            }
            
            CFRelease(volumeDisk);
        }
        
        // Copy mountpoints to DriveInfo
        info.mountpointsCount = (int)[mountpoints count];
        if (info.mountpointsCount > 0) {
            info.mountpoints = (char**)malloc(sizeof(char*) * info.mountpointsCount);
            for (int i = 0; i < info.mountpointsCount; i++) {
                info.mountpoints[i] = strdup([[mountpoints objectAtIndex:i] UTF8String]);
            }
        }
        
        [drives addObject:[NSValue valueWithBytes:&info objCType:@encode(DriveInfo)]];
        
        CFRelease(diskDescription);
        CFRelease(disk);
    }
    
    CFRelease(session);
    
    // Copy drives to result
    result->count = (int)[drives count];
    result->drives = (DriveInfo*)malloc(sizeof(DriveInfo) * result->count);
    for (int i = 0; i < result->count; i++) {
        DriveInfo info;
        [[drives objectAtIndex:i] getValue:&info];
        result->drives[i] = info;
    }
    
    return result;
}

void FreeDriveList(DriveList* list) {
    if (list == NULL) return;
    
    for (int i = 0; i < list->count; i++) {
        DriveInfo *info = &list->drives[i];
        free(info->device);
        free(info->displayName);
        free(info->description);
        free(info->raw);
        if (info->mountpoints) {
            for (int j = 0; j < info->mountpointsCount; j++) {
                free(info->mountpoints[j]);
            }
            free(info->mountpoints);
        }
    }
    
    free(list->drives);
    free(list);
}
