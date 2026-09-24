#!/usr/bin/env bash

main () {
    local score="$1"
    local command="$2"
    local allergens=(eggs peanuts shellfish strawberries tomatoes chocolate pollen cats)

    if [ "$command" = "allergic_to" ]; then
        local target="$3"
        for i in "${!allergens[@]}"; do
            if [ "${allergens[$i]}" = "$target" ]; then
                if (( score & (1 << i) )); then
                    echo "true"
                else
                    echo "false"
                fi
            fi
        done
    elif [ "$command" = "list" ]; then
        local result=()
        for i in "${!allergens[@]}"; do
            if (( score & (1 << i) )); then
                result+=("${allergens[$i]}")
            fi
        done
        echo "${result[@]}"
    fi
}

main "$@"