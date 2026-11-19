#!/bin/sh
set -eu
encode() {
    LC_ALL=C printf '%s' "$1" | od -An -v -tx1 | tr -d ' \n' | sed 's/../%&/g'
}
user="$(encode "$DB_USER")"
password="$(encode "$DB_PASSWORD")"
name="$(encode "$DB_NAME")"
exec migrate -path /migrations \
    -database "postgres://$user:$password@$DB_HOST:$DB_PORT/$name?sslmode=disable" up
