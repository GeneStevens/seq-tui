#!/bin/sh
# make play: a local seq server on a PostgreSQL that keeps its data, serving the
# world (Thornwick), and this TUI on it. The TUI asks which character to play.
# make play-testgrounds (SEQ_TESTGROUNDS=1): the old setup for tests and quick
# checks: the same server on a throwaway PostgreSQL, new characters starting in
# Testgrounds (the engine's test zone), and the dev characters me-1 and me-2.
#
# make play-web (SEQ_PLAY_WEB=1) is the same server and kept world, but it builds
# the browser version (make web) and has seqd serve it itself (-web-dir) over
# plain HTTP on loopback (-plain-http), on http://localhost:8480, so there is no
# certificate, proxy or Caddyfile and no clash with make play's 8443. It prints
# the address and, with sign-in off, the development token to paste. Stytch's
# settings (SEQ_STYTCH_PROJECT_ID, SEQ_STYTCH_DOMAIN, SEQ_STYTCH_PUBLIC_TOKEN,
# SEQ_STYTCH_SECRET) pass through from the environment to seqd, so the real
# sign-in can be tried on a Stytch Test project (docs/browser.md). Do not run it
# together with make play: they share the kept world's database container.
#
# Needs: docker (OrbStack's), and access to the private seq module. The script
# downloads the module itself; if that fails it says what the one-time setup is
# (GOPRIVATE, and git over ssh; the README has the same text).
# Nothing here is a secret: the database is a container on 127.0.0.1, the
# certificate is made in memory, the token is for a development account.
#
# Where the world lives (default mode): the docker volume
# <SEQ_PLAY_PROJECT>_pgdata (SEQ_PLAY_PROJECT defaults to seq-tui-world); the
# characters are the server's (the TUI's picker asks it for the account's list).
# Quitting stops the containers and keeps the volume. make play-reset
# (scripts/play-reset.sh) wipes it.
#
#   SEQ_TESTGROUNDS=1   the Testgrounds setup (same as make play-testgrounds)
#   SEQ_PG_PORT=55433   host port of the database (55432 in the Testgrounds setup)
#   SEQ_ADDR=127.0.0.1:8443   where the server listens (8480 with SEQ_PLAY_WEB=1)
#   SEQ_PLAY_WEB=1      make play-web: the browser build instead of the terminal (see below)
#   SEQ_CHARACTER=ID    skip the picker and play this character (me-1 or me-2 in Testgrounds)
#   SEQ_NEW=1           skip the picker and make a new character
#   SEQ_WORLD_PACKS=DIR  serve the packs in DIR (a seq-world checkout under work) instead of the pinned seq-world's
#   SEQ_PLAY_PROJECT=NAME  the docker compose project (its volume is the world)
#   SEQ_PLAY_DRY=1      print what would run (mode, project, packs, start zone, files) and stop
set -eu

cd "$(dirname "$0")/.."
. scripts/play-env.sh
web="${SEQ_PLAY_WEB:-}"
if [ "$web" = 1 ]; then addr="${SEQ_ADDR:-127.0.0.1:8480}"; else addr="${SEQ_ADDR:-127.0.0.1:8443}"; fi
# The address people open: Stytch's Redirect URLs name localhost, and match exactly,
# so the page, the hints and the activate link use localhost, not the loopback IP.
webaddr="localhost:${addr##*:}"
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
# The world (D54) is seq-world, at the version go.mod pins: the server serves its
# packs; the TUI reads everything about the zones from the server (the zone map).
world=$(go list -m -f '{{.Dir}}' github.com/GeneStevens/seq-world)
packs="${SEQ_WORLD_PACKS:-$world/packs}"
if [ ! -d "$packs" ]; then
	echo "make play: seq-world's packs are not at '$packs'; try: go mod download github.com/GeneStevens/seq-world" >&2
	exit 1
fi
mkdir -p tmp
log=tmp/seqd.log
export SEQ_PG_PORT="$port" SEQ_PG_MOUNT="$mount"

# What seqd runs with. Without -start-space it starts new characters in the start
# zone of the packs it serves (Thornwick, from seq-world's thornwick pack).
set -- -listen "$addr" -packs "$packs" -dev-account "$account"
if [ -n "$startspace" ]; then set -- "$@" -start-space "$startspace"; fi
# The kept Thornwick world holds only the characters made in it: no Testgrounds
# testers (player-1, player-2) on the account. Testgrounds mode needs them.
if [ "$mode" = world ]; then set -- "$@" -dev-testers=false; fi

if [ "$web" = 1 ]; then set -- "$@" -web-dir "$PWD/web/dist" -plain-http; fi

tui_args="-insecure -url wss://$addr/seq/v1"
if [ "${SEQ_NEW:-}" = 1 ]; then
	tui_args="$tui_args -new"
elif [ -n "${SEQ_CHARACTER:-}" ]; then
	tui_args="$tui_args -character $SEQ_CHARACTER"
elif [ "$mode" = testgrounds ]; then
	tui_args="$tui_args -character me-1"
fi

if [ "${SEQ_PLAY_DRY:-}" = 1 ]; then
	echo "mode: $mode"
	echo "docker project: $project ($mount database on 127.0.0.1:$port)"
	echo "packs: $packs"
	echo "seqd $*"
	if [ "$web" = 1 ]; then echo "web http://$webaddr/"; else echo "seq-tui $tui_args"; fi
	exit 0
fi
if ! docker info >/dev/null 2>&1; then
	echo "make play: docker is not running (start OrbStack); the server's database runs in a container" >&2
	exit 1
fi

cleanup() {
	[ -n "${seqd_pid:-}" ] && kill "$seqd_pid" 2>/dev/null || true
	# down keeps the volume: the world survives. (Not down -v; that is play-reset.)
	docker compose -p "$project" -f "$seqdir/deploy/docker-compose.yml" down >/dev/null 2>&1 || true
}
trap cleanup EXIT INT TERM

if [ "$mode" = world ]; then
	echo "the world is kept in the docker volume ${project}_pgdata (make play-reset wipes it)"
fi
echo "starting PostgreSQL on 127.0.0.1:$port ..."
docker compose -p "$project" -f "$seqdir/deploy/docker-compose.yml" up -d --wait >/dev/null

echo "building seqd $seqver ..."
GOBIN="$PWD/tmp" go install "github.com/GeneStevens/seq/cmd/seqd@$seqver"

# seqd applies the migrations still pending on the database before it serves, so
# a kept world is brought up to this seqd's schema on the way in.
SEQ_DATABASE_URL="postgres://seq:seq@127.0.0.1:$port/seq?sslmode=disable" \
	tmp/seqd "$@" >"$log" 2>&1 &
seqd_pid=$!

token=""
for _ in $(seq 1 100); do
	token=$(sed -n "s/^development token for $account: //p" "$log")
	[ -n "$token" ] && break
	kill -0 "$seqd_pid" 2>/dev/null || { cat "$log"; exit 1; }
	sleep 0.2
done
[ -n "$token" ] || { echo "seqd did not start; see $log"; exit 1; }
if [ "$mode" = world ] && ! grep -q '"start_space":"thornwick"' "$log"; then
	echo "make play: the server did not start new characters in Thornwick (see $log):" >&2
	grep 'packs loaded' "$log" >&2 || true
	exit 1
fi

if [ "$web" = 1 ]; then
	echo
	echo "SEQ in the browser: http://$webaddr/    (Ctrl+C stops it and keeps the world)"
	if [ -n "${SEQ_STYTCH_PROJECT_ID:-}" ] && [ "${SEQ_STYTCH:-}" != false ]; then
		echo "sign-in: Stytch project $SEQ_STYTCH_PROJECT_ID; add http://$webaddr/auth/callback as a Redirect URL and http://$webaddr as an authorized domain (docs/browser.md)"
	else
		echo "sign-in is off (no SEQ_STYTCH_PROJECT_ID). Paste this development token on the page:"
	fi
	echo "  token: $token"
	wait "$seqd_pid"
	exit 0
fi

go build -o tmp/seq-tui ./cmd/seq-tui
echo "on $addr (? for keys, q to quit, Ctrl+C on the picker) ..."
# shellcheck disable=SC2086 # the arguments hold no spaces
SEQ_SESSION_TOKEN="$token" tmp/seq-tui $tui_args
