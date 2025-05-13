#ifndef GODRIVELIST_DARWIN_DISKLIST_H
#define GODRIVELIST_DARWIN_DISKLIST_H

#include <stdint.h>
#include <stdbool.h>

// Defines the structure for mountpoint information
typedef struct {
    char *path;
    char *label;
} mountpoint_t;

// Defines the structure for drive information
typedef struct {
    char *device;
    char *display_name;
    char *description;
    uint64_t size;
    mountpoint_t *mountpoints;
    int mountpoints_count;
    char *raw;
    bool protected;
    bool system;
    bool removable;
    bool virtual_drive;
    bool internal;
    uint32_t block_size;
} drive_t;

// Defines the return structure for the GetDriveList function
typedef struct {
    drive_t *drives;
    int count;
    char *error;
} drive_list_t;

// Main function to get the list of drives
drive_list_t GetDriveList();

// Function to free the memory allocated by GetDriveList
void FreeDriveList(drive_list_t list);

#endif // GODRIVELIST_DARWIN_DISKLIST_H 