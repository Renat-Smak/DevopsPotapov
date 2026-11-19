#!/usr/bin/env bash
set -Eeuo pipefail
cd -- "$(dirname -- "$0")"
id="lr8-cron-${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}-$$"
volume="$id-data"
image="$id:lab"
first="$id-first"
second="$id-second"
owned=()
cleanup() {
  for container in "${owned[@]}"; do docker rm -f "$container" >/dev/null 2>&1 || true; done
  docker volume rm "$volume" >/dev/null 2>&1 || true
  docker image rm "$image" >/dev/null 2>&1 || true
}
trap cleanup EXIT

wait_for_lines() {
  local container="$1" target="$2" count=0
  local deadline=$((SECONDS + 95))
  while (( SECONDS < deadline )); do
    count="$(docker exec "$container" sh -c 'if [ -f /data/dates.log ]; then wc -l < /data/dates.log; else echo 0; fi')"
    if (( count >= target )); then
      return
    fi
    sleep 2
  done
  docker logs "$container" >&2
  printf 'Timed out waiting for %s lines on %s (last count %s)\n' "$target" "$container" "$count" >&2
  return 1
}

printf '\n>>> CRON_SETUP\n'
docker build --quiet -t "$image" ./cron
docker volume create "$volume"
docker run -d --name "$first" --mount "type=volume,src=$volume,dst=/data" "$image"
owned+=("$first")
docker exec "$first" crontab -l

wait_for_lines "$first" 1
printf '\n>>> CRON_FIRST\n'
docker exec "$first" cat /data/dates.log

wait_for_lines "$first" 2
printf '\n>>> CRON_GROWTH\n'
before="$(docker exec "$first" cat /data/dates.log)"
printf '%s\n' "$before"
printf '%s\n' "$before" | awk 'NR==1{first=$0} NR==2 && $0<=first{exit 1} END{if(NR<2)exit 1}'

docker rm -f "$first" >/dev/null
docker run -d --name "$second" --mount "type=volume,src=$volume,dst=/data" "$image" >/dev/null
owned+=("$second")
printf '\n>>> CRON_PERSISTENCE\n'
count="$(printf '%s\n' "$before" | wc -l)"
docker exec "$second" sh -c 'head -n "$1" /data/dates.log' sh "$count"
test "$(docker exec "$second" sh -c 'head -n "$1" /data/dates.log' sh "$count")" = "$before"
wait_for_lines "$second" "$((count + 1))"
printf 'Lines after recreation: '
docker exec "$second" sh -c 'wc -l < /data/dates.log'
docker volume inspect "$volume" --format '{{.Name}} | driver={{.Driver}} | {{.Mountpoint}}'
printf '\n>>> CRON_OK\n'
