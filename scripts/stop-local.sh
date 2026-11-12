#!/bin/bash
set -euo pipefail
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$project_dir"
for service in server worker; do
  binary_name="$service"
  if [[ "$service" == server ]]; then
    binary_name=litloop-go-server
  fi
  pid_file="var/run/$service.pid"
  [[ -f "$pid_file" ]] || continue
  service_pid="$(cat "$pid_file")"
  if [[ "$service_pid" =~ ^[0-9]+$ ]] && kill -0 "$service_pid" 2>/dev/null; then
    process_command="$(ps -p "$service_pid" -o command=)"
    if [[ "$process_command" == "$project_dir/bin/$binary_name"* ]]; then
      kill "$service_pid"
      echo "$service stopped (PID $service_pid)"
    else
      echo "PID $service_pid belongs to another process; preserved." >&2
    fi
  fi
  rm -- "$pid_file"
done
