// src/types.ts

export interface Beacon {
  ID: number;
  BeaconID: string;
  OS: string;
  Arch: string;
  Hostname: string;
  Username: string;
  InternalIP: string;
  LastSeen: string;
  Status: string;
  FirstSeen: string;
  Listener: string;
  Sleep: number;
}

export interface Task {
  TaskID: string;
  Command: string;
  Arguments: string;
  Status: string;
  Output: string;
  CreatedAt: string;
  UpdatedAt: string;
}
