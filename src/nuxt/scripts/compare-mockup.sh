#!/usr/bin/env bash
# Renderar den byggda lunchsidan och jämför geometrin med mockupen.
#
#   ./scripts/compare-mockup.sh [mockup.png]
#
# Mockupen (1080x1920) mättes med OCR + pixelanays. Måtten nedan är
# uttag ur mockupen och används som facit. Sidan har sedan ändrats till
# ett tvåkolumnslayout (måndag–torsdag till vänster, fredag–söndag till
# höger) med veckans soppa och vegetariska rätt under söndag, så delar av
# facit nedan gäller rubriken och logotypen.
set -euo pipefail

cd "$(dirname "$0")/.."

mockup="${1:-/home/simon/Downloads/Meny skärm Hovet/1.png}"
out=/tmp/lunch-shot.png

NUXT_PAGES_BUILD=true NUXT_APP_BASE_URL=/ node node_modules/nuxt/bin/nuxt.mjs generate >/dev/null 2>&1

(cd .output/public && python3 -m http.server 8123 >/dev/null 2>&1 &)
sleep 1

chromium --headless --disable-gpu --no-sandbox --hide-scrollbars \
	--force-device-scale-factor=1 --window-size=1080,1920 \
	--screenshot="${out}" http://localhost:8123/lunch/ 2>/dev/null

echo "=== MITT PÅ EGEN RENDERING ==="
tesseract "${out}" stdout -l eng --psm 6 tsv 2>/dev/null |
	awk -F'\t' '$12!="" {printf "%-14s x=%-5s y=%-5s w=%-5s h=%-4s\n",$12,$7,$8,$9,$10}' |
	grep -iE "Lunchmeny|^v[0-9]+|M.ndag|Tisdag|Onsdag|Torsdag|Fredag|L.rdag|S.ndag|Veckans|RESTAURANG|Lasarett|HOVET"

echo
echo "=== FACIT (mockup) ==="
cat <<'EOF'
Lunchmeny          x=282  y=88   w=360  h=80   färg #767477
V45                x=680  y=88   w=120  h=62   färg #767477

-- Två kolumner (ny layout) --
Måndag 2/11        x=102  y=242  vänster kolumn
Tisdag 3/11        x=100  y=571  vänster kolumn
Onsdag 4/11        x=101  y=919  vänster kolumn
Torsdag 5/11       x=100  y=1266 vänster kolumn
Fredag 6/11        x=572  y=243  höger kolumn
Lördag 7/11        x=572  y=648  höger kolumn
Söndag 8/11        x=571  y=927  höger kolumn
Veckans soppa      x=570  y=1248 under söndag i höger kolumn
Veckans veget.     x=570  y=1350 under söndag i höger kolumn

HOVET/RESTAURANG   x=732  y=1842 w=240  h=113  färg #000
EOF
