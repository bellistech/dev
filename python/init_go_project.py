#!/usr/bin/env python3
"""init_go_project.py

Go project scaffold (modules-first) with optional Docker/Compose/Postgres bits.

Examples:
  ./init_go_project.py -n myapp
  ./init_go_project.py -n myapp -m github.com/you/myapp --features makefile,docker
  ./init_go_project.py -n myapp --features repo,git
"""

from __future__ import annotations

import argparse
import shutil
import subprocess
from pathlib import Path


def sh(cmd: list[str], cwd: Path | None = None) -> None:
    subprocess.run(cmd, cwd=str(cwd) if cwd else None, check=True)


def w(p: Path, s: str) -> None:
    p.parent.mkdir(parents=True, exist_ok=True)
    p.write_text(s, encoding="utf-8")


def main() -> int:
    ap = argparse.ArgumentParser(description="Go project scaffold")
    ap.add_argument("-n", "--name", required=True, help="Project dir name")
    ap.add_argument("-m", "--module", default="", help="Go module path (default: github.com/yourname/<name>)")
    ap.add_argument("-o", "--out", default=".", help="Parent output dir")
    ap.add_argument("-b", "--bin", default="", help="Binary name (default: name)")
    ap.add_argument("-f", "--force", action="store_true", help="Overwrite existing dir")
    ap.add_argument("--features", default="", help="comma list: makefile,docker,compose,pgx,repo,git,all")
    a = ap.parse_args()

    name = a.name.strip()
    bin_name = a.bin.strip() or name
    module = a.module.strip() or f"github.com/yourname/{name}"

    feats = {x.strip().lower() for x in a.features.split(",") if x.strip()}
    all_on = "all" in feats
    def on(k: str) -> bool: return all_on or (k in feats)

    add_make = on("makefile")
    add_docker = on("docker") or on("compose")
    add_compose = on("compose") or on("repo")
    add_pgx = on("pgx") or on("repo")
    add_repo = on("repo")
    add_git = on("git")

    target = (Path(a.out).expanduser().resolve() / name)
    if target.exists():
        if a.force:
            shutil.rmtree(target)
        else:
            raise SystemExit(f"ERROR: target exists: {target} (use -f/--force)")

    (target / f"cmd/{bin_name}").mkdir(parents=True, exist_ok=True)
    (target / "internal").mkdir(parents=True, exist_ok=True)
    (target / "pkg").mkdir(parents=True, exist_ok=True)

    sh(["go", "mod", "init", module], cwd=target)

    w(target / f"cmd/{bin_name}/main.go", f'''package main

import "log"

func main() {{
    log.Println("✅ {name} is alive")
}}
''')

    if add_pgx:
        (target / "pkg/database").mkdir(parents=True, exist_ok=True)
        sh(["go", "get", "github.com/jackc/pgx/v5/pgxpool"], cwd=target)
        w(target / "pkg/database/postgres.go", '''package database

import (
    "context"
    "fmt"
    "os"
    "time"

    "github.com/jackc/pgx/v5/pgxpool"
)

func NewPostgresPool(ctx context.Context) (*pgxpool.Pool, error) {
    dsn := os.Getenv("DATABASE_URL")
    if dsn == "" {
        return nil, fmt.Errorf("DATABASE_URL is not set")
    }
    cfg, err := pgxpool.ParseConfig(dsn)
    if err != nil { return nil, err }

    cfg.MaxConns = 10
    cfg.MinConns = 1
    cfg.MaxConnLifetime = 30 * time.Minute

    pool, err := pgxpool.NewWithConfig(ctx, cfg)
    if err != nil { return nil, err }
    if err := pool.Ping(ctx); err != nil {
        pool.Close()
        return nil, err
    }
    return pool, nil
}
''')

        w(target / f"cmd/{bin_name}/main.go", f'''package main

import (
    "context"
    "log"
    "time"

    "{module}/pkg/database"
)

func main() {{
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    pool, err := database.NewPostgresPool(ctx)
    if err != nil {{
        log.Fatalf("DB connection failed: %v", err)
    }}
    defer pool.Close()

    log.Println("✅ connected to Postgres via pgxpool")
}}
''')

    if add_repo:
        (target / "internal/model").mkdir(parents=True, exist_ok=True)
        (target / "internal/repository").mkdir(parents=True, exist_ok=True)
        (target / "init-db").mkdir(parents=True, exist_ok=True)

        w(target / "internal/model/user.go", '''package model

import "time"

type User struct {
    ID        int64
    Email     string
    CreatedAt time.Time
}
''')

        w(target / "internal/repository/user_repository.go", f'''package repository

import (
    "context"

    "{module}/internal/model"
    "github.com/jackc/pgx/v5/pgxpool"
)

type UserRepository struct {{ db *pgxpool.Pool }}

func NewUserRepository(db *pgxpool.Pool) *UserRepository {{ return &UserRepository{{db: db}} }}

func (r *UserRepository) Create(ctx context.Context, email string) (*model.User, error) {{
    const q = "INSERT INTO users (email) VALUES ($1) RETURNING id, email, created_at"
    u := &model.User{{}}
    err := r.db.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.CreatedAt)
    if err != nil {{ return nil, err }}
    return u, nil
}}
''')

        w(target / "init-db/01_init.sql", '''CREATE TABLE IF NOT EXISTS users (
  id SERIAL PRIMARY KEY,
  email TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
''')

    if add_docker:
        w(target / "Dockerfile", f'''FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/bin/{bin_name} ./cmd/{bin_name}

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /app/bin/{bin_name} .
CMD ["./{bin_name}"]
''')

    if add_compose:
        compose = f'''services:
  app:
    build: .
    environment:
      - DATABASE_URL=postgres://user:pass@db:5432/{name}?sslmode=disable
    depends_on:
      db:
        condition: service_healthy

  db:
    image: postgres:17-alpine
    environment:
      - POSTGRES_USER=user
      - POSTGRES_PASSWORD=pass
      - POSTGRES_DB={name}
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U user -d {name}"]
      interval: 5s
      timeout: 5s
      retries: 10
'''
        if add_repo:
            compose += "    volumes:\n      - ./init-db:/docker-entrypoint-initdb.d\n"
        w(target / "docker-compose.yaml", compose)

    if add_make:
        w(target / "Makefile", f'''BINARY={bin_name}

.PHONY: build run test clean docker-build up down
build:
	go build -o bin/$(BINARY) ./cmd/$(BINARY)

run:
	go run ./cmd/$(BINARY)/main.go

test:
	go test ./...

clean:
	rm -rf bin/

docker-build:
	docker build -t $(BINARY):latest .

up:
	docker compose up --build

down:
	docker compose down
''')

    w(target / "README.md", f'''# {name}

Generated with init_go_project.py

## Run
```bash
go run ./cmd/{bin_name}
```
''')

    if add_git:
        sh(["git", "init"], cwd=target)

    sh(["go", "mod", "tidy"], cwd=target)

    print(f"✅ Created: {target}")
    print(f"   Module:  {module}")
    print(f"   Binary:  {bin_name}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
