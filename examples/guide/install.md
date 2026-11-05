---
title: Install the lab
description: Start the five services and the two fakes, and check that each one answers.
---

# Install the lab

| | |
|---|---|
| **Time** | 10 minutes (about 3 with the images already pulled) |
| **You need** | Docker 24 or later, ports 8080 to 8085 free |

## Step 1: Get the lab files

```bash
git clone https://github.com/northwind-example/order-journey-lab.git
cd order-journey-lab
```

## Step 2: Start the stack

```bash
docker compose up -d --wait
```

You should see seven containers reported as `Healthy`. The first start pulls
about 600 MB of images.

> [!TIP]
> Pull the images before a workshop with `docker compose pull`: on a shared
> conference network the first start can take ten minutes.

## Step 3: Check every station

```bash
for p in 8080 8081 8082 8083 8084 8085; do
  curl -s -o /dev/null -w "$p %{http_code}\n" localhost:$p/health
done
```

Every line should end in `200`. A line with `000` means the service is still
starting: wait ten seconds and run the loop again.

Next, [place the first order](happy/place.md).
