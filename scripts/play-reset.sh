#!/bin/sh
# make play-reset: wipe the kept play world (the docker volume; the characters
# are in it), after asking. Type "yes" to go ahead; anything else stops.
#
#   SEQ_PLAY_PROJECT=NAME  the docker compose project whose volume is wiped
set -eu

cd "$(dirname "$0")/.."
unset SEQ_TESTGROUNDS # the throwaway database has nothing to wipe
. scripts/play-env.sh

echo "This deletes every character, item and bit of progress in the play world:"
echo "  the docker volume ${project}_pgdata"
printf 'Type yes to wipe it: '
answer=""
read -r answer || true
if [ "$answer" != yes ]; then
	echo "nothing wiped."
	exit 1
fi
if docker info >/dev/null 2>&1; then
	# Remove the project's containers (a make play still running loses its
	# database), then its volume; a volume that was never made is no error.
	ids=$(docker ps -aq --filter "label=com.docker.compose.project=$project")
	[ -z "$ids" ] || docker rm -f $ids >/dev/null
	docker volume rm "${project}_pgdata" >/dev/null 2>&1 || true
else
	echo "docker is not running; nothing could be removed" >&2
fi
echo "wiped. make play starts a new world."
