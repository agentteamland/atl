---
knowledge-base-summary: "mkcert for trusted self-signed certificates. Reverse proxy (nginx/traefik) for HTTPS termination. Required for: OAuth callbacks, secure cookies, PWA testing. Optional — most projects work without it."
---
# SSL/TLS for Local Development

HTTPS on localhost using mkcert and a reverse proxy (Traefik or nginx). Required when OAuth providers, secure cookies, or PWA service workers need HTTPS.

## When Do You Need Local HTTPS?

### Required

| Use Case | Why HTTPS? |
|----------|-----------|
| OAuth callbacks (Google, GitHub) | OAuth providers reject `http://` redirect URIs |
| Secure cookies (`SameSite=None; Secure`) | Browser blocks secure cookies over HTTP |
| PWA Service Worker | Service workers require secure context |
| WebAuthn / FIDO2 | Web authentication API requires HTTPS |
| Cross-origin requests with credentials | Some CORS + cookies setups need HTTPS |

### NOT Required

| Use Case | Why HTTP Is Fine |
|----------|-----------------|
| Internal admin panel | No external OAuth, no public access |
| API development (Postman/curl) | Tools don't care about HTTPS |
| Mobile app + API | App connects to `http://localhost` directly |
| Background services (Worker, LogIngest) | No browser involved |

**Rule: Don't add HTTPS complexity unless you actually need it.** Most development workflows work fine with HTTP.

## Step 1: Install mkcert

mkcert creates locally-trusted development certificates. No browser warnings, no self-signed cert issues.

### macOS

```bash
brew install mkcert
brew install nss        # For Firefox support

# Install the local CA (one-time)
mkcert -install
```

### Linux (Ubuntu/Debian)

```bash
sudo apt install libnss3-tools
# Download mkcert binary from https://github.com/FiloSottile/mkcert/releases
chmod +x mkcert-v*-linux-amd64
sudo mv mkcert-v*-linux-amd64 /usr/local/bin/mkcert

mkcert -install
```

### Windows

```powershell
choco install mkcert
mkcert -install
```

## Step 2: Generate Certificates

```bash
# Create a certs directory in your project
mkdir -p certs

# Generate cert for localhost and common local domains
mkcert -key-file certs/local-key.pem -cert-file certs/local-cert.pem \
  localhost \
  127.0.0.1 \
  ::1 \
  "*.localhost" \
  "api.localhost" \
  "app.localhost"
```

This produces:
- `certs/local-cert.pem` -- the certificate
- `certs/local-key.pem` -- the private key

### .gitignore

```
# SSL certificates (generated per machine)
certs/
```

Certificates are machine-specific (trusted only by the local CA that generated them). Each developer runs `mkcert` on their own machine.

## Option A: Traefik Reverse Proxy (Recommended)

Traefik is a modern reverse proxy that integrates well with Docker. It auto-discovers services via Docker labels.

### Docker Compose with Traefik

```yaml
services:
  traefik:
    image: traefik:v3.1
    container_name: ${PROJECT_PREFIX}-traefik
    command:
      - "--api.insecure=true"                        # Traefik dashboard on :8080
      - "--providers.docker=true"                    # Auto-discover Docker services
      - "--providers.docker.exposedbydefault=false"  # Only route labeled services
      - "--entrypoints.web.address=:80"
      - "--entrypoints.websecure.address=:443"
      - "--entrypoints.web.http.redirections.entryPoint.to=websecure"  # HTTP -> HTTPS
      - "--providers.file.directory=/etc/traefik/dynamic"
    ports:
      - "80:80"
      - "443:443"
      - "8080:8080"     # Traefik dashboard
    volumes:
      - /var/run/docker.sock:/var/run/docker.sock:ro
      - ./certs:/etc/traefik/certs:ro
      - ./traefik/dynamic:/etc/traefik/dynamic:ro
    restart: unless-stopped

  api:
    build:
      context: .
      dockerfile: src/ExampleApp.Api/Dockerfile.dev
    container_name: ${PROJECT_PREFIX}-api
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`api.localhost`)"
      - "traefik.http.routers.api.entrypoints=websecure"
      - "traefik.http.routers.api.tls=true"
      - "traefik.http.services.api.loadbalancer.server.port=3000"
    # No need to expose ports directly -- Traefik handles routing
    # ports:
    #   - "3000:3000"   <-- Remove this when using Traefik
```

### Traefik Dynamic Configuration (TLS Certificates)

```yaml
# traefik/dynamic/tls.yml
tls:
  stores:
    default:
      defaultCertificate:
        certFile: /etc/traefik/certs/local-cert.pem
        keyFile: /etc/traefik/certs/local-key.pem

  certificates:
    - certFile: /etc/traefik/certs/local-cert.pem
      keyFile: /etc/traefik/certs/local-key.pem
```

### Multiple Services with Traefik

```yaml
services:
  api:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.api.rule=Host(`api.localhost`)"
      - "traefik.http.routers.api.entrypoints=websecure"
      - "traefik.http.routers.api.tls=true"
      - "traefik.http.services.api.loadbalancer.server.port=3000"

  socket:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.socket.rule=Host(`socket.localhost`)"
      - "traefik.http.routers.socket.entrypoints=websecure"
      - "traefik.http.routers.socket.tls=true"
      - "traefik.http.services.socket.loadbalancer.server.port=3002"

  # React/Flutter web app
  web:
    labels:
      - "traefik.enable=true"
      - "traefik.http.routers.web.rule=Host(`app.localhost`)"
      - "traefik.http.routers.web.entrypoints=websecure"
      - "traefik.http.routers.web.tls=true"
      - "traefik.http.services.web.loadbalancer.server.port=5173"
```

### Resulting URLs

```
https://api.localhost     -> API (port 3000 inside container)
https://socket.localhost  -> Socket (port 3002 inside container)
https://app.localhost     -> Web app (port 5173 inside container)
http://localhost:8080     -> Traefik dashboard
```

## Option B: nginx Reverse Proxy

For simpler setups or if you prefer nginx.

### nginx Configuration

```nginx
# nginx/nginx.conf
events {
    worker_connections 1024;
}

http {
    # HTTP -> HTTPS redirect
    server {
        listen 80;
        server_name localhost api.localhost app.localhost;
        return 301 https://$host$request_uri;
    }

    # API
    server {
        listen 443 ssl;
        server_name api.localhost;

        ssl_certificate     /etc/nginx/certs/local-cert.pem;
        ssl_certificate_key /etc/nginx/certs/local-key.pem;

        location / {
            proxy_pass http://api:3000;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }

    # Socket (WebSocket support)
    server {
        listen 443 ssl;
        server_name socket.localhost;

        ssl_certificate     /etc/nginx/certs/local-cert.pem;
        ssl_certificate_key /etc/nginx/certs/local-key.pem;

        location / {
            proxy_pass http://socket:3002;
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;
        }
    }

    # Web app
    server {
        listen 443 ssl;
        server_name app.localhost;

        ssl_certificate     /etc/nginx/certs/local-cert.pem;
        ssl_certificate_key /etc/nginx/certs/local-key.pem;

        location / {
            proxy_pass http://web:5173;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
            proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
            proxy_set_header X-Forwarded-Proto $scheme;

            # HMR WebSocket support
            proxy_http_version 1.1;
            proxy_set_header Upgrade $http_upgrade;
            proxy_set_header Connection "upgrade";
        }
    }
}
```

### Docker Compose with nginx

```yaml
services:
  nginx:
    image: nginx:alpine
    container_name: ${PROJECT_PREFIX}-nginx
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf:ro
      - ./certs:/etc/nginx/certs:ro
    depends_on:
      - api
      - socket
    restart: unless-stopped
```

## /etc/hosts Setup

For subdomain-based routing (`api.localhost`, `app.localhost`), add entries to `/etc/hosts`:

```bash
# /etc/hosts
127.0.0.1  api.localhost
127.0.0.1  app.localhost
127.0.0.1  socket.localhost
```

On macOS/Linux:

```bash
# Add entries (one-time)
echo "127.0.0.1  api.localhost app.localhost socket.localhost" | sudo tee -a /etc/hosts
```

**Note:** Chrome and most modern browsers resolve `*.localhost` to 127.0.0.1 automatically. The `/etc/hosts` entry may not be needed for some browsers, but adding it ensures universal compatibility.

## Full Setup Guide (Step by Step)

### 1. Install mkcert and create CA

```bash
brew install mkcert nss
mkcert -install
```

### 2. Generate certificates

```bash
mkdir -p certs
mkcert -key-file certs/local-key.pem -cert-file certs/local-cert.pem \
  localhost 127.0.0.1 ::1 "*.localhost"
```

### 3. Add .gitignore entry

```bash
echo "certs/" >> .gitignore
```

### 4. Choose Traefik or nginx (see sections above)

### 5. Create dynamic config (Traefik) or nginx.conf

### 6. Add reverse proxy service to docker-compose.yml

### 7. Update /etc/hosts if using subdomains

```bash
echo "127.0.0.1  api.localhost app.localhost socket.localhost" | sudo tee -a /etc/hosts
```

### 8. Start

```bash
docker compose up
```

### 9. Verify

```bash
curl -v https://api.localhost/health
# Should show: SSL certificate verify ok
```

## Troubleshooting

### "Your connection is not private" in browser

- mkcert CA not installed: run `mkcert -install`
- Certificate doesn't cover the domain: regenerate with correct domains
- Firefox: needs NSS tools installed (`brew install nss`)

### "Connection refused" on HTTPS port

- Reverse proxy not running: `docker compose logs traefik` or `docker compose logs nginx`
- Port 443 already in use: `lsof -i :443`

### WebSocket connection fails through proxy

- Ensure `Upgrade` and `Connection` headers are forwarded
- For nginx: add `proxy_http_version 1.1` and WebSocket headers (see nginx config above)
- For Traefik: WebSocket is supported by default

### OAuth callback not working

- Verify redirect URI uses `https://` in OAuth provider settings
- Cookie `SameSite` and `Secure` flags match the HTTPS context
- Check browser console for mixed content warnings
