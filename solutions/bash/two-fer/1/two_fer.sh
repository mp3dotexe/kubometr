#!/usr/bin/env bash

main () {
    name="$1"
    if [ -z "$name" ]; then
        name="you"
    fi
    echo "One for $name, one for me."
}

main "$@"
