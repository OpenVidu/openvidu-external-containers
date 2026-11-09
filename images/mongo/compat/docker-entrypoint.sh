#!/bin/bash
# Entrypoint of the -compat variant of openvidu/mongodb. It accepts what the
# docker-entrypoint.sh of the official mongo image accepts as OpenVidu runs it:
#
#   docker-entrypoint.sh mongod [mongod options]
#
# - Started as root, it hands /data/db and /data/configdb to the mongodb user
#   (UID 999) and runs itself again as that user with setpriv.
# - MONGO_INITDB_ROOT_USERNAME and MONGO_INITDB_ROOT_PASSWORD create the root
#   user when the data directory is empty, on a temporary mongod that listens
#   on 127.0.0.1:27017 without --auth, --keyFile and --replSet. The *.sh and
#   *.js files of /docker-entrypoint-initdb.d run against it afterwards.
# - The final mongod gets the options as given, plus --auth when there is a
#   root user.
set -Eeuo pipefail

if [ "${1:0:1}" = '-' ]; then
	set -- mongod "$@"
fi

if [ "$1" = 'mongod' ] && [ "$(id -u)" = '0' ]; then
	find -L /data/configdb /data/db \! -user mongodb -exec chown mongodb: '{}' +
	# The temporary mongod logs to the standard output of this process
	chown --dereference mongodb "/proc/$$/fd/1" "/proc/$$/fd/2" || :
	exec setpriv --reuid=mongodb --regid=mongodb --init-groups "${BASH_SOURCE[0]}" "$@"
fi

if [ "$1" != 'mongod' ]; then
	exec "$@"
fi

# Value of option $1 in the rest of the arguments ("--opt value" or "--opt=value").
arg_value() {
	local name="$1" previous=''
	shift
	for arg in "$@"; do
		case "$arg" in
			"$name="*) echo "${arg#*=}"; return 0 ;;
		esac
		if [ "$previous" = "$name" ]; then
			echo "$arg"
			return 0
		fi
		previous="$arg"
	done
	return 1
}

# The arguments without option $1 and, when $2 is "value", without its value.
without_arg() {
	local name="$1" takes_value="$2" skip=''
	shift 2
	for arg in "$@"; do
		if [ -n "$skip" ]; then
			skip=''
			continue
		fi
		case "$arg" in
			"$name") [ "$takes_value" = value ] && skip=1; continue ;;
			"$name="*) continue ;;
		esac
		printf '%s\0' "$arg"
	done
}

if [ -n "${MONGO_INITDB_ROOT_USERNAME:-}" ] || [ -n "${MONGO_INITDB_ROOT_PASSWORD:-}" ]; then
	if [ -z "${MONGO_INITDB_ROOT_USERNAME:-}" ] || [ -z "${MONGO_INITDB_ROOT_PASSWORD:-}" ]; then
		echo >&2 'error: MONGO_INITDB_ROOT_USERNAME and MONGO_INITDB_ROOT_PASSWORD must be set together'
		exit 1
	fi
	has_root_user=1
else
	has_root_user=''
fi

db_path="$(arg_value --dbpath "$@" || echo /data/db)"
has_data=''
for f in "$db_path/WiredTiger" "$db_path/journal" "$db_path/storage.bson"; do
	if [ -e "$f" ]; then has_data=1; fi
done

initdb_scripts=()
for f in /docker-entrypoint-initdb.d/*; do
	case "$f" in *.sh | *.js) initdb_scripts+=("$f") ;; esac
done

if [ -z "$has_data" ] && { [ -n "$has_root_user" ] || [ "${#initdb_scripts[@]}" -gt 0 ]; }; then
	temp_args=("${@:2}")
	for opt in --auth --bind_ip_all; do
		mapfile -d '' temp_args < <(without_arg "$opt" flag "${temp_args[@]}")
	done
	for opt in --keyFile --replSet --bind_ip --port --logpath --pidfilepath; do
		mapfile -d '' temp_args < <(without_arg "$opt" value "${temp_args[@]}")
	done
	pidfile="$(mktemp)"
	mongod "${temp_args[@]}" --bind_ip 127.0.0.1 --port 27017 \
		--logpath "/proc/$$/fd/1" --logappend --pidfilepath "$pidfile" --fork

	mongo=(mongosh --host 127.0.0.1 --port 27017 --quiet)
	for _ in $(seq 30); do
		if "${mongo[@]}" --eval 'quit(db.adminCommand({ ping: 1 }).ok ? 0 : 2)' >/dev/null 2>&1; then break; fi
		sleep 1
	done

	if [ -n "$has_root_user" ]; then
		"${mongo[@]}" admin --eval '
			db.createUser({
				user: process.env.MONGO_INITDB_ROOT_USERNAME,
				pwd: process.env.MONGO_INITDB_ROOT_PASSWORD,
				roles: [{ role: "root", db: "admin" }]
			})'
		mongo+=(--username "$MONGO_INITDB_ROOT_USERNAME" --password "$MONGO_INITDB_ROOT_PASSWORD" --authenticationDatabase admin)
	fi

	export MONGO_INITDB_DATABASE="${MONGO_INITDB_DATABASE:-test}"
	for f in "${initdb_scripts[@]}"; do
		case "$f" in
			*.sh)
				echo "$0: running $f"
				# shellcheck disable=SC1090
				. "$f"
				;;
			*.js) echo "$0: running $f"; "${mongo[@]}" "$MONGO_INITDB_DATABASE" "$f" ;;
		esac
	done

	mongod "${temp_args[@]}" --shutdown
	rm -f "$pidfile"
	echo 'MongoDB init process complete; ready for start up.'
fi

if [ -n "$has_root_user" ] && ! printf '%s\n' "$@" | grep -qx -- '--auth'; then
	set -- "$@" --auth
fi

exec "$@"
