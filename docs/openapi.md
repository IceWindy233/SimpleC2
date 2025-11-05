```yaml
openapi: 3.0.3
info:
  title: SimpleC2 TeamServer API
  description: The API for interacting with the SimpleC2 TeamServer to manage beacons, listeners, and tasks.
  version: 1.0.0
servers:

  - url: http://localhost:8080/api
    description: Local development server

components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
      bearerFormat: JWT

  schemas:
    Beacon:
      type: object
      properties:
        ID:
          type: integer
          format: uint
          readOnly: true
        CreatedAt:
          type: string
          format: date-time
          readOnly: true
        UpdatedAt:
          type: string
          format: date-time
          readOnly: true
        BeaconID:
          type: string
          format: uuid
          description: The unique identifier for the beacon.
        Listener:
          type: string
          description: The name of the listener the beacon is connected to.
        RemoteAddr:
          type: string
          description: The source network address of the beacon.
        Status:
          type: string
          description: The current status of the beacon.
          enum: [active, inactive, lost, exiting]
          default: active
        FirstSeen:
          type: string
          format: date-time
          description: Timestamp of the first time the beacon checked in.
        LastSeen:
          type: string
          format: date-time
          description: Timestamp of the most recent beacon check-in.
        Sleep:
          type: integer
          description: The current sleep interval of the beacon in seconds.
        OS:
          type: string
          description: The operating system of the beacon's host.
        Arch:
          type: string
          description: The architecture of the beacon's host.
        Username:
          type: string
          description: The username the beacon is running as.
        Hostname:
          type: string
          description: The hostname of the beacon's host.
        InternalIP:
          type: string
          description: The internal IP address of the beacon's host.
        ProcessName:
          type: string
          description: The name of the beacon's process.
        PID:
          type: integer
          format: int32
          description: The process ID of the beacon.
        IsHighIntegrity:
          type: boolean
          description: Whether the beacon is running with high integrity (e.g., root/Administrator).

    Task:
      type: object
      properties:
        ID:
          type: integer
          format: uint
          readOnly: true
        CreatedAt:
          type: string
          format: date-time
          readOnly: true
        UpdatedAt:
          type: string
          format: date-time
          readOnly: true
        TaskID:
          type: string
          format: uuid
          description: The unique identifier for the task.
        BeaconID:
          type: string
          format: uuid
          description: The ID of the beacon this task is for.
        Command:
          type: string
          description: The command to be executed.
          enum: [shell, download, upload, exit, sleep, browse]
        Arguments:
          type: string
          description: The arguments for the command.
        Status:
          type: string
          description: The current status of the task.
          enum: [queued, dispatched, completed, error, Timeout]
        Output:
          type: string
          description: The output of the task after execution.
    
    Listener:
      type: object
      properties:
        ID:
          type: integer
          format: uint
          readOnly: true
        CreatedAt:
          type: string
          format: date-time
          readOnly: true
        UpdatedAt:
          type: string
          format: date-time
          readOnly: true
        Name:
          type: string
          description: The unique name for the listener.
        Type:
          type: string
          description: The type of the listener (e.g., http, dns).
        Config:
          type: string
          description: Listener-specific configuration, stored as a JSON string.
    
    Operator:
      type: object
      properties:
        ID:
          type: integer
          format: uint
          readOnly: true
        Username:
          type: string
          description: The username of the operator.
        PasswordHash:
          type: string
          description: The hashed password of the operator.
          readOnly: true
    
    AuthRequest:
      type: object
      required:
        - username
        - password
      properties:
        username:
          type: string
        password:
          type: string
          format: password
    
    CreateTaskRequest:
      type: object
      required:
        - command
      properties:
        command:
          type: string
          description: The command to execute.
        arguments:
          type: string
          description: Arguments for the command.
    
    CreateListenerRequest:
      type: object
      required:
        - name
        - type
      properties:
        name:
          type: string
        type:
          type: string
        config:
          type: string
    
    ErrorResponse:
      type: object
      properties:
        error:
          type: string

security:

  - BearerAuth: []

paths:
  /auth/register:
    post:
      summary: Register a new operator
      tags: [Authentication]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AuthRequest'
      responses:
        '201':
          description: Operator created successfully
        '400':
          description: Invalid request body
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'

  /auth/login:
    post:
      summary: Log in and get a JWT
      tags: [Authentication]
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/AuthRequest'
      responses:
        '200':
          description: Successful login
          content:
            application/json:
              schema:
                type: object
                properties:
                  token:
                    type: string
        '401':
          description: Invalid credentials
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'

  /beacons:
    get:
      summary: Get all active beacons
      tags: [Beacons]
      security:
        - BearerAuth: []
      responses:
        '200':
          description: A list of beacons
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Beacon'
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'

  /beacons/{beacon_id}:
    get:
      summary: Get a single beacon by ID
      tags: [Beacons]
      security:
        - BearerAuth: []
      parameters:
        - name: beacon_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Beacon details
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    $ref: '#/components/schemas/Beacon'
        '404':
          description: Beacon not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
    delete:
      summary: Task a beacon to exit and soft-delete it
      tags: [Beacons]
      description: >
        This endpoint queues an 'exit' task for the specified beacon, updates its status to 'exiting',
        and then soft-deletes the beacon record from the database. The beacon will appear removed from the UI
        and will terminate itself upon its next check-in.
      security:
        - BearerAuth: []
      parameters:
        - name: beacon_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '204':
          description: Beacon successfully tasked to exit and soft-deleted
        '404':
          description: Beacon not found
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'
        '500':
          description: Internal server error
          content:
            application/json:
              schema:
                $ref: '#/components/schemas/ErrorResponse'

  /beacons/{beacon_id}/tasks:
    post:
      summary: Create a new task for a beacon
      tags: [Tasks]
      security:
        - BearerAuth: []
      parameters:
        - name: beacon_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateTaskRequest'
      responses:
        '201':
          description: Task created successfully
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    $ref: '#/components/schemas/Task'
        '400':
          description: Invalid request body
        '404':
          description: Beacon not found
        '500':
          description: Internal server error

  /tasks/{task_id}:
    get:
      summary: Get a single task by ID
      tags: [Tasks]
      security:
        - BearerAuth: []
      parameters:
        - name: task_id
          in: path
          required: true
          schema:
            type: string
            format: uuid
      responses:
        '200':
          description: Task details
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    $ref: '#/components/schemas/Task'
        '404':
          description: Task not found

  /listeners:
    get:
      summary: Get all listeners
      tags: [Listeners]
      security:
        - BearerAuth: []
      responses:
        '200':
          description: A list of listeners
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    type: array
                    items:
                      $ref: '#/components/schemas/Listener'
        '500':
          description: Internal server error
    post:
      summary: Create a new listener
      tags: [Listeners]
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          application/json:
            schema:
              $ref: '#/components/schemas/CreateListenerRequest'
      responses:
        '201':
          description: Listener created successfully
          content:
            application/json:
              schema:
                type: object
                properties:
                  data:
                    $ref: '#/components/schemas/Listener'
        '400':
          description: Invalid request body
        '500':
          description: Internal server error

  /listeners/{name}:
    delete:
      summary: Delete a listener by name
      tags: [Listeners]
      security:
        - BearerAuth: []
      parameters:
        - name: name
          in: path
          required: true
          schema:
            type: string
      responses:
        '204':
          description: Listener deleted successfully
        '404':
          description: Listener not found
        '500':
          description: Internal server error

  /upload:
    post:
      summary: Upload a file to the TeamServer
      tags: [Files]
      description: Uploads a file to the 'teamserver/uploads' directory for later use in 'download' tasks for beacons.
      security:
        - BearerAuth: []
      requestBody:
        required: true
        content:
          multipart/form-data:
            schema:
              type: object
              properties:
                file:
                  type: string
                  format: binary
      responses:
        '200':
          description: File uploaded successfully
          content:
            application/json:
              schema:
                type: object
                properties:
                  filepath:
                    type: string
        '400':
          description: File not provided
        '500':
          description: Failed to save file

  /loot/{filename}:
    get:
      summary: Download a looted file
      tags: [Files]
      description: Downloads a file that was uploaded from a beacon to the 'teamserver/loot' directory.
      security:
        - BearerAuth: []
      parameters:
        - name: filename
          in: path
          required: true
          schema:
            type: string
      responses:
        '200':
          description: The file content
          content:
            application/octet-stream:
              schema:
                type: string
                format: binary
        '404':
          description: File not found
```

