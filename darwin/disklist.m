#include "disklist.h"
#include <IOKit/IOKitLib.h>
#include <IOKit/storage/IOStorageDeviceCharacteristics.h>
#include <IOKit/storage/IOMedia.h>
#include <IOKit/storage/IOBlockStorageDriver.h>
#include <IOKit/IOBSD.h>
#include <DiskArbitration/DiskArbitration.h>
#include <sys/mount.h>
#include <stdlib.h>
#include <string.h>

// Utility function to create a copy of a string
static char* copy_string(const char* str) {
    if (str == NULL) return NULL;
    size_t len = strlen(str);
    char* result = (char*)malloc(len + 1);
    if (result) {
        strcpy(result, str);
    }
    return result;
}

// Function to get mountpoints for a disk
static void get_mountpoints(const char* bsd_name, mountpoint_t** mountpoints, int* count) {
    struct statfs* mounts;
    int num_mounts = getmntinfo(&mounts, MNT_WAIT);
    
    // First, count the number of mountpoints for this disk
    *count = 0;
    for (int i = 0; i < num_mounts; i++) {
        if (strstr(mounts[i].f_mntfromname, bsd_name) != NULL) {
            (*count)++;
        }
    }
    
    if (*count == 0) {
        *mountpoints = NULL;
        return;
    }
    
    // Allocate memory for mountpoints
    *mountpoints = (mountpoint_t*)malloc(sizeof(mountpoint_t) * (*count));
    if (*mountpoints == NULL) {
        *count = 0;
        return;
    }
    
    // Fill the mountpoints
    int index = 0;
    for (int i = 0; i < num_mounts; i++) {
        if (strstr(mounts[i].f_mntfromname, bsd_name) != NULL) {
            (*mountpoints)[index].path = copy_string(mounts[i].f_mntonname);
            (*mountpoints)[index].label = NULL; // No label info in statfs
            index++;
        }
    }
}

// Check if a disk is a system disk
static bool is_system_disk(const char* bsd_name) {
    struct statfs* mounts;
    int num_mounts = getmntinfo(&mounts, MNT_WAIT);
    
    for (int i = 0; i < num_mounts; i++) {
        if (strstr(mounts[i].f_mntfromname, bsd_name) != NULL && 
            strcmp(mounts[i].f_mntonname, "/") == 0) {
            return true;
        }
    }
    
    return false;
}

// Get drive information
drive_list_t GetDriveList() {
    drive_list_t result = {0};
    
    // Initialize the count to 0
    result.count = 0;
    result.drives = NULL;
    result.error = NULL;
    
    // Initialize DiskArbitration framework
    DASessionRef session = DASessionCreate(kCFAllocatorDefault);
    if (session == NULL) {
        result.error = copy_string("Failed to create DiskArbitration session");
        return result;
    }
    
    // Get the list of disks
    mach_port_t masterPort;
    kern_return_t kernResult = IOMasterPort(MACH_PORT_NULL, &masterPort);
    if (kernResult != KERN_SUCCESS) {
        DASessionUnscheduleFromRunLoop(session, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
        CFRelease(session);
        result.error = copy_string("Failed to get IO master port");
        return result;
    }
    
    // Create a matching dictionary for IOMedia objects
    CFMutableDictionaryRef matchingDict = IOServiceMatching(kIOMediaClass);
    if (matchingDict == NULL) {
        DASessionUnscheduleFromRunLoop(session, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
        CFRelease(session);
        result.error = copy_string("Failed to create matching dictionary");
        return result;
    }
    
    // Add matching criteria - only want whole media, not partitions
    CFDictionarySetValue(matchingDict, CFSTR(kIOMediaWholeKey), kCFBooleanTrue);
    
    // Get an iterator for matching IOMedia objects
    io_iterator_t iter;
    kernResult = IOServiceGetMatchingServices(masterPort, matchingDict, &iter);
    if (kernResult != KERN_SUCCESS) {
        DASessionUnscheduleFromRunLoop(session, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
        CFRelease(session);
        result.error = copy_string("Failed to get matching services");
        return result;
    }
    
    // Count the number of disks
    io_service_t disk;
    while ((disk = IOIteratorNext(iter)) != IO_OBJECT_NULL) {
        result.count++;
        IOObjectRelease(disk);
    }
    
    // Reset the iterator
    IOIteratorReset(iter);
    
    // Allocate memory for drives
    result.drives = (drive_t*)malloc(sizeof(drive_t) * result.count);
    if (result.drives == NULL) {
        DASessionUnscheduleFromRunLoop(session, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
        CFRelease(session);
        IOObjectRelease(iter);
        result.error = copy_string("Failed to allocate memory for drives");
        result.count = 0;
        return result;
    }
    
    // Initialize drives
    for (int i = 0; i < result.count; i++) {
        result.drives[i].device = NULL;
        result.drives[i].display_name = NULL;
        result.drives[i].description = NULL;
        result.drives[i].size = 0;
        result.drives[i].mountpoints = NULL;
        result.drives[i].mountpoints_count = 0;
        result.drives[i].raw = NULL;
        result.drives[i].protected = false;
        result.drives[i].system = false;
        result.drives[i].removable = false;
        result.drives[i].virtual_drive = false;
        result.drives[i].internal = true;
        result.drives[i].block_size = 0;
    }
    
    // Get information for each disk
    int index = 0;
    while ((disk = IOIteratorNext(iter)) != IO_OBJECT_NULL && index < result.count) {
        // Get the BSD name
        CFStringRef bsdNameRef = IORegistryEntryCreateCFProperty(disk, CFSTR(kIOBSDNameKey), kCFAllocatorDefault, 0);
        if (bsdNameRef != NULL) {
            char bsd_name[128];
            CFStringGetCString(bsdNameRef, bsd_name, sizeof(bsd_name), kCFStringEncodingUTF8);
            CFRelease(bsdNameRef);
            
            // Set the device and raw paths
            char device_path[256];
            sprintf(device_path, "/dev/%s", bsd_name);
            result.drives[index].device = copy_string(device_path);
            
            char raw_path[256];
            sprintf(raw_path, "/dev/r%s", bsd_name);
            result.drives[index].raw = copy_string(raw_path);
            result.drives[index].display_name = copy_string(device_path);
            
            // Get the disk size
            CFNumberRef sizeRef = IORegistryEntryCreateCFProperty(disk, CFSTR(kIOMediaSizeKey), kCFAllocatorDefault, 0);
            if (sizeRef != NULL) {
                CFNumberGetValue(sizeRef, kCFNumberSInt64Type, &result.drives[index].size);
                CFRelease(sizeRef);
            }
            
            // Get the block size
            CFNumberRef blockSizeRef = IORegistryEntryCreateCFProperty(disk, CFSTR(kIOMediaPreferredBlockSizeKey), kCFAllocatorDefault, 0);
            if (blockSizeRef != NULL) {
                uint32_t block_size;
                CFNumberGetValue(blockSizeRef, kCFNumberSInt32Type, &block_size);
                result.drives[index].block_size = block_size;
                CFRelease(blockSizeRef);
            } else {
                result.drives[index].block_size = 512; // Default
            }
            
            // Check if it's a system disk
            result.drives[index].system = is_system_disk(bsd_name);
            
            // Get mountpoints
            get_mountpoints(bsd_name, &result.drives[index].mountpoints, &result.drives[index].mountpoints_count);
            
            // Get disk properties from DiskArbitration
            DADiskRef dadisk = DADiskCreateFromBSDName(kCFAllocatorDefault, session, bsd_name);
            if (dadisk != NULL) {
                CFDictionaryRef description = DADiskCopyDescription(dadisk);
                if (description != NULL) {
                    // Get the model/vendor name
                    CFStringRef model = CFDictionaryGetValue(description, kDADiskDescriptionDeviceModelKey);
                    if (model != NULL) {
                        char model_str[256];
                        CFStringGetCString(model, model_str, sizeof(model_str), kCFStringEncodingUTF8);
                        result.drives[index].description = copy_string(model_str);
                    } else {
                        result.drives[index].description = copy_string("Unknown");
                    }
                    
                    // Check if it's removable
                    CFBooleanRef removable = CFDictionaryGetValue(description, kDADiskDescriptionMediaRemovableKey);
                    if (removable != NULL) {
                        result.drives[index].removable = CFBooleanGetValue(removable);
                    }
                    
                    // Check if it's a virtual disk
                    CFBooleanRef virtual = CFDictionaryGetValue(description, kDADiskDescriptionMediaVirtualKey);
                    if (virtual != NULL) {
                        result.drives[index].virtual_drive = CFBooleanGetValue(virtual);
                    }
                    
                    // Check if it's write-protected
                    CFBooleanRef writeable = CFDictionaryGetValue(description, kDADiskDescriptionMediaWritableKey);
                    if (writeable != NULL) {
                        result.drives[index].protected = !CFBooleanGetValue(writeable);
                    }
                    
                    // Check if it's internal
                    CFBooleanRef internal = CFDictionaryGetValue(description, kDADiskDescriptionDeviceInternalKey);
                    if (internal != NULL) {
                        result.drives[index].internal = CFBooleanGetValue(internal);
                    }
                    
                    CFRelease(description);
                }
                CFRelease(dadisk);
            }
            
            index++;
        }
        
        IOObjectRelease(disk);
    }
    
    // Update the count if we found fewer disks than expected
    result.count = index;
    
    // Clean up
    IOObjectRelease(iter);
    DASessionUnscheduleFromRunLoop(session, CFRunLoopGetCurrent(), kCFRunLoopDefaultMode);
    CFRelease(session);
    
    return result;
}

// Free the memory allocated by GetDriveList
void FreeDriveList(drive_list_t list) {
    for (int i = 0; i < list.count; i++) {
        free(list.drives[i].device);
        free(list.drives[i].display_name);
        free(list.drives[i].description);
        free(list.drives[i].raw);
        
        for (int j = 0; j < list.drives[i].mountpoints_count; j++) {
            free(list.drives[i].mountpoints[j].path);
            free(list.drives[i].mountpoints[j].label);
        }
        
        free(list.drives[i].mountpoints);
    }
    
    free(list.drives);
    free(list.error);
} 