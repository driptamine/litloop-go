#!/bin/bash
set -euo pipefail
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$project_dir"
export PATH="$project_dir/bin:$PATH"
mkdir -p var/log var/run var/media
if [[ ! -f .env ]]; then
  echo 'Missing .env. Configure PostgreSQL and run migrations first.' >&2
  exit 1
fi
if [[ ! -x bin/detach ]]; then
  go build -o bin/detach ./scripts/detach.go
fi
for service in server worker; do
  binary_name="$service"
  if [[ "$service" == server ]]; then
    binary_name=litloop-go-server
  fi
  if [[ ! -x "bin/$binary_name" ]]; then
    go build -o "bin/$binary_name" "./cmd/$binary_name"
  fi
  pid_file="var/run/$service.pid"
  if [[ -f "$pid_file" ]]; then
    service_pid="$(cat "$pid_file")"
    if [[ "$service_pid" =~ ^[0-9]+$ ]] && kill -0 "$service_pid" 2>/dev/null; then
      process_command="$(ps -p "$service_pid" -o command=)"
      if [[ "$process_command" == "$project_dir/bin/$binary_name"* ]]; then
        echo "$service already running (PID $service_pid)"
        continue
      fi
    fi
  fi
  "$project_dir/bin/detach" "$project_dir/bin/$binary_name" "$project_dir/var/log/$service.log" >"$pid_file"
  echo "$service started (PID $(cat "$pid_file"))"
done
