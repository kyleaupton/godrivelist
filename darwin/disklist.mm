#import <Foundation/Foundation.h>
#import <DiskArbitration/DiskArbitration.h>
#include "disklist.h"
#include <string>
#include <vector>

bool IsDiskPartition(NSString *disk) {
    NSPredicate *partitionRegEx = [NSPredicate predicateWithFormat:@"SELF MATCHES %@", @"disk\\d+s\\d+"];
    return [partitionRegEx evaluateWithObject:disk];
}

NSNumber *DictionaryGetNumber(CFDictionaryRef dict, const void *key) {
    return (NSNumber*)CFDictionaryGetValue(dict, key);
}

DriveInfo CreateDriveInfo(NSString *diskBsdName, CFDictionaryRef diskDescription) {
    DriveInfo info = {};
    
    std::string devicePath = "/dev/" + std::string([diskBsdName UTF8String]);
    info.device = strdup(devicePath.c_str());
    info.displayName = strdup(devicePath.c_str());
    
    NSString *mediaContent = (NSString*)CFDictionaryGetValue(diskDescription, kDADiskDescriptionMediaContentKey);
    NSString *mediaName = (NSString*)CFDictionaryGetValue(diskDescription, kDADiskDescriptionMediaNameKey);
    std::string description;
    if (mediaContent) {
        description = [mediaContent UTF8String];
        if (mediaName) {
            description += " ";
            description += [mediaName UTF8String];
        }
    } else if (mediaName) {
        description = [mediaName UTF8String];
    }
    info.description = strdup(description.c_str());
    
    info.size = [DictionaryGetNumber(diskDescription, kDADiskDescriptionMediaSizeKey) unsignedLongValue];
    
    std::string rawPath = "/dev/r" + std::string([diskBsdName UTF8String]);
    info.raw = strdup(rawPath.c_str());
    
    info.protected = ![DictionaryGetNumber(diskDescription, kDADiskDescriptionMediaWritableKey) boolValue];
    
    bool isInternal = [DictionaryGetNumber(diskDescription, kDADiskDescriptionDeviceInternalKey) boolValue];
    bool isRemovable = [DictionaryGetNumber(diskDescription, kDADiskDescriptionMediaRemovableKey) boolValue];
    info.system = isInternal && !isRemovable;
    
    return info;
}

extern "C" {

DriveList* GetDriveList(void) {
    DriveList* result = (DriveList*)malloc(sizeof(DriveList));
    std::vector<DriveInfo> drives;
    
    DASessionRef session = DASessionCreate(kCFAllocatorDefault);
    if (session == nil) {
        result->drives = NULL;
        result->count = 0;
        return result;
    }
    
    NSArray *volumeKeys = [NSArray arrayWithObjects:NSURLVolumeNameKey, NSURLVolumeLocalizedNameKey, nil];
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
        std::vector<std::string> mountpoints;
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
            
            std::string partitionBsdName = std::string(bsdnameChar);
            std::string diskBsdName = partitionBsdName.substr(0, partitionBsdName.find("s", 5));
            
            if (diskBsdName == [path UTF8String]) {
                mountpoints.push_back([[volumePath path] UTF8String]);
            }
            
            CFRelease(volumeDisk);
        }
        
        // Copy mountpoints to DriveInfo
        info.mountpointsCount = mountpoints.size();
        if (info.mountpointsCount > 0) {
            info.mountpoints = (char**)malloc(sizeof(char*) * info.mountpointsCount);
            for (int i = 0; i < info.mountpointsCount; i++) {
                info.mountpoints[i] = strdup(mountpoints[i].c_str());
            }
        } else {
            info.mountpoints = NULL;
        }
        
        drives.push_back(info);
        
        CFRelease(diskDescription);
        CFRelease(disk);
    }
    
    CFRelease(session);
    
    // Copy drives to result
    result->count = drives.size();
    result->drives = (DriveInfo*)malloc(sizeof(DriveInfo) * result->count);
    memcpy(result->drives, drives.data(), sizeof(DriveInfo) * result->count);
    
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

}