#!/bin/bash

DUALSENSECTL="/usr/bin/dualsensectl"

output="$(timeout 5s $DUALSENSECTL battery 2>&1 | grep -m 1 -E "(discharging|charging|full|No device found)" || echo "Error collecting status")"

text=""

class=""

if echo "$output" | grep -q "No device found"; then
    text="   󰐔 "
    class="ds-nodualsense" # Custom class for styling
elif echo "$output" | grep -qE "[0-9]+ (discharging|charging|full)"; then
    percentage=$(echo "$output" | grep -oE "[0-9]+")
    status=$(echo "$output" | grep -oE "(discharging|charging|full)")

    text="${percentage}% 󰐔"

    case "$status" in
        "discharging")
            class="ds-discharging"
            if [ "$percentage" -le 20 ]; then
                class="${class} ds-low"
            elif [ "$percentage" -le 50 ]; then
                class="${class} ds-medium"
            else
                class="${class} ds-high"
            fi
            ;;
        "charging")
            class="ds-charging"
            ;;
        "full")
            class="ds-full"
            ;;
        *)
            class="ds-unknown"
            ;;
    esac
else
    text="? 󰐔"
    class="ds-unknown"
fi

cat << EOF
{"text": "$text","class": "$class"}
EOF
