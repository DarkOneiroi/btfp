#!/bin/bash

# setup_waybar.sh - Integrates BTFP with Waybar
# Usage: ./setup_waybar.sh [binary_path]

BTFP_PATH="${1:-$HOME/go/bin/btfp}"
CONFIG_PATH="$HOME/.config/waybar/config.jsonc"
STYLE_PATH="$HOME/.config/waybar/style.css"

# Try to find config if default doesn't exist
if [ ! -f "$CONFIG_PATH" ]; then
    CONFIG_PATH="$HOME/.config/waybar/config"
fi

if [ ! -f "$CONFIG_PATH" ]; then
    echo "Waybar config not found, skipping integration."
    exit 0
fi

echo "Checking Waybar integration..."

# Check if already integrated
if grep -q '"custom/btfp":' "$CONFIG_PATH"; then
    echo "Waybar already configured for btfp. Updating path if necessary..."
    sed -i "s|\"exec\": \".*btfp --waybar\"|\"exec\": \"$BTFP_PATH --waybar\"|" "$CONFIG_PATH"
    sed -i "s|\"on-click\": \".*btfp --remote pause\"|\"on-click\": \"$BTFP_PATH --remote pause\"|" "$CONFIG_PATH"
else
    echo "Integrating btfp with Waybar config..."
    
    # Add custom/btfp to modules-center if not present and if modules-center exists
    if grep -q '"modules-center": \[' "$CONFIG_PATH"; then
        if ! grep -q '"custom/btfp"' "$CONFIG_PATH"; then
            sed -i 's/"modules-center": \[/"modules-center": ["custom\/btfp", /' "$CONFIG_PATH"
        fi
    fi
    
    BTFP_DEF='  "custom/btfp": {
    "exec": "'$BTFP_PATH' --waybar",
    "return-type": "json",
    "on-click": "'$BTFP_PATH' --remote pause",
    "interval": 1,
    "tooltip": true
  },'

    # Insert after the first opening brace
    printf "%s\n" "$BTFP_DEF" > /tmp/btfp_def.json
    sed -i "0,/{/r /tmp/btfp_def.json" "$CONFIG_PATH"
    rm /tmp/btfp_def.json
    echo "Updated Waybar config."
fi

# Style integration
if [ -f "$STYLE_PATH" ]; then
    if ! grep -q "#custom-btfp" "$STYLE_PATH"; then
        echo "Updating Waybar style..."
        cat >> "$STYLE_PATH" <<EOF

#custom-btfp {
    margin: 0 10px;
    padding: 0 8px;
    font-weight: bold;
    color: #b4befe;
}

#custom-btfp.playing {
    color: #a6e3a1;
}
EOF
        echo "Updated Waybar style."
    fi
fi
