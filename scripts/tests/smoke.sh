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
CYCLES=0
while [ $# -gt 0 ]; do
	case "$1" in
	--reload)
		RELOADS=${2:-100}
		shift 2
		;;
	--modules-cycles)
		CYCLES=${2:-100}
		shift 2
		;;
	*)
		echo "unknown option: $1"
		exit 2
		;;
	esac
done

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
ln -s "$ROOT/scripts/bin/modules" "$TMP/hang/modules"
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
ln -s "$ROOT/scripts/bin/modules" "$TMP/fail/modules"
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

# ---- 4. Modules ----
wait_log() { # wait_log pattern seconds: wait until the shell log has a line matching pattern
	local i
	for i in $(seq 1 $(($2 * 20))); do
		if grep -q -- "$1" "$TMP/shell.log"; then return 0; fi
		sleep 0.05
	done
	return 1
}

echo "4. modules: discovery, enabling, unloading"
UM="$TMP/user-modules"
mkdir -p "$UM"
cp -r "$ROOT/examples/hello-module" "$UM/"
rm -f "$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UM"; then
	wait_log '"msg":"modules scanned"' 3 && ok "modules are scanned at start" || bad "modules are scanned at start"
	if grep -q 'hello loaded' "$TMP/shell.log"; then bad "a user module must not run before it is enabled"; else ok "user module is discovered but not loaded"; fi

	t0=$(now_ms)
	printf '{"modules":{"hello":{"enabled":true}}}\n' >"$LUMEN_CONFIG_FILE"
	if wait_log 'hello loaded' 3; then
		t1=$(now_ms)
		ok "enabled module loaded after $((t1 - t0)) ms"
		[ $((t1 - t0)) -lt 1000 ] && ok "module loads in under 1 s" || bad "module took $((t1 - t0)) ms to load"
	else
		bad "enabled module did not load"
	fi

	printf '{}\n' >"$LUMEN_CONFIG_FILE"
	wait_log 'module:hello","msg":"unloaded"' 3 && ok "removing the override unloads the module" || bad "removing the override unloads the module"
	check "the shell is still running" "$LUMEN" version
else
	bad "shell did not start with a user module directory"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 5. Module API ----
echo "5. modules: the public interface and permissions"
MODFIX="$ROOT/scripts/tests/fixtures/modules"
UM2="$TMP/user-modules-api"
mkdir -p "$UM2"
cp -r "$MODFIX/api-probe" "$MODFIX/api-denied" "$UM2/"
printf '{"modules":{"api-probe":{"enabled":true},"api-denied":{"enabled":true}}}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UM2"; then
	wait_log 'probe run state' 4 || true
	wait_log 'denied done' 4 || true
	for pattern in \
		'probe id api-probe apiVersion 1.0' \
		'probe theme background #' \
		'probe style spacing.md 12' \
		'probe settings.greeting hi' \
		'manifest yes' \
		'probe config.value theme.source' \
		'probe run state ok message probe helper ok'; do
		check "api-probe: $pattern" grep -q -- "$pattern" "$TMP/shell.log"
	done
	check "module log lines carry the source module:api-probe" grep -q '"source":"module:api-probe"' "$TMP/shell.log"
	check "api.log.recent returned entries" bash -c "grep -E 'probe log.recent [1-9]' '$TMP/shell.log'"
	check "api-denied: run is denied" grep -q 'denied run state denied' "$TMP/shell.log"
	for perm in helpers config.read config.write log.read modules.read-settings; do
		check "api-denied: refusal names permission $perm" grep -q "was refused .*permission $perm not declared" "$TMP/shell.log"
	done
	check "refusals are logged with source module:api-denied" bash -c "grep 'was refused' '$TMP/shell.log' | grep -q '\"source\":\"module:api-denied\"'"
	check "a refused module keeps running" bash -c "! grep -q 'module:api-denied.*failed' '$TMP/shell.log'"

	"$LUMEN" theme tokyo-night >/dev/null 2>&1
	wait_log 'probe paletteId tokyo-night' 1 && ok "theme change reaches a module within 1 s (tokyo-night)" || bad "theme change reaches a module within 1 s (tokyo-night)"
	"$LUMEN" theme catppuccin-latte >/dev/null 2>&1
	wait_log 'probe paletteId catppuccin-latte' 1 && ok "theme change reaches a module within 1 s (catppuccin-latte)" || bad "theme change reaches a module within 1 s (catppuccin-latte)"
else
	bad "shell did not start for the module API checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 6. Module settings ----
echo "6. modules: settings"
UM3="$TMP/user-modules-settings"
mkdir -p "$UM3"
cp -r "$MODFIX/api-probe" "$MODFIX/qml-good-window" "$UM3/"
printf '{"modules":{"api-probe":{"enabled":true},"qml-good-window":{"enabled":true},"gone":{"x":1}}}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UM3"; then
	wait_log 'probe run state' 4 || true

	printf '{"modules":{"api-probe":{"enabled":true,"greeting":"Yo"},"qml-good-window":{"enabled":true},"gone":{"x":1}}}\n' >"$LUMEN_CONFIG_FILE"
	wait_log 'probe settings changed greeting=Yo' 1 && ok "a setting edit reaches the module within 1 s" || bad "a setting edit reaches the module within 1 s"

	printf '{"modules":{"api-probe":{"enabled":true,"greeting":"Yo","size":99},"qml-good-window":{"enabled":true},"gone":{"x":1}}}\n' >"$LUMEN_CONFIG_FILE"
	wait_log 'probe setting errors size:' 1 && ok "an invalid setting is reported to the module" || bad "an invalid setting is reported to the module"
	check "the warning names the module and the key" grep -q 'modules.api-probe.size' "$TMP/shell.log"
	check "the valid setting next to it is kept" grep -q 'probe settings changed greeting=Yo size=3' "$TMP/shell.log"
	check "the other module is unaffected" bash -c "! grep -q 'module:qml-good-window.*\(failed\|unloaded\)' '$TMP/shell.log'"
	"$LUMEN" version >/dev/null 2>&1 && ok "the shell keeps running" || bad "the shell keeps running"

	printf '{"modules":{"api-probe":{"enabled":true},"qml-good-window":{"enabled":true},"gone":{"x":1}}}\n' >"$LUMEN_CONFIG_FILE"
	wait_log 'probe settings changed greeting=hi size=3' 1 && ok "removing the overrides falls back to the defaults" || bad "removing the overrides falls back to the defaults"
	[ "$(cat "$LUMEN_CONFIG_FILE")" = '{"modules":{"api-probe":{"enabled":true},"qml-good-window":{"enabled":true},"gone":{"x":1}}}' ] && ok "the file holds only what the test wrote" || bad "the file holds only what the test wrote"
else
	bad "shell did not start for the settings checks"
	cat "$TMP/shell.log"
fi
stop_shell

echo "6b. modules: a module writes its own settings"
UM4="$TMP/user-modules-writer"
mkdir -p "$UM4"
cp -r "$MODFIX/settings-writer" "$UM4/"
printf '{"modules":{"settings-writer":{"enabled":true},"gone":{"x":1}},"custom":true}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UM4"; then
	wait_log 'writer reset size state' 4 || true
	check "undeclared setting is refused" grep -q 'writer set undeclared false' "$TMP/shell.log"
	check "invalid value is rejected" grep -q 'writer set size 99 state failed' "$TMP/shell.log"
	check "valid value is saved and applied" grep -q 'writer set size 4 state ok value 4' "$TMP/shell.log"
	check "reset removes the override" grep -q 'writer reset size state ok value 2' "$TMP/shell.log"
	check "reset leaves no leftovers in the file" bash -c "! grep -q 'size' '$LUMEN_CONFIG_FILE'"
	check "unknown keys survive the writes" bash -c "grep -q '\"gone\"' '$LUMEN_CONFIG_FILE' && grep -q '\"custom\"' '$LUMEN_CONFIG_FILE'"
else
	bad "shell did not start for the settings-writer checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 7. Isolation: a broken module only breaks itself ----
count_log() { grep -c -- "$1" "$TMP/shell.log"; }
wait_count() { # wait_count pattern count seconds
	local i
	for i in $(seq 1 $(($3 * 20))); do
		if [ "$(count_log "$1")" -ge "$2" ]; then return 0; fi
		sleep 0.05
	done
	return 1
}

echo "7. modules: isolation of broken modules"
UM5="$TMP/user-modules-broken"
mkdir -p "$UM5"
for m in qml-good-window err-bad-json err-missing-main err-newer-minor dup-a dup-b qml-syntax-error qml-missing-import qml-runtime-once helper-fails helper-hangs qml-error-storm; do
	cp -r "$MODFIX/$m" "$UM5/"
done
cp -r "$MODFIX/bad name!" "$UM5/"
printf '{"modules":{"qml-good-window":{"enabled":true},"err-missing-main":{"enabled":true},"err-newer-minor":{"enabled":true},"dup":{"enabled":true},"qml-syntax-error":{"enabled":true},"qml-missing-import":{"enabled":true},"qml-runtime-once":{"enabled":true},"helper-fails":{"enabled":true},"helper-hangs":{"enabled":true},"qml-error-storm":{"enabled":true}}}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UM5"; then
	t0=$(now_ms)
	"$LUMEN" version >/dev/null 2>&1
	t1=$(now_ms)
	[ $((t1 - t0)) -lt 500 ] && ok "the shell answers in $((t1 - t0)) ms while modules fail" || bad "the shell is slow while modules fail ($((t1 - t0)) ms)"
	wait_log 'fixture good-window loaded' 3 && ok "the good module loads next to the broken ones" || bad "the good module loads next to the broken ones"
	wait_log 'fixture helper-hangs state timeout' 5 || true
	wait_count 'module:qml-error-storm","msg":"failed' 1 5 || true
	sleep 1

	check "bad JSON is reported with a dir: id" grep -q 'module:dir:err-bad-json","msg":"invalid: .*JSON' "$TMP/shell.log"
	check "missing main file is reported with the module id" grep -q 'module:err-missing-main","msg":"invalid: main' "$TMP/shell.log"
	check "newer interface is reported with both versions" grep -q 'module:err-newer-minor","msg":"invalid: api: requires interface 1.9, Lumen provides 1.0' "$TMP/shell.log"
	check "duplicate id is reported with both directories" bash -c "grep 'invalid: id: duplicate' '$TMP/shell.log' | grep -q 'dup-a' "
	check "odd directory name is shown as dir:bad_name_" grep -q 'module:dir:bad_name_' "$TMP/shell.log"
	check "QML syntax error is reported with the module id" grep -q 'module:qml-syntax-error","msg":"failed: ' "$TMP/shell.log"
	check "QML missing import is reported with the module id" grep -q 'module:qml-missing-import","msg":"failed: .*does-not-exist' "$TMP/shell.log"
	check "a runtime error is attributed to its module" grep -q 'module:qml-runtime-once","msg":"runtime error: .*fixture runtime error once' "$TMP/shell.log"
	check "one runtime error does not unload the module" bash -c "! grep -q 'module:qml-runtime-once\",\"msg\":\"\(unloaded\|failed\)' '$TMP/shell.log'"
	check "a failing helper is named with its module" grep -q 'module:helper-fails helper h failed' "$TMP/shell.log"
	check "the failing helper's module keeps running" grep -q 'fixture helper-fails state failed' "$TMP/shell.log"
	check "a hanging helper is stopped and named" grep -q 'module:helper-hangs helper h timeout' "$TMP/shell.log"
	check "the hanging helper's module sees the timeout" grep -q 'fixture helper-hangs state timeout' "$TMP/shell.log"
	check "an error storm unloads the module" grep -q 'module:qml-error-storm","msg":"unloaded"' "$TMP/shell.log"
	check "an error storm marks it failed" grep -q 'module:qml-error-storm","msg":"failed: 3 runtime errors within 5 s' "$TMP/shell.log"
	sleep 3
	[ "$(count_log 'module:qml-error-storm","msg":"loaded"')" -eq 1 ] && ok "the error-storm module is not loaded again by itself" || bad "the error-storm module was loaded again"
	"$LUMEN" version >/dev/null 2>&1 && ok "the shell is still responsive" || bad "the shell is still responsive"
else
	bad "shell did not start for the isolation checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 7c. Module commands ----
echo "7c. modules: lumen modules commands"
UM7="$TMP/user-modules-cmds"
mkdir -p "$UM7"
cp -r "$ROOT/examples/hello-module" "$MODFIX/qml-syntax-error" "$UM7/"
rm -f "$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UM7"; then
	"$LUMEN" modules list >"$TMP/list.txt" 2>&1
	grep -Eq '^test-surface +1\.0\.0 +bundled +loaded' "$TMP/list.txt" && ok "list shows the bundled test surface as loaded" || bad "list shows the bundled test surface as loaded"
	grep -Eq '^hello +1\.0\.0 +user +disabled +permissions: helpers' "$TMP/list.txt" && ok "list shows a new user module as disabled with its permissions" || bad "list shows a new user module as disabled with its permissions"
	grep -Eq '^qml-syntax-error +1\.0\.0 +user +disabled' "$TMP/list.txt" && ok "list shows a second user module as disabled" || bad "list shows a second user module as disabled"
	[ ! -e "$LUMEN_CONFIG_FILE" ] && ok "listing writes nothing" || bad "listing writes nothing"

	t0=$(now_ms)
	"$LUMEN" modules enable hello >"$TMP/out.txt" 2>&1
	code=$?
	t1=$(now_ms)
	[ $code -eq 0 ] && grep -q 'hello: loaded' "$TMP/out.txt" && ok "enable loads the module and says so" || bad "enable loads the module and says so (exit $code: $(cat "$TMP/out.txt"))"
	[ $((t1 - t0)) -lt 1000 ] && ok "enable took $((t1 - t0)) ms (< 1000)" || bad "enable took $((t1 - t0)) ms"
	check "enable saved the override" grep -q '"enabled": true' "$LUMEN_CONFIG_FILE"
	"$LUMEN" modules list 2>&1 | grep -Eq '^hello +.* loaded' && ok "list shows hello as loaded" || bad "list shows hello as loaded"
	"$LUMEN" modules show hello >"$TMP/show.txt" 2>&1
	check "show prints the state and permissions" bash -c "grep -q 'state: loaded' '$TMP/show.txt' && grep -q 'permissions: helpers' '$TMP/show.txt' && grep -q 'greeting' '$TMP/show.txt'"

	"$LUMEN" modules disable hello >/dev/null 2>&1
	[ $? -eq 0 ] && ok "disable exits 0" || bad "disable exits 0"
	check "disabling a user module removes the override (the default is off)" bash -c "! grep -q hello '$LUMEN_CONFIG_FILE'"
	"$LUMEN" modules list 2>&1 | grep -Eq '^hello +.* disabled' && ok "list shows hello as disabled again" || bad "list shows hello as disabled again"

	"$LUMEN" modules disable test-surface >/dev/null 2>&1
	check "disabling a bundled module writes enabled=false" grep -q '"enabled": false' "$LUMEN_CONFIG_FILE"
	"$LUMEN" modules enable test-surface >/dev/null 2>&1
	check "enabling it again removes the override" bash -c "! grep -q test-surface '$LUMEN_CONFIG_FILE'"

	"$LUMEN" modules enable qml-syntax-error >"$TMP/out.txt" 2>&1
	code=$?
	[ $code -eq 6 ] && grep -q 'qml-syntax-error' "$TMP/out.txt" && ok "enabling a broken module exits 6 and names it" || bad "enabling a broken module exits 6 and names it (exit $code: $(cat "$TMP/out.txt"))"
	"$LUMEN" modules list 2>&1 | grep -Eq '^qml-syntax-error +.* failed' && ok "list shows it as failed" || bad "list shows it as failed"

	# Fix the module and reload it: no shell restart (recovery).
	cp "$MODFIX/qml-good-window/Main.qml" "$UM7/qml-syntax-error/Main.qml"
	before=$(cat "$LUMEN_CONFIG_FILE")
	"$LUMEN" modules reload qml-syntax-error >"$TMP/out.txt" 2>&1
	code=$?
	[ $code -eq 0 ] && grep -q 'qml-syntax-error: loaded' "$TMP/out.txt" && ok "a fixed module loads again after reload, without a shell restart" || bad "a fixed module loads again after reload (exit $code: $(cat "$TMP/out.txt"))"
	[ "$(cat "$LUMEN_CONFIG_FILE")" = "$before" ] && ok "reload writes nothing" || bad "reload writes nothing"

	# Rescan picks up a new directory without writing anything.
	cp -r "$MODFIX/api-denied" "$UM7/"
	"$LUMEN" modules list 2>&1 | grep -q '^api-denied' && bad "a new module must not appear before a rescan" || ok "a new module does not appear before a rescan"
	"$LUMEN" modules rescan >/dev/null 2>&1
	[ $? -eq 0 ] && ok "rescan exits 0" || bad "rescan exits 0"
	"$LUMEN" modules list 2>&1 | grep -Eq '^api-denied +.* disabled' && ok "rescan finds the new module (disabled)" || bad "rescan finds the new module (disabled)"
	[ "$(cat "$LUMEN_CONFIG_FILE")" = "$before" ] && ok "rescan writes nothing" || bad "rescan writes nothing"

	# A loaded module whose directory disappears stays loaded until reloaded.
	"$LUMEN" modules enable hello >/dev/null 2>&1
	rm -rf "$UM7/hello-module"
	"$LUMEN" modules rescan >/dev/null 2>&1
	"$LUMEN" modules list 2>&1 | grep -Eq '^hello +.* loaded.*directory missing|^hello +.* loaded.*1 warning' && ok "a vanished loaded module stays loaded with a warning" || bad "a vanished loaded module stays loaded with a warning"
	"$LUMEN" modules reload hello >"$TMP/out.txt" 2>&1
	code=$?
	[ $code -eq 6 ] && grep -q 'directory is missing' "$TMP/out.txt" && ok "reloading a vanished module marks it failed" || bad "reloading a vanished module marks it failed (exit $code: $(cat "$TMP/out.txt"))"
	"$LUMEN" modules rescan >/dev/null 2>&1
	"$LUMEN" modules list 2>&1 | grep -q '^hello' && bad "the vanished module is dropped by the next rescan" || ok "the vanished module is dropped by the next rescan"

	"$LUMEN" modules show nope-module >/dev/null 2>&1
	[ $? -eq 7 ] && ok "unknown module exits 7" || bad "unknown module exits 7"
	"$LUMEN" modules enable "Bad Id" >/dev/null 2>&1
	[ $? -eq 2 ] && ok "bad id exits 2" || bad "bad id exits 2"
	"$LUMEN" version >/dev/null 2>&1 && ok "the shell is still running" || bad "the shell is still running"
else
	bad "shell did not start for the module command checks"
	cat "$TMP/shell.log"
fi
stop_shell
"$LUMEN" modules list >/dev/null 2>&1
[ $? -eq 3 ] && ok "with no shell, modules list exits 3" || bad "with no shell, modules list exits 3"

# ---- 8. Module load/unload memory check ----
if [ "$CYCLES" -gt 0 ]; then
	echo "8. $CYCLES module load/unload cycles"
	UM6="$TMP/user-modules-cycles"
	mkdir -p "$UM6"
	cp -r "$MODFIX/qml-good-window" "$UM6/"
	printf '{}\n' >"$LUMEN_CONFIG_FILE"
	if start_shell LUMEN_USER_MODULES_DIR="$UM6"; then
		first=""
		for i in $(seq 1 "$CYCLES"); do
			printf '{"modules":{"qml-good-window":{"enabled":true}}}\n' >"$LUMEN_CONFIG_FILE"
			wait_count 'module:qml-good-window","msg":"loaded"' "$i" 3 || { bad "cycle $i: module did not load"; break; }
			printf '{}\n' >"$LUMEN_CONFIG_FILE"
			wait_count 'module:qml-good-window","msg":"unloaded"' "$i" 3 || { bad "cycle $i: module did not unload"; break; }
			if [ "$i" -eq 1 ]; then
				sleep 0.5
				first=$(ps -o rss= -p "$SHELL_PID" | tr -d ' ')
			fi
		done
		sleep 1
		last=$(ps -o rss= -p "$SHELL_PID" | tr -d ' ')
		if [ -n "$first" ]; then
			limit=$((first + first / 10))
			echo "  rss after first cycle: ${first} kB, after $CYCLES: ${last} kB (limit ${limit} kB)"
			[ "$last" -le "$limit" ] && ok "memory within 10% after $CYCLES load/unload cycles" || bad "memory grew more than 10% over $CYCLES cycles"
		fi
	else
		bad "shell did not start for the module cycle check"
	fi
	stop_shell
fi

# ---- 9. Reload memory check ----
if [ "$RELOADS" -gt 0 ]; then
	echo "9. $RELOADS reloads"
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
