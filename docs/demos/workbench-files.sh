#!/bin/sh
# The files workbench.md presents: a short talk served by si play
# (broker-talk/), a two-page queue dashboard for its browser windows and the
# loop that grows the talk's app.log (broker-dash/), and a home with a plain
# bash prompt for the drawer's shells (broker-home/). cleanup: removes them.
set -e
rm -rf broker-talk broker-dash broker-home
mkdir -p broker-talk/demos broker-dash broker-home

cat > broker-home/.bashrc <<'EOF'
PS1='broker-talk $ '
export BASH_SILENCE_DEPRECATION_WARNING=1
EOF
cp broker-home/.bashrc broker-home/.bash_profile

cat > broker-talk/talk.md <<'EOF'
---
title: Ship a message broker
theme: slate
runbook: runbook.json
demos: demos/
browser: Queues=http://127.0.0.1:7402/, Consumers=http://127.0.0.1:7402/consumers.html
---

# Ship a message broker

From a laptop to production in one afternoon

Notes: Today we take a message broker from a laptop to production, in one afternoon.

---

## Ship it in one afternoon

From a laptop to production, every command on stage

- Run it: one compose file, one config
- Size it: memory and disk limits
- Watch it: queues, consumers and the log
- Change it live, in front of the room

Notes: Four steps, and every one of them happens live, on this machine.

---

## Three things to get right

- Durable queues, so a restart loses nothing
- Memory limits, so a burst never takes the node down
- Consumers that keep up, watched live

Notes: Three things decide whether a broker survives its first week. [CLICK]

---

## One config file

```ini
listeners.tcp.default = 5672
vm_memory_high_watermark.absolute = 512MB
queue_master_locator = min-masters
```

Notes: All of it lives in one small config file, and we change it live.
EOF

cat > broker-talk/rabbitmq.conf <<'EOF'
# rabbitmq.conf: the broker for the orders service
listeners.tcp.default = 5672
management.tcp.port = 15672

## memory and disk
vm_memory_high_watermark.absolute = 512MB
disk_free_limit.absolute = 2GB

## queues
queue_master_locator = min-masters
consumer_timeout = 1800000
heartbeat = 30
log.console.level = info
EOF

cat > broker-talk/docker-compose.yml <<'EOF'
services:
  rabbitmq:
    image: rabbitmq:3.13-management
    ports:
      - "5672:5672"
      - "15672:15672"
    volumes:
      - ./rabbitmq.conf:/etc/rabbitmq/rabbitmq.conf:ro
      - data:/var/lib/rabbitmq
  orders:
    build: ./orders
    environment:
      AMQP_URL: amqp://guest:guest@rabbitmq:5672/
    depends_on: [rabbitmq]
volumes:
  data: {}
EOF

cat > broker-talk/runbook.json <<'EOF'
{ "name": "Broker runbook", "categories": [
  { "name": "Health", "entries": [
    { "label": "Deliveries so far", "description": "how many messages the orders service delivered", "command": "grep -c delivered app.log", "target": "bash" },
    { "label": "Last warnings", "description": "the newest warnings in the log", "command": "grep WARN app.log | tail -n 3", "target": "bash" } ] },
  { "name": "Config", "entries": [
    { "label": "Memory limit", "description": "the watermark the broker runs with", "command": "grep watermark rabbitmq.conf", "target": "bash" } ] } ] }
EOF

cat > broker-talk/demos/smoke.md <<'EOF'
# Smoke test the broker

## Is it delivering?

```bash
tail -n 3 app.log
```

## What does it listen on?

```bash
grep listeners rabbitmq.conf
```
EOF

cat > broker-dash/style.css <<'EOF'
body { margin: 0; font: 15px/1.5 system-ui, sans-serif; background: #0f1720; color: #e6edf3; }
nav { display: flex; gap: 0; background: #16212c; border-bottom: 1px solid #24323f; }
nav a { color: #9fb3c8; text-decoration: none; padding: 14px 22px; font-weight: 600; }
nav a.on { color: #fff; box-shadow: inset 0 -3px #f59e0b; }
main { padding: 22px 26px; }
h1 { font-size: 22px; margin: 0 0 16px; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: left; padding: 9px 10px; border-bottom: 1px solid #24323f; }
th { color: #9fb3c8; font-weight: 600; font-size: 13px; text-transform: uppercase; letter-spacing: .04em; }
td.n { font-variant-numeric: tabular-nums; }
.ok { color: #4ade80; } .warn { color: #fbbf24; }
EOF

cat > broker-dash/index.html <<'EOF'
<!doctype html><meta charset="utf-8"><title>Queues · orders broker</title><link rel="stylesheet" href="style.css">
<nav><a class="on" href="index.html">Queues</a><a href="consumers.html">Consumers</a></nav>
<main><h1>Queues on orders-broker</h1>
<table><tr><th>Queue</th><th>Ready</th><th>Unacked</th><th>Rate in</th><th>State</th></tr>
<tr><td>orders.created</td><td class="n">12</td><td class="n">4</td><td class="n">180/s</td><td class="ok">running</td></tr>
<tr><td>orders.paid</td><td class="n">3</td><td class="n">1</td><td class="n">95/s</td><td class="ok">running</td></tr>
<tr><td>orders.shipped</td><td class="n">240</td><td class="n">0</td><td class="n">60/s</td><td class="warn">backlog</td></tr>
<tr><td>orders.dead-letter</td><td class="n">0</td><td class="n">0</td><td class="n">0/s</td><td class="ok">idle</td></tr>
</table></main>
EOF

cat > broker-dash/consumers.html <<'EOF'
<!doctype html><meta charset="utf-8"><title>Consumers · orders broker</title><link rel="stylesheet" href="style.css">
<nav><a href="index.html">Queues</a><a class="on" href="consumers.html">Consumers</a></nav>
<main><h1>Consumers on orders-broker</h1>
<table><tr><th>Consumer</th><th>Queue</th><th>Prefetch</th><th>Acked</th></tr>
<tr><td>billing-1</td><td>orders.created</td><td class="n">50</td><td class="n">18,204</td></tr>
<tr><td>billing-2</td><td>orders.created</td><td class="n">50</td><td class="n">17,961</td></tr>
<tr><td>shipping-1</td><td>orders.paid</td><td class="n">20</td><td class="n">9,410</td></tr>
<tr><td>mailer-1</td><td>orders.shipped</td><td class="n">10</td><td class="n">2,077</td></tr>
</table></main>
EOF

cat > broker-dash/logger.sh <<'EOF'
#!/bin/sh
# app.log grows by a line a second, as a running service's log does
i=0
while :; do
  i=$((i + 1))
  if [ $((i % 7)) -eq 0 ]; then
    echo "$(date +%H:%M:%S) WARN  orders.shipped backlog $((200 + i)) messages" >> broker-talk/app.log
  else
    echo "$(date +%H:%M:%S) INFO  orders.created delivered message $((4700 + i)) to billing-$((i % 2 + 1))" >> broker-talk/app.log
  fi
  sleep 1
done
EOF
i=0
while [ $i -lt 24 ]; do
  i=$((i + 1))
  echo "09:5$((i / 10)):$((i % 10))0 INFO  orders.created delivered message $((4600 + i)) to billing-$((i % 2 + 1))" >> broker-talk/app.log
done
