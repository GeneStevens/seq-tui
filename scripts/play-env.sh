#!/bin/sh
# Sourced by play.sh and play-reset.sh: which setup this is, and the names it
# uses. Sets mode, project, mount, port, account, startspace and roster.
if [ "${SEQ_TESTGROUNDS:-}" = 1 ]; then
	mode=testgrounds
	project="${SEQ_PLAY_PROJECT:-seq-tui-play}" # the throwaway database (tmpfs)
	mount=tmpfs
	port="${SEQ_PG_PORT:-55432}"
	account=me
	startspace=testgrounds
else
	mode=world
	project="${SEQ_PLAY_PROJECT:-seq-tui-world}" # its volume, <project>_pgdata, is the world
	mount=volume
	port="${SEQ_PG_PORT:-55433}"
	account=player
	startspace="" # seqd's own: the start zone of the packs (Thornwick)
fi
roster="${SEQ_ROSTER:-$PWD/seq-data/roster.json}"
# the Testgrounds characters are thrown away with the database; their list is too
if [ "$mode" = testgrounds ]; then roster="${SEQ_ROSTER:-$PWD/tmp/testgrounds-roster.json}"; fi
