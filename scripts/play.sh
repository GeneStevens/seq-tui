#!/bin/sh
# make play: a local seq server on a throwaway PostgreSQL, and this TUI on it.
#
# Needs: docker (OrbStack's), and access to the private seq module. The script
# downloads the module itself; if that fails it says what the one-time setup is
# (GOPRIVATE, and git over ssh; the README has the same text).
# Nothing here is a secret: the database is a tmpfs container on 127.0.0.1, the
# certificate is made in memory, the token is for an account that lives as long
# as the server does.
#
#   SEQ_PG_PORT=55432   host port of the database
#   SEQ_ADDR=127.0.0.1:8443   where the server listens
#   SEQ_CHARACTER=me-1  the character to play (me-1 or me-2)
#   SEQ_NEW=1           make a new character on the creation screen instead
#   SEQ_WORLD_PACKS=DIR  serve and draw the packs in DIR (a seq-world checkout under work) instead of the pinned seq-world's
set -eu

cd "$(dirname "$0")/.."
port="${SEQ_PG_PORT:-55432}"
addr="${SEQ_ADDR:-127.0.0.1:8443}"
character="${SEQ_CHARACTER:-me-1}"
module=github.com/GeneStevens/seq

setup_help() {
	cat >&2 <<HELP

make play: could not download the private module $module.

One-time setup (the module is private, so go must skip the public proxy and
fetch it with git over ssh, with a key GitHub accepts):

  go env -w GOPRIVATE='github.com/GeneStevens/*'
  git config --global url."git@github.com:".insteadOf "https://github.com/"
  ssh -T git@github.com            # should greet you by name

Then run make play again.
HELP
}

fetch() { go mod download "$module" github.com/GeneStevens/seq-world >"$dlog" 2>&1; }

dlog=$(mktemp)
if ! fetch; then
	# Retry once as the one-time setup would have it, for this run only.
	case ",${GOPRIVATE:-}," in
	*"github.com/GeneStevens"*) ;;
	*) export GOPRIVATE="${GOPRIVATE:+$GOPRIVATE,}github.com/GeneStevens/*" ;;
	esac
	export GIT_CONFIG_COUNT=1 GIT_CONFIG_KEY_0='url.ssh://git@github.com/.insteadOf' GIT_CONFIG_VALUE_0='https://github.com/'
	export GIT_SSH_COMMAND="${GIT_SSH_COMMAND:-ssh -o BatchMode=yes}"
	if ! fetch; then
		cat "$dlog" >&2
		rm -f "$dlog"
		setup_help
		exit 1
	fi
fi
rm -f "$dlog"
seqdir=$(go list -m -f '{{.Dir}}' "$module")
seqver=$(go list -m -f '{{.Version}}' "$module")
if [ -z "$seqdir" ] || [ ! -f "$seqdir/deploy/docker-compose.yml" ]; then
	echo "make play: the $module module is not where go says it is (dir '$seqdir'); try: go mod download $module" >&2
	exit 1
fi
if ! docker info >/dev/null 2>&1; then
	echo "make play: docker is not running (start OrbStack); the server's database runs in a container" >&2
	exit 1
fi
# The world (D54) is seq-world, at the version go.mod pins: the server serves its
# packs, and the TUI reads the zones' areas and patrol paths from the same ones
# (the protocol does not carry them yet).
world=$(go list -m -f '{{.Dir}}' github.com/GeneStevens/seq-world)
packs="${SEQ_WORLD_PACKS:-$world/packs}"
if [ ! -d "$packs" ]; then
	echo "make play: seq-world's packs are not at '$packs'; try: go mod download github.com/GeneStevens/seq-world" >&2
	exit 1
fi
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
	tmp/seqd -listen "$addr" -packs "$packs" -dev-account me >"$log" 2>&1 &
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
if [ "${SEQ_NEW:-}" = 1 ]; then
	echo "making a character on $addr (? for keys once playing, Ctrl+C to quit) ..."
	SEQ_SESSION_TOKEN="$token" tmp/seq-tui -insecure -world "$packs" -url "wss://$addr/seq/v1" -new
else
	echo "playing $character on $addr (? for keys, q to quit) ..."
	SEQ_SESSION_TOKEN="$token" tmp/seq-tui -insecure -world "$packs" -url "wss://$addr/seq/v1" -character "$character"
fi
