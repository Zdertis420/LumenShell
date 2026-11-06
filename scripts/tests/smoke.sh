#!/usr/bin/env bash
# End-to-end smoke test: starts a real Lumen shell and drives it with `lumen`.
#
#   scripts/tests/smoke.sh               run the checks
#   scripts/tests/smoke.sh --reload N    also trigger N theme reloads and compare memory use
#
# Needs a niri session (a Wayland display), Quickshell, and built binaries (make build).
# It uses a temporary config file, so your own configuration is never touched.

set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
LUMEN=$ROOT/scripts/bin/lumen
FIXTURES=$ROOT/scripts/tests/fixtures
RELOADS=0
if [ "${1:-}" = "--reload" ]; then
	RELOADS=${2:-100}
fi

if [ -z "${NIRI_SOCKET:-}" ] || [ -z "${WAYLAND_DISPLAY:-}" ]; then
	echo "SKIP: smoke test needs a niri session (NIRI_SOCKET and WAYLAND_DISPLAY)"
	exit 0
fi
for bin in "$LUMEN" "$ROOT/scripts/bin/config" "$ROOT/scripts/bin/palette"; do
	if [ ! -x "$bin" ]; then
		echo "missing $bin; run make build"
		exit 1
	fi
done
if pgrep -f "qs -p $ROOT" >/dev/null; then
	echo "a Lumen shell is already running; stop it before running the smoke test"
	exit 1
fi

TMP=$(mktemp -d)
export LUMEN_CONFIG_FILE=$TMP/config.json
SHELL_PID=""
FAILED=0
PASSED=0

cleanup() {
	stop_shell
	rm -rf "$TMP"
}
trap cleanup EXIT

ok() {
	PASSED=$((PASSED + 1))
	echo "  ok   $1"
}
bad() {
	FAILED=$((FAILED + 1))
	echo "  FAIL $1"
}
check() { # check "description" command...
	local desc=$1
	shift
	if "$@" >/dev/null 2>&1; then ok "$desc"; else bad "$desc"; fi
}
now_ms() { echo $(($(date +%s%N) / 1000000)); }

start_shell() { # start_shell [VAR=value ...]
	env "$@" "$LUMEN" start >"$TMP/shell.log" 2>&1 &
	SHELL_PID=$!
	local i
	for i in $(seq 1 50); do
		if env "$@" "$LUMEN" version 2>/dev/null | grep -q "shell: running, protocol 1"; then
			return 0
		fi
		sleep 0.1
	done
	return 1
}

stop_shell() {
	if [ -n "$SHELL_PID" ] && kill -0 "$SHELL_PID" 2>/dev/null; then
		kill "$SHELL_PID" 2>/dev/null
		wait "$SHELL_PID" 2>/dev/null
	fi
	SHELL_PID=""
}

# ---- 1. Normal shell ----
echo "1. start, commands, config"
rm -f "$LUMEN_CONFIG_FILE"
if start_shell; then ok "shell starts and answers lumen version right away"; else
	bad "shell did not start"
	cat "$TMP/shell.log"
	exit 1
fi

"$LUMEN" start >/dev/null 2>&1
[ $? -eq 4 ] && ok "second start exits 4" || bad "second start exits 4"

t0=$(now_ms)
"$LUMEN" theme tokyo-night >/dev/null 2>&1
code=$?
t1=$(now_ms)
[ $code -eq 0 ] && ok "theme tokyo-night exits 0" || bad "theme tokyo-night exits 0 (got $code)"
[ $((t1 - t0)) -lt 1000 ] && ok "valid command took $((t1 - t0)) ms (< 1000)" || bad "valid command took $((t1 - t0)) ms"
check "override saved to the user file" grep -q '"tokyo-night"' "$LUMEN_CONFIG_FILE"
before=$(cat "$LUMEN_CONFIG_FILE")

"$LUMEN" theme catppuccin-latte --temp >/dev/null 2>&1
[ $? -eq 0 ] && ok "theme --temp exits 0" || bad "theme --temp exits 0"
[ "$(cat "$LUMEN_CONFIG_FILE")" = "$before" ] && ok "--temp leaves the file unchanged" || bad "--temp leaves the file unchanged"

t0=$(now_ms)
"$LUMEN" theme nope >/dev/null 2>&1
code=$?
t1=$(now_ms)
[ $code -eq 2 ] && ok "unknown theme exits 2" || bad "unknown theme exits 2 (got $code)"
[ $((t1 - t0)) -lt 1000 ] && ok "invalid command failed in $((t1 - t0)) ms (< 1000)" || bad "invalid command took $((t1 - t0)) ms"
[ "$(cat "$LUMEN_CONFIG_FILE")" = "$before" ] && ok "invalid theme leaves the file unchanged" || bad "invalid theme leaves the file unchanged"

printf '{"custom":{"keep":true},"theme":{"preset":"tokyo-night","source":"preset"}}\n' >"$LUMEN_CONFIG_FILE"
"$LUMEN" theme --reset >/dev/null 2>&1
[ $? -eq 0 ] && ok "theme --reset exits 0" || bad "theme --reset exits 0"
check "reset keeps unknown keys" grep -q '"custom"' "$LUMEN_CONFIG_FILE"
check "reset removes the theme override" bash -c "! grep -q 'tokyo-night' '$LUMEN_CONFIG_FILE'"

printf '{"log":{"level":"loud"}}\n' >"$LUMEN_CONFIG_FILE"
sleep 0.6
check "invalid config value is reported with its key" grep -q 'log.level' "$TMP/shell.log"
"$LUMEN" version >/dev/null 2>&1
[ $? -eq 0 ] && ok "shell keeps running after an invalid config" || bad "shell keeps running after an invalid config"

printf '{broken' >"$LUMEN_CONFIG_FILE"
sleep 0.6
[ "$(cat "$LUMEN_CONFIG_FILE")" = "{broken" ] && ok "invalid JSON file is never overwritten" || bad "invalid JSON file is never overwritten"

stop_shell
"$LUMEN" theme tokyo-night >/dev/null 2>&1
code=$?
[ $code -eq 3 ] && ok "no shell: exit 3" || bad "no shell: exit 3 (got $code)"
[ "$(cat "$LUMEN_CONFIG_FILE")" = "{broken" ] && ok "no shell: nothing written" || bad "no shell: nothing written"

# ---- 2. A helper that hangs ----
echo "2. hanging helper"
mkdir -p "$TMP/hang"
ln -s "$ROOT/scripts/bin/config" "$TMP/hang/config"
ln -s "$FIXTURES/helper-hang.sh" "$TMP/hang/palette"
rm -f "$LUMEN_CONFIG_FILE"
if start_shell LUMEN_HELPER_DIR="$TMP/hang"; then
	ok "shell starts while the palette helper hangs"
	t0=$(now_ms)
	LUMEN_HELPER_DIR="$TMP/hang" "$LUMEN" version >/dev/null 2>&1
	t1=$(now_ms)
	[ $((t1 - t0)) -lt 500 ] && ok "shell answers in $((t1 - t0)) ms during the hang" || bad "shell slow during the hang ($((t1 - t0)) ms)"
	sleep 4
	check "hung helper is stopped and named" grep -q 'helper palette timeout' "$TMP/shell.log"
	LUMEN_HELPER_DIR="$TMP/hang" "$LUMEN" version >/dev/null 2>&1
	[ $? -eq 0 ] && ok "shell responsive after the timeout" || bad "shell responsive after the timeout"
else
	bad "shell did not start with a hanging helper"
fi
stop_shell

# ---- 3. A helper that fails ----
echo "3. failing helper"
mkdir -p "$TMP/fail"
ln -s "$FIXTURES/helper-fail.sh" "$TMP/fail/config"
ln -s "$ROOT/scripts/bin/palette" "$TMP/fail/palette"
rm -f "$LUMEN_CONFIG_FILE"
if start_shell LUMEN_HELPER_DIR="$TMP/fail"; then
	LUMEN_HELPER_DIR="$TMP/fail" "$LUMEN" theme tokyo-night >/dev/null 2>&1
	code=$?
	[ $code -eq 6 ] && ok "failing helper: lumen exits 6" || bad "failing helper: lumen exits 6 (got $code)"
	check "failure names the helper" grep -q 'helper config failed' "$TMP/shell.log"
	LUMEN_HELPER_DIR="$TMP/fail" "$LUMEN" version >/dev/null 2>&1
	[ $? -eq 0 ] && ok "shell responsive after the failure" || bad "shell responsive after the failure"
else
	bad "shell did not start with a failing helper"
fi
stop_shell

# ---- 4. Reload memory check ----
if [ "$RELOADS" -gt 0 ]; then
	echo "4. $RELOADS reloads"
	rm -f "$LUMEN_CONFIG_FILE"
	if start_shell; then
		first=""
		for i in $(seq 1 "$RELOADS"); do
			if [ $((i % 2)) -eq 0 ]; then theme=tokyo-night; else theme=catppuccin-latte; fi
			"$LUMEN" theme "$theme" --temp >/dev/null 2>&1
			if [ "$i" -eq 1 ]; then
				sleep 0.5
				first=$(ps -o rss= -p "$SHELL_PID" | tr -d ' ')
			fi
		done
		sleep 1
		last=$(ps -o rss= -p "$SHELL_PID" | tr -d ' ')
		limit=$((first + first / 10))
		echo "  rss after first reload: ${first} kB, after $RELOADS: ${last} kB (limit ${limit} kB)"
		[ "$last" -le "$limit" ] && ok "memory within 10% after $RELOADS reloads" || bad "memory grew more than 10%"
	else
		bad "shell did not start for the reload test"
	fi
	stop_shell
fi

echo
echo "passed: $PASSED, failed: $FAILED"
[ "$FAILED" -eq 0 ]
