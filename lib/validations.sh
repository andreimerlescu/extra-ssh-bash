#!/bin/bash

[[ -z "${INSIDE_ESB}" ]] && { echo "FATAL ERROR: This function cannot be executed on its own."; exit 1; }

# Function to ensure directory is writable
# is_writable_dir "$(mktemp -d)" # returns:""
function is_writable_dir() {
    local dir=$1
    { [[ ! -w "$(dirname "$dir")" ]] && fatal "Directory is not writable: $dir"; } || true
}

# Function to ensure that provided param is not empty
# param_required "parent" # returns:""
function param_required(){
    local p=$1
    [[ "${p}" == "password" ]] && [[ ${#params[$p]} -le 3 ]] && retrieve_password
    { [[ -z "${p}" ]] && fatal "--${p} required"; } || true
}

# Function to ensure provided param is a boolean flag
# flag_required "sudo" # returns:""
function flag_required(){
    local f=$1
    { [[ "${f}" == false ]] && fatal "--${f} must be true"; } || true
}

# Function to require a provided param given the true set flag
# require_param_if_flag_true "password" "encrypt"
function require_param_if_flag_true() {
    local p=$1
    local f=$2
    [[ "${f}" == "encrypt" ]] && encrypted && [[ ${#params[$p]} -le 3 ]] && retrieve_password
    { [[ "${params[$f]}" == true ]] && param_required $p; } || true
}
