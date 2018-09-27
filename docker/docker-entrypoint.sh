#!/bin/bash

set -e

INPUT_DATA_FILE="/tmp/input-data.yml"
CONF_FILE_1="/etc/nxs-backup/conf.d/service.conf"
TPL_CONF_FILE_1="/usr/share/nxs-backup/service.conf.j2"

IMAGE_USER="root"

# usage: file_env VAR [DEFAULT]
#    ie: file_env 'XYZ_DB_PASSWORD' 'example'
# (will allow for "$XYZ_DB_PASSWORD_FILE" to fill in the value of
#  "$XYZ_DB_PASSWORD" from a file, especially for Docker's secrets feature)
file_env() {
	local var="$1"
	local fileVar="${var}_FILE"
	local def="${2:-}"
	if [ "${!var:-}" ] && [ "${!fileVar:-}" ]; then
		echo >&2 "error: both $var and $fileVar are set (but are exclusive)"
		exit 1
	fi
	local val="$def"
	if [ "${!var:-}" ]; then
		val="${!var}"
	elif [ "${!fileVar:-}" ]; then
		val="$(< "${!fileVar}")"
	fi
	export "$var"="$val"
	unset "$fileVar"
}

function file_env_input_data()
{
	local var="$1"
	file_env "${var}"

	local value="${!var}"
	local key=`echo "${var}" | tr '[:upper:]' '[:lower:]'`

	if [ ! -z "${value}" ];
	then
		echo "${key}: ${value}" >> ${INPUT_DATA_FILE}
	fi
}

# Setup Ssmtp

file_env 'SMTP_MAILHUB_ADDR'
file_env 'SMTP_MAILHUB_PORT'
file_env 'SMTP_USE_TLS'
file_env 'SMTP_AUTH_USER'
file_env 'SMTP_AUTH_PASS'
file_env 'SMTP_FROM_LINE_OVERRIDE'

if [ -z "${SMTP_USE_TLS}" ];
then
	SMTP_USE_TLS="YES"
fi

if [ -z "${SMTP_FROM_LINE_OVERRIDE}" ];
then
	SMTP_FROM_LINE_OVERRIDE="YES"
fi

if [ ! -z "${SMTP_MAILHUB_ADDR}" ] && \
   [ ! -z "${SMTP_MAILHUB_PORT}" ] && \
   [ ! -z "${SMTP_AUTH_USER}" ] && \
   [ ! -z "${SMTP_AUTH_PASS}" ];
then
	cat <<EOF > /etc/ssmtp/ssmtp.conf
mailhub=${SMTP_MAILHUB_ADDR}:${SMTP_MAILHUB_PORT}
UseTLS=${SMTP_USE_TLS}
AuthUser=${SMTP_AUTH_USER}
AuthPass=${SMTP_AUTH_PASS}
FromLineOverride=${SMTP_FROM_LINE_OVERRIDE}
EOF

	if [ ! -z "${IMAGE_USER}" ];
	then
		echo "${IMAGE_USER}:${SMTP_AUTH_USER}:${SMTP_MAILHUB_ADDR}:${SMTP_MAILHUB_PORT}" > /etc/ssmtp/revaliases
	fi

fi

# Preparing config files for nxs-backup

if [ -f "${INPUT_DATA_FILE}" ];
then
	rm -f ${INPUT_DATA_FILE}
fi

file_env_input_data 'DB_HOST'
file_env_input_data 'DB_PORT'
file_env_input_data 'DB_NAME'
file_env_input_data 'DB_USER'
file_env_input_data 'DB_PASSWORD'

if [ -f "${INPUT_DATA_FILE}" ];
then
	jinja2 ${TPL_CONF_FILE_1} ${INPUT_DATA_FILE} > ${CONF_FILE_1}
fi

exec "$@"
