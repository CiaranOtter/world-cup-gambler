# World Cup Gambler ⚽

Live World Cup 2026 scores + a sweepstake game where players draw a random team
and compete on a combined leaderboard (points × 3 + goals scored + goal difference + knockout wins × 5).

Data is pulled from [worldcup26.ir](https://worldcup26.ir) and cached in-memory,
refreshing every 30 seconds.

---

## Running with Docker (recommended)

### Prerequisites
- Docker + Docker Compose installed on your server

### 1. Get the files onto your server

```bash
# Clone / copy your project, then cd into it
cd world_cup_gambler
```

### 2. Build and start

```bash
docker compose up -d --build
```

The app will be available at `http://your-server:8080`.

### 3. Persist user data

Player profiles are stored in `/app/data/users.json` inside the container.
The `docker-compose.yml` mounts this as a named volume (`worldcup_data`) so
data survives restarts and image rebuilds automatically.

### 4. Optional: API token

If you register at worldcup26.ir and get a JWT token:

```bash
cp .env.example .env
# Edit .env and set WORLDCUP_API_TOKEN=your_token_here
docker compose up -d --build
```

---

## Useful commands

```bash
# View logs
docker compose logs -f

# Stop
docker compose down

# Rebuild after code changes
docker compose up -d --build

# Back up user data
docker run --rm -v worldcup_gambler_worldcup_data:/data \
  alpine tar czf - /data > users_backup.tar.gz
```

---

## Running behind a reverse proxy (nginx / Caddy)

### Caddy (automatic HTTPS)
```
your-domain.com {
    reverse_proxy localhost:8080
}
```

### nginx
```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass         http://localhost:8080;
        proxy_set_header   Host $host;
        proxy_set_header   X-Real-IP $remote_addr;
    }
}
```

---

## API endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/matches` | All matches (supports `?group=A`, `?matchday=1`, `?finished=true`) |
| GET | `/api/matches/{id}` | Single match |
| GET | `/api/teams` | All 48 teams |
| GET | `/api/stadiums` | All stadiums |
| GET | `/api/groups` | Group standings |
| GET | `/api/leaderboard` | Ranked player leaderboard |
| GET | `/api/status` | Cache freshness / health check |
| POST | `/api/users` | Register `{"name": "..."}` → assigned a random team |
| GET | `/api/users/{id}` | Get player profile |
| POST | `/api/users/{id}/reroll` | Re-roll team (once per player) |

---

## Project structure

```
.
├── Dockerfile
├── docker-compose.yml
├── .env.example
├── go.mod
├── main.go
├── models.go
├── cache.go
├── worldcup_client.go
├── handlers_matches.go
├── handlers_users.go
├── store.go
├── leaderboard.go
└── static/
    ├── index.html
    ├── css/style.css
    └── js/app.js
```