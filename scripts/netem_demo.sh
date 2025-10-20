#!/usr/bin/env bash
# Usage: sudo bash scripts/netem_demo.sh up|down [delay_ms] [rate_mbit] [loss_pct]
set -euo pipefail
CMD="${1:-up}"
DELAY="${2:-40}"   # one-way delay (ms)
RATE="${3:-50}"    # rate limit (Mbit/s)
LOSS="${4:-0.5}"   # loss (%)

nsadd(){ ip netns add "$1" 2>/dev/null || true; }
nsdel(){ ip netns del "$1" 2>/dev/null || true; }
nsexec(){ ip netns exec "$@"; }

if [ "$CMD" = "up" ]; then
  # create namespaces
  nsadd client; nsadd edge1
  # veth pair
  ip link add veth-c type veth peer name veth-e || true
  ip link set veth-c netns client
  ip link set veth-e netns edge1
  # addresses
  nsexec client ip addr add 10.10.0.1/24 dev veth-c
  nsexec edge1  ip addr add 10.10.0.2/24 dev veth-e
  nsexec client ip link set lo up
  nsexec edge1  ip link set lo up
  nsexec client ip link set veth-c up
  nsexec edge1  ip link set veth-e up
  # routing
  nsexec client ip route add 10.10.0.0/24 dev veth-c || true
  nsexec edge1  ip route add 10.10.0.0/24 dev veth-e || true
  # netem: apply both directions
  nsexec client tc qdisc add dev veth-c root handle 1: netem delay ${DELAY}ms loss ${LOSS}%
  nsexec client tc qdisc add dev veth-c parent 1:1 handle 10: tbf rate ${RATE}mbit burst 32kbit latency 400ms
  nsexec edge1  tc qdisc add dev veth-e root handle 1: netem delay ${DELAY}ms loss ${LOSS}%
  nsexec edge1  tc qdisc add dev veth-e parent 1:1 handle 10: tbf rate ${RATE}mbit burst 32kbit latency 400ms
  echo "netem up: delay=${DELAY}ms one-way, rate=${RATE}mbit, loss=${LOSS}%  (client<->edge1)"
  echo "test: sudo ip netns exec client ping -c 3 10.10.0.2"
elif [ "$CMD" = "down" ]; then
  # cleanup qdiscs (best effort)
  for ns in client edge1; do
    ip netns exec "$ns" tc qdisc del dev veth-c root 2>/dev/null || true
    ip netns exec "$ns" tc qdisc del dev veth-e root 2>/dev/null || true
  done
  # delete namespaces (removes links)
  nsdel client; nsdel edge1
  echo "netem down: namespaces removed"
else
  echo "usage: sudo bash scripts/netem_demo.sh up|down [delay_ms] [rate_mbit] [loss_pct]"
  exit 1
fi
