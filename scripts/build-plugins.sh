#!/bin/bash

# Build all dynamic plugins for Zephyr MCP server

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
PLUGINS_DIR="$PROJECT_ROOT/plugins"

echo "Building Zephyr MCP Server plugins..."
echo "Project root: $PROJECT_ROOT"
echo "Plugins directory: $PLUGINS_DIR"

# Check if plugins directory exists
if [ ! -d "$PLUGINS_DIR" ]; then
    echo "Error: Plugins directory not found: $PLUGINS_DIR"
    exit 1
fi

# Build function for a single plugin
build_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    
    if [ ! -d "$plugin_dir" ]; then
        echo "Warning: Plugin directory not found: $plugin_dir"
        return 1
    fi
    
    if [ ! -f "$plugin_dir/Makefile" ]; then
        echo "Warning: No Makefile found in $plugin_dir"
        return 1
    fi
    
    echo "Building plugin: $plugin_name"
    cd "$plugin_dir"
    
    if make build; then
        echo "✅ Successfully built plugin: $plugin_name"
        
        # Check if .so file was created
        if [ -f "$plugin_name.so" ]; then
            echo "   Plugin binary: $plugin_dir/$plugin_name.so"
            echo "   Size: $(ls -lh "$plugin_name.so" | awk '{print $5}')"
        fi
    else
        echo "❌ Failed to build plugin: $plugin_name"
        return 1
    fi
    
    echo ""
}

# Clean function for a single plugin
clean_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    
    if [ ! -d "$plugin_dir" ]; then
        return 0
    fi
    
    if [ ! -f "$plugin_dir/Makefile" ]; then
        return 0
    fi
    
    echo "Cleaning plugin: $plugin_name"
    cd "$plugin_dir"
    make clean > /dev/null 2>&1 || true
}

# Test function for a single plugin
test_plugin() {
    local plugin_name="$1"
    local plugin_dir="$PLUGINS_DIR/$plugin_name"
    
    if [ ! -d "$plugin_dir" ]; then
        return 1
    fi
    
    if [ ! -f "$plugin_dir/Makefile" ]; then
        return 1
    fi
    
    echo "Testing plugin: $plugin_name"
    cd "$plugin_dir"
    
    if make test; then
        echo "✅ Plugin test passed: $plugin_name"
    else
        echo "❌ Plugin test failed: $plugin_name"
        return 1
    fi
}

# Get list of available plugins
get_plugins() {
    find "$PLUGINS_DIR" -maxdepth 1 -type d -name ".*" -prune -o -type d -print | \
    grep -v "^$PLUGINS_DIR$" | \
    xargs -I {} basename {}
}

# Main execution based on command
case "${1:-build}" in
    "build")
        echo "Building all plugins..."
        failed_plugins=()
        
        for plugin in $(get_plugins); do
            if ! build_plugin "$plugin"; then
                failed_plugins+=("$plugin")
            fi
        done
        
        echo "Build summary:"
        if [ ${#failed_plugins[@]} -eq 0 ]; then
            echo "✅ All plugins built successfully"
        else
            echo "❌ Failed plugins: ${failed_plugins[*]}"
            exit 1
        fi
        ;;
        
    "clean")
        echo "Cleaning all plugins..."
        for plugin in $(get_plugins); do
            clean_plugin "$plugin"
        done
        echo "✅ All plugins cleaned"
        ;;
        
    "test")
        echo "Testing all plugins..."
        failed_tests=()
        
        for plugin in $(get_plugins); do
            if ! test_plugin "$plugin"; then
                failed_tests+=("$plugin")
            fi
        done
        
        echo "Test summary:"
        if [ ${#failed_tests[@]} -eq 0 ]; then
            echo "✅ All plugin tests passed"
        else
            echo "❌ Failed tests: ${failed_tests[*]}"
            exit 1
        fi
        ;;
        
    "list")
        echo "Available plugins:"
        for plugin in $(get_plugins); do
            plugin_dir="$PLUGINS_DIR/$plugin"
            if [ -f "$plugin_dir/plugin.json" ]; then
                version=$(grep '"version"' "$plugin_dir/plugin.json" | sed 's/.*"version".*:.*"\([^"]*\)".*/\1/')
                description=$(grep '"description"' "$plugin_dir/plugin.json" | sed 's/.*"description".*:.*"\([^"]*\)".*/\1/')
                echo "  📦 $plugin v$version - $description"
                
                if [ -f "$plugin_dir/$plugin.so" ]; then
                    echo "      ✅ Built ($(ls -lh "$plugin_dir/$plugin.so" | awk '{print $5}'))"
                else
                    echo "      ❌ Not built"
                fi
            else
                echo "  📦 $plugin (no metadata)"
            fi
        done
        ;;
        
    "help"|"-h"|"--help")
        echo "Usage: $0 [command]"
        echo ""
        echo "Commands:"
        echo "  build    Build all plugins (default)"
        echo "  clean    Clean all plugin build artifacts"
        echo "  test     Test all plugins compilation"
        echo "  list     List all available plugins with status"
        echo "  help     Show this help message"
        echo ""
        echo "Examples:"
        echo "  $0                # Build all plugins"
        echo "  $0 build         # Build all plugins"
        echo "  $0 clean         # Clean all plugins"
        echo "  $0 list          # List available plugins"
        ;;
        
    *)
        echo "Unknown command: $1"
        echo "Use '$0 help' for usage information"
        exit 1
        ;;
esac 