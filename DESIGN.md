# NAS Docker Controller - System Design Document

## 1. Overview

A **personal application controller** running on **Unraid 7.2.2** with Docker enabled. The system manages self-owned web applications through a **single controller container** that provides:

- Modern web UI (responsive, works on mobile)
- App lifecycle management (start/stop/restart/rebuild)
- Local Docker image builds
- GitHub-based app onboarding (public repos with Dockerfile required)

This system acts as a **lightweight personal PaaS**, not a full orchestration platform.

---

## 2. Goals & Non-Goals

### Goals

- Single controller app with modern Web UI
- Manage multiple Dockerized web apps
- Add new apps via GitHub repository URL + branch
- Build Docker images locally on Unraid
- Start / stop / rebuild apps
- Repos must provide Dockerfile (controller will not auto-generate)
- Display app metadata (name, icon, repo, port, status)
- Real-time log streaming
- Port conflict detection and automatic allocation
- Clean REST API for future iOS app integration
- Expose localhost with port (user handles Cloudflare Tunnel manually)
- Fit naturally into Unraid's Docker ecosystem
- Password authentication (auto-generated on first run, updatable)
- Storage overview (DB size, logs, repo sizes)

### Non-Goals

- Multi-user access or RBAC
- Kubernetes-level orchestration
- Verifying repo ownership (trust all provided repos)
- Full CI/CD pipelines
- Auto proxy or DNS management
- Cross-platform mobile framework
- Private repository support
- Dockerfile auto-generation
- GitHub webhooks
- Cloudflare Tunnel auto-configuration

---

## 3. Tech Stack

### Backend: Go

- Native Docker SDK (`github.com/docker/docker/client`) - no shell commands
- Single binary deployment, low memory footprint
- Efficient resource usage (target system: 16GB RAM, i7-11800H, 1GB SSD, 20TB HDD)
- Clean REST API design for future iOS consumption
- Single build at a time (simpler), multiple containers run concurrently

### Frontend: React + TypeScript + Vite

- Best ecosystem for complex, stateful UIs
- Excellent developer tooling
- Strong typing catches errors early
- Served as static files embedded in Go binary
- Large ecosystem and community

### UI Components: shadcn/ui + Tailwind CSS

- Modern, accessible components
- Highly customizable
- Excellent dark mode support
- Responsive design for mobile

### Database: SQLite

- Single-file database
- Concurrent access support
- No external dependencies

### Future iOS App Strategy

- Native Swift/SwiftUI for iOS
- Shares the same REST API as web frontend
- No cross-platform compromise on performance

---

## 4. System Architecture

```
+-------------------------------------------------------------+
|                    Browser / iOS App                         |
+----------------------------+--------------------------------+
                             | HTTPS / REST API
                             v
+-------------------------------------------------------------+
|              Controller Container (Go)                       |
|  +-----------+  +----------+  +--------------------+        |
|  |  REST API |  | WebSocket|  |  Static File Server |        |
|  |  /api/*   |  |  /ws     |  |  (React SPA)        |        |
|  +-----+-----+  +----+-----+  +--------------------+        |
|        |             |                                       |
|  +-----+-------------+------+  +------------------+         |
|  |      Core Services        |  |   SQLite DB      |         |
|  |  - App Manager            |  |   (controller.db)|         |
|  |  - Build Service          |  +------------------+         |
|  |  - Port Allocator         |                               |
|  |  - Git Service            |  +------------------+         |
|  |  - Log Streamer           |  |  /data/repos     |         |
|  |  - Auth Service           |  |  (cloned repos)  |         |
|  +---------------------------+  +------------------+         |
|              |                                               |
|  +-----------+---------------------------------------------+ |
|  |           Docker Client (via docker.sock)                | |
|  +----------------------------------------------------------+ |
+-------------------------------------------------------------+
                             |
                             v
+-------------------------------------------------------------+
|                   Docker Engine                              |
|  +---------+  +---------+  +---------+  +---------+         |
|  | App A   |  | App B   |  | App C   |  | App D   |         |
|  | :13001  |  | :13002  |  | :13003  |  | :13004  |         |
|  +---------+  +---------+  +---------+  +---------+         |
+-------------------------------------------------------------+
```

---

## 5. App Data Model

```json
{
  "id": "uuid-v4",
  "name": "Hugo Web Tools",
  "slug": "hugowebtools",
  "description": "Collection of handy online tools",
  "icon": "/api/v1/apps/{id}/icon",

  "repo": {
    "url": "https://github.com/0HugoHu/hugowebtools",
    "branch": "main",
    "lastCommit": "abc123",
    "lastPulled": "2024-01-15T10:30:00Z"
  },

  "build": {
    "dockerfilePath": "./Dockerfile",
    "context": ".",
    "buildArgs": {}
  },

  "container": {
    "imageName": "hugowebtools:latest",
    "name": "hugowebtools",
    "ports": [
      { "internal": 80, "external": 13001, "protocol": "tcp" }
    ],
    "volumes": [],
    "restartPolicy": "unless-stopped"
  },

  "env": {
    "NODE_ENV": "production"
  },

  "status": "running",

  "metadata": {
    "createdAt": "2024-01-01T00:00:00Z",
    "updatedAt": "2024-01-15T10:30:00Z",
    "lastBuild": "2024-01-15T10:25:00Z",
    "lastBuildDuration": "45s",
    "lastBuildSuccess": true,
    "imageSize": "256MB"
  }
}
```

---

## 6. Standardized App Manifest (Optional)

For repos you control, add `nas-controller.json` in the repo root for auto-configuration:

```json
{
  "name": "Hugo Web Tools",
  "description": "Collection of handy online tools for developers",
  "icon": "./public/icon.png",
  "defaultPort": 8080,
  "env": {
    "NODE_ENV": "production"
  }
}
```

The controller auto-detects and uses this manifest when present. Keeps configuration simple.

---

## 7. API Design (REST)

### Authentication

```
POST   /api/v1/auth/login              # Login with password
POST   /api/v1/auth/logout             # Logout
PUT    /api/v1/auth/password           # Update password
GET    /api/v1/auth/check              # Check if authenticated
```

### Apps

```
GET    /api/v1/apps                    # List all apps
POST   /api/v1/apps                    # Add new app from GitHub URL
GET    /api/v1/apps/:id                # Get app details
PUT    /api/v1/apps/:id                # Update app configuration
DELETE /api/v1/apps/:id                # Remove app (stops container, deletes image)
GET    /api/v1/apps/:id/icon           # Get app icon

POST   /api/v1/apps/:id/build          # Trigger image build
POST   /api/v1/apps/:id/start          # Start container
POST   /api/v1/apps/:id/stop           # Stop container
POST   /api/v1/apps/:id/restart        # Restart container
POST   /api/v1/apps/:id/pull           # Pull latest from GitHub and rebuild

GET    /api/v1/apps/:id/logs           # Get container logs (query: lines, since)
WS     /api/v1/apps/:id/logs/stream    # Stream logs via WebSocket
DELETE /api/v1/apps/:id/logs           # Clear logs for this app

GET    /api/v1/apps/:id/build-logs     # Get build logs
WS     /api/v1/apps/:id/build/stream   # Stream build progress via WebSocket
```

### System

```
GET    /api/v1/system/info             # Controller version, uptime, Docker info
GET    /api/v1/system/ports            # List used/available ports
GET    /api/v1/system/storage          # Storage usage (DB, repos, logs, images)
POST   /api/v1/system/prune            # Cleanup unused Docker images
GET    /api/v1/system/health           # Controller health check
```

---

## 8. GitHub to App Workflow

```
1. User provides GitHub repo URL and branch
          |
          v
2. Controller validates URL and checks accessibility
          |
          v
3. Clone repository to /data/repos/{slug}
          |
          v
4. Verify Dockerfile exists
   - If no Dockerfile found, reject with error
          |
          v
5. Check for nas-controller.json (use if present for defaults)
          |
          v
6. User reviews/modifies configuration (ports, env vars)
          |
          v
7. Build Docker image
   - Stream build logs to UI in real-time
   - Tag as {slug}:latest
          |
          v
8. Register app in database
          |
          v
9. User starts container
          |
          v
10. Container runs, accessible at configured port
```

---

## 9. Port Management

### Allocation Strategy

- Reserved range: `13001-13999` for managed apps
- Controller UI: `13000` (configurable via `--port` flag)
- On app creation, automatically assign next available port in range
- Validate port availability before container start
- Store port assignments in database

### Conflict Resolution

1. Before starting container, check if port is in use
2. If conflict detected:
   - Automatically select next available port from reserved pool
   - Update app configuration with new port
   - Log the port change
3. No user intervention required

---

## 10. Container Naming

- Keep original container names from the repo/app slug
- Example: `hugowebtools`, `hdrive`, `hugoshare`
- No prefixes added

---

## 11. Volume Management

### Controller Volumes

```
/var/run/docker.sock:/var/run/docker.sock   # Docker control
/mnt/user/appdata/nas-controller/data:/data  # App repos, DB, logs
```

### Data Directory Structure

```
/data/
  controller.db           # SQLite database
  password.txt            # Auto-generated password (first run)
  repos/                  # Cloned repositories
    hugowebtools/
    hdrive/
  logs/                   # Build and container logs
    build-{app-id}.log
  icons/                  # Cached app icons
    {app-id}.png
```

---

## 12. Error Handling & Recovery

### Build Failures

- Stream build output to UI in real-time
- Persist full build log to file
- Mark app status as `build-failed`
- Display error in UI
- Allow retry

### Missing Dockerfile

- Reject app addition with clear error message
- Guide user to add Dockerfile to their repo

### Container Crashes

- Detect via Docker API
- Mark status as `stopped`
- Log details
- One-click restart available

### Controller Restart

- All state persisted in SQLite
- On startup, reconcile DB state with Docker reality
- Detect containers that died while controller was down

---

## 13. UI Design

### Design Principles

- Clean, modern aesthetic
- Responsive (works on desktop and mobile)
- System theme (dark/light based on OS preference)
- Minimal clicks to perform actions

### Main Dashboard

```
+--------------------------------------------------------------+
| NAS Controller                                    [Settings] |
+--------------------------------------------------------------+
|                                                              |
| Storage: DB 2.5MB | Repos 1.2GB | Logs 45MB | Images 2.8GB   |
|                                                              |
| [+ Add New App]                     4 apps, 2 running        |
|                                                              |
| +----------------------------------------------------------+ |
| |  [Icon] Hugo Web Tools                        * Running  | |
| |  github.com/0HugoHu/hugowebtools  |  :13001              | |
| |  256MB  |  Uptime: 2d 5h                                 | |
| |                                                          | |
| |  [Open] [Stop] [Rebuild] [Logs] [Config] [Delete]        | |
| +----------------------------------------------------------+ |
|                                                              |
| +----------------------------------------------------------+ |
| |  [Icon] HDrive                                o Stopped  | |
| |  github.com/0HugoHu/HDrive  |  :13002                    | |
| |  180MB                                                   | |
| |                                                          | |
| |  [Open] [Start] [Rebuild] [Logs] [Config] [Delete]       | |
| +----------------------------------------------------------+ |
|                                                              |
| +----------------------------------------------------------+ |
| |  [Icon] Hugo Share                           ~ Building  | |
| |  github.com/0HugoHu/hugoshare  |  :13003                 | |
| |  [=======>                              ] 42%            | |
| |                                                          | |
| |  [Cancel] [View Build Log]                               | |
| +----------------------------------------------------------+ |
|                                                              |
+--------------------------------------------------------------+
```

### Add App Modal

```
+----------------------------------------------------------+
|  Add New App                                        [X]   |
+----------------------------------------------------------+
|                                                          |
|  GitHub URL:                                             |
|  [https://github.com/username/repo                    ]  |
|                                                          |
|  Branch:                                                 |
|  [main                                                ]  |
|                                                          |
|  [Clone & Verify ->]                                     |
|                                                          |
+----------------------------------------------------------+
```

### App Configuration Modal

```
+----------------------------------------------------------+
|  Configure: Hugo Web Tools                          [X]   |
+----------------------------------------------------------+
|                                                          |
|  Name: [Hugo Web Tools                               ]   |
|                                                          |
|  Dockerfile: ./Dockerfile (detected)                     |
|                                                          |
|  Port:                                                   |
|  Internal: [80    ]  External: [13001 ] (auto-assigned)  |
|                                                          |
|  Environment Variables:                                  |
|  +----------------------------------------------+        |
|  | NODE_ENV    | production              | [X]  |        |
|  +----------------------------------------------+        |
|  [+ Add Variable]                                        |
|                                                          |
|  [Cancel]                        [Build & Create App]    |
|                                                          |
+----------------------------------------------------------+
```

### Logs View

```
+----------------------------------------------------------+
|  Logs: Hugo Web Tools                     [Download] [X]  |
+----------------------------------------------------------+
|  [Container Logs] [Build Logs]                           |
|  Lines: [100 v]  [Auto-scroll: ON]  [Clear Logs]         |
+----------------------------------------------------------+
|  2024-01-15 10:30:01 | Server started on port 80         |
|  2024-01-15 10:30:02 | Connected to database             |
|  2024-01-15 10:30:05 | Ready to accept connections       |
|  2024-01-15 10:31:00 | GET /api/health 200 5ms           |
|  ...                                                      |
+----------------------------------------------------------+
```

### Settings Page

```
+----------------------------------------------------------+
|  Settings                                           [X]   |
+----------------------------------------------------------+
|                                                          |
|  Password                                                |
|  Current: [**********                               ]    |
|  New:     [                                         ]    |
|  Confirm: [                                         ]    |
|  [Update Password]                                       |
|                                                          |
|  Storage                                                 |
|  +----------------------------------------------+        |
|  | Database     | 2.5 MB                        |        |
|  | Repositories | 1.2 GB                        |        |
|  | Logs         | 45 MB          [Clear All]    |        |
|  | Docker Images| 2.8 GB         [Prune Unused] |        |
|  +----------------------------------------------+        |
|                                                          |
|  Controller                                              |
|  Port: 13000                                             |
|  Version: 1.0.0                                          |
|                                                          |
+----------------------------------------------------------+
```

---

## 14. Authentication

### First Run

1. Controller generates random 16-character password
2. Password saved to `/data/password.txt`
3. Password displayed in container logs on first startup
4. User must login with this password

### Login Flow

1. Single password (no username)
2. Session stored in HTTP-only cookie
3. Session expires after 7 days of inactivity
4. Password can be updated via Settings page

### API Protection

- All `/api/v1/*` endpoints require authentication (except `/api/v1/auth/*`)
- Returns 401 if not authenticated
- Frontend redirects to login page

---

## 15. Security Considerations

- Controller only accessible on local network
- Docker socket access is privileged - controller is trusted
- No execution of arbitrary code outside Docker builds
- Container isolation via Docker's default security
- Password authentication required
- Session cookies are HTTP-only

---

## 16. Deployment

### Controller Dockerfile

```dockerfile
# Build frontend
FROM node:20-alpine AS frontend
WORKDIR /app/frontend
COPY frontend/package*.json ./
RUN npm ci
COPY frontend/ ./
RUN npm run build

# Build backend
FROM golang:1.22-alpine AS backend
WORKDIR /app
RUN apk add --no-cache gcc musl-dev
COPY go.* ./
RUN go mod download
COPY . .
COPY --from=frontend /app/frontend/dist ./frontend/dist
RUN CGO_ENABLED=1 go build -o controller ./cmd/controller

# Final image
FROM alpine:3.19
RUN apk add --no-cache git docker-cli ca-certificates
WORKDIR /app
COPY --from=backend /app/controller .
EXPOSE 13000
ENTRYPOINT ["/app/controller"]
```

### Docker Run Command

```bash
docker run -d \
  --name nas-controller \
  --restart unless-stopped \
  -p 13000:13000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v /mnt/user/appdata/nas-controller/data:/data \
  nas-controller:latest
```

### Unraid Template (XML)

```xml
<?xml version="1.0"?>
<Container version="2">
  <Name>NAS-Controller</Name>
  <Repository>nas-controller:latest</Repository>
  <Network>bridge</Network>
  <Privileged>false</Privileged>
  <Overview>Personal app controller for managing Dockerized web applications</Overview>
  <Category>Tools:</Category>
  <WebUI>http://[IP]:[PORT:13000]/</WebUI>
  <Config Name="Web UI Port" Target="13000" Default="13000" Mode="tcp" Description="Controller web interface port" Type="Port" Display="always" Required="true">13000</Config>
  <Config Name="Docker Socket" Target="/var/run/docker.sock" Default="/var/run/docker.sock" Mode="rw" Description="Docker socket for container management" Type="Path" Display="always" Required="true">/var/run/docker.sock</Config>
  <Config Name="Data Directory" Target="/data" Default="/mnt/user/appdata/nas-controller/data" Mode="rw" Description="Persistent data storage" Type="Path" Display="always" Required="true">/mnt/user/appdata/nas-controller/data</Config>
</Container>
```

---

## 17. Project Structure

```
nas-controller/
  cmd/
    controller/
      main.go               # Entry point
  internal/
    api/
      router.go             # HTTP router setup
      middleware.go         # Auth middleware
      handlers/
        apps.go             # App CRUD handlers
        auth.go             # Auth handlers
        system.go           # System handlers
        websocket.go        # WebSocket handlers
    services/
      app_manager.go        # App lifecycle management
      build_service.go      # Docker image building
      git_service.go        # Git clone/pull operations
      port_allocator.go     # Port management
      auth_service.go       # Password management
    docker/
      client.go             # Docker SDK wrapper
    database/
      sqlite.go             # SQLite operations
      migrations.go         # DB migrations
    models/
      app.go                # App data structures
  frontend/
    src/
      components/
        AppCard.tsx
        AddAppModal.tsx
        ConfigModal.tsx
        LogViewer.tsx
        Header.tsx
        StorageBar.tsx
      pages/
        Dashboard.tsx
        Login.tsx
        Settings.tsx
      api/
        client.ts           # API client
      hooks/
        useAuth.ts
        useApps.ts
      App.tsx
      main.tsx
    package.json
    vite.config.ts
    tailwind.config.js
  Dockerfile
  docker-compose.yml        # For development
  go.mod
  go.sum
  README.md
```

---

## 18. Example Apps

| App | Tech Stack | Dockerfile | Notes |
|-----|-----------|------------|-------|
| hugowebtools | Vue 3 + Vite | Yes | Fork of IT-Tools |
| HDrive | Go + SolidJS | Needs adding | File manager/WebDAV |
| hugoshare | Ember.js + WebRTC | Yes | P2P file sharing |

---

## 19. Future Enhancements (Low Priority)

- iOS native app using same REST API
- Resource usage graphs and history
