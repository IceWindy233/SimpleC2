export interface Beacon {
    BeaconID: string
    Listener: string
    RemoteAddr: string
    Status: string
    FirstSeen: string
    LastSeen: string
    Sleep: number
    OS: string
    Arch: string
    Username: string
    Hostname: string
    InternalIP: string
    ProcessName: string
    PID: number
    IsHighIntegrity: boolean
}

export interface Tunnel {
    ID: string
    BeaconID: string
    Target: string
    OperatorID: string
    Status: string // 'pending', 'active', 'closed', 'error'
    CreatedAt: string
    LastActivity: string
}

export interface Process {
    pid: number
    parent_pid?: number
    name: string
    executable?: string
    user?: string
    status?: string
    cpu?: string
    memory?: string
}

export interface SysInfo {
    hostname: string
    os: string
    arch: string
    username: string
    internal_ip: string
    num_cpu: number
    go_version: string
    current_cmd: string
    is_high_integrity: boolean
}
