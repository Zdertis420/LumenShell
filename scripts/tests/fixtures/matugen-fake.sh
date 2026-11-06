#!/bin/sh
# Fake matugen for tests. It imitates the output of matugen 4.2.0 (verified with a real
# matugen, task T035a): `matugen image <path> --json hex --type <t> --mode <m> --prefer <p> --dry-run`
# prints {"colors": {"<name>": {"dark": {"color": "#.."}, "default": {...}, "light": {...}}, ...}}.
# Like the real one, it fails when --prefer is missing.
#
# Behavior is chosen with FAKE_MATUGEN:
#   (unset)      valid output
#   lowcontrast  valid output, but on_primary is almost the same as primary
#   fail         exits with status 1
#   badjson      prints something that is not JSON
#   missing      valid JSON without the surface_container color
#   hang         sleeps for a long time

mode=dark
prefer=""
while [ $# -gt 0 ]; do
	case "$1" in
	--mode) mode="$2" ;;
	--prefer) prefer="$2" ;;
	esac
	shift
done

if [ -z "$prefer" ]; then
	echo "Error: Multiple source colors found, no preference was inputted, and a terminal was not detected." >&2
	exit 1
fi

case "$FAKE_MATUGEN" in
fail)
	echo "fake matugen: failing on purpose" >&2
	exit 1
	;;
badjson)
	echo "this is not json"
	exit 0
	;;
hang)
	sleep 30
	exit 0
	;;
esac

# Each name has a dark and a light value: name=dark,light
colors="
primary=#a8c8ff,#005ac1
on_primary=#00315e,#ffffff
secondary=#bcc7dc,#575e71
on_secondary=#263141,#ffffff
tertiary=#dabce2,#715573
on_tertiary=#3d2846,#ffffff
error=#ffb4ab,#ba1a1a
on_error=#690005,#ffffff
background=#121316,#faf9fd
on_background=#e3e2e6,#1a1b20
surface=#121316,#faf9fd
on_surface=#e3e2e6,#1a1b20
surface_variant=#44474e,#e1e2ec
on_surface_variant=#c4c6d0,#44474f
surface_container=#1e2022,#eeedf1
outline=#8e9099,#74777f
"

printf '{"image":"x","is_dark_mode":%s,"mode":"%s","colors":{' "$([ "$mode" = dark ] && echo true || echo false)" "$mode"
first=1
for line in $colors; do
	name=${line%%=*}
	pair=${line#*=}
	dark=${pair%%,*}
	light=${pair#*,}
	[ "$FAKE_MATUGEN" = "missing" ] && [ "$name" = "surface_container" ] && continue
	if [ "$FAKE_MATUGEN" = "lowcontrast" ] && [ "$name" = "on_primary" ]; then
		dark="#a0c0f8"
		light="#2a6bc8"
	fi
	[ $first -eq 1 ] || printf ','
	first=0
	printf '"%s":{"dark":{"color":"%s"},"default":{"color":"%s"},"light":{"color":"%s"}}' "$name" "$dark" "$dark" "$light"
done
printf '}}\n'
