#!/bin/bash

find /home/rafa/projects/fiware/scripts/domains -name 'userroles_*.json' | while read f; do
  content=$(cat "$f")
  trimmed=$(echo "$content" | tr -d '[:space:]')
  if [ "$trimmed" = "{}" ]; then
    dir=$(dirname "$f")
    fname=$(basename "$f")
    uid="${fname#userroles_}"
    uid="${uid%.json}"
    rolemap="$dir/rolemap.json"
    if [ -f "$rolemap" ]; then
      folder=$(basename "$dir")
      uname=$(jq -r --arg id "$uid" '.users[] | select(.id == $id) | .name' "$rolemap" 2>/dev/null)
      echo "$folder: $uname (user ID: $uid)"
    fi
  fi
done
