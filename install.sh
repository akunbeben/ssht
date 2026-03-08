#!/bin/sh
set -e

# ssht installer
# This script detects your OS and architecture, downloads the latest binary,
# and installs it to /usr/local/bin/ssht.

REPO="akunbeben/ssht"
BINARY_NAME="ssht"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin) OS="darwin" ;;
    linux) OS="linux" ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

# Detect Architecture
ARCH=$(uname -m)
case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported Architecture: $ARCH"; exit 1 ;;
esac

# Construct download URL
# Note: This points to the latest release
URL="https://github.com/$REPO/releases/latest/download/${BINARY_NAME}-${OS}-${ARCH}"

echo "Downloading $BINARY_NAME for $OS-$ARCH..."
curl -sL "$URL" -o "$BINARY_NAME"

echo "Setting permissions..."
chmod +x "$BINARY_NAME"

# Determine installation directory
INSTALL_DIR=""
if [ -d "$HOME/.local/bin" ]; then
    INSTALL_DIR="$HOME/.local/bin"
elif [ -d "$HOME/bin" ]; then
    INSTALL_DIR="$HOME/bin"
else
    INSTALL_DIR="/usr/local/bin"
fi

echo "Installing to $INSTALL_DIR..."

# Check if directory is writable
if [ ! -d "$INSTALL_DIR" ]; then
    echo "Creating directory $INSTALL_DIR..."
    mkdir -p "$INSTALL_DIR" || (echo "Need sudo to create $INSTALL_DIR..." && sudo mkdir -p "$INSTALL_DIR")
fi

if [ -w "$INSTALL_DIR" ]; then
    mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
else
    echo "Need sudo to move binary to $INSTALL_DIR..."
    sudo mv "$BINARY_NAME" "$INSTALL_DIR/$BINARY_NAME"
fi

echo "Successfully installed $BINARY_NAME to $INSTALL_DIR/$BINARY_NAME"

# Check if the directory is in PATH
case ":$PATH:" in
    *":$INSTALL_DIR:"*) ;;
    *) 
        # Detect shell config file
        SHELL_CONF=""
        case "$SHELL" in
            */zsh) SHELL_CONF="$HOME/.zshrc" ;;
            */bash) 
                if [ -f "$HOME/.bashrc" ]; then
                    SHELL_CONF="$HOME/.bashrc"
                else
                    SHELL_CONF="$HOME/.bash_profile"
                fi
                ;;
            *) SHELL_CONF="$HOME/.profile" ;;
        esac

        if [ -n "$SHELL_CONF" ]; then
            echo "Adding $INSTALL_DIR to PATH in $SHELL_CONF..."
            # Create file if it doesn't exist
            touch "$SHELL_CONF"
            # Append if not already present in the file (basic check)
            if ! grep -q "$INSTALL_DIR" "$SHELL_CONF"; then
                printf "\n# ssht binary path\nexport PATH=\"\$PATH:$INSTALL_DIR\"\n" >> "$SHELL_CONF"
                echo "Successfully updated $SHELL_CONF. Please run 'source $SHELL_CONF' or restart your terminal."
            fi
        else
            echo "\nWARNING: $INSTALL_DIR is not in your PATH. You may need to add it manually."
        fi
        ;;
esac

"$INSTALL_DIR/$BINARY_NAME" version
