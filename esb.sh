#!/bin/bash

#set -e  # BEST PRACTICES: Exit immediately if a command exits with a non-zero status.
[ "${DEBUG:-0}" == "1" ] && set -x  # DEVELOPER EXPERIENCE: Enable debug mode, printing each command before it's executed.
set -C  # SECURITY: Prevent existing files from being overwritten using the '>' operator.

# Early parsing of debug flag
for arg in "$@"; do
  case $arg in
    --debug)
      DEBUG=true
      set -x
      ;;
  esac
done

# Required for any dependency to load
declare INSIDE_ESB=true 

# Load dependencies
files=("functions.sh" "header.sh" "validations.sh" "actions.sh")
declare -A icns=(
    ["functions.sh"]="⚀"
    ["header.sh"]="⚁"
    ["validations.sh"]="⚂"
    ["actions.sh"]="⚃"
    #[""]="⚄" # placeholder for 5
    #[""]="⚅" # placeholder for 6
)
for file in "${files[@]}"; do
    ((idx++))
    found_file=$(find ./lib -maxdepth 1 -name "$file" -print -quit)
    found_file="$(realpath $found_file)"
    { [[ -n "$found_file" ]] && source "$found_file"; printf "%s" "${icns[$file]}"; } || { echo "$found_file not found in the current directory."; exit 1; }
done
printf "%s\n" " Loaded application!"


function main() {
    parse_arguments "$@" || fatal "Failed to parse arguments"

    for p in "${!params[@]}"; do
        debug "${p} = ${params[$p]}"
    done

    # Welcome! It's very good to have you in the source code =D
    banner_success "WELCOME TO EXTRA SSH BASH (ESB)!"

    case "${params[action]}" in
        #pa|pass|passwd|password) action_passwd ;;
        *) fatal "unsupported action chosen $1" ;;
    esac

    [[ -n "${params[log]}" ]] && [[ -f "${params[log]}" ]] && $SUDO chown -R $(whoami):$(whoami) "${params[log]}"
    success "Script has completed execution!"

}

main "$@"