#!/bin/bash

set -eu

# Use this script to download the latest Prysm release binary.
# Usage: ./prysm.sh PROCESS FLAGS
#   PROCESS can be one of beacon-chain or validator.
#   FLAGS are the flags or arguments passed to the PROCESS.
# Downloaded binaries are saved to ./dist. 
# Use USE_PRYSM_VERSION to specify a specific release version. 
#   Example: USE_PRYSM_VERSION=v0.3.3 ./prysm.sh beacon-chain

function color() {
      # Usage: color "31;5" "string"
      # Some valid values for color:
      # - 5 blink, 1 strong, 4 underlined
      # - fg: 31 red,  32 green, 33 yellow, 34 blue, 35 purple, 36 cyan, 37 white
      # - bg: 40 black, 41 red, 44 blue, 45 purple
      printf '\033[%sm%s\033[0m\n' "$@"
}

# `readlink -f` that works on OSX too.
function get_realpath() {
    if [ "$(uname -s)" == "Darwin" ]; then
        local queue="$1"
        if [[ "${queue}" != /* ]] ; then
            # Make sure we start with an absolute path.
            queue="${PWD}/${queue}"
        fi
        local current=""
        while [ -n "${queue}" ]; do
            # Removing a trailing /.
            queue="${queue#/}"
            # Pull the first path segment off of queue.
            local segment="${queue%%/*}"
            # If this is the last segment.
            if [[ "${queue}" != */* ]] ; then
                segment="${queue}"
                queue=""
            else
                # Remove that first segment.
                queue="${queue#*/}"
            fi
            local link="${current}/${segment}"
            if [ -h "${link}" ] ; then
                link="$(readlink "${link}")"
                queue="${link}/${queue}"
                if [[ "${link}" == /* ]] ; then
                    current=""
                fi
            else
                current="${link}"
            fi
        done

        echo "${current}"
    else
        readlink -f "$1"
    fi
}

# Complain if no arguments were provided.
if [ "$#" -lt 1 ]; then
    color "31" "Usage: ./prysm.sh PROCESS FLAGS."
    color "31" "PROCESS can be beacon-chain or validator."
    exit 1
fi


readonly wrapper_dir="$(dirname "$(get_realpath "${BASH_SOURCE[0]}")")/dist"
arch=$(uname -m)
arch=${arch/x86_64/amd64}
arch=${arch/aarch64/arm64}
readonly os_arch_suffix="$(uname -s | tr '[:upper:]' '[:lower:]')-$arch"

mkdir -p $wrapper_dir

function get_prysm_version() {
  if [[ -n ${USE_PRYSM_VERSION:-} ]]; then
    readonly reason="specified in \$USE_PRYSM_VERSION"
    readonly prysm_version="${USE_PRYSM_VERSION}"
  else
    # Find the latest Prysm version available for download.
    readonly reason="automatically selected latest available version"
    prysm_version=$(curl -s https://api.github.com/repos/prysmaticlabs/prysm/releases/latest | grep "tag_name" | cut -d : -f 2,3 | tr -d \" | tr -d , | tr -d [:space:])
    readonly prysm_version
  fi
}

get_prysm_version

color "37" "Latest Prysm version is $prysm_version."

BEACON_CHAIN_REAL="${wrapper_dir}/beacon-chain-${prysm_version}-${os_arch_suffix}"
VALIDATOR_REAL="${wrapper_dir}/validator-${prysm_version}-${os_arch_suffix}"

if [[ ! -x $BEACON_CHAIN_REAL ]]; then 
    color "34" "Downloading beacon chain@${prysm_version} to ${BEACON_CHAIN_REAL} (${reason})"

    curl -L "https://github.com/prysmaticlabs/prysm/releases/download/${prysm_version}/beacon-chain-${prysm_version}-${os_arch_suffix}" -o $BEACON_CHAIN_REAL
    chmod +x $BEACON_CHAIN_REAL
else
    color "37" "Beacon chain is up to date."
fi

if [[ ! -x $VALIDATOR_REAL ]]; then 
    color "34" "Downloading validator@${prysm_version} to ${VALIDATOR_REAL} (${reason})"

    curl -L "https://github.com/prysmaticlabs/prysm/releases/download/${prysm_version}/validator-${prysm_version}-${os_arch_suffix}" -o $VALIDATOR_REAL
    chmod +x $VALIDATOR_REAL
else
    color "37" "Validator is up to date."
fi

case $1 in
  beacon-chain)
    readonly process=$BEACON_CHAIN_REAL
    ;;

  validator)
    readonly process=$VALIDATOR_REAL
    ;;

  *)
    color "31" "Usage: ./prysm.sh PROCESS FLAGS."
    color "31" "PROCESS can be beacon-chain or validator."
    ;;
esac

color "36" "Starting Prysm $1 ${@:2}"
exec -a "$0" "${process}" "${@:2}"