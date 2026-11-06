/**
 * Hämtar lunchmenyn för Hovet Restaurang & Café från Mat och Mat.
 *
 * Menyn publiceras veckovis i matochmat.se:s REST-API:
 *
 *   GET https://www.matochmat.se/rest/v3/menus?restaurant=<id>&week=<vecka>&year=<år>
 *
 * Svaret innehåller en veckas rätter per vardag (måndag..söndag).
 * Skriptet skriver resultatet till app/data/lunch.json, som sidan /lunch
 * importerar. Därför behöver den statiska sidan ingen backend.
 *
 * Hämtningslogiken ligger i app/utils/lunchSource.ts så att startsidan kan
 * hämta samma data direkt i webbläsaren.
 *
 * Miljövariabler:
 *   LUNCH_RESTAURANT_ID  - restaurangens id (default 729)
 *   LUNCH_DAYS           - antal dagar i menyn (default 7 = hela veckan)
 *   LUNCH_OUTPUT         - sökväg till JSON-filen (default app/data/lunch.json)
 */

import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import { buildLunchMenu } from '../app/utils/lunchSource'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const nuxtDir = resolve(scriptDir, '..')

const RESTAURANT_ID = Number(process.env.LUNCH_RESTAURANT_ID ?? 729)
const DAYS_AHEAD = Number(process.env.LUNCH_DAYS ?? 7)
const OUTPUT = resolve(nuxtDir, process.env.LUNCH_OUTPUT ?? 'app/data/lunch.json')

async function main(): Promise<void> {
  const menu = await buildLunchMenu({
    restaurantId: RESTAURANT_ID,
    days: DAYS_AHEAD,
  })

  await mkdir(dirname(OUTPUT), { recursive: true })
  await writeFile(OUTPUT, `${JSON.stringify(menu, null, 2)}\n`, 'utf8')

  const published = menu.days.filter((day) => day.isPublished).length
  console.log(
    `Skrev ${OUTPUT} (${menu.days.length} dagar, ${published} publicerade)`,
  )
}

main().catch((error: unknown) => {
  console.error('Kunde inte hämta lunchmenyn:', error)
  process.exitCode = 1
})
