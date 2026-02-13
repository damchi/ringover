#!/usr/bin/env sh
set -eu

MYSQL_HOST="${MYSQL_HOST:-127.0.0.1}"
MYSQL_PORT="${MYSQL_PORT:-3306}"
MYSQL_DATABASE="${MYSQL_DATABASE:-ringover}"
MYSQL_TEST_DATABASE="${MYSQL_TEST_DATABASE:-${MYSQL_DATABASE}_test}"
MYSQL_ROOT_USER="${MYSQL_ROOT_USER:-root}"
MYSQL_ROOT_PASSWORD="${MYSQL_ROOT_PASSWORD:-root}"

echo "Creating ${MYSQL_DATABASE} and ${MYSQL_TEST_DATABASE} if needed..."

SQL_CREATE_DATABASES="CREATE DATABASE IF NOT EXISTS \`${MYSQL_DATABASE}\`; CREATE DATABASE IF NOT EXISTS \`${MYSQL_TEST_DATABASE}\`;"

# During MySQL first init, socket access is available before TCP, and root auth can
# still be transitioning. Try local socket first (no password, then password), then TCP.
if mysql -u "${MYSQL_ROOT_USER}" -e "${SQL_CREATE_DATABASES}" >/dev/null 2>&1; then
  :
elif MYSQL_PWD="${MYSQL_ROOT_PASSWORD}" mysql -u "${MYSQL_ROOT_USER}" -e "${SQL_CREATE_DATABASES}" >/dev/null 2>&1; then
  :
elif mysql -h "${MYSQL_HOST}" -P "${MYSQL_PORT}" -u "${MYSQL_ROOT_USER}" --protocol=TCP -e "${SQL_CREATE_DATABASES}" >/dev/null 2>&1; then
  :
else
  MYSQL_PWD="${MYSQL_ROOT_PASSWORD}" mysql \
    -h "${MYSQL_HOST}" \
    -P "${MYSQL_PORT}" \
    -u "${MYSQL_ROOT_USER}" \
    --protocol=TCP \
    -e "${SQL_CREATE_DATABASES}"
fi

echo "Done."
