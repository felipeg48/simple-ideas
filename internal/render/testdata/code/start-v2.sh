#!/bin/sh
set -e
docker compose pull
docker compose up -d --wait
