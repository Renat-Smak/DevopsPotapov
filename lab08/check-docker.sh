#!/usr/bin/env bash
set -Eeuo pipefail
lab="$(cd -- "$(dirname -- "$0")" && pwd)"
root="$(cd -- "$lab/.." && pwd)"
if [[ ! -f "$root/go.mod" ]]; then root="$root/lab06-07"; fi
test -f "$root/go.mod"
id="lr8-${GITHUB_RUN_ID:-local}-${GITHUB_RUN_ATTEMPT:-1}-$$"
label="devops.lab08=$id"
network="$id-appnet"
volume="$id-pgdata"
web="$id-web"
bind="$id-bind"
db="$id-db"
api="$id-api"
before="$id-pharmacy:before"
after="$id-pharmacy:after"
scratch="$(mktemp -d)"
export PG_LAB_PASSWORD="lab-${RANDOM}-${RANDOM}"
cleanup() {
  local ids
  ids="$(docker ps -aq --filter "label=$label")"
  if [[ -n "$ids" ]]; then printf '%s\n' "$ids" | xargs docker rm -f >/dev/null; fi
  docker volume rm "$volume" >/dev/null 2>&1 || true
  docker network rm "$network" >/dev/null 2>&1 || true
  docker image rm "$before" "$after" >/dev/null 2>&1 || true
  rm -f -- "$scratch/bind/index.html" "$scratch/create.json" "$scratch/before.log" "$scratch/after.log"
  if [[ -d "$scratch/context" ]]; then find "$scratch/context" -depth -delete; fi
  if [[ -d "$scratch/bind" ]]; then rmdir -- "$scratch/bind"; fi
  rmdir -- "$scratch"
}
trap cleanup EXIT
section() { printf '\n>>> %s\n' "$1"; }
port() { docker port "$1" "$2" | awk -F: 'NR==1{print $NF}'; }
get() { curl -fsS --max-time 4 --retry 12 --retry-delay 1 --retry-connrefused "$@"; }

section DOCKER_VERSION
docker version --format 'Client={{.Client.Version}} | Server={{.Server.Version}}'
section HELLO_WORLD
docker run --rm --label "$label" hello-world | sed -n '/Hello from Docker/,+2p'
section IMAGE
docker pull nginx:alpine
docker image ls nginx:alpine
section NGINX
docker network create --label "$label" "$network" >/dev/null
docker run -d --name "$web" --network "$network" --network-alias web --label "$label" -p 127.0.0.1::80 nginx:alpine >/dev/null
url="http://127.0.0.1:$(port "$web" 80/tcp)"
get "$url" | grep -F '<title>Welcome to nginx!</title>'
docker ps --filter "label=$label" --format 'table {{.Names}}\t{{.Status}}\t{{.Ports}}'
docker logs --tail 4 "$web" 2>&1
docker stop "$web" >/dev/null
docker start "$web" >/dev/null
docker restart "$web" >/dev/null
section EXEC
docker exec "$web" hostname
docker exec "$web" sh -c 'printf "%s\n" lab08 > /tmp/check.txt && cat /tmp/check.txt'

section BIND_MOUNT
mkdir "$scratch/bind"
echo '<h1>Lab 8 from host</h1>' > "$scratch/bind/index.html"
docker run -d --name "$bind" --label "$label" -p 127.0.0.1::80 \
  --mount "type=bind,src=$scratch/bind,dst=/usr/share/nginx/html,readonly" nginx:alpine >/dev/null
bindurl="http://127.0.0.1:$(port "$bind" 80/tcp)"
get "$bindurl"
echo '<h1>Updated on host</h1>' > "$scratch/bind/index.html"
get "$bindurl" | grep -Fx '<h1>Updated on host</h1>'

section POSTGRES_VOLUME
docker volume create --label "$label" "$volume"
start_db() {
  docker run -d --name "$db" --network "$network" --network-alias db --label "$label" \
    -e "POSTGRES_PASSWORD=$PG_LAB_PASSWORD" -e POSTGRES_DB=lab08 \
    --mount "type=volume,src=$volume,dst=/var/lib/postgresql/data" postgres:16 >/dev/null
  for ((i=0;i<45;i++)); do
    if docker exec "$db" pg_isready -h 127.0.0.1 -U postgres -d lab08 >/dev/null 2>&1; then return; fi
    sleep 2
  done
  docker logs "$db" >&2
  return 1
}
sql() { docker exec "$db" psql -v ON_ERROR_STOP=1 -U postgres -d lab08 -Atc "$1"; }
start_db
sql "CREATE TABLE lab_items (name text NOT NULL); INSERT INTO lab_items VALUES ('Paracetamol');"
printf 'Before recreation: '; sql 'SELECT name FROM lab_items;'
docker rm -f "$db" >/dev/null
start_db
printf 'After recreation: '; sql 'SELECT name FROM lab_items;'
test "$(sql 'SELECT name FROM lab_items;')" = Paracetamol
docker volume inspect "$volume" --format '{{.Name}} | driver={{.Driver}} | {{.Mountpoint}}'

section BRIDGE_NETWORK
docker run --rm --network "$network" alpine:3.23 wget -qO- http://web | grep -F '<title>Welcome to nginx!</title>'
docker network inspect "$network" --format '{{.Name}} | {{.Driver}} | {{range .Containers}}{{.Name}} {{.IPv4Address}}; {{end}}'

section BUILD_CONTEXT
mkdir "$scratch/context"
cp -R "$root/api" "$root/cmd" "$root/internal" "$root/orders" "$root/pharmacy" "$scratch/context/"
cp "$root/go.mod" "$root/go.sum" "$root/Dockerfile" "$scratch/context/"
head -c 524288 /dev/urandom > "$scratch/context/unused-sample.docx"
build() {
  if ! docker build --progress=plain -t "$1" "$scratch/context" >"$2" 2>&1; then
    tail -60 "$2" >&2
    return 1
  fi
  local step
  step="$(awk '/load build context/{print $1; exit}' "$2")"
  grep -F "$step transferring context:" "$2" | tail -1
  docker image inspect "$1" --format '{{index .RepoTags 0}} | image bytes={{.Size}}'
}
printf 'Without .dockerignore: '; build "$before" "$scratch/before.log"
cp "$root/.dockerignore" "$scratch/context/.dockerignore"
printf 'With .dockerignore: '; build "$after" "$scratch/after.log"

section PHARMACY_HTTP_DNS
docker run -d --name "$api" --network "$network" --network-alias api --label "$label" -p 127.0.0.1::8080 "$after" >/dev/null
apiurl="http://127.0.0.1:$(port "$api" 8080/tcp)"
get -H 'Content-Type: application/json' \
  -d '{"name":"Paracetamol","manufacturer":"Lab","price":12.5,"prescription":false,"quantity":3}' \
  "$apiurl/medicines" >"$scratch/create.json"
printf 'POST /medicines: '; cat "$scratch/create.json"
medicine="$(jq -er '.id' "$scratch/create.json")"
printf 'GET /medicines/{id}: '; get "$apiurl/medicines/$medicine" | jq -c '.medicine'
get -H 'Content-Type: application/json' -d '{"count":1}' "$apiurl/medicines/$medicine/sell" >/dev/null
printf 'GET after Sell: '; get "$apiurl/medicines/$medicine" | jq -ce '.medicine | select(.quantity == 2)'
docker exec "$api" nslookup db
docker network inspect "$network" --format '{{.Name}} | {{range .Containers}}{{.Name}}; {{end}}'

section DISK_BEFORE
docker system df
cleanup
trap - EXIT
section DISK_AFTER
docker system df
section DOCKER_OK
