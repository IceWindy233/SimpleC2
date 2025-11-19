// src/types.ts

export interface Beacon {
  ID: number;
  CreatedAt: string;
  UpdatedAt: string;
  DeletedAt: string | null;
  BeaconID: string;
  SessionKey: string | null;
  Listener: string;
  RemoteAddr: string;
  Status: string;
  FirstSeen: string;
  LastSeen: string;
  Sleep: number;
  OS: string;
  Arch: string;
  Username: string;
  Hostname: string;
  InternalIP: string;
  ProcessName: string;
  PID: number;
  IsHighIntegrity: boolean;
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

