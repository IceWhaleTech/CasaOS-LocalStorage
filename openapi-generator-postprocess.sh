#!/usr/bin/bash

set -e

if grep -q "api_default.go" <<< "$1"; then
    API_DEFAULT_GO=$1

    cat < "${API_DEFAULT_GO}" | grep -v "github.com/go-chi/chi/v5" > "${API_DEFAULT_GO}.tmp"

    mv "${API_DEFAULT_GO}.tmp" "${API_DEFAULT_GO}"
fi
