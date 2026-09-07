#!/bin/bash
set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
REGISTRY_DIR="$PROJECT_ROOT/skills/registry"
PLUGIN_DIR="$PROJECT_ROOT/skills/plugin"

mkdir -p "$PLUGIN_DIR"

if [ ! -d "$REGISTRY_DIR" ]; then
    echo "Error: Registry directory not found at $REGISTRY_DIR"
    exit 1
fi

config_files=$(ls "$REGISTRY_DIR"/*.yaml 2>/dev/null || ls "$REGISTRY_DIR"/*.yml 2>/dev/null)

if [ -z "$config_files" ]; then
    echo "No plugin config files found in $REGISTRY_DIR"
    exit 0
fi

for config_file in $config_files; do
    echo "Processing $(basename "$config_file")..."
    
    name=$(grep -E '^name:' "$config_file" | head -1 | sed 's/name:\s*//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    repo_url=$(grep -E '^repo_url:' "$config_file" | head -1 | sed 's/repo_url:\s*//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    local_path=$(grep -E '^local_path:' "$config_file" | head -1 | sed 's/local_path:\s*//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    branch=$(grep -E '^branch:' "$config_file" | head -1 | sed 's/branch:\s*//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')
    version=$(grep -E '^version:' "$config_file" | head -1 | sed 's/version:\s*//' | sed 's/^[[:space:]]*//;s/[[:space:]]*$//')

    if [ -z "$name" ]; then
        echo "  Skipping: no name found in config"
        continue
    fi

    if [ -z "$repo_url" ]; then
        echo "  Skipping: no repo_url found for $name"
        continue
    fi

    plugin_path="$PROJECT_ROOT/${local_path:-skills/plugin/$name}"
    
    echo "  Plugin: $name (version: $version)"
    echo "  Repo: $repo_url"
    echo "  Path: $plugin_path"

    if [ -d "$plugin_path/.git" ]; then
        echo "  Updating existing plugin..."
        if [ -n "$branch" ]; then
            git -C "$plugin_path" checkout "$branch"
            git -C "$plugin_path" pull origin "$branch"
        else
            git -C "$plugin_path" pull
        fi
    else
        echo "  Cloning new plugin..."
        if [ -n "$branch" ]; then
            git clone --branch "$branch" "$repo_url" "$plugin_path"
        else
            git clone "$repo_url" "$plugin_path"
        fi
    fi

    echo "  Done!"
    echo ""
done

echo "All plugins have been built successfully!"