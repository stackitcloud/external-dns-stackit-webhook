#!/bin/sh
RECORD_TYPE="$1"
RECORD_NAME="$2"
EXPECTED_RESULT="$3"

# 1. Dynamically fetch one of the authoritative nameservers for your zone
AUTH_NS=$(dig +short NS "${ZONE_NAME}" | head -n 1)

if [ -z "$AUTH_NS" ]; then
  echo "Error: Could not determine authoritative nameserver for ${ZONE_NAME}"
  exit 1
fi

# 2. Query the authoritative nameserver directly
RESULT=$(dig "@$AUTH_NS" -t "$RECORD_TYPE" +short "${RECORD_NAME}")

# 3. Check for deletion or creation
if [ -z "$EXPECTED_RESULT" ]; then
  # Deletion Case
  if [ -z "$RESULT" ]; then
    echo "DNS $RECORD_TYPE record $RECORD_NAME successfully deleted from $AUTH_NS!"
    exit 0
  fi
else
  # Creation Case (Using grep to handle quotes around TXT records or trailing dots on CNAMEs)
  if echo "$RESULT" | grep -q "$EXPECTED_RESULT"; then
    echo "DNS $RECORD_TYPE record $RECORD_NAME successfully verified on $AUTH_NS!"
    exit 0
  fi
fi

# 4. If not matched, print wait message, sleep, and exit 1 so Kuttl retries
echo "Waiting for DNS update... (Type: $RECORD_TYPE, Name: $RECORD_NAME, Expected: '$EXPECTED_RESULT', Got: '$RESULT')"
exit 1