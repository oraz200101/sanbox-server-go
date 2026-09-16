#!/usr/bin/env bash

cd "$(dirname "$0")" || exit 131

if ! docker-compose down ; then
  echo "%%%"
  echo "%%% ERROR of : docker-compose down"
  echo "%%%"
  exit 1
fi

if ! docker-compose up -d ; then
  echo "%%%"
  echo "%%% ERROR of : docker-compose up -d"
  echo "%%%"
  exit 1
fi

echo "%%%"
echo "%%% ГОТОВО"
echo "%%%"
