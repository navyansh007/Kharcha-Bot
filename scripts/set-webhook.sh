#!/usr/bin/env bash
set -euo pipefail

: "${TELEGRAM_BOT_TOKEN:?}"
: "${PUBLIC_BASE_URL:?}"
: "${WEBHOOK_PATH_TOKEN:?}"
: "${WEBHOOK_SECRET:?}"

curl --fail --silent --show-error --config - <<CONFIG
url = "https://api.telegram.org/bot${TELEGRAM_BOT_TOKEN}/setWebhook"
data-urlencode = "url=${PUBLIC_BASE_URL%/}/webhook/${WEBHOOK_PATH_TOKEN}"
data-urlencode = "secret_token=${WEBHOOK_SECRET}"
data-urlencode = "allowed_updates=[\"message\",\"callback_query\",\"my_chat_member\"]"
data-urlencode = "drop_pending_updates=false"
CONFIG
echo
