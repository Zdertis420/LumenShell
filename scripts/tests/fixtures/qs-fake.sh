#!/bin/sh
# Fake `qs` for controller tests. It records every call in $FAKE_QS_DIR/calls.log and
# behaves according to these environment variables:
#   FAKE_QS_RUNNING   1 (default) a shell is running, 0 none
#   FAKE_QS_VERSION   reply of version() (default 1)
#   FAKE_QS_REPLY     reply of setTheme/resetTheme (default "queued 7")
#   FAKE_QS_STATUS    comma-separated replies of status(), one per call, the last repeats
#                     (default "pending 7,done 7")
#   FAKE_QS_HANG      1 makes setTheme/resetTheme hang
#   FAKE_QS_LIST_FAIL 1 makes `list` fail

dir="${FAKE_QS_DIR:-/tmp}"
echo "$*" >>"$dir/calls.log"

# Drop "-p <root>".
if [ "$1" = "-p" ]; then
	shift 2
fi

case "$1" in
list)
	if [ "$FAKE_QS_LIST_FAIL" = "1" ]; then
		echo "list failed" >&2
		exit 1
	fi
	if [ "${FAKE_QS_RUNNING:-1}" = "1" ]; then
		echo '[{"config_path":"x/shell.qml","id":"abc","pid":1}]'
	else
		echo 'No running instances for "x/shell.qml"'
		echo 'Use --all to list all instances.'
	fi
	exit 0
	;;
-n)
	echo "STARTED"
	exit 0
	;;
ipc)
	# ipc call lumen <function> [args...]
	fn="$4"
	case "$fn" in
	version)
		echo "${FAKE_QS_VERSION:-1}"
		;;
	log)
		echo "ok"
		;;
	setTheme | resetTheme)
		if [ "$FAKE_QS_HANG" = "1" ]; then
			sleep 30
		fi
		echo "${FAKE_QS_REPLY:-queued 7}"
		;;
	modulesList)
		# FAKE_QS_MODULES_LIST holds the JSON reply.
		if [ -n "$FAKE_QS_MODULES_LIST" ]; then
			echo "$FAKE_QS_MODULES_LIST"
		else
			echo '{"attribution":"active","modules":[]}'
		fi
		;;
	moduleShow)
		id="$5"
		if [ -n "$FAKE_QS_MODULES_DIR" ] && [ -f "$FAKE_QS_MODULES_DIR/$id.json" ]; then
			cat "$FAKE_QS_MODULES_DIR/$id.json"
		else
			echo "error: unknown: unknown module '$id'"
		fi
		;;
	moduleCall)
		# FAKE_QS_CALL_REPLY holds the reply; the arguments are recorded by the call log.
		echo "${FAKE_QS_CALL_REPLY:-{\"ok\":true,\"result\":\"ok\"\}}"
		;;
	moduleEnable | moduleDisable | moduleReload | modulesRescan)
		id="$5"
		if [ "$fn" != "modulesRescan" ] && [ -n "$FAKE_QS_MODULES_DIR" ] && [ ! -f "$FAKE_QS_MODULES_DIR/$id.json" ]; then
			echo "error: unknown: unknown module '$id'"
		else
			if [ "$FAKE_QS_HANG" = "1" ]; then
				sleep 30
			fi
			echo "${FAKE_QS_REPLY:-queued 7}"
		fi
		;;
	status)
		seq="${FAKE_QS_STATUS:-pending 7,done 7}"
		n=0
		[ -f "$dir/status.idx" ] && n=$(cat "$dir/status.idx")
		count=$(printf '%s' "$seq" | awk -F, '{print NF}')
		pick=$((n + 1))
		[ "$pick" -gt "$count" ] && pick=$count
		printf '%s' "$seq" | awk -F, -v i="$pick" '{print $i}'
		echo $((n + 1)) >"$dir/status.idx"
		;;
	*)
		echo "unknown function $fn" >&2
		exit 1
		;;
	esac
	exit 0
	;;
esac
exit 1
