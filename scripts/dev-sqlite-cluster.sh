#!/usr/bin/env bash
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
BIN="${PB_BIN:-$ROOT_DIR/bin/postgrebase}"
BASE_DIR="${PB_CLUSTER_DIR:-$ROOT_DIR/pb_data/cluster-dev}"
HOST="${PB_CLUSTER_HOST:-127.0.0.1}"
DEFAULT_NODE_COUNT="${PB_CLUSTER_NODES:-3}"
PORT_BASE="${PB_CLUSTER_PORT_BASE:-8090}"
PORTS=()

usage() {
  cat <<EOF
Usage: $0 {start|stop|restart|status|logs|clean} [node_count]

Environment:
  PB_BIN             PostgreBase binary path (default: $ROOT_DIR/bin/postgrebase)
  PB_CLUSTER_DIR     Runtime dir for data/logs/pids (default: $ROOT_DIR/pb_data/cluster-dev)
  PB_CLUSTER_HOST    Host used for --http and peers (default: 127.0.0.1)
  PB_CLUSTER_NODES   Default node count when [node_count] is omitted (default: 3)
  PB_CLUSTER_PORT_BASE First generated port when PB_CLUSTER_PORTS is omitted (default: 8090)
  PB_CLUSTER_PORTS   Space-separated ports. Overrides generated ports.

Examples:
  $0 start
  $0 start 5
  $0 status
  $0 status 5
  $0 logs
  $0 stop
  PB_BIN=./bin/postgrebase PB_CLUSTER_PORTS="8090 8091" $0 start
EOF
}

is_positive_int() {
  [[ "${1:-}" =~ ^[1-9][0-9]*$ ]]
}

configure_ports() {
  local count="${1:-$DEFAULT_NODE_COUNT}"
  if ! is_positive_int "$count"; then
    echo "node_count must be a positive integer, got: $count" >&2
    exit 1
  fi

  if [[ -n "${PB_CLUSTER_PORTS:-}" ]]; then
    read -r -a PORTS <<<"$PB_CLUSTER_PORTS"
    if [[ "${#PORTS[@]}" -ne "$count" ]]; then
      echo "PB_CLUSTER_PORTS has ${#PORTS[@]} ports but node_count is $count" >&2
      exit 1
    fi
    return
  fi

  PORTS=()
  local i
  for ((i = 0; i < count; i++)); do
    PORTS+=("$((PORT_BASE + i))")
  done
}

node_name() {
  echo "node-$1"
}

pid_file() {
  echo "$BASE_DIR/pids/$(node_name "$1").pid"
}

log_file() {
  echo "$BASE_DIR/logs/$(node_name "$1").log"
}

node_dir() {
  echo "$BASE_DIR/data/$(node_name "$1")"
}

node_addr() {
  echo "http://$HOST:${PORTS[$1]}"
}

peers_for() {
  local idx="$1"
  local peers=()
  local i
  for i in "${!PORTS[@]}"; do
    if [[ "$i" != "$idx" ]]; then
      peers+=("$(node_addr "$i")")
    fi
  done
  local IFS=,
  echo "${peers[*]}"
}

is_running() {
  local pid="$1"
  [[ -n "$pid" ]] && kill -0 "$pid" >/dev/null 2>&1
}

ensure_binary() {
  if [[ ! -x "$BIN" ]]; then
    echo "PostgreBase binary not found or not executable: $BIN" >&2
    echo "Build it first, for example: make build" >&2
    exit 1
  fi
}

start_node() {
  local idx="$1"
  local pid_path
  pid_path="$(pid_file "$idx")"

  if [[ -f "$pid_path" ]]; then
    local old_pid
    old_pid="$(cat "$pid_path" 2>/dev/null || true)"
    if is_running "$old_pid"; then
      echo "$(node_name "$idx") already running: $old_pid"
      return
    fi
  fi

  local dir db log peers addr
  dir="$(node_dir "$idx")"
  db="$dir/data.db"
  log="$(log_file "$idx")"
  peers="$(peers_for "$idx")"
  addr="$(node_addr "$idx")"

  mkdir -p "$dir" "$(dirname "$log")" "$(dirname "$pid_path")"

  "$BIN" serve \
    --dir="$dir/files" \
    --dataDsn="sqlite://$db" \
    --http="$HOST:${PORTS[$idx]}" \
    --node-id="$(node_name "$idx")" \
    --node-addr="$addr" \
    --peers="$peers" \
    >"$log" 2>&1 &

  echo "$!" >"$pid_path"
  echo "started $(node_name "$idx") pid=$! addr=$addr log=$log"
}

start_cluster() {
  ensure_binary
  local i
  for i in "${!PORTS[@]}"; do
    start_node "$i"
  done
  echo
  echo "Admin UI:"
  for i in "${!PORTS[@]}"; do
    echo "  $(node_name "$i"): $(node_addr "$i")/_/"
  done
}

stop_node() {
  local idx="$1"
  local pid_path pid
  pid_path="$(pid_file "$idx")"
  if [[ ! -f "$pid_path" ]]; then
    echo "$(node_name "$idx") not running"
    return
  fi
  pid="$(cat "$pid_path" 2>/dev/null || true)"
  if is_running "$pid"; then
    kill "$pid" >/dev/null 2>&1 || true
    local waited=0
    while is_running "$pid" && [[ "$waited" -lt 30 ]]; do
      sleep 0.2
      waited=$((waited + 1))
    done
    if is_running "$pid"; then
      kill -9 "$pid" >/dev/null 2>&1 || true
    fi
    echo "stopped $(node_name "$idx") pid=$pid"
  else
    echo "$(node_name "$idx") stale pid=$pid"
  fi
  rm -f "$pid_path"
}

stop_cluster() {
  local i
  for i in "${!PORTS[@]}"; do
    stop_node "$i"
  done
}

status_cluster() {
  local i
  for i in "${!PORTS[@]}"; do
    local pid_path pid state
    pid_path="$(pid_file "$i")"
    pid=""
    state="stopped"
    if [[ -f "$pid_path" ]]; then
      pid="$(cat "$pid_path" 2>/dev/null || true)"
      if is_running "$pid"; then
        state="running"
      else
        state="stale"
      fi
    fi
    printf "%-8s %-8s pid=%-8s addr=%s log=%s\n" "$(node_name "$i")" "$state" "${pid:-"-"}" "$(node_addr "$i")" "$(log_file "$i")"
  done
}

logs_cluster() {
  mkdir -p "$BASE_DIR/logs"
  local files=()
  local i
  for i in "${!PORTS[@]}"; do
    if [[ -f "$(log_file "$i")" ]]; then
      files+=("$(log_file "$i")")
    fi
  done
  if [[ "${#files[@]}" -eq 0 ]]; then
    echo "no log files found under $BASE_DIR/logs"
    return
  fi
  tail -n 80 -f "${files[@]}"
}

clean_cluster() {
  stop_cluster
  rm -rf "$BASE_DIR"
  echo "removed $BASE_DIR"
}

cmd="${1:-}"
node_count="${2:-$DEFAULT_NODE_COUNT}"
case "$cmd" in
  start)
    configure_ports "$node_count"
    start_cluster
    ;;
  stop)
    configure_ports "$node_count"
    stop_cluster
    ;;
  restart)
    configure_ports "$node_count"
    stop_cluster
    start_cluster
    ;;
  status)
    configure_ports "$node_count"
    status_cluster
    ;;
  logs)
    configure_ports "$node_count"
    logs_cluster
    ;;
  clean)
    configure_ports "$node_count"
    clean_cluster
    ;;
  -h|--help|help|"")
    usage
    ;;
  *)
    usage >&2
    exit 1
    ;;
esac
