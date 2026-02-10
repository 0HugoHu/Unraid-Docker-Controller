# Unraid Docker Controller

A personal Docker app controller for managing containerized web applications on Unraid/NAS systems. Add apps via GitHub URL, build Docker images locally, and manage container lifecycle through a clean web UI.

## Features

- **App Management**: Add apps by GitHub URL + branch, auto-clone and build
- **Container Lifecycle**: Start, stop, restart, rebuild containers
- **Port Management**: Automatic port allocation (13001-13999) with conflict detection
- **Build Logs**: View build output and container logs
- **Storage Overview**: Monitor disk usage, prune unused images
- **REST API**: Clean API for future mobile app integration
- **Authentication**: Password-protected access (auto-generated on first run)

## Screenshots

*Coming soon*

## Requirements

- Docker installed on host system
- Repositories must include a `Dockerfile`

## Quick Start

### Using Docker (Recommended)

```bash
docker run -d \
  --name nas-controller \
  -p 13000:13000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v nas-controller-data:/data \
  nas-controller
```

### Using Docker Compose

```bash
docker-compose up -d
```

### Build from Source

```bash
# Build the image
docker build -t nas-controller .

# Run
docker run -d \
  --name nas-controller \
  -p 13000:13000 \
  -v /var/run/docker.sock:/var/run/docker.sock \
  -v nas-controller-data:/data \
  nas-controller
```

## Usage

1. Access the web UI at `http://your-server:13000`
2. On first run, check the container logs for the generated password:
   ```bash
   docker logs nas-controller | grep password
   ```
3. Login with the password
4. Click "Add App" to add a new application:
   - Enter the GitHub repository URL
   - Select the branch
   - Configure port and environment variables
5. Build and start your app

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `DATA_DIR` | Data storage directory | `/data` |
| `PORT` | Controller port | `13000` |

### Port Ranges

- Controller: `13000`
- Managed apps: `13001-13999`

## API

The controller exposes a REST API for all operations:

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/api/v1/auth/login` | POST | Login |
| `/api/v1/auth/logout` | POST | Logout |
| `/api/v1/apps` | GET | List all apps |
| `/api/v1/apps` | POST | Create app |
| `/api/v1/apps/:id` | GET | Get app details |
| `/api/v1/apps/:id` | PUT | Update app |
| `/api/v1/apps/:id` | DELETE | Delete app |
| `/api/v1/apps/:id/build` | POST | Build app |
| `/api/v1/apps/:id/start` | POST | Start app |
| `/api/v1/apps/:id/stop` | POST | Stop app |
| `/api/v1/apps/:id/restart` | POST | Restart app |
| `/api/v1/apps/:id/logs` | GET | Get container logs |
| `/api/v1/system/info` | GET | Get system info |
| `/api/v1/system/storage` | GET | Get storage info |
| `/api/v1/system/prune` | POST | Prune unused images |

## Tech Stack

- **Backend**: Go 1.24, Gin, Docker SDK, SQLite
- **Frontend**: React 18, TypeScript, Vite, Tailwind CSS
- **Deployment**: Multi-stage Docker build (Alpine)

## Development

### Backend

```bash
# Run backend locally
go run ./cmd/controller
```

### Frontend

```bash
cd frontend
npm install
npm run dev
```

## License

MIT
