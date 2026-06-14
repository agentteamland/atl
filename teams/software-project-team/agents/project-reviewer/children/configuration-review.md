---
knowledge-base-summary: "Check .env.example completeness, hardcoded config values, health checks, Docker Compose correctness, environment-specific configs, and secret exposure."
---
# Configuration Review

## .env.example Completeness

### What to Check

Every environment variable used in the project must be documented in `.env.example`:

1. **Scan docker-compose.yml** for all `${VARIABLE}` references
2. **Scan appsettings.json** for placeholder values
3. **Scan code** for `Environment.GetEnvironmentVariable()` calls
4. **Compare** against `.env.example` entries

### How to Verify

```bash
# Find all env vars referenced in docker-compose
grep -oP '\$\{[^}]+\}' docker-compose.yml | sort -u

# Find all env vars in code
grep -rn "GetEnvironmentVariable\|IConfiguration\[" src/ --include="*.cs"

# Check .env.example contents
cat .env.example
```

### Required Information per Variable

Each entry in `.env.example` should have:
- Variable name
- Example value (not a real secret)
- Comment explaining what it's for

```env
# PostgreSQL connection
POSTGRES_USER=example_app
POSTGRES_PASSWORD=your_password_here
POSTGRES_DB=example_app

# JWT Configuration
JWT_SECRET=your_jwt_secret_at_least_32_characters
JWT_ISSUER=example-app
JWT_AUDIENCE=example-app-clients
```

### Common Findings

- Variable used in docker-compose but missing from .env.example
- Variable has no description comment
- Example value is a real secret (copy-pasted from .env)
- .env.example doesn't exist at all

## Hardcoded Config Values in Code

### What to Look For

Configuration values that should be in appsettings.json or environment variables but are hardcoded:

```csharp
// BAD: Hardcoded connection string
var conn = "Host=localhost;Database=mydb;Username=admin;Password=secret";

// BAD: Hardcoded timeout
var timeout = TimeSpan.FromSeconds(30);

// BAD: Hardcoded URL
var apiUrl = "https://api.example.com/v1";

// BAD: Hardcoded feature toggle
var enableCache = true;

// CORRECT: From configuration
var timeout = TimeSpan.FromSeconds(_settings.TimeoutSeconds);
var apiUrl = _settings.ExternalApiUrl;
```

### How to Detect

```bash
# Find hardcoded strings that look like config
grep -rn '"localhost\|"127.0.0.1\|"http://\|"https://' src/ --include="*.cs"

# Find hardcoded timeouts
grep -rn "TimeSpan.From\|new TimeSpan" src/ --include="*.cs"

# Find magic numbers used as config
grep -rn "const int\|const long\|const double" src/ --include="*.cs"
```

## Health Checks

### What to Check

Every service should have a health check endpoint or mechanism:

| Service | Health Check Method |
|---------|-------------------|
| API | `/health` endpoint (ASP.NET Health Checks) |
| Database | EF Core connection check |
| Redis | Connection ping |
| RabbitMQ | Connection status |
| Elasticsearch | Cluster health API |
| External services | HTTP ping or dedicated health endpoint |

### Docker Compose Health Checks

Every service in docker-compose should have a healthcheck:

```yaml
services:
  api:
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:3000/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 40s
```

### Common Findings

- No health check endpoint on the API
- Docker services without healthcheck configuration
- Health check that only checks "is the process running" but not "can it serve requests"
- Health check that doesn't verify external dependencies (DB, Redis, RMQ)
- Missing `start_period` causing false failures during startup

## Docker Compose Correctness

### What to Check

1. **Service definitions:** Are all documented services present?
2. **Port mappings:** Are ports correct and non-conflicting?
3. **Volume mounts:** Are data volumes persistent where needed?
4. **Dependency ordering:** Are `depends_on` with conditions set correctly?
5. **Network configuration:** Are services on the correct networks?
6. **Environment variables:** Are all required env vars set?
7. **Build context:** Are Dockerfiles referenced correctly?
8. **Resource limits:** Are memory/CPU limits set?

### Common Findings

- Service depends on another but no `depends_on` configured
- Port conflict (two services on the same port)
- Data volume not marked as persistent (lost on `docker compose down`)
- Missing restart policy (`restart: unless-stopped`)
- Build context too broad (sending entire repo to Docker daemon)
- No `.dockerignore` file (slow builds)
- Hardcoded image tags without version pinning (`latest` tag)

### How to Verify

```bash
# Validate compose file syntax
docker compose config

# Check for port conflicts
grep -n "ports:" docker-compose.yml -A 2

# Check for volume definitions
grep -n "volumes:" docker-compose.yml -A 5
```

## Environment-Specific Configs

### What to Check

1. **Development vs Production separation:** Are there different configs for each environment?
2. **appsettings hierarchy:** `appsettings.json` (base) + `appsettings.Development.json` (dev overrides) + `appsettings.Production.json` (prod overrides)
3. **Docker profiles:** Are there separate compose files or profiles for dev vs prod?
4. **Logging levels:** Debug in dev, Warning in production?
5. **CORS origins:** Permissive in dev, restricted in production?

### Common Findings

- Only one appsettings.json with no environment overrides
- Development settings (verbose logging, debug mode) left in base config
- No production compose file or profile
- Same CORS policy for all environments

## Secret Exposure Check

### What to Look For

1. **Secrets in source control:**
   ```bash
   # Check for .env files committed
   git ls-files | grep -i "\.env$"
   
   # Check for potential secrets in tracked files
   grep -rn "password\|secret\|api.key\|token" --include="*.json" --include="*.yml" --include="*.yaml"
   ```

2. **Secrets in docker-compose.yml:**
   ```yaml
   # BAD: Secret directly in compose file
   environment:
     - DB_PASSWORD=actualPassword123
   
   # CORRECT: Reference from .env
   environment:
     - DB_PASSWORD=${DB_PASSWORD}
   ```

3. **Secrets in appsettings.json:**
   ```json
   // BAD: Real connection string in tracked file
   "ConnectionStrings": {
     "Default": "Host=prod-db;Password=realPassword"
   }
   ```

4. **.gitignore coverage:**
   ```bash
   # Verify .env is gitignored
   grep "\.env" .gitignore
   
   # Verify appsettings with secrets are gitignored
   grep "appsettings.*\.json" .gitignore
   ```

### Severity

- Real secrets in source control: Red (Critical)
- Secrets in docker-compose (not using env vars): Red
- .env file not gitignored: Red
- Missing .env.example: Amber

## Review Checklist

- [ ] .env.example exists and contains all required variables
- [ ] Every variable has a description comment
- [ ] No real secrets in .env.example
- [ ] No hardcoded config values in source code
- [ ] Health check endpoints exist for all services
- [ ] Docker Compose services have healthcheck configuration
- [ ] Docker Compose is valid (docker compose config passes)
- [ ] No port conflicts
- [ ] Environment-specific configs exist (dev/prod)
- [ ] No secrets committed to source control
- [ ] .env is in .gitignore
- [ ] Docker images use specific version tags (not :latest)
- [ ] Restart policies configured
