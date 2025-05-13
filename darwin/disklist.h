#ifndef DISKLIST_H
#define DISKLIST_H

#include <stdbool.h>
#include <stdint.h>

typedef struct
{
  char *path;
} Mountpoint;

typedef struct
{
  char *device;
  char *displayName;
  char *description;
  uint64_t size;
  Mountpoint *mountpoints;
  int mountpointsCount;
  char *raw;
  bool is_protected;
  bool system;
} DriveInfo;

typedef struct
{
  DriveInfo *drives;
  int count;
} DriveList;

DriveList *GetDriveList(void);
void FreeDriveList(DriveList *list);

#endif /* DISKLIST_H */