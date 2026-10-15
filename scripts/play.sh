#!/bin/sh
# make play: a local seq server on a throwaway PostgreSQL, and this TUI on it.
#
# Needs: docker (OrbStack's), and access to the private seq module (go fetches
# it over ssh; set GOPRIVATE=github.com/GeneStevens/* if it does not).
# Nothing here is a secret: the database is a tmpfs container on 127.0.0.1, the
# certificate is made in memory, the token is for an account that lives as long
# as the server does.
#
#   SEQ_PG_PORT=55432   host port of the database
#   SEQ_ADDR=127.0.0.1:8443   where the server listens
#   SEQ_CHARACTER=me-1  the character to play (me-1 or me-2)
set -eu

cd "$(dirname "$0")/.."
port="${SEQ_PG_PORT:-55432}"
addr="${SEQ_ADDR:-127.0.0.1:8443}"
character="${SEQ_CHARACTER:-me-1}"
seqdir=$(go list -m -f '{{.Dir}}' github.com/GeneStevens/seq)
seqver=$(go list -m -f '{{.Version}}' github.com/GeneStevens/seq)
mkdir -p tmp
log=tmp/seqd.log
export SEQ_PG_PORT="$port"

cleanup() {
	[ -n "${seqd_pid:-}" ] && kill "$seqd_pid" 2>/dev/null || true
	docker compose -p seq-tui-play -f "$seqdir/deploy/docker-compose.yml" down >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

echo "starting PostgreSQL on 127.0.0.1:$port ..."
docker compose -p seq-tui-play -f "$seqdir/deploy/docker-compose.yml" up -d --wait >/dev/null

echo "building seqd $seqver ..."
GOBIN="$PWD/tmp" go install "github.com/GeneStevens/seq/cmd/seqd@$seqver"

SEQ_DATABASE_URL="postgres://seq:seq@127.0.0.1:$port/seq?sslmode=disable" \
	tmp/seqd -listen "$addr" -dev-account me >"$log" 2>&1 &
seqd_pid=$!

token=""
for _ in $(seq 1 100); do
	token=$(sed -n 's/^development token for me: //p' "$log")
	[ -n "$token" ] && break
	kill -0 "$seqd_pid" 2>/dev/null || { cat "$log"; exit 1; }
	sleep 0.2
done
[ -n "$token" ] || { echo "seqd did not start; see $log"; exit 1; }

go build -o tmp/seq-tui ./cmd/seq-tui
echo "playing $character on $addr (? for keys, q to quit) ..."
SEQ_SESSION_TOKEN="$token" tmp/seq-tui -insecure -url "wss://$addr/seq/v1" -character "$character"
