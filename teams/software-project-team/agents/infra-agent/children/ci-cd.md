---
knowledge-base-summary: "GitHub Actions templates: build, test, push image, deploy. Multi-stage Docker build for production images. Environment-specific deployments (staging, production). Secrets management in CI."
---
# CI/CD Pipeline

GitHub Actions workflows for build, test, image push, and deployment. All builds happen inside Docker -- CI mirrors the development experience.

## Workflow Overview

```
Push to main ──────> Build & Test ──> Push Image (GHCR) ──> Deploy (Production)
Push to develop ───> Build & Test ──> Push Image (GHCR) ──> Deploy (Staging)
Pull Request ──────> Build & Test (no push, no deploy)
```

## Workflow 1: Build & Test

Runs on every push and PR. Builds the .NET solution inside Docker and runs tests.

```yaml
# .github/workflows/build-and-test.yml
name: Build & Test

on:
  push:
    branches: [main, develop]
  pull_request:
    branches: [main, develop]

env:
  DOTNET_VERSION: "9.0"

jobs:
  build-and-test:
    runs-on: ubuntu-latest

    services:
      postgres:
        image: postgres:17-alpine
        env:
          POSTGRES_USER: testuser
          POSTGRES_PASSWORD: testpassword
          POSTGRES_DB: testdb
        ports:
          - 5432:5432
        options: >-
          --health-cmd="pg_isready -U testuser"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=5

      redis:
        image: redis:7-alpine
        ports:
          - 6379:6379
        options: >-
          --health-cmd="redis-cli ping"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=5

      rabbitmq:
        image: rabbitmq:3-management-alpine
        ports:
          - 5672:5672
        options: >-
          --health-cmd="rabbitmq-diagnostics -q ping"
          --health-interval=10s
          --health-timeout=5s
          --health-retries=5

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Setup .NET
        uses: actions/setup-dotnet@v4
        with:
          dotnet-version: ${{ env.DOTNET_VERSION }}

      - name: Cache NuGet packages
        uses: actions/cache@v4
        with:
          path: ~/.nuget/packages
          key: nuget-${{ runner.os }}-${{ hashFiles('**/*.csproj') }}
          restore-keys: |
            nuget-${{ runner.os }}-

      - name: Restore dependencies
        run: dotnet restore ExampleApp.sln

      - name: Build
        run: dotnet build ExampleApp.sln --no-restore --configuration Release

      - name: Test
        run: dotnet test ExampleApp.sln --no-build --configuration Release --verbosity normal
        env:
          ConnectionStrings__DefaultConnection: "Host=localhost;Port=5432;Database=testdb;Username=testuser;Password=testpassword"
          Redis__ConnectionString: "localhost:6379"
          RabbitMQ__Host: "localhost"
          RabbitMQ__Port: "5672"
```

## Workflow 2: Docker Image Build & Push

Builds production Docker images and pushes to GitHub Container Registry (GHCR).

```yaml
# .github/workflows/docker-publish.yml
name: Docker Publish

on:
  push:
    branches: [main, develop]
    tags: ["v*"]

env:
  REGISTRY: ghcr.io
  IMAGE_PREFIX: ${{ github.repository_owner }}/example-app

jobs:
  build-and-push:
    runs-on: ubuntu-latest
    permissions:
      contents: read
      packages: write

    strategy:
      matrix:
        service:
          - name: api
            dockerfile: src/ExampleApp.Api/Dockerfile
            context: .
          - name: socket
            dockerfile: src/ExampleApp.Socket/Dockerfile
            context: .
          - name: worker
            dockerfile: src/ExampleApp.Worker/Dockerfile
            context: .
          - name: log-ingest
            dockerfile: src/ExampleApp.LogIngest/Dockerfile
            context: .
          - name: mail-sender
            dockerfile: src/ExampleApp.MailSender/Dockerfile
            context: .

    steps:
      - name: Checkout
        uses: actions/checkout@v4

      - name: Set up Docker Buildx
        uses: docker/setup-buildx-action@v3

      - name: Log in to GHCR
        uses: docker/login-action@v3
        with:
          registry: ${{ env.REGISTRY }}
          username: ${{ github.actor }}
          password: ${{ secrets.GITHUB_TOKEN }}

      - name: Extract metadata
        id: meta
        uses: docker/metadata-action@v5
        with:
          images: ${{ env.REGISTRY }}/${{ env.IMAGE_PREFIX }}-${{ matrix.service.name }}
          tags: |
            type=ref,event=branch
            type=sha,prefix=
            type=semver,pattern={{version}}
            type=raw,value=latest,enable=${{ github.ref == 'refs/heads/main' }}

      - name: Build and push
        uses: docker/build-push-action@v6
        with:
          context: ${{ matrix.service.context }}
          file: ${{ matrix.service.dockerfile }}
          push: true
          tags: ${{ steps.meta.outputs.tags }}
          labels: ${{ steps.meta.outputs.labels }}
          cache-from: type=gha
          cache-to: type=gha,mode=max
```

### Image Tags Produced

| Trigger | Tags |
|---------|------|
| Push to `main` | `latest`, `main`, `<sha>` |
| Push to `develop` | `develop`, `<sha>` |
| Tag `v1.2.3` | `1.2.3`, `<sha>` |

## Workflow 3: Deploy

Deploys to server via SSH. Pulls the latest image and restarts the service.

```yaml
# .github/workflows/deploy.yml
name: Deploy

on:
  workflow_run:
    workflows: ["Docker Publish"]
    types: [completed]
    branches: [main, develop]

jobs:
  deploy-staging:
    if: >
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.head_branch == 'develop'
    runs-on: ubuntu-latest
    environment: staging

    steps:
      - name: Deploy to staging
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.STAGING_HOST }}
          username: ${{ secrets.STAGING_USER }}
          key: ${{ secrets.STAGING_SSH_KEY }}
          script: |
            cd /opt/example-app
            docker compose pull
            docker compose up -d --remove-orphans
            docker image prune -f

  deploy-production:
    if: >
      github.event.workflow_run.conclusion == 'success' &&
      github.event.workflow_run.head_branch == 'main'
    runs-on: ubuntu-latest
    environment: production

    steps:
      - name: Deploy to production
        uses: appleboy/ssh-action@v1
        with:
          host: ${{ secrets.PRODUCTION_HOST }}
          username: ${{ secrets.PRODUCTION_USER }}
          key: ${{ secrets.PRODUCTION_SSH_KEY }}
          script: |
            cd /opt/example-app
            docker compose -f docker-compose.yml -f docker-compose.prod.yml pull
            docker compose -f docker-compose.yml -f docker-compose.prod.yml up -d --remove-orphans
            docker image prune -f
            echo "Deployed at $(date)"
```

## Environment Secrets

Configure these in GitHub repository settings under Settings -> Environments.

### Required Secrets

| Secret | Where | Purpose |
|--------|-------|---------|
| `GITHUB_TOKEN` | Automatic | GHCR authentication (built-in) |
| `STAGING_HOST` | staging environment | Staging server IP/hostname |
| `STAGING_USER` | staging environment | SSH username |
| `STAGING_SSH_KEY` | staging environment | SSH private key |
| `PRODUCTION_HOST` | production environment | Production server IP/hostname |
| `PRODUCTION_USER` | production environment | SSH username |
| `PRODUCTION_SSH_KEY` | production environment | SSH private key |

### Environment Protection Rules

- **staging**: no restrictions, deploys automatically on develop push
- **production**: require manual approval, restrict to main branch

## Multi-Service Build Strategy

All .NET services share the same solution file and project references. The matrix strategy builds each service in parallel.

### Why Matrix?

```yaml
strategy:
  matrix:
    service:
      - name: api
        dockerfile: src/ExampleApp.Api/Dockerfile
        context: .
```

- Each service gets its own image: `ghcr.io/owner/example-app-api`
- Parallel builds -- all services build simultaneously
- Independent tagging -- each service has its own version lifecycle
- Shared context (`.`) because projects reference each other via solution

### Alternative: Single Multi-Target Dockerfile

For simpler setups, one Dockerfile with build targets:

```dockerfile
# Build all projects in one restore/build step
FROM mcr.microsoft.com/dotnet/sdk:9.0 AS build
WORKDIR /src
COPY . .
RUN dotnet restore
RUN dotnet publish src/ExampleApp.Api -c Release -o /app/api
RUN dotnet publish src/ExampleApp.Worker -c Release -o /app/worker
```

This reduces build time (single restore) but produces larger intermediate images.

## Caching Strategy

### Docker Layer Cache (GitHub Actions Cache)

```yaml
cache-from: type=gha
cache-to: type=gha,mode=max
```

Uses GitHub Actions cache backend for Docker layers. Significantly speeds up rebuilds when only application code changes (base image and NuGet restore layers are cached).

### NuGet Cache

```yaml
- name: Cache NuGet packages
  uses: actions/cache@v4
  with:
    path: ~/.nuget/packages
    key: nuget-${{ runner.os }}-${{ hashFiles('**/*.csproj') }}
    restore-keys: |
      nuget-${{ runner.os }}-
```

Cache key is based on .csproj files -- only invalidated when package references change.

## Branch Strategy

```
main     ──────────> Production (auto-deploy with approval)
  |
develop  ──────────> Staging (auto-deploy)
  |
feature/xyz ───────> PR to develop (build & test only)
```

### Rules

1. **Never push directly to main** -- always merge from develop
2. **Feature branches** target develop
3. **Hotfixes** can branch from main, PR to both main and develop
4. **Tags** (`v1.0.0`) trigger versioned image builds

## Workflow for Alternative Registry (DockerHub)

Replace the GHCR login step:

```yaml
- name: Log in to DockerHub
  uses: docker/login-action@v3
  with:
    username: ${{ secrets.DOCKERHUB_USERNAME }}
    password: ${{ secrets.DOCKERHUB_TOKEN }}

# And change the image prefix:
env:
  REGISTRY: docker.io
  IMAGE_PREFIX: yourusername/example-app
```

## Health Check After Deploy

Add a post-deploy verification step:

```yaml
- name: Verify deployment
  run: |
    sleep 10
    for i in {1..5}; do
      if curl -sf https://${{ secrets.PRODUCTION_HOST }}/health; then
        echo "Health check passed"
        exit 0
      fi
      echo "Attempt $i failed, retrying in 10s..."
      sleep 10
    done
    echo "Health check failed after 5 attempts"
    exit 1
```
