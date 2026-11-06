#!/usr/bin/env bash
# Renderar den byggda lunchsidan och jämför geometrin med mockupen.
#
#   ./scripts/compare-mockup.sh [mockup.png]
#
# Mockupen (1080x1920) mättes med OCR + pixelanays. Måtten nedan är
# uttag ur mockupen och används som facit.
set -euo pipefail

cd "$(dirname "$0")/.."

mockup="${1:-/home/simon/Downloads/Meny skärm Hovet/1.png}"
out=/tmp/lunch-shot.png

NUXT_PAGES_BUILD=true NUXT_APP_BASE_URL=/ bun run generate >/dev/null 2>&1

(cd .output/public && python3 -m http.server 8123 >/dev/null 2>&1 &)
sleep 1

chromium --headless --disable-gpu --no-sandbox --hide-scrollbars \
	--force-device-scale-factor=1 --window-size=1080,1920 \
	--screenshot="${out}" http://localhost:8123/lunch/ 2>/dev/null

echo "=== MITT PÅ EGEN RENDERING ==="
tesseract "${out}" stdout -l eng --psm 6 tsv 2>/dev/null |
	awk -F'\t' '$12!="" {printf "%-14s x=%-5s y=%-5s w=%-5s h=%-4s\n",$12,$7,$8,$9,$10}' |
	grep -iE "Lunchmeny|^v[0-9]+|M.ndag|Tisdag|Onsdag|Torsdag|Fredag|L.rdag|S.ndag|Lunchbuff|Hovet$|RESTAURANG|Lasarett|HOVET"

echo
echo "=== FACIT (mockup) ==="
cat <<'EOF'
Lunchmeny      x=282  y=88   w=360  h=80   färg #767477
V45            x=680  y=88   w=120  h=62   färg #767477
Måndag 2/11    x=183  y=246  w=139  h=27   färg #000
Lunchbuffé     x=183  y=301  w=98   h=15
beskrivning    x=183  y=325  w=~470 h=19   färg #767477
HOVET          x=574  y=1780 w=376  h=65   färg #000
EOF
