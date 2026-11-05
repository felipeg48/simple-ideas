---
title: Introduction
description: What you will build, what you need, and how the guide is laid out.
---

# Introduction

Order #4711 was placed at 21:03 on a Monday and delivered on Thursday at
14:32, as promised. On the way, its card was declined once and the warehouse
fell behind once. This guide runs the same journey on your laptop: five
services in Docker, one fake bank, one fake courier, and you.

| | |
|---|---|
| **Time** | About an hour, in six pages |
| **You need** | A laptop with Docker and 4 GB of free memory |
| **You will see** | One order placed, paid, picked, packed and delivered; then a declined card and a backlog, and how each recovers |

## How the guide is laid out

- **Getting started** installs the lab and checks it: [Install the lab](install.md).
- **The happy path** sends order #4711 through every station.
- **When it breaks** makes the bank slow and the warehouse busy, the two
  failures the real order met.
- **Reset the lab** puts everything back, so you can start again.

Each page ends with the next one. Your progress is kept in this browser.

## The stations

| Station | Service | Port |
|---|---|---|
| Checkout | `shop-checkout` | 8080 |
| Payment | `shop-payments` (talks to `fake-bank` on 8081) | 8082 |
| Warehouse | `shop-warehouse` | 8083 |
| Courier | `fake-courier` | 8084 |
| Tracking | `shop-tracking`, the page the customer sees | 8085 |
