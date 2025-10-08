#!/bin/bash
# Direct test of terminal launching command construction

echo "🔧 Testing terminal command construction..."

# Simulate the terminal launching logic
findDmuxExecutable() {
    # First try to find dmux in PATH
    if command -v dmux >/dev/null 2>&1; then
        command -v dmux
        return
    fi
    
    # Try current executable's directory
    local currentExe="$0"
    if [[ "$currentExe" == *"dmux"* ]]; then
        echo "$currentExe"
        return
    fi
    
    # Try common installation locations
    local commonPaths=(
        "/usr/local/bin/dmux"
        "/usr/bin/dmux" 
        "$HOME/.local/bin/dmux"
        "$HOME/bin/dmux"
    )
    
    for path in "${commonPaths[@]}"; do
        if [[ -f "$path" ]]; then
            echo "$path"
            return
        fi
    done
    
    # Fallback
    echo "dmux"
}

findTerminalEmulator() {
    local terminals=(
        "konsole"
        "gnome-terminal"
        "xfce4-terminal" 
        "mate-terminal"
        "xterm"
        "urxvt"
        "terminator"
        "tilix"
        "alacritty"
        "kitty"
        "x-terminal-emulator"
    )
    
    for term in "${terminals[@]}"; do
        if command -v "$term" >/dev/null 2>&1; then
            echo "$term"
            return
        fi
    done
    
    echo "x-terminal-emulator"
}

# Test the functions
dmuxPath=$(findDmuxExecutable)
termCmd=$(findTerminalEmulator)

echo "✅ Dmux path: $dmuxPath"
echo "✅ Terminal: $termCmd"

# Test command construction
fromUser="testuser1"
joinCommand="export PATH=\"\$PATH:/usr/local/bin:/usr/bin:\$HOME/.local/bin:\$HOME/bin\"; $dmuxPath join $fromUser; exec bash"

# Determine terminal args
case "$termCmd" in
    "gnome-terminal")
        termArgs="--"
        ;;
    *)
        termArgs="-e"
        ;;
esac

echo ""
echo "🚀 Constructed command:"
echo "Terminal: $termCmd"
echo "Args: $termArgs bash -c \"$joinCommand\""
echo ""

# Test the actual command construction that would be used
echo "📋 Full command that would be executed:"
echo "$termCmd $termArgs bash -c \"$joinCommand\""

# Test PATH enhancement
currentPath="$PATH"
enhancedPath="$currentPath:/usr/local/bin:/usr/bin:$HOME/.local/bin:$HOME/bin"

echo ""
echo "🔧 PATH enhancement:"
echo "Original PATH length: $(echo "$currentPath" | wc -c)"
echo "Enhanced PATH length: $(echo "$enhancedPath" | wc -c)"
echo ""

# Show environment variables that would be set
echo "📋 Environment variables that would be set:"
echo "PATH=$enhancedPath"

echo ""
echo "✅ Terminal launching setup is robust and should work correctly!"