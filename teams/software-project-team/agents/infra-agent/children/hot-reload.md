---
knowledge-base-summary: "Every service supports hot reload in development. .NET: dotnet watch run. React/Vite: HMR. Flutter: hot reload (native, not Docker). File watcher configuration. Volume mount requirements for hot reload to work."
---
# Hot Reload

Every service in the stack supports live code reloading during development. Edit code on the host, the container picks up the change automatically. No manual rebuild, no restart.

## .NET Hot Reload: dotnet watch

### How It Works

1. Source code is mounted from the host via bind mount: `./src:/src/src`
2. The container runs `dotnet watch run` instead of `dotnet run`
3. `dotnet watch` monitors all `.cs`, `.razor`, `.json`, `.csproj` files for changes
4. On change detection: recompiles the affected assembly and restarts the application
5. For supported changes (method bodies, lambdas), it applies "hot reload" without restart

### Container Configuration

```yaml
# docker-compose.yml
api:
  build:
    context: .
    dockerfile: src/ExampleApp.Api/Dockerfile.dev
  volumes:
    - ./src:/src/src                              # Source code from host
    - ./ExampleApp.sln:/src/ExampleApp.sln    # Solution file
    - api_bin:/src/src/ExampleApp.Api/bin        # Shadow volume (isolation)
    - api_obj:/src/src/ExampleApp.Api/obj        # Shadow volume (isolation)
    - nuget_api:/root/.nuget/packages             # NuGet cache (persistence)
```

```dockerfile
# Dockerfile.dev
FROM mcr.microsoft.com/dotnet/sdk:9.0
WORKDIR /src

# Copy csproj + restore (cached layer)
COPY src/ExampleApp.Domain/*.csproj src/ExampleApp.Domain/
# ... (all projects)
COPY ExampleApp.sln .
RUN dotnet restore

# Source is NOT copied — mounted at runtime
ENTRYPOINT ["dotnet", "watch", "run", "--project", "src/ExampleApp.Api/ExampleApp.Api.csproj", "--urls", "http://0.0.0.0:3000"]
```

### What Triggers Recompilation

| File Change | Behavior |
|------------|----------|
| `.cs` file modified | Recompile + restart (or hot reload if supported) |
| `.csproj` modified | Full rebuild |
| `appsettings.json` modified | Restart (configuration reload) |
| NuGet package added/removed | Requires `docker compose build` + restart |
| New project reference added | Requires `docker compose build` + restart |

### What dotnet watch Cannot Handle

These changes require a manual `docker compose build`:

1. **New NuGet package** — the package needs to be restored, which happens during build
2. **New project added to solution** — the .csproj needs to be copied and restored
3. **Dockerfile.dev changes** — obviously needs a rebuild
4. **Solution structure changes** — new project references, solution-level changes

```bash
# After adding a NuGet package or new project:
docker compose build api
docker compose up -d api
```

---

## React/Vite Hot Reload: HMR

### How It Works

1. Source code is mounted from the host: `./web:/app`
2. Vite dev server runs with `--host 0.0.0.0` (required for Docker)
3. Vite's Hot Module Replacement (HMR) detects file changes via WebSocket
4. Browser receives the update and patches the module in-place (no full page reload)

### Container Configuration

```yaml
# docker-compose.yml
web:
  build:
    context: ./web
    dockerfile: Dockerfile.dev
  container_name: wfm-web
  ports:
    - "${WEB_PORT:-3001}:3001"
  environment:
    VITE_API_URL: http://localhost:${API_PORT:-3000}
  volumes:
    - ./web:/app                                   # Source from host
    - web_node_modules:/app/node_modules           # Shadow volume for node_modules
```

```dockerfile
# web/Dockerfile.dev
FROM node:22-alpine
WORKDIR /app

COPY package.json package-lock.json ./
RUN npm ci

# Source is NOT copied — mounted at runtime
EXPOSE 3001
CMD ["npx", "vite", "--host", "0.0.0.0", "--port", "3001"]
```

### Critical: node_modules Shadow Volume

```yaml
volumes:
  - ./web:/app                          # Mount everything from host
  - web_node_modules:/app/node_modules  # BUT overlay node_modules with container's own
```

Without the shadow volume, the host's `node_modules` (installed for macOS/native binaries) would be mounted into the Linux container, causing binary incompatibility errors. The shadow volume ensures the container has its own `node_modules` with Linux-compatible binaries.

### Vite Configuration for Docker

```typescript
// web/vite.config.ts
import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  server: {
    host: '0.0.0.0',      // Required: listen on all interfaces (not just localhost)
    port: 3001,
    watch: {
      usePolling: true,    // Required on some Docker setups (especially Docker Desktop)
      interval: 1000,      // Poll every 1 second
    },
    hmr: {
      clientPort: 3001,    // HMR WebSocket connects to this host port
    },
  },
});
```

### What Triggers HMR

| File Change | Behavior |
|------------|----------|
| `.tsx`, `.jsx`, `.ts`, `.js` | HMR (instant, no page reload) |
| `.css`, `.scss` | HMR (instant style update) |
| `vite.config.ts` | Full restart of Vite |
| `package.json` | Requires `docker compose build web` |
| `tailwind.config.js` | Full restart of Vite |

---

## Flutter: Native on Host (NOT Docker)

Flutter is the **only exception** to the "everything runs in Docker" rule. Flutter runs natively on the host machine.

### Why Not Docker?

1. **iOS Simulator** requires macOS + Xcode — cannot run in a Linux container
2. **Android Emulator** requires KVM or hardware acceleration — impractical in Docker
3. **USB debugging** for physical devices requires host-level USB access
4. **Flutter hot reload** communicates with the simulator/emulator via host networking

### Developer Workflow

```bash
# Flutter runs directly on the host machine
flutter run                # iOS simulator or Android emulator
flutter run -d chrome      # Web target (for testing)
```

### API Connection from Flutter

Flutter on the host connects to the API running in Docker via localhost:

```dart
// lib/config/api_config.dart
const apiBaseUrl = 'http://localhost:3000';  // Docker-exposed API port
```

For physical Android devices, `localhost` does not work (it refers to the phone). Use the host machine's IP:

```dart
// For physical Android device:
const apiBaseUrl = 'http://10.0.2.2:3000';  // Android emulator's host alias
// OR
const apiBaseUrl = 'http://192.168.1.100:3000';  // Actual host IP
```

---

## Volume Mount Requirements Summary

For hot reload to work, these volume mounts are mandatory:

### .NET Service

```yaml
volumes:
  - ./src:/src/src                              # Source code (hot reload source)
  - ./{SolutionName}.sln:/src/{SolutionName}.sln  # Solution file
  - {service}_bin:/src/src/{Project}/bin         # Shadow (prevents Mac/Linux conflict)
  - {service}_obj:/src/src/{Project}/obj         # Shadow (prevents Mac/Linux conflict)
  - nuget_{service}:/root/.nuget/packages       # Cache (prevents re-download)
```

### Frontend Service

```yaml
volumes:
  - ./web:/app                                  # Source code (hot reload source)
  - web_node_modules:/app/node_modules          # Shadow (prevents Mac/Linux conflict)
```

Missing any of these will either break hot reload or cause cross-platform build errors.

---

## File Watcher Limits

### Linux (Docker Desktop on Mac uses a Linux VM)

If file watching stops working or you see "too many open files" errors:

```bash
# Check current inotify watch limit:
docker compose exec api cat /proc/sys/fs/inotify/max_user_watches

# If it is low (e.g., 8192), increase it:
# On the Docker Desktop Linux VM, this is usually not needed.
# But if running Docker on a Linux host:
echo "fs.inotify.max_user_watches=524288" | sudo tee -a /etc/sysctl.conf
sudo sysctl -p
```

### macOS with Docker Desktop

Docker Desktop for macOS handles file watching through its filesystem sync layer (VirtioFS or gRPC FUSE). If watching is unreliable:

1. Check Docker Desktop settings: Settings > General > File Sharing > VirtioFS (recommended)
2. For Vite, enable `usePolling: true` in vite.config.ts (fallback mechanism)
3. For .NET, `dotnet watch` uses its own polling — usually works without configuration

---

## Troubleshooting: "My Changes Are Not Reflected"

### Symptom: Code change is saved but the container does not reload

**Check 1: Is the source actually mounted?**

```bash
# Verify the file exists inside the container:
docker compose exec api cat src/ExampleApp.Api/Program.cs

# If the file content does not match your local file, the volume mount is wrong
```

**Check 2: Is dotnet watch running?**

```bash
# Check the container logs:
docker compose logs -f api

# You should see: "dotnet watch: Started" and later "dotnet watch: Hot reload"
# If you see nothing, the entrypoint is wrong
```

**Check 3: Is the bin/obj shadow volume working?**

```bash
# If you see errors about assembly versions or "file not found":
# Reset the shadow volumes:
docker compose down
docker volume rm $(docker volume ls -q | grep -E '(api_bin|api_obj)')
docker compose up
```

**Check 4: Is the NuGet cache stale?**

```bash
# If restore fails or packages are not found:
docker volume rm $(docker volume ls -q | grep nuget_api)
docker compose build api
docker compose up -d api
```

**Check 5: Did you add a new NuGet package?**

```bash
# dotnet watch cannot detect new packages — rebuild:
docker compose build api
docker compose up -d api
```

### Symptom: Vite HMR not working (page does not update)

**Check 1: Is the WebSocket connection established?**

Open browser DevTools (Network tab, WS filter). You should see a WebSocket connection to `ws://localhost:3001`. If not:
- Verify `--host 0.0.0.0` is set in the Vite command
- Verify `hmr.clientPort` matches the exposed host port

**Check 2: Is the node_modules shadow volume correct?**

```bash
# If you see import errors or "module not found":
docker compose down
docker volume rm $(docker volume ls -q | grep web_node_modules)
docker compose build web
docker compose up -d web
```

**Check 3: Is file polling enabled?**

For Docker Desktop on macOS, Vite's native file watcher may not detect changes. Enable polling in `vite.config.ts`:

```typescript
server: {
  watch: {
    usePolling: true,
    interval: 1000,
  },
}
```

---
