#!/bin/sh
# checkout against a provider that misbehaves on purpose
set -e
docker compose up --build --wait
curl -s -X POST localhost:8082/orders/4711/pay
docker compose logs payments-fake | grep -c dropped
