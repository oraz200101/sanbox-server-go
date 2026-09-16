#!/usr/bin/env bash

cd "$(dirname "$0")" || exit 131

VOLUMES="$HOME/volumes/sandbox"

docker-compose down

docker run --rm -v "$VOLUMES/:/data" \
       busybox:1.28 \
       find /data -mindepth 1 -maxdepth 1 -exec \
       rm -rf {} \;

mkdir -p "$VOLUMES/elasticsearch"
# на Windows + Docker Desktop это no-op, права раздаёт сам Docker Desktop
chmod -R 777 "$VOLUMES/elasticsearch" 2>/dev/null

if ! docker-compose up -d ; then
  echo "%%%"
  echo "%%% ERROR of : docker-compose up -d"
  echo "%%%"
  exit 1
fi

echo "%%%"
echo "%%% ГОТОВО"
echo "%%%"
