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
 * Miljövariabler:
 *   LUNCH_RESTAURANT_ID  - restaurangens id (default 729)
 *   LUNCH_DAYS           - antal dagar framåt som ska ingå (default 3)
 *   LUNCH_OUTPUT         - sökväg till JSON-filen (default app/data/lunch.json)
 */

import { mkdir, writeFile } from 'node:fs/promises'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const scriptDir = dirname(fileURLToPath(import.meta.url))
const nuxtDir = resolve(scriptDir, '..')

const RESTAURANT_ID = Number(process.env.LUNCH_RESTAURANT_ID ?? 729)
const DAYS_AHEAD = Number(process.env.LUNCH_DAYS ?? 3)
const OUTPUT = resolve(nuxtDir, process.env.LUNCH_OUTPUT ?? 'app/data/lunch.json')

const TIME_ZONE = 'Europe/Stockholm'
const API_URL = 'https://www.matochmat.se/rest/v3/menus'
const RESTAURANT_URL =
  'https://www.matochmat.se/restauranger/sundsvall/lunch/hovet-restaurang-och-cafe/'

/** Nycklarna matochmat.se använder för veckans dagar. */
const WEEKDAY_KEYS = [
  'mandag',
  'tisdag',
  'onsdag',
  'torsdag',
  'fredag',
  'lordag',
  'sondag',
]

/** Vardagarna som de skrivs på svenska. */
const WEEKDAY_NAMES = [
  'måndag',
  'tisdag',
  'onsdag',
  'torsdag',
  'fredag',
  'lördag',
  'söndag',
]

/** En rätt i menyn. */
interface Dish {
  name: string
  description: string
  price: string
  tags: string[]
}

/** En dag i menyn. */
interface MenuDay {
  date: string
  weekday: string
  isToday: boolean
  isPublished: boolean
  dishes: Dish[]
}

/** Filen som skrivs till disk. */
interface LunchMenu {
  restaurant: {
    id: number
    name: string
    url: string
  }
  timeZone: string
  generatedAt: string
  days: MenuDay[]
}

/** Ett svar från matochmat.se. */
interface ApiResponse {
  status: boolean
  data?: Array<{
    content: string
    isDefaultMenu: boolean
    lastUpdated: string
    restaurantId: number
    week: number
    year: number
  }>
}

/**
 * Returnerar dagens datum (YYYY-MM-DD) i tidszonen anläggningen ligger i.
 */
function todayInTimeZone(): string {
  const formatter = new Intl.DateTimeFormat('sv-SE', {
    timeZone: TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })

  return formatter.format(new Date())
}

/**
 * Räknar ut ISO-veckans nummer och år för ett datum.
 */
function isoWeek(date: Date): { week: number; year: number } {
  const thursday = new Date(
    Date.UTC(date.getUTCFullYear(), date.getUTCMonth(), date.getUTCDate()),
  )
  const weekday = (thursday.getUTCDay() + 6) % 7
  thursday.setUTCDate(thursday.getUTCDate() - weekday + 3)

  const isoYear = thursday.getUTCFullYear()
  const firstThursday = new Date(Date.UTC(isoYear, 0, 4))
  const firstWeekday = (firstThursday.getUTCDay() + 6) % 7
  firstThursday.setUTCDate(firstThursday.getUTCDate() - firstWeekday + 3)

  const week =
    1 + Math.round((thursday.getTime() - firstThursday.getTime()) / (7 * 86_400_000))

  return { week, year: isoYear }
}

/**
 * Lägger till dagar på ett datum (YYYY-MM-DD).
 */
function addDays(date: string, days: number): string {
  const parsed = new Date(`${date}T00:00:00Z`)
  parsed.setUTCDate(parsed.getUTCDate() + days)

  return parsed.toISOString().slice(0, 10)
}

/**
 * Hämtar menyn för en ISO-vecka.
 */
interface WeekMenu {
  days: Map<string, Dish[]>
  isDefaultMenu: boolean
}

async function fetchWeek(week: number, year: number): Promise<WeekMenu> {
  const url = `${API_URL}?restaurant=${RESTAURANT_ID}&week=${week}&year=${year}`
  const response = await fetch(url, {
    headers: { Accept: 'application/json' },
  })

  if (!response.ok) {
    throw new Error(`HTTP ${response.status} för ${url}`)
  }

  const payload = (await response.json()) as ApiResponse
  if (!payload.status || !payload.data || payload.data.length === 0) {
    return { days: new Map(), isDefaultMenu: true }
  }

  const entry = payload.data[0]
  const content = JSON.parse(entry.content) as Record<string, Dish[]>

  const days = new Map<string, Dish[]>()
  for (const [key, dishes] of Object.entries(content)) {
    days.set(key, dishes.map((dish) => ({
      name: dish.name ?? '',
      description: dish.description ?? '',
      price: dish.price ?? '',
      tags: dish.tags ?? [],
    })))
  }

  return { days, isDefaultMenu: entry.isDefaultMenu }
}

/**
 * Bygger menyn för de dagar som ska visas.
 */
async function buildMenu(): Promise<LunchMenu> {
  const today = todayInTimeZone()
  const dates: string[] = []
  for (let offset = 0; offset <= DAYS_AHEAD; offset += 1) {
    dates.push(addDays(today, offset))
  }

  const weeks = new Map<string, { week: number; year: number }>()
  for (const date of dates) {
    const { week, year } = isoWeek(new Date(`${date}T00:00:00Z`))
    weeks.set(`${week}-${year}`, { week, year })
  }

  const menus = new Map<string, WeekMenu>()
  for (const { week, year } of weeks.values()) {
    menus.set(`${week}-${year}`, await fetchWeek(week, year))
  }

  const days: MenuDay[] = dates.map((date) => {
    const { week, year } = isoWeek(new Date(`${date}T00:00:00Z`))
    const weekdayIndex = (new Date(`${date}T00:00:00Z`).getUTCDay() + 6) % 7
    const key = WEEKDAY_KEYS[weekdayIndex]

    const weekMenu = menus.get(`${week}-${year}`)
    const dishes = weekMenu?.days.get(key) ?? []

    return {
      date,
      weekday: WEEKDAY_NAMES[weekdayIndex],
      isToday: date === today,
      // En standardmeny innehåller bara lunchbuffén, vilket betyder att
      // den riktiga menyn för veckan ännu inte är publicerad.
      isPublished: dishes.length > 0 && weekMenu?.isDefaultMenu === false,
      dishes,
    }
  })

  return {
    restaurant: {
      id: RESTAURANT_ID,
      name: 'Hovet Restaurang & Café',
      url: RESTAURANT_URL,
    },
    timeZone: TIME_ZONE,
    generatedAt: new Date().toISOString(),
    days,
  }
}

async function main(): Promise<void> {
  const menu = await buildMenu()

  await mkdir(dirname(OUTPUT), { recursive: true })
  await writeFile(OUTPUT, `${JSON.stringify(menu, null, 2)}\n`, 'utf8')

  const published = menu.days.filter((day) => day.isPublished).length
  console.log(
    `Skrev ${OUTPUT} (${menu.days.length} dagar, ${published} publicerade)`,
  )
}

main().catch((error: unknown) => {
  console.error('fetch-lunch misslyckades:', error)
  process.exit(1)
})
