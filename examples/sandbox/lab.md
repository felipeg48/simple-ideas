---
type: lab
title: Put a Proxy in Front of the Shop
author: Ada Lindqvist · Platform
date: Northwind Logistics · hands-on lab
theme: studio
sandbox: docker:28-dind
sandbox-files: env
sandbox-setup: docker compose pull -q
sandbox-cleanup: docker compose down
sandbox-ports: 8080 Shop
sandbox-limits: memory=2g cpus=2
---

# Put a Proxy in Front of the Shop

The shop front runs as one web server. Before the holiday rush it needs a
proxy in front, so we can add web servers without anyone noticing. You do it
in your own sandbox: a machine with Docker inside it, the lab's files already
in `/lab`.

| | |
|---|---|
| **Time** | 15 minutes |
| **Files** | `compose.yaml`, `proxy.conf`, `site/` |
| **Shop** | [the shop front](sandbox:8080), once it runs |

## Start the shop

Bring up the web server and the proxy:

```bash
docker compose up -d
```

<!-- check: docker compose ps --services --status running · expect "proxy" -->

## Look at it

Open [the shop front](sandbox:8080). Every answer goes through the proxy,
which says which web server served it:

```bash
curl -sI http://localhost:8080 | grep X-Served-By
```

<!-- check: wget -qO- http://localhost:8080 · expect "Northwind shop front" -->

## Add two web servers

Three web servers share the work; the proxy finds them by name:

```bash
docker compose up -d --scale web=3
docker compose restart proxy
```

<!-- check: docker compose ps -q web | wc -l · expect "3" -->

## What changed

The shop never went down. The proxy is the one address the world knows, and
the web servers behind it can come and go.
