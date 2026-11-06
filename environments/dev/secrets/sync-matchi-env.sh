#!/usr/bin/env bash
#
# Dekrypterar MATCHi-hemligheterna från den SOPS-krypterade filen i prod
# till en gitignoras env-fil som kustomize använder för att generera
# Secreten "go-api-secrets".
#
# Env-filen innehåller klartext och får ALDRIG checkas in.
#
# Körs automatiskt av Tilt (se ../Tiltfile) men kan även köras manuellt:
#
#   bash environments/dev/secrets/sync-matchi-env.sh

set -euo pipefail

script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
repo_root="$(cd "${script_dir}/../../.." && pwd)"

encrypted="${repo_root}/environments/prod/secrets/go-api-secrets.enc.yaml"
target="${script_dir}/matchi.env"

if ! command -v sops >/dev/null 2>&1; then
	echo "FEL: sops måste vara installerat för att dekryptera ${encrypted}" >&2
	exit 1
fi

# SOPS letar efter age-nyckeln i ~/.config/sops/age/keys.txt, men på vissa
# maskiner ligger den i ~/.config/sops-age/keys.txt.
if [[ -z "${SOPS_AGE_KEY_FILE:-}" ]]; then
	for candidate in "${HOME}/.config/sops/age/keys.txt" "${HOME}/.config/sops-age/keys.txt"; do
		if [[ -f "${candidate}" ]]; then
			export SOPS_AGE_KEY_FILE="${candidate}"
			break
		fi
	done
fi

if [[ -z "${SOPS_AGE_KEY_FILE:-}" ]]; then
	echo "FEL: hittade ingen age-nyckel (SOPS_AGE_KEY_FILE)" >&2
	exit 1
fi

if [[ ! -f "${encrypted}" ]]; then
	echo "FEL: ${encrypted} saknas" >&2
	exit 1
fi

sops --decrypt --extract '["stringData"]["MATCHI_USERNAME"]' "${encrypted}" \
	| tr -d '\n' > /tmp/matchi-username.$$
sops --decrypt --extract '["stringData"]["MATCHI_PASSWORD"]' "${encrypted}" \
	| tr -d '\n' > /tmp/matchi-password.$$

cat > "${target}" <<EOF
# Genererad av $(basename "$0") - gitignoras, innehåller klartext.
MATCHI_USERNAME=$(cat /tmp/matchi-username.$$)
MATCHI_PASSWORD=$(cat /tmp/matchi-password.$$)
EOF

rm -f /tmp/matchi-username.$$ /tmp/matchi-password.$$

echo "Skrev ${target}"
