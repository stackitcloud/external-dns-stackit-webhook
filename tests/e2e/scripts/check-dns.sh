#!/bin/sh
set -x

RECORD_TYPE="$1"
RECORD_NAME="$2"
EXPECTED_RESULT="$3"
ZONE_NAME="$4" # Grab it from the 4th argument!

echo "=== DEBUG: DNS CHECK STARTED ==="
echo "Type: $RECORD_TYPE | Name: $RECORD_NAME | Expected: '$EXPECTED_RESULT' | Zone: '$ZONE_NAME'"

AUTH_NS=$(dig +short NS "${ZONE_NAME}" | head -n 1)
echo "DEBUG: Discovered Auth NS: '$AUTH_NS'"

if [ -z "$AUTH_NS" ]; then
  echo "ERROR: Could not determine authoritative nameserver for ${ZONE_NAME}"
  sleep 5
  exit 1
fi

RESULT=$(dig "@$AUTH_NS" -t "$RECORD_TYPE" +short "${RECORD_NAME}")
echo "DEBUG: Dig Result: '$RESULT'"

if [ -z "$EXPECTED_RESULT" ]; then
  if [ -z "$RESULT" ]; then
    echo "SUCCESS: $RECORD_TYPE record $RECORD_NAME successfully deleted!"
    exit 0
  fi
else
  if echo "$RESULT" | grep -q "$EXPECTED_RESULT"; then
    echo "SUCCESS: $RECORD_TYPE record $RECORD_NAME verified!"
    exit 0
  fi
fi

echo "FAILED: Condition not met. Retrying in 5s..."
sleep 5
exit 1