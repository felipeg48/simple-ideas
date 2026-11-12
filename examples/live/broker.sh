#!/bin/sh
# A stand-in for the orders broker the talk shows live: it says when it is
# ready, then reports how many orders wait in .live/orders every few seconds.
trap 'echo "broker stopped"; exit 0' TERM INT
echo "broker ready: $(cat .live/orders) order(s) waiting"
while :; do
  sleep 3
  echo "$(date +%H:%M:%S) $(cat .live/orders 2>/dev/null || echo 0) order(s) waiting"
done
