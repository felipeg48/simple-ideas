---
type: demo
title: {{title}}
session: 1
slides: 1-3
duration: 6
---

## What this demo must accomplish

The one thing the room should see work.

## Before the session (T-30 min)

```bash
docker compose up -d --wait
```

## The script

### Slide 2, "The problem" (3 min)

> Say: "Let me show you what happens when the bank is slow."

```bash
curl localhost:8080/health
```

```text
{"status":"ok"}
```

### Slide 3, "The fix" (3 min)

> Say: "Now the same request, with the fix in place."

## If it goes wrong

## Reset between rehearsals
