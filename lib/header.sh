#!/bin/bash

[[ -z "${INSIDE_ESB}" ]] && { echo "FATAL ERROR: This function cannot be executed on its own."; exit 1; }

# Welcome! It's very good to have you in the source code =D
banner_success "WELCOME TO ELWORK (encrypted luks workspace)!"

# Arrays to store parameter / command line arguments
declare -A params=()
declare -A documentation=()

# Use --sudo to enable sudo to be appended before each command executed
SUDO=""

# Use --debug to set DEBUG="--debug" into various external script executions as well as enable set -x on runtime
DEBUG=""

# Define color and style codes
BOLD=$(tput bold)
NORMAL=$(tput sgr0)

RED=$(tput setaf 1)
YELLOW=$(tput setaf 3)
GREEN=$(tput setaf 2)
WHITE=$(tput setaf 7)
BLACK=$(tput setaf 0)

WHITE_BG=$(tput setab 7)
YELLOW_BG=$(tput setab 3)
RED_BG=$(tput setab 1)
GREEN_BG=$(tput setab 2)

SELF="$(basename $0)"
APP="${SELF/.sh/}"

# Command Line Argument Registration
params[log]="./logs/${APP,,}.$(date +"%Y-%m-%d").log"
documentation[log]="Path to log file"

params[sudo]=false
documentation[sudo]="Flag to enable sudo before running commands"

params[trace]=false
documentation[trace]="Flag to enable stack traces in console output"
