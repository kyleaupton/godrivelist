#ifndef DISKLIST_H
#define DISKLIST_H

#include <stdbool.h>
#include <stdint.h>

typedef struct {
    char *device;
    char *displayName;
    char *description;
    uint64_t size;
    char **mountpoints;
    int mountpointsCount;
    char *raw;
    bool protected;
    bool system;
} DriveInfo;

typedef struct {
    DriveInfo *drives;
    int count;
} DriveList;

#ifdef __cplusplus
extern "C" {
#endif

DriveList* GetDriveList(void);
void FreeDriveList(DriveList* list);

#ifdef __cplusplus
}
#endif

#endif // DISKLIST_H