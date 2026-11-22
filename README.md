# Real-Time Forum

A single-page forum built with Go, SQLite, HTML, CSS, and vanilla JavaScript. Users can register and log in, publish categorized posts, comment and react, and exchange private messages in real time.

## Features

- Account registration, login, and logout
- Post feed with categories, comments, and reactions
- Private messaging with online status and live updates over WebSockets
- Responsive interface served directly by the Go application

## Requirements

- Go 1.25.5 or later
- CGO and a C compiler when building locally (required by `go-sqlite3`)

## Run locally

```powershell
go run .
```

Open [http://localhost:8080](http://localhost:8080). The app creates `forum.db` in the project directory if it does not already exist.

Configuration is available through environment variables:

| Variable | Default | Purpose |
| --- | --- | --- |
| `PORT` | `8080` | HTTP port |
| `DATABASE_PATH` | `./forum.db` | SQLite database file |

## Deploy to Render

This repository includes a `render.yaml` Blueprint and a Dockerfile. To deploy:

1. Push this project to a new GitHub repository that you own.
2. In Render, choose **New → Blueprint**, connect that repository, and deploy the Blueprint.
3. Render builds the Docker image and creates the web service.

The Blueprint uses Render's **free** plan in Frankfurt and stores SQLite at `/tmp/forum.db` on the service's ephemeral filesystem. No persistent disk is attached, so data can be lost whenever Render restarts or redeploys the service. This is intentional for this setup. Render's free web services can also spin down after a period without traffic, so the first visit afterward may take longer while the service starts again.

Render supplies `PORT` to the service. The app also defaults to port `8080` for local use.

## Push to your GitHub

If this folder is not already a Git repository, create an empty repository on GitHub and run these commands from the project directory:

```powershell
git init
git add .
git commit -m "Initial commit"
git branch -M main
git remote add origin https://github.com/<YOUR-USERNAME>/<YOUR-REPOSITORY>.git
git push -u origin main
```

The local SQLite database, local environment files, and compiled binaries are excluded from Git and from the Docker build context.
