#!/bin/sh

if [ -f .env ]; then
  echo "Loading environment variables from .env file"
. ./.env
else
  echo "No .env file found. Please create one with the necessary environment variables."
  exit 1
fi

bin/client $*