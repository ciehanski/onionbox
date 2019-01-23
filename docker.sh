#!/usr/bin/env bash
export EXTERNAL_IP="dig +short myip.opendns.com @resolver1.opendns.com"
docker-compose up -d