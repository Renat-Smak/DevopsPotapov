#!/usr/bin/env bash
set -Eeuo pipefail
cd -- "$(dirname -- "$0")"
project="lr10-${GITHUB_RUN_ID:-local}-$$"
trial="$project-race"
scratch="$(mktemp -d)"
settings="$scratch/compose.env"
port="${LAB10_HTTP_PORT:-18080}"
new_port="${LAB10_ALT_HTTP_PORT:-18081}"
grpc_port="${LAB10_GRPC_PORT:-19090}"
password="$(openssl rand -hex 12):test@lab"
image="$project:test"

write_env() {
  printf 'POSTGRES_USER=pharmacy\nPOSTGRES_PASSWORD=%s\nPOSTGRES_DB=pharmacy\nAPP_PORT=%s\nGRPC_PORT=%s\nAPP_IMAGE=%s\n' \
    "$password" "$1" "$grpc_port" "$image" > "$settings"
}
write_env "$port"
chmod 0600 "$settings"
c() { docker compose --env-file "$settings" --project-name "$project" -f compose.yaml "$@"; }
ok() { docker compose --env-file "$settings" --project-name "$trial" -f compose.yaml "$@"; }
race() { docker compose --env-file "$settings" --project-name "$trial" -f compose.yaml -f compose.failure.yaml "$@"; }
cleanup() {
  c down -v --remove-orphans >/dev/null 2>&1 || true
  ok down -v --remove-orphans >/dev/null 2>&1 || true
  docker image rm "$image" >/dev/null 2>&1 || true
  rm -f -- "$settings"
  rmdir -- "$scratch"
}
trap cleanup EXIT
section() { printf '\n>>> %s\n' "$1"; }
query() { c exec -T db psql -U pharmacy -d pharmacy -At -v ON_ERROR_STOP=1 -c "$1"; }
request() { curl -fsS --max-time 5 "$@"; }
code() { curl -sS --max-time 5 -o /dev/null -w '%{http_code}' "$1"; }
url="http://127.0.0.1:$port"
seed='00000000-0000-4000-8000-000000000010'
volume="${project}_pgdata"
network="${project}_backend"

section COMPOSE_UP
docker compose version
c up -d --build --wait --wait-timeout 100
c ps -a
test "$(code "$url/health")" = 200
section MIGRATE_SEED
c logs --no-color --no-log-prefix migrate
printf 'schema_migrations (version|dirty): '; query 'SELECT version, dirty FROM schema_migrations;'
test "$(query 'SELECT version, dirty FROM schema_migrations;')" = '2|f'
query "SELECT name, quantity FROM medicines ORDER BY id;"
test "$(query 'SELECT count(*) FROM medicines;')" = 2
printf 'GET seed: '; request "$url/medicines/$seed"
request "$url/medicines/$seed" | jq -e '.medicine.name == "Paracetamol" and .medicine.quantity == 20' >/dev/null

section API_HTTP
created="$(request -H 'Content-Type: application/json' -d \
  '{"name":"Amoxicillin","manufacturer":"Lab","price":24.50,"prescription":true,"quantity":5}' "$url/medicines")"
printf 'POST /medicines: %s\n' "$created"
id="$(jq -er '.id' <<<"$created")"
read_quantity() {
  local response
  response="$(request "$url/medicines/$id")"
  jq -e --argjson quantity "$1" '.medicine.quantity == $quantity' <<<"$response" >/dev/null
  printf 'GET /medicines/{id}: %s\n' "$response"
}
read_quantity 5
printf 'POST /medicines/{id}/sell: '
request -H 'Content-Type: application/json' -d '{"count":2}' "$url/medicines/$id/sell"
read_quantity 3
http="$(curl -sS --max-time 5 -o /dev/null -w '%{http_code}' -H 'Content-Type: application/json' \
  -d '{"count":9}' "$url/medicines/$id/sell")"
printf 'Sell over stock: HTTP %s\n' "$http"
test "$http" = 409

section DEBUG
if timeout 2s docker compose --env-file "$settings" --project-name "$project" -f compose.yaml logs -f --no-color --tail 8 app; then
  :
else
  test "$?" = 124
fi
c exec -T app sh -c 'printf "exec app sh: uid="; id -u; printf "database host=%s\n" "$DB_HOST"'
c config --format json | jq -c '{
    ports: [.services.app.ports[] | {published, target, host_ip}],
    depends_on: .services.app.depends_on | map_values(.condition),
    volume: (.volumes|keys), network: (.networks|keys),
    environment_keys: (.services.app.environment|keys)
  }'

section STOP_START
app_id="$(c ps -q app)"
c stop app
c ps -a app
test "$(docker inspect --format '{{.State.Status}}' "$app_id")" = exited
c start --wait --wait-timeout 35 app
test "$app_id" = "$(c ps -q app)"
printf 'Same app container after start: %.12s\n' "$app_id"

section ENV_PORT
printf 'Old HTTP port: %s\n' "$port"
old_image="$(docker inspect --format '{{.Image}}' "$(c ps -q app)")"
write_env "$new_port"
c up -d --no-build --wait --wait-timeout 60
url="http://127.0.0.1:$new_port"
c port app 8080
printf 'New HTTP port: %s (GET /health = %s)\n' "$new_port" "$(code "$url/health")"
test "$(code "$url/health")" = 200
test "$(docker inspect --format '{{.Image}}' "$(c ps -q app)")" = "$old_image"
printf 'Application image ID unchanged\n'

section DOWN_KEEP
c down
test -z "$(c ps -aq)"
docker volume inspect "$volume" --format 'Volume kept: {{.Name}}'
if docker network inspect "$network" >/dev/null 2>&1; then exit 1; fi
printf 'Compose containers and network removed\n'
c up -d --wait --wait-timeout 80
printf 'After down/up: '; read_quantity 3

section DOWN_REMOVE
c down -v
if docker volume inspect "$volume" >/dev/null 2>&1; then exit 1; fi
printf 'Volume removed: %s\n' "$volume"
c up -d --wait --wait-timeout 80
missing="$(code "$url/medicines/$id")"
seed_status="$(code "$url/medicines/$seed")"
printf 'After fresh up: custom row HTTP %s; seed row HTTP %s\n' "$missing" "$seed_status"
test "$missing" = 404
test "$seed_status" = 200
test "$(query 'SELECT count(*) FROM medicines;')" = 2
c down -v

section DB_RACE
race up -d --no-build db app
cid="$(race ps -a -q app)"
test -n "$cid"
for ((i=0;i<25;i++)); do
  if [[ "$(docker inspect --format '{{.State.Status}}' "$cid")" == exited ]]; then break; fi
  sleep 1
done
test "$(docker inspect --format '{{.State.ExitCode}}' "$cid")" = 1
race logs --no-color app | grep -F 'cannot connect to db'
printf 'App without readiness guard exited with status 1\n'
race down -v

section DB_RECOVERY
ok up -d --no-build --wait --wait-timeout 80
ok ps -a
test "$(code "$url/health")" = 200
printf 'With readiness restored: /health HTTP 200\n'
section LAB10_OK
