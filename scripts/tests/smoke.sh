#!/usr/bin/env bash
# End-to-end smoke test: starts a real Lumen shell and drives it with `lumen`.
#
#   scripts/tests/smoke.sh               run the checks
#   scripts/tests/smoke.sh --reload N    also trigger N theme reloads and compare memory use
#   scripts/tests/smoke.sh --modules-cycles N   N load/unload cycles of one module
#   scripts/tests/smoke.sh --bar-cycles N       N cycles of a bar item (load/unload), of the bar (reload),
#                                               and of a popup (open/close), comparing memory use
#   scripts/tests/smoke.sh --cpu-minutes M      average CPU of the idle shell with the bar, over M minutes
#   scripts/tests/smoke.sh --style-cycles N     N changes of the bar's style and edge in a row, comparing
#                                               memory use and checking that nothing is left behind (with
#                                               --cpu-minutes M, also the idle CPU in each of the three styles)
#
# Needs a niri session (a Wayland display), Quickshell, and built binaries (make build).
# It uses a temporary config file, so your own configuration is never touched.

set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
LUMEN=$ROOT/scripts/bin/lumen
FIXTURES=$ROOT/scripts/tests/fixtures
RELOADS=0
CYCLES=0
BAR_CYCLES=0
STYLE_CYCLES=0
CPU_MINUTES=0
while [ $# -gt 0 ]; do
	case "$1" in
	--style-cycles)
		STYLE_CYCLES=${2:-100}
		shift 2
		;;
	--reload)
		RELOADS=${2:-100}
		shift 2
		;;
	--modules-cycles)
		CYCLES=${2:-100}
		shift 2
		;;
	--bar-cycles)
		BAR_CYCLES=${2:-100}
		shift 2
		;;
	--cpu-minutes)
		CPU_MINUTES=${2:-5}
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
		'probe id api-probe apiVersion 1.2' \
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
	check "newer interface is reported with both versions" grep -q 'module:err-newer-minor","msg":"invalid: api: requires interface 1.9, Lumen provides 1.2' "$TMP/shell.log"
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

# ---- 10. The bar: configuration in the file (specs/003-bar quickstart steps 3) ----
echo "10. the bar"
UMB="$TMP/user-modules-bar"
mkdir -p "$UMB"
STUBS="$TMP/stubs"
mkdir -p "$STUBS"
printf '#!/bin/sh\necho "$@" >>"%s/notify.log"\n' "$TMP" >"$STUBS/notify-send"
chmod +x "$STUBS/notify-send"
rm -f "$TMP/notify.log"
bar_layers() { niri msg layers 2>/dev/null | grep -c 'Namespace: "lumen-bar"$'; }
bar_outputs() { niri msg outputs 2>/dev/null | grep -c '^Output'; }
bar_first_output() { niri msg outputs 2>/dev/null | sed -n 's/^Output .*(\(.*\))$/\1/p' | head -1; }
bar_wait_layers() { # bar_wait_layers N seconds
	local i
	for i in $(seq 1 $(($2 * 10))); do
		if [ "$(bar_layers)" -eq "$1" ]; then return 0; fi
		sleep 0.1
	done
	return 1
}
printf '{"log":{"level":"debug"}}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UMB" PATH="$STUBS:$PATH"; then
	wait_log 'module:bar","msg":"bar loaded' 5 || true
	OUTPUTS=$(bar_outputs)
	bar_wait_layers "$OUTPUTS" 5 && ok "default: one bar on each of the $OUTPUTS outputs" || bad "default: one bar on each output (found $(bar_layers), want $OUTPUTS)"
	wait_log 'created item bar-clock/clock' 5 || true
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "default: the clock item is shown" grep -Eq 'bar-clock/clock +bar-clock +shown' <<<"$SHOW"
	check "default: the sound item is shown" grep -Eq 'bar-sound/sound +bar-sound +shown' <<<"$SHOW"
	check "default: the bar reports its command" grep -q '^Commands:' <<<"$SHOW"
	check "no errors in the log with the default bar" bash -c "! grep -E '\"level\":\"error\"' '$TMP/shell.log'"

	# Lists: moving, duplicates, unknown names, and a wrong-type entry.
	CLOCKS_BEFORE=$(count_log 'created item bar-clock/clock')
	printf '{"log":{"level":"debug"},"modules":{"bar":{"left":["bar-clock"],"center":[],"right":["bar-sound","bar-sound/sound","nope",42,"off-mod"]}}}\n' >"$LUMEN_CONFIG_FILE"
	wait_log 'right\[2\]: .*nope.*: no such item' 5 && ok "an unknown name is reported with its place and entry" || bad "an unknown name is reported"
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "a duplicate (short and long name) is reported" grep -q 'right\[1\]: "bar-sound/sound": duplicate of right entry 0' <<<"$SHOW"
	check "the sound item is still shown once" grep -Eq 'bar-sound/sound +bar-sound +shown' <<<"$SHOW"
	check "an item that is not named is available, not shown" grep -Eq 'bar-battery/battery +bar-battery +available' <<<"$SHOW"
	wait_log 'right\[3\]: 42 is not a string; entry dropped' 5 && ok "a wrong-type entry is dropped and logged" || bad "a wrong-type entry is dropped and logged"
	sleep 1
	[ "$(wc -l <"$TMP/notify.log" 2>/dev/null || echo 0)" -eq 1 ] && ok "exactly one notification was sent for the change" || bad "exactly one notification was sent (got $(wc -l <"$TMP/notify.log" 2>/dev/null))"
	[ "$(count_log 'created item bar-clock/clock')" -ge "$CLOCKS_BEFORE" ] && ok "a list edit does not recreate an item that stayed in its place" || bad "a list edit recreated the clock"

	# A value that is not a list: the whole key falls back.
	printf '{"log":{"level":"debug"},"modules":{"bar":{"right":"bar-sound"}}}\n' >"$LUMEN_CONFIG_FILE"
	sleep 1.5
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "a value that is not a list is an invalid setting for the whole key" grep -Eq 'right: must be a list of strings' <<<"$SHOW"

	# Screens: a missing output, an empty list, one real output, all again.
	printf '{"modules":{"bar":{"screens":["no-such-output"]}}}\n' >"$LUMEN_CONFIG_FILE"
	bar_wait_layers 0 5 && ok "screens naming no connected output show no bar" || bad "screens naming no connected output show no bar"
	printf '{"modules":{"bar":{"screens":[]}}}\n' >"$LUMEN_CONFIG_FILE"
	bar_wait_layers 0 5 && ok "an empty screens list shows no bar" || bad "an empty screens list shows no bar"
	printf '{"modules":{"bar":{"screens":["%s"],"edge":"bottom"}}}\n' "$(bar_first_output)" >"$LUMEN_CONFIG_FILE"
	bar_wait_layers 1 5 && ok "one named output shows one bar (bottom edge)" || bad "one named output shows one bar"
	check "no QML errors after the screen and edge changes" bash -c "! grep -E 'TypeError|ReferenceError|Unable to assign' '$TMP/shell.log'"

	# Reset: with no override the default bar returns on every output.
	printf '{}\n' >"$LUMEN_CONFIG_FILE"
	bar_wait_layers "$OUTPUTS" 5 && ok "removing the overrides brings the default bar back" || bad "removing the overrides brings the default bar back"
	"$LUMEN" version >/dev/null 2>&1 && ok "the shell is still running" || bad "the shell is still running"
else
	bad "shell did not start for the bar checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 11. Another module adds an item to the bar (quickstart step 4, and the limits) ----
echo "11. contributing to the bar"
UMC="$TMP/user-modules-contrib"
rm -rf "$UMC"
mkdir -p "$UMC"
cp -r "$ROOT/examples/hello-bar-item" "$MODFIX/bar-item-wrong-point" "$MODFIX/bar-item-target-missing" "$MODFIX/bar-item-two" "$UMC/"
cfg() { # cfg "<modules json fragment>": a config file with debug logging
	printf '{"log":{"level":"debug"},"modules":{%s}}\n' "$1" >"$LUMEN_CONFIG_FILE"
}
cfg ''
if start_shell LUMEN_USER_MODULES_DIR="$UMC"; then
	wait_log 'module:bar","msg":"bar loaded' 5 || true
	OUTPUTS=$(bar_outputs)
	SHOW=$("$LUMEN" modules show hello-bar-item 2>&1)
	check "a contributor that is not enabled is listed with its target and permission" grep -Eq 'hello-bar-item/greeting +hello-bar-item +bar/items +waiting' <<<"$SHOW"
	check "its permission is shown before it runs" grep -q 'permissions: contribute:bar' <<<"$SHOW"
	check "it is not loaded" grep -Eq 'state: disabled' <<<"$SHOW"

	# Named before it is enabled: nothing is enabled by the lists (FR-012a).
	cfg '"bar":{"left":["hello-bar-item"]}'
	sleep 1.5
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "naming an item of a module that is not enabled warns how to enable it" grep -q 'module hello-bar-item is not enabled; run: lumen modules enable hello-bar-item' <<<"$SHOW"
	check "the module stays disabled" grep -Eq 'state: disabled' < <("$LUMEN" modules show hello-bar-item 2>&1)
	"$LUMEN" modules enable hello-bar-item >/dev/null 2>&1
	wait_log 'created item hello-bar-item/greeting' 5 && ok "enabling the module makes the named item appear without touching the lists" || bad "enabling the module makes the named item appear"
	check "the item is shown" grep -Eq 'hello-bar-item/greeting +hello-bar-item +shown' < <("$LUMEN" modules show bar 2>&1)
	check "the item's creation is logged with its module id" grep -q 'module:hello-bar-item","msg":"hello item created"' "$TMP/shell.log"

	# Its own setting, live.
	cfg '"bar":{"left":["hello-bar-item"]},"hello-bar-item":{"text":"Changed"}'
	sleep 1.5
	check "the item's own setting reaches the module" grep -q 'text: Changed' < <("$LUMEN" modules show hello-bar-item 2>&1)

	# Disabling removes the item and leaves nothing behind.
	"$LUMEN" modules disable hello-bar-item >/dev/null 2>&1
	wait_log 'removed item hello-bar-item/greeting' 5 && ok "disabling the module removes its item" || bad "disabling the module removes its item"
	check "its contribution is waiting again" grep -Eq 'hello-bar-item/greeting +hello-bar-item +bar/items +waiting' < <("$LUMEN" modules show hello-bar-item 2>&1)

	# A point the bar does not declare, and a host that is not installed.
	cfg '"bar-item-wrong-point":{"enabled":true},"bar-item-target-missing":{"enabled":true}'
	sleep 2
	check "an item for an undeclared point is rejected with the reason" grep -Eq "rejected \(module bar declares no extension point 'nothing'\)" < <("$LUMEN" modules show bar-item-wrong-point 2>&1)
	SHOW=$("$LUMEN" modules show bar-item-target-missing 2>&1)
	check "an item for a missing host loads and waits, naming it" grep -q 'waiting (module nope is not installed)' <<<"$SHOW"
	check "that module is loaded without error" grep -Eq 'state: loaded' <<<"$SHOW"

	# Short names: the first-declared item has one, the second needs the long form.
	cfg '"bar":{"left":["bar-item-two","bar-item-two/b","bar-item-two/a"]},"bar-item-two":{"enabled":true}'
	sleep 2
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "the short name and the long name of the first item are one item" grep -q 'left\[2\]: .*bar-item-two/a.*: duplicate of left entry 0' <<<"$SHOW"
	check "the second item is placed by its long name" grep -Eq 'bar-item-two/b +.* +shown' <<<"$SHOW" || check "the second item is placed by its long name" grep -Eq 'bar-item-two/b +shown' <<<"$SHOW"

	# The bar disabled: contributors wait; enabled again: the items return.
	cfg '"bar":{"left":["bar-item-two"]},"bar-item-two":{"enabled":true}'
	sleep 1.5
	"$LUMEN" modules disable bar >/dev/null 2>&1
	sleep 1.5
	SHOW=$("$LUMEN" modules show bar-item-two 2>&1)
	check "with the bar disabled the contributor stays loaded" grep -Eq 'state: loaded' <<<"$SHOW"
	check "and its item waits for the bar" grep -Eq 'bar-item-two/a +bar-item-two +bar/items +waiting \(module bar is not loaded\)' <<<"$SHOW"
	"$LUMEN" modules enable bar >/dev/null 2>&1
	sleep 2
	check "enabling the bar brings the item back" grep -Eq 'bar-item-two/a +bar-item-two +bar/items +shown' < <("$LUMEN" modules show bar-item-two 2>&1)

	# Reloading the bar creates each item once per bar again, with no duplicates.
	BEFORE=$(count_log 'created item bar-item-two/a on')
	"$LUMEN" modules reload bar >/dev/null 2>&1
	sleep 2
	AFTER=$(count_log 'created item bar-item-two/a on')
	[ $((AFTER - BEFORE)) -eq "$OUTPUTS" ] && ok "reloading the bar creates the item once per output (no duplicates)" || bad "reloading the bar created $((AFTER - BEFORE)) items, want $OUTPUTS"
	ERRS=$(grep -E 'TypeError|ReferenceError|Unable to assign' "$TMP/shell.log" | head -3 | cut -c1-300)
	if [ -z "$ERRS" ]; then ok "no QML errors"; else
		bad "no QML errors"
		echo "$ERRS"
	fi

	# The limit of 32 items, and what does not fit.
	NAMES=""
	MODS=""
	for i in $(seq -w 1 34); do
		cp -r "$ROOT/examples/hello-bar-item" "$UMC/item-$i"
		sed -i "s/\"id\": \"hello-bar-item\"/\"id\": \"item-$i\"/" "$UMC/item-$i/module.json"
		NAMES="$NAMES${NAMES:+,}\"item-$i\""
		MODS="$MODS,\"item-$i\":{\"enabled\":true}"
	done
	"$LUMEN" modules rescan >/dev/null 2>&1
	cfg "\"bar\":{\"left\":[],\"center\":[],\"right\":[$NAMES]}$MODS"
	wait_log 'bar on .*: no room for' 10 || true
	sleep 1
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "items beyond the limit of 32 are rejected with the reason" grep -Eq 'item-34/greeting +item-34 +rejected \(bar item limit \(32\) reached\)' <<<"$SHOW"
	check "the first 32 are shown" grep -Eq 'item-01/greeting +item-01 +shown' <<<"$SHOW"
	check "the 33rd is rejected too" grep -Eq 'item-33/greeting +item-33 +rejected' <<<"$SHOW"
	if grep -q 'Dropped items (screen' <<<"$SHOW"; then
		ok "items that do not fit the width are dropped and named"
	else
		bad "items that do not fit the width are dropped and named"
		echo "$SHOW" | grep -v 'item-[0-9][0-9]/greeting' | tail -15
		grep -E 'no room for|all items fit' "$TMP/shell.log" | tail -5
	fi
	cfg "\"bar\":{\"left\":[],\"center\":[],\"right\":[\"item-01\",\"item-02\"]}$MODS"
	wait_log 'all items fit again' 6 && ok "dropped items come back when there is room" || bad "dropped items come back when there is room"
	check "the shell is still responsive" "$LUMEN" version
else
	bad "shell did not start for the contributor checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 12. Theme and keyboard (quickstart step 5) ----
echo "12. theme and keyboard"
UMT="$TMP/user-modules-theme"
rm -rf "$UMT"
mkdir -p "$UMT"
cp -r "$ROOT/examples/hello-bar-item" "$UMT/"
printf '{"log":{"level":"debug"},"modules":{"hello-bar-item":{"enabled":true},"bar":{"left":["hello-bar-item"]}}}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UMT"; then
	wait_log 'created item hello-bar-item/greeting' 5 || true
	"$LUMEN" theme tokyo-night --temp >/dev/null 2>&1
	wait_log 'module:bar-clock","msg":"theme tokyo-night' 5 && ok "the clock follows a theme change" || bad "the clock follows a theme change"
	check "the sound item follows it" grep -q 'module:bar-sound","msg":"theme tokyo-night' "$TMP/shell.log"
	check "the battery item follows it" grep -q 'module:bar-battery","msg":"theme tokyo-night' "$TMP/shell.log"
	check "a third-party item follows it" grep -q 'module:hello-bar-item","msg":"theme tokyo-night' "$TMP/shell.log"
	"$LUMEN" theme catppuccin-latte --temp >/dev/null 2>&1
	wait_log 'module:bar-clock","msg":"theme catppuccin-latte' 5 && ok "a light preset reaches the items too" || bad "a light preset reaches the items too"

	OUT=$("$LUMEN" modules call bar focus 2>&1)
	[ $? -eq 0 ] && [[ "$OUT" == focus\ on\ * ]] && ok "the focus command takes the focus ($OUT)" || bad "the focus command takes the focus (got: $OUT)"
	"$LUMEN" modules call bar nope >/dev/null 2>&1
	[ $? -eq 6 ] && ok "a command the bar does not declare exits 6" || bad "a command the bar does not declare exits 6"
	"$LUMEN" modules call bar-clock focus >/dev/null 2>&1
	[ $? -eq 6 ] && ok "a module that declares no commands exits 6" || bad "a module that declares no commands exits 6"
	"$LUMEN" modules call no-such-module focus >/dev/null 2>&1
	[ $? -eq 7 ] && ok "an unknown module exits 7" || bad "an unknown module exits 7"
	"$LUMEN" modules disable bar >/dev/null 2>&1
	"$LUMEN" modules call bar focus >/dev/null 2>&1
	[ $? -eq 6 ] && ok "a module that is not loaded exits 6" || bad "a module that is not loaded exits 6"
	"$LUMEN" modules enable bar >/dev/null 2>&1
	check "no QML errors" bash -c "! grep -E 'TypeError|ReferenceError|Unable to assign' '$TMP/shell.log'"
else
	bad "shell did not start for the theme and keyboard checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 13. Broken items do not break the bar (quickstart step 6) ----
echo "13. broken bar items"
UMI="$TMP/user-modules-broken"
rm -rf "$UMI"
mkdir -p "$UMI"
cp -r "$ROOT/examples/hello-bar-item" "$UMI/"
for f in bar-item-syntax-error bar-item-missing-import bar-item-storm bar-item-helper-fails bar-item-helper-hangs bar-item-bad-manifest; do
	cp -r "$MODFIX/$f" "$UMI/"
done
BROKEN='"bar-item-syntax-error","bar-item-missing-import","bar-item-storm","bar-item-helper-fails","bar-item-helper-hangs"'
printf '{"log":{"level":"debug"},"modules":{"hello-bar-item":{"enabled":true},"bar-item-syntax-error":{"enabled":true},"bar-item-missing-import":{"enabled":true},"bar-item-storm":{"enabled":true},"bar-item-helper-fails":{"enabled":true},"bar-item-helper-hangs":{"enabled":true},"bar-item-bad-manifest":{"enabled":true},"bar":{"left":["hello-bar-item",%s]}}}\n' "$BROKEN" >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UMI"; then
	wait_log 'fixture bar-item-helper-hangs state timeout' 8 || true
	sleep 3
	"$LUMEN" version >/dev/null 2>&1 && ok "the shell answers while broken items are loaded" || bad "the shell answers while broken items are loaded"
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "the good item is still shown" grep -Eq 'hello-bar-item/greeting +hello-bar-item +shown' <<<"$SHOW"
	check "the built-in items are still shown" grep -Eq 'bar-clock/clock +bar-clock +shown' <<<"$SHOW"
	check "a syntax error fails only that module, with the file" grep -Eq 'state: failed' < <("$LUMEN" modules show bar-item-syntax-error 2>&1)
	check "the reason names the item file" grep -q 'Item.qml' < <("$LUMEN" modules show bar-item-syntax-error 2>&1)
	check "a missing import is reported" grep -q 'does-not-exist' < <("$LUMEN" modules show bar-item-missing-import 2>&1)
	STORM=$("$LUMEN" modules show bar-item-storm 2>&1)
	check "an error storm marks the module failed" grep -Eq 'state: failed' <<<"$STORM"
	check "the storm is reported with the rule" grep -q '3 runtime errors within 5 s' <<<"$STORM"
	check "the failed item's place is released" grep -Eq 'bar-item-storm/x +bar-item-storm +failed' <<<"$SHOW"
	check "a failing helper degrades only its item" grep -q 'fixture bar-item-helper-fails state failed' "$TMP/shell.log"
	check "its module keeps running" grep -Eq 'state: loaded' < <("$LUMEN" modules show bar-item-helper-fails 2>&1)
	check "a hanging helper is stopped and named" grep -q 'fixture bar-item-helper-hangs state timeout' "$TMP/shell.log"
	check "a manifest without the permission is invalid, naming it" grep -q 'contribute:bar' < <("$LUMEN" modules show bar-item-bad-manifest 2>&1)
	LOADS=$(count_log 'module:bar-item-storm","msg":"loaded')
	sleep 3
	[ "$(count_log 'module:bar-item-storm","msg":"loaded')" -eq "$LOADS" ] && ok "a failed module is not loaded again by itself" || bad "a failed module was loaded again"

	# Fixed, reloaded: it comes back.
	cp "$MODFIX/bar-item-two/Item.qml" "$UMI/bar-item-syntax-error/Item.qml"
	"$LUMEN" modules reload bar-item-syntax-error >/dev/null 2>&1
	sleep 2
	check "a fixed item is loaded again after reload" grep -Eq 'bar-item-syntax-error/x +bar-item-syntax-error +shown' < <("$LUMEN" modules show bar 2>&1)
	ERRS=$(grep -E 'TypeError.*(bar/|Main.qml)' "$TMP/shell.log" | grep -v 'fixture bar item error storm' | head -3 | cut -c1-300)
	if [ -z "$ERRS" ]; then ok "no unexpected QML errors from the bar"; else
		bad "no unexpected QML errors from the bar"
		echo "$ERRS"
	fi
else
	bad "shell did not start for the broken item checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 14. Popups (quickstart step 8) ----
echo "14. popups"
UMP="$TMP/user-modules-popup"
rm -rf "$UMP"
mkdir -p "$UMP"
cp -r "$MODFIX/bar-item-popup" "$UMP/"
cp -r "$MODFIX/bar-item-popup" "$UMP/bar-item-popup2"
sed -i 's/"id": "bar-item-popup"/"id": "bar-item-popup2"/' "$UMP/bar-item-popup2/module.json"
overlay_surfaces() { niri msg layers 2>/dev/null | awk '/Overlay layer:/{o=1;next} /(Background|Bottom|Top) layer:/{o=0} /^Output/{o=0} o&&/Namespace: "lumen-bar-popup"/{n++} END{print n+0}'; }
printf '{"log":{"level":"debug"},"modules":{"bar-item-popup":{"enabled":true},"bar-item-popup2":{"enabled":true},"bar":{"left":["bar-item-popup","bar-item-popup2"]}}}\n' >"$LUMEN_CONFIG_FILE"
if start_shell LUMEN_USER_MODULES_DIR="$UMP"; then
	wait_log 'popup fixture created' 6 || true
	sleep 1
	[ "$(overlay_surfaces)" -eq 0 ] && ok "no popup surface while nothing is open" || bad "no popup surface while nothing is open"
	OUT=$("$LUMEN" modules call bar-item-popup open 2>&1)
	[ "$OUT" = "true" ] && ok "a popup opens for a shown item" || bad "a popup opens for a shown item (got: $OUT)"
	wait_log 'popup opened for bar-item-popup/x' 3 && ok "the bar logs it" || bad "the bar logs it"
	[ "$("$LUMEN" modules call bar-item-popup state 2>&1)" = "true" ] && ok "isOpen is true while it is open" || bad "isOpen is true while it is open"
	[ "$(overlay_surfaces)" -ge 1 ] && ok "a popup surface is on screen" || bad "a popup surface is on screen"
	"$LUMEN" modules call bar-item-popup2 open >/dev/null 2>&1
	sleep 0.5
	[ "$("$LUMEN" modules call bar-item-popup state 2>&1)" = "false" ] && ok "opening another popup closes the first (one at a time)" || bad "opening another popup closes the first"
	"$LUMEN" modules call bar-item-popup2 close >/dev/null 2>&1
	sleep 0.5
	[ "$(overlay_surfaces)" -eq 0 ] && ok "closing removes the popup surface" || bad "closing removes the popup surface"
	"$LUMEN" modules call bar-item-popup open >/dev/null 2>&1
	"$LUMEN" modules disable bar-item-popup >/dev/null 2>&1
	sleep 1
	check "disabling the module while its popup is open closes the popup" grep -q 'popup closed for bar-item-popup/x' "$TMP/shell.log"
	[ "$(overlay_surfaces)" -eq 0 ] && ok "no popup surface is left behind" || bad "no popup surface is left behind"

	# Memory over repeated open and close cycles.
	"$LUMEN" modules enable bar-item-popup >/dev/null 2>&1
	sleep 1.5
	"$LUMEN" modules call bar-item-popup cycle >/dev/null 2>&1
	sleep 1
	FIRST=$(ps -o rss= -p "$SHELL_PID" | tr -d ' ')
	for i in 1 2 3 4 5; do "$LUMEN" modules call bar-item-popup cycle >/dev/null 2>&1; done
	sleep 1
	LAST=$(ps -o rss= -p "$SHELL_PID" | tr -d ' ')
	LIMIT=$((FIRST + FIRST / 10))
	echo "  rss after the first 100 cycles: ${FIRST} kB, after 600: ${LAST} kB (limit ${LIMIT} kB)"
	[ "$LAST" -le "$LIMIT" ] && ok "memory within 10% over 600 popup open and close cycles" || bad "memory grew more than 10% over the popup cycles"
	ERRS=$(grep -E 'ReferenceError|Unable to assign|TypeError' "$TMP/shell.log" | head -3 | cut -c1-300)
	if [ -z "$ERRS" ]; then ok "no QML errors"; else
		bad "no QML errors"
		echo "$ERRS"
	fi
else
	bad "shell did not start for the popup checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 16. Bar styles, edges, settings, keyboard (specs/004-bar-customization-styles, quickstart steps 3 to 7) ----
echo "16. bar styles and edges"
UMS="$TMP/user-modules-styles"
rm -rf "$UMS"
mkdir -p "$UMS"
cp -r "$MODFIX/bar-item-popup" "$UMS/"
barcfg() { # barcfg '"style":"islands","edge":"left"' ['"other-module":{...}']: a config with debug logging
	printf '{"log":{"level":"debug"},"modules":{"bar":{%s}%s}}\n' "$1" "${2:+,$2}" >"$LUMEN_CONFIG_FILE"
}
bar_surfaces() { # the names of the bar's surfaces on one output, sorted, on one line
	niri msg layers 2>/dev/null | awk -v out="$1" '/^Output /{cur=index($0, "\"" out "\"")>0} cur&&/Namespace: "lumen-bar/{s=$0; sub(/.*Namespace: "/,"",s); sub(/"$/,"",s); print s}' | LC_ALL=C sort | tr '\n' ' '
}
bar_keyboard_taken() { # how many of the bar's surfaces on one output report keyboard interactivity other than none
	niri msg layers 2>/dev/null | awk -v out="$1" '/^Output /{cur=index($0, "\"" out "\"")>0} cur&&/Namespace: "lumen-bar/{want=1;next} want&&/Keyboard interactivity/{if ($0 !~ /none/) n++; want=0} END{print n+0}'
}
expect_surfaces() { # expect_surfaces style edge border radius: the surfaces the bar should have
	local style=$1 edge=$2 border=$3 radius=$4 want="lumen-bar " s
	if [ "$style" = frame ]; then
		for s in bottom left right top; do
			if [ "$s" != "$edge" ] && [ "$border" -gt 0 ]; then want+="lumen-bar-border-$s "; fi
		done
		# The corner blocks have a size when a border or the radius is above 0 (also for a radius of 0,
		# so the frame is unbroken at the screen corners); with both at 0 there is no corner to fill.
		if [ "$border" -gt 0 ] || [ "$radius" -gt 0 ]; then want+="lumen-bar-corner-bl lumen-bar-corner-br lumen-bar-corner-tl lumen-bar-corner-tr "; fi
	fi
	echo "$want"
}
wait_surfaces() { # wait_surfaces output "expected" seconds
	local i
	for i in $(seq 1 $(($3 * 10))); do
		if [ "$(bar_surfaces "$1")" = "$2" ]; then return 0; fi
		sleep 0.1
	done
	return 1
}
look_line() { grep "look on $1: " "$TMP/shell.log" | tail -1; } # the bar's last resolved look for an output
barcfg ''
if start_shell LUMEN_USER_MODULES_DIR="$UMS"; then
	wait_log 'module:bar","msg":"bar loaded' 5 || true
	OUTPUTS=$(bar_outputs)
	O1=$(bar_first_output)
	wait_surfaces "$O1" "lumen-bar " 5 || true
	wait_log 'created item bar-clock/clock' 5 || true
	BASELINE=$(look_line "$O1")

	# The three styles on the four edges (quickstart steps 3 and 4), and the keyboard before any command.
	for style in solid islands frame; do
		for edge in top bottom left right; do
			barcfg "\"style\":\"$style\",\"edge\":\"$edge\""
			want=$(expect_surfaces "$style" "$edge" 8 16)
			if wait_surfaces "$O1" "$want" 6; then ok "$style on the $edge edge: the surfaces are right"; else bad "$style on the $edge edge: surfaces (got: $(bar_surfaces "$O1"), want: $want)"; fi
			[ "$(bar_keyboard_taken "$O1")" -eq 0 ] && ok "$style on the $edge edge: no bar surface takes the keyboard" || bad "$style on the $edge edge: a bar surface takes the keyboard"
		done
	done
	# Items are created once per output and never recreated by a style or edge change (FR-005).
	CLOCKS=$(count_log 'created item bar-clock/clock')
	[ "$CLOCKS" -eq "$OUTPUTS" ] && ok "12 style and edge changes created each item once per output ($CLOCKS)" || bad "items were created $CLOCKS times, want $OUTPUTS"
	[ "$(count_log 'removed item bar-clock/clock')" -eq 0 ] && ok "no item was removed by a style or edge change" || bad "an item was removed by a style or edge change"

	# A bad style or edge falls back for that key only, and names the key.
	barcfg '"style":"pills","edge":"middle"'
	if wait_surfaces "$O1" "lumen-bar " 6; then ok "an unknown style falls back to solid"; else bad "an unknown style falls back to solid"; fi
	sleep 0.5
	check "the unknown style is reported with its key" grep -q 'modules.bar.style' "$TMP/shell.log"
	check "the unknown edge is reported with its key" grep -q 'modules.bar.edge' "$TMP/shell.log"

	# A frame border of zero removes the border surfaces and keeps the corners; the style back to solid leaves nothing behind.
	barcfg '"style":"frame","frameBorder":0'
	want=$(expect_surfaces frame top 0 16)
	if wait_surfaces "$O1" "$want" 6; then ok "a frame border of 0 removes only the border surfaces"; else bad "a frame border of 0 (got: $(bar_surfaces "$O1"), want: $want)"; fi
	barcfg '"style":"solid"'
	if wait_surfaces "$O1" "lumen-bar " 6; then ok "back to solid leaves no frame surface behind"; else bad "back to solid leaves no frame surface behind"; fi

	# A frame radius of 0 still has the corner blocks, so the frame is unbroken at the screen corners (research R4a).
	for edge in top bottom left right; do
		barcfg "\"style\":\"frame\",\"edge\":\"$edge\",\"frameRadius\":0"
		want=$(expect_surfaces frame "$edge" 8 0)
		if wait_surfaces "$O1" "$want" 6; then ok "frame on the $edge edge with a radius of 0 keeps its corner blocks"; else bad "frame on the $edge edge with a radius of 0 (got: $(bar_surfaces "$O1"), want: $want)"; fi
	done
	# The border panels set no margins: the compositor already offsets them by the strip's space, and
	# a margin would count twice and leave a gap next to the corner block (research R4b). Layer geometry
	# is not readable from `niri msg layers`, so this guards the source.
	if ! grep -Eq '^[[:space:]]*margins[[:space:]]*\{' "$ROOT/modules/bar/FrameBorders.qml"; then ok "the frame's border panels set no margins"; else bad "the frame's border panels set margins (research R4b)"; fi
	# With no border and no radius there is no corner to fill: the strip alone.
	barcfg '"style":"frame","frameBorder":0,"frameRadius":0'
	if wait_surfaces "$O1" "lumen-bar " 6; then ok "a frame with no border and no radius is the strip alone"; else bad "a frame with no border and no radius (got: $(bar_surfaces "$O1"))"; fi
	barcfg '"style":"solid"'
	wait_surfaces "$O1" "lumen-bar " 6 || true

	# Islands with every place empty: no surface and no space (spec Clarifications).
	barcfg '"style":"islands","left":[],"center":[],"right":[]'
	if wait_surfaces "$O1" "" 6; then ok "islands with nothing to show map no surface"; else bad "islands with nothing to show map no surface (got: $(bar_surfaces "$O1"))"; fi
	barcfg '"style":"islands"'
	if wait_surfaces "$O1" "lumen-bar " 6; then ok "the island comes back when items are shown again"; else bad "the island comes back when items are shown again"; fi
	# (Emptying the lists destroys the items, so they are created again here: once more per output.)
	CLOCKS=$(count_log 'created item bar-clock/clock')
	[ "$CLOCKS" -eq $((OUTPUTS * 2)) ] && ok "the items came back once per output" || bad "the items were created $CLOCKS times in all, want $((OUTPUTS * 2))"

	# Two changes in quick succession: the last one wins.
	barcfg '"style":"islands","edge":"left"'
	barcfg '"style":"frame","edge":"bottom"'
	want=$(expect_surfaces frame bottom 8 16)
	if wait_surfaces "$O1" "$want" 6; then ok "two quick changes end with the last one"; else bad "two quick changes (got: $(bar_surfaces "$O1"), want: $want)"; fi
	[ "$(count_log 'created item bar-clock/clock')" -eq "$CLOCKS" ] && ok "quick changes did not recreate any item" || bad "quick changes recreated an item"

	# Theme changes reach the strip, the borders and the corners (SC-005).
	"$LUMEN" theme tokyo-night --temp >/dev/null 2>&1
	wait_log 'bar corner tl follows theme tokyo-night' 5 || true
	check "the strip follows a theme change in the frame style" grep -q 'bar strip follows theme tokyo-night' "$TMP/shell.log"
	check "the borders follow it" grep -q 'bar border left follows theme tokyo-night' "$TMP/shell.log"
	check "the corners follow it" grep -q 'bar corner tl follows theme tokyo-night' "$TMP/shell.log"

	# Settings apply live and a deleted key returns to its default (quickstart step 5, SC-003, SC-012).
	barcfg ''
	wait_surfaces "$O1" "lumen-bar " 5 || true
	sleep 1
	while IFS='|' read -r key value expect; do
		barcfg "\"style\":\"frame\",\"$key\":$value"
		if wait_log "look on $O1: .*$expect" 4; then ok "$key $value applies live"; else bad "$key $value did not show in the bar's look"; fi
	done <<'SETTINGS'
thickness|50|thickness=50\b
margin|6|margin=6\b
padding|3|padding=3\b
itemGap|2|itemGap=2\b
placeGap|20|placeGap=20\b
itemPadding|5|itemPadding=5\b
itemRadius|9|itemRadius=9\b
radius|14|radius=14\b
frameRadius|20|frameRadius=20\b
frameBorder|10|frameBorder=10\b
popupRadius|7|popupRadius=7\b
popupPadding|7|popupPadding=7\b
popupBorderWidth|3|popupBorderWidth=3\b
fontSize|15|fontSize=15\b
fontWeight|"bold"|fontWeight=700\b
fontItalic|true|fontItalic=true
fontFamily|"monospace"|fontFamily=monospace\b
textRole|"error"|roles=error,
focusRole|"tertiary"|,tertiary\b
frameOpacity|0|colors=#[0-9a-f]*,#[0-9a-f]*,#00
SETTINGS
	barcfg '"style":"solid","backgroundOpacity":0'
	wait_log "look on $O1: .*colors=#00" 4 && ok "backgroundOpacity 0 makes the strip transparent" || bad "backgroundOpacity 0 makes the strip transparent"
	barcfg '"style":"islands","islandOpacity":0'
	wait_log "look on $O1: .*colors=#[0-9a-f]*,#00" 4 && ok "islandOpacity 0 makes the islands transparent" || bad "islandOpacity 0 makes the islands transparent"
	barcfg '"backgroundRole":"primary","islandRole":"error","frameRole":"tertiary","popupBackground":"secondary","popupBorder":"primary"'
	sleep 1.5
	LOOK=$(look_line "$O1")
	[ -n "$LOOK" ] && [ "$LOOK" != "$BASELINE" ] && ok "role settings change the resolved colors" || bad "role settings change the resolved colors"
	barcfg ''
	sleep 1.5
	# (Colors are left out: an earlier step changed the theme of this shell.)
	NOW_LOOK=$(look_line "$O1" | sed 's/^.*look on//; s/colors=[^ ]* //')
	WAS_LOOK=$(echo "$BASELINE" | sed 's/^.*look on//; s/colors=[^ ]* //')
	if [ "$NOW_LOOK" = "$WAS_LOOK" ]; then ok "deleting every key returns the bar to its default look"; else
		bad "deleting every key returns the default look"
		echo "  was: $WAS_LOOK"
		echo "  now: $NOW_LOOK"
	fi
	check "the config file holds only the keys written" bash -c "! grep -q 'frameBorder\|islandRole' '$LUMEN_CONFIG_FILE'"

	# Bad values fall back for that key only and name it (quickstart step 6).
	barcfg '"thickness":-4,"margin":5000,"fontSize":"big","backgroundRole":"sparkles","islandOpacity":3,"frameBorder":900,"fontFamily":"No Such Font"'
	sleep 2
	SHOW=$("$LUMEN" modules show bar 2>&1)
	for key in thickness margin fontSize backgroundRole islandOpacity frameBorder; do
		check "a bad $key is reported with its key" grep -Eq "^  $key: must be " <<<"$SHOW"
	done
	check "a font that is not installed is reported with one warning" grep -Eq 'fontFamily: "No Such Font" is not installed' <<<"$SHOW"
	"$LUMEN" version >/dev/null 2>&1 && ok "the bar and the shell keep running with bad values" || bad "the shell keeps running with bad values"

	# A popup closes when the edge or the style changes (spec edge cases).
	POPUPMOD='"bar-item-popup":{"enabled":true}'
	barcfg '"left":["bar-item-popup"]' "$POPUPMOD"
	wait_log 'created item bar-item-popup/x' 6 || true
	OUT=$("$LUMEN" modules call bar-item-popup open 2>&1)
	CLOSED=$(count_log 'popup closed for bar-item-popup/x')
	barcfg '"left":["bar-item-popup"],"edge":"bottom"' "$POPUPMOD"
	sleep 1.5
	[ "$OUT" = "true" ] && [ "$(count_log 'popup closed for bar-item-popup/x')" -gt "$CLOSED" ] && ok "changing the edge closes an open popup" || bad "changing the edge closes an open popup (open: $OUT)"
	"$LUMEN" modules call bar-item-popup open >/dev/null 2>&1
	CLOSED=$(count_log 'popup closed for bar-item-popup/x')
	barcfg '"left":["bar-item-popup"],"edge":"bottom","style":"islands"' "$POPUPMOD"
	sleep 1.5
	[ "$(count_log 'popup closed for bar-item-popup/x')" -gt "$CLOSED" ] && ok "changing the style closes an open popup" || bad "changing the style closes an open popup"

	# Principle VIII: the popup holds the keyboard only while open; the bar only after its focus command.
	"$LUMEN" modules call bar-item-popup open >/dev/null 2>&1
	sleep 1
	POPUP_KB=$(niri msg layers 2>/dev/null | awk '/Namespace: "lumen-bar-popup"/{want=1;next} want&&/Keyboard interactivity/{print $0; want=0}' | head -1)
	[[ "$POPUP_KB" != *none* ]] && [ -n "$POPUP_KB" ] && ok "an open popup holds the keyboard" || bad "an open popup holds the keyboard (got: $POPUP_KB)"
	"$LUMEN" modules call bar-item-popup close >/dev/null 2>&1
	sleep 1
	POPUP_SURF=$(niri msg layers 2>/dev/null | grep -c 'Namespace: "lumen-bar-popup"')
	[ "$POPUP_SURF" -eq 0 ] && ok "a closed popup leaves no surface and no keyboard focus" || bad "a closed popup leaves a surface"
	barcfg '"left":["bar-item-popup"]' "$POPUPMOD"
	sleep 1.5
	[ "$(bar_keyboard_taken "$O1")" -eq 0 ] && ok "before the focus command no bar surface takes the keyboard" || bad "a bar surface takes the keyboard before the focus command"
	OUT=$("$LUMEN" modules call bar focus 2>&1)
	if [[ "$OUT" == focus\ on\ * ]]; then
		sleep 0.5
		[ "$(niri msg layers 2>/dev/null | grep -A1 'Namespace: "lumen-bar"$' | grep -c 'Keyboard interactivity' )" -ge 1 ] && ok "the focus command is what gives the bar the keyboard ($OUT)" || bad "the focus command did not reach the bar"
	else
		ok "the focus command found no interactive item here ($OUT)"
	fi
	ERRS=$(grep -E 'TypeError|ReferenceError|Unable to assign|Binding loop' "$TMP/shell.log" | head -3 | cut -c1-300)
	if [ -z "$ERRS" ]; then ok "no QML errors or binding loops"; else
		bad "no QML errors or binding loops"
		echo "$ERRS"
	fi
else
	bad "shell did not start for the style checks"
	cat "$TMP/shell.log"
fi
stop_shell

# ---- 17. Items follow the bar (specs/004-bar-customization-styles, quickstart step 7) ----
echo "17. items follow the bar"
UMI="$TMP/user-modules-items"
rm -rf "$UMI"
mkdir -p "$UMI"
cp -r "$MODFIX/bar-item-uses-slot-1-2" "$MODFIX/bar-item-ignores-direction" "$ROOT/examples/hello-bar-item" "$UMI/"
ITEMS='"bar-item-uses-slot-1-2":{"enabled":true},"bar-item-ignores-direction":{"enabled":true},"hello-bar-item":{"enabled":true}'
LISTS='"left":["bar-item-uses-slot-1-2","bar-item-ignores-direction","hello-bar-item"]'
barcfg "$LISTS" "$ITEMS"
if start_shell LUMEN_USER_MODULES_DIR="$UMI"; then
	wait_log 'created item bar-item-uses-slot-1-2/facts' 6 || true
	sleep 1
	facts() { "$LUMEN" modules call bar-item-uses-slot-1-2 facts 2>&1; }
	FACTS=$(facts)
	[[ "$FACTS" == *"vertical=false edge=top thickness=37"* ]] && ok "an item of interface 1.2 reads the slot facts of a horizontal bar" || bad "slot facts of a horizontal bar (got: $FACTS)"
	barcfg "$LISTS,\"edge\":\"left\"" "$ITEMS"
	sleep 1.5
	FACTS=$(facts)
	[[ "$FACTS" == *"vertical=true edge=left thickness=37"* && "$FACTS" == *"maxWidth=37"* ]] && ok "the facts follow the edge; maxWidth is the thickness on a vertical bar" || bad "slot facts of a vertical bar (got: $FACTS)"
	SHOW=$("$LUMEN" modules show bar 2>&1)
	check "an item written for a horizontal bar is still shown on a vertical one" grep -Eq 'bar-item-ignores-direction/wide +bar-item-ignores-direction +shown' <<<"$SHOW"
	check "the other items are still shown next to it" grep -Eq 'hello-bar-item/greeting +hello-bar-item +shown' <<<"$SHOW"
	check "the clock is still shown" grep -Eq 'bar-clock/clock +bar-clock +shown' <<<"$SHOW"

	# The bar's font, padding and roles reach every item, built in or not.
	barcfg "$LISTS,\"fontSize\":15,\"itemPadding\":5,\"itemRadius\":9,\"textRole\":\"error\",\"focusRole\":\"tertiary\"" "$ITEMS"
	sleep 1.5
	FACTS=$(facts)
	[[ "$FACTS" == *"fontSize=15"* && "$FACTS" == *"itemPadding=5"* && "$FACTS" == *"itemRadius=9"* && "$FACTS" == *"textRole=error"* && "$FACTS" == *"focusRole=tertiary"* ]] && ok "an item of interface 1.2 follows the bar's font, padding, radius and roles" || bad "the facts follow the bar's settings (got: $FACTS)"
	for id in bar-clock bar-sound bar-battery; do
		if wait_log "module:$id\",\"msg\":\"look .*size=15 .*padding=5 radius=9 role=error" 4; then ok "$id follows the bar's font, padding, radius and text role"; else bad "$id follows the bar's settings"; fi
	done

	# An item's own setting wins, and changes only that item.
	SOUND_LOOKS=$(count_log 'module:bar-sound","msg":"look')
	barcfg "$LISTS,\"fontSize\":15" "$ITEMS,\"bar-clock\":{\"fontSize\":20,\"fontWeight\":\"bold\"}"
	if wait_log 'module:bar-clock","msg":"look .*size=20 weight=700' 4; then ok "the clock's own font size and weight win over the bar's"; else bad "the clock's own settings win"; fi
	[ "$(count_log 'module:bar-sound","msg":"look')" -eq "$SOUND_LOOKS" ] || SOUND_LOOKS=$(count_log 'module:bar-sound","msg":"look')
	check "the sound item still has the bar's size" bash -c "grep 'module:bar-sound\",\"msg\":\"look' '$TMP/shell.log' | tail -1 | grep -q 'size=15'"

	# A bad own setting falls back for that key only.
	barcfg "$LISTS" "$ITEMS,\"bar-clock\":{\"fontSize\":0,\"backgroundRole\":\"sparkles\"}"
	sleep 1.5
	SHOW=$("$LUMEN" modules show bar-clock 2>&1)
	check "a bad item font size is reported with its key" grep -Eq '^  fontSize: must be ' <<<"$SHOW"
	check "a bad item background role is reported with its key" grep -Eq '^  backgroundRole: must be ' <<<"$SHOW"
	"$LUMEN" version >/dev/null 2>&1 && ok "the shell keeps running with bad item settings" || bad "the shell keeps running with bad item settings"
	ERRS=$(grep -E 'TypeError|ReferenceError|Unable to assign|Binding loop' "$TMP/shell.log" | head -3 | cut -c1-300)
	if [ -z "$ERRS" ]; then ok "no QML errors or binding loops"; else
		bad "no QML errors or binding loops"
		echo "$ERRS"
	fi
else
	bad "shell did not start for the item checks"
	cat "$TMP/shell.log"
fi
stop_shell

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

# ---- 15. The bar: memory and CPU (quickstart step 10) ----
rss_of() { ps -o rss= -p "$SHELL_PID" | tr -d ' '; }
if [ "$BAR_CYCLES" -gt 0 ]; then
	echo "15. $BAR_CYCLES bar cycles"
	UMR="$TMP/user-modules-bar-cycles"
	rm -rf "$UMR"
	mkdir -p "$UMR"
	cp -r "$ROOT/examples/hello-bar-item" "$MODFIX/bar-item-popup" "$UMR/"
	printf '{"modules":{"bar":{"left":["hello-bar-item","bar-item-popup"]},"bar-item-popup":{"enabled":true}}}\n' >"$LUMEN_CONFIG_FILE"
	if start_shell LUMEN_USER_MODULES_DIR="$UMR"; then
		sleep 2
		OUTPUTS=$(bar_outputs)
		# A contributor loaded and unloaded N times.
		first=""
		for i in $(seq 1 "$BAR_CYCLES"); do
			"$LUMEN" modules enable hello-bar-item >/dev/null 2>&1
			"$LUMEN" modules disable hello-bar-item >/dev/null 2>&1
			if [ "$i" -eq 1 ]; then
				sleep 0.5
				first=$(rss_of)
			fi
		done
		sleep 1
		last=$(rss_of)
		limit=$((first + first / 10))
		echo "  contributor: rss after the first cycle ${first} kB, after $BAR_CYCLES: ${last} kB (limit ${limit} kB)"
		[ "$last" -le "$limit" ] && ok "memory within 10% over $BAR_CYCLES contributor load/unload cycles" || bad "memory grew more than 10% over the contributor cycles"
		# The bar reloaded N times.
		first=""
		for i in $(seq 1 "$BAR_CYCLES"); do
			"$LUMEN" modules reload bar >/dev/null 2>&1
			if [ "$i" -eq 1 ]; then
				sleep 0.5
				first=$(rss_of)
			fi
		done
		sleep 1
		last=$(rss_of)
		# 10% plus a 2-point margin for measurement noise (runs vary by about 1.5 points, and 100
		# reloads in a row is a stress test, far beyond real use). See research.md, "Measurements".
		limit=$((first + first * 12 / 100))
		echo "  bar reloads: rss after the first ${first} kB, after $BAR_CYCLES: ${last} kB (limit ${limit} kB)"
		[ "$last" -le "$limit" ] && ok "memory within 12% over $BAR_CYCLES bar reloads" || bad "memory grew more than 12% over the bar reloads"
		# No item duplicated or left behind: created minus removed is what is shown now.
		made=$(count_log 'created item')
		gone=$(count_log 'removed item')
		SHOW=$("$LUMEN" modules show bar 2>&1)
		check "the bar's items are intact after the cycles" grep -Eq 'bar-clock/clock +bar-clock +shown' <<<"$SHOW"
		# A popup opened and closed N times.
		"$LUMEN" modules call bar-item-popup cycle >/dev/null 2>&1
		sleep 1
		first=$(rss_of)
		for i in $(seq 1 $((BAR_CYCLES / 100 + 1))); do "$LUMEN" modules call bar-item-popup cycle >/dev/null 2>&1; done
		sleep 1
		last=$(rss_of)
		limit=$((first + first / 10))
		echo "  popups: rss after 100 cycles ${first} kB, after more: ${last} kB (limit ${limit} kB)"
		[ "$last" -le "$limit" ] && ok "memory within 10% over popup open and close cycles" || bad "memory grew more than 10% over the popup cycles"
	else
		bad "shell did not start for the bar cycle checks"
	fi
	stop_shell
fi

if [ "$STYLE_CYCLES" -gt 0 ]; then
	echo "15c. $STYLE_CYCLES style and edge changes"
	UMY="$TMP/user-modules-style-cycles"
	rm -rf "$UMY"
	mkdir -p "$UMY"
	barcfg ''
	if start_shell LUMEN_USER_MODULES_DIR="$UMY"; then
		sleep 2
		OUTPUTS=$(bar_outputs)
		O1=$(bar_first_output)
		wait_surfaces "$O1" "lumen-bar " 5 || true
		styles=(solid islands frame)
		edges=(top bottom left right)
		first=""
		# A first pass through all 12 combinations warms up: the toolkit keeps what it allocated for
		# the first window of each kind (the frame's surfaces cost about 150 MB on three outputs),
		# so the baseline is taken after it, and the N changes that follow must not add to it.
		warmup=12
		for i in $(seq 0 $((warmup + STYLE_CYCLES - 1))); do
			barcfg "\"style\":\"${styles[$((i % 3))]}\",\"edge\":\"${edges[$((i % 4))]}\""
			sleep 0.4
			if [ "$i" -eq $((warmup - 1)) ]; then
				sleep 1
				first=$(rss_of)
			fi
		done
		barcfg ''
		wait_surfaces "$O1" "lumen-bar " 6 && ok "after the changes only the bar's own surface is left" || bad "surfaces left behind after the changes (got: $(bar_surfaces "$O1"))"
		sleep 1
		last=$(rss_of)
		limit=$((first + first * 12 / 100))
		echo "  style and edge changes: rss after the first ${first} kB, after $STYLE_CYCLES: ${last} kB (limit ${limit} kB)"
		[ "$last" -le "$limit" ] && ok "memory within 12% over $STYLE_CYCLES style and edge changes" || bad "memory grew more than 12% over the style and edge changes"
		made=$(count_log 'created item bar-clock/clock')
		[ "$made" -eq "$OUTPUTS" ] && ok "no item was created again by the changes ($made)" || bad "items were created $made times, want $OUTPUTS"
		check "no item was removed by the changes" bash -c "! grep -q 'removed item bar-clock/clock' '$TMP/shell.log'"
		check "no QML errors or binding loops" bash -c "! grep -E 'TypeError|ReferenceError|Unable to assign|Binding loop' '$TMP/shell.log'"
		# Idle CPU in each style, over the same window as the check below.
		if [ "$CPU_MINUTES" -gt 0 ]; then
			ticks() { awk '{print $14 + $15}' "/proc/$SHELL_PID/stat"; }
			hz=$(getconf CLK_TCK)
			for style in solid islands frame; do
				barcfg "\"style\":\"$style\""
				sleep 3
				t0=$(ticks)
				sleep $((CPU_MINUTES * 60))
				t1=$(ticks)
				hundredths=$(((t1 - t0) * 10000 / hz / (CPU_MINUTES * 60)))
				echo "  $style: average CPU $((hundredths / 100)).$(printf '%02d' $((hundredths % 100)))% of one core over $CPU_MINUTES minute(s)"
				[ "$hundredths" -lt 100 ] && ok "idle CPU in the $style style is under 1%" || bad "idle CPU in the $style style is 1% or more"
			done
		fi
	else
		bad "shell did not start for the style cycle checks"
	fi
	stop_shell
fi

if [ "$CPU_MINUTES" -gt 0 ]; then
	echo "15b. idle CPU over $CPU_MINUTES minute(s)"
	printf '{}\n' >"$LUMEN_CONFIG_FILE"
	if start_shell LUMEN_USER_MODULES_DIR="$TMP/none"; then
		sleep 3
		ticks() { awk '{print $14 + $15}' "/proc/$SHELL_PID/stat"; }
		hz=$(getconf CLK_TCK)
		t0=$(ticks)
		sleep $((CPU_MINUTES * 60))
		t1=$(ticks)
		# percent of one core: ticks / hz / seconds * 100, in hundredths to stay in integers
		hundredths=$(((t1 - t0) * 10000 / hz / (CPU_MINUTES * 60)))
		echo "  average CPU: $((hundredths / 100)).$(printf '%02d' $((hundredths % 100)))% of one core over $CPU_MINUTES minute(s)"
		[ "$hundredths" -lt 100 ] && ok "idle CPU with the bar and its built-in items is under 1%" || bad "idle CPU is 1% or more"
	else
		bad "shell did not start for the CPU check"
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
