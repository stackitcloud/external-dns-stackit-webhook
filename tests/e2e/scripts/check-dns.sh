#!/bin/sh

# Enable shell debugging: prints every command and variable expansion to stdout
set -x

RECORD_TYPE="$1"
RECORD_NAME="$2"
EXPECTED_RESULT="$3"

echo "=== DEBUG: DNS CHECK STARTED ==="
echo "Type: $RECORD_TYPE | Name: $RECORD_NAME | Expected: '$EXPECTED_RESULT' | Zone: '${ZONE_NAME}'"

# 1. Fetch Auth NS
AUTH_NS=$(dig +short NS "${ZONE_NAME}" | head -n 1)
echo "DEBUG: Discovered Auth NS: '$AUTH_NS'"

if [ -z "$AUTH_NS" ]; then
  echo "ERROR: Could not determine authoritative nameserver for ${ZONE_NAME}"
  # Sleep so the log isn't spammed 100 times a second
  sleep 5
  exit 1
fi

# 2. Query the authoritative nameserver
RESULT=$(dig "@$AUTH_NS" -t "$RECORD_TYPE" +short "${RECORD_NAME}")
echo "DEBUG: Dig Result: '$RESULT'"

# 3. Check for deletion or creation
if [ -z "$EXPECTED_RESULT" ]; then
  # Deletion Case
  if [ -z "$RESULT" ]; then
    echo "SUCCESS: $RECORD_TYPE record $RECORD_NAME successfully deleted!"
    exit 0
  fi
else
  # Creation Case
  if echo "$RESULT" | grep -q "$EXPECTED_RESULT"; then
    echo "SUCCESS: $RECORD_TYPE record $RECORD_NAME verified!"
    exit 0
  fi
fi

echo "FAILED: Condition not met. Retrying in 5s..."
sleep 5
exit 1