/**
 * Hämtning av lunchmenyn från Mat och Mat.
 *
 * Logiken ligger här så att både byggskriptet (scripts/fetch-lunch.ts) och
 * webbläsaren (startsidan) använder exakt samma tolkning av API:t.
 *
 * API:t:
 *
 *   GET https://www.matochmat.se/rest/v3/menus?restaurant=<id>&week=<vecka>&year=<år>
 *
 * Svaret innehåller en veckas rätter per vardag (måndag..söndag).
 */

export interface LunchDish {
  name: string
  description: string
  price: string
  tags: string[]
}

export interface LunchDay {
  /** Datumet i formatet YYYY-MM-DD. */
  date: string
  /** Veckedagen på svenska, t.ex. "måndag". */
  weekday: string
  /** Sant om dagen är idag. */
  isToday: boolean
  /** Sant om restaurangen har publicerat menyn för dagen. */
  isPublished: boolean
  /** Rätterna som serveras. */
  dishes: LunchDish[]
}

export interface LunchMenu {
  restaurant: {
    id: number
    name: string
    url: string
    address: string
  }
  timeZone: string
  /** När menyn senast hämtades, ISO 8601. */
  generatedAt: string
  days: LunchDay[]
}

export const LUNCH_API_URL = 'https://www.matochmat.se/rest/v3/menus'
export const LUNCH_RESTAURANT_ID = 729
export const LUNCH_RESTAURANT_URL =
  'https://www.matochmat.se/restauranger/sundsvall/lunch/hovet-restaurang-och-cafe/'
export const LUNCH_RESTAURANT_NAME = 'Hovet Restaurang & Café'
export const LUNCH_RESTAURANT_ADDRESS = 'Lasarettsvägen 22, Sundsvall'
export const LUNCH_TIME_ZONE = 'Europe/Stockholm'

/** Lunchbuffén hör inte till menyn och filtreras bort. */
const BUFFET = /lunchbuff/i

/** Nycklarna matochmat.se använder för veckans dagar. */
export const WEEKDAY_KEYS = [
  'mandag',
  'tisdag',
  'onsdag',
  'torsdag',
  'fredag',
  'lordag',
  'sondag',
]

/** Vardagarna som de skrivs på svenska. */
export const WEEKDAY_NAMES = [
  'måndag',
  'tisdag',
  'onsdag',
  'torsdag',
  'fredag',
  'lördag',
  'söndag',
]

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

export interface WeekMenu {
  days: Map<string, LunchDish[]>
  isDefaultMenu: boolean
}

/**
 * Returnerar dagens datum (YYYY-MM-DD) i tidszonen anläggningen ligger i.
 */
export function todayInTimeZone(): string {
  const formatter = new Intl.DateTimeFormat('sv-SE', {
    timeZone: LUNCH_TIME_ZONE,
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
  })

  return formatter.format(new Date())
}

/**
 * Returnerar måndagen i den ISO-vecka datumet tillhör.
 */
export function mondayOf(date: string): string {
  const parsed = new Date(`${date}T00:00:00Z`)
  const weekday = (parsed.getUTCDay() + 6) % 7

  parsed.setUTCDate(parsed.getUTCDate() - weekday)

  return parsed.toISOString().slice(0, 10)
}

/**
 * Lägger till dagar på ett datum (YYYY-MM-DD).
 */
export function addDays(date: string, days: number): string {
  const parsed = new Date(`${date}T00:00:00Z`)
  parsed.setUTCDate(parsed.getUTCDate() + days)

  return parsed.toISOString().slice(0, 10)
}

/**
 * Räknar ut ISO-veckans nummer och år för ett datum.
 */
export function isoWeekOf(date: string): { week: number; year: number } {
  const parsed = new Date(`${date}T00:00:00Z`)
  const thursday = new Date(parsed.getTime())
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
 * Räknar ut ISO-veckans nummer för ett datum.
 */
export function isoWeekNumber(date: string): number {
  return isoWeekOf(date).week
}

/**
 * Hämtar menyn för en ISO-vecka.
 */
export async function fetchWeekMenu(
  week: number,
  year: number,
  restaurantId: number = LUNCH_RESTAURANT_ID,
): Promise<WeekMenu> {
  const url = `${LUNCH_API_URL}?restaurant=${restaurantId}&week=${week}&year=${year}`
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
  const content = JSON.parse(entry.content) as Record<string, ApiDish[]>

  const days = new Map<string, LunchDish[]>()
  for (const [key, dishes] of Object.entries(content)) {
    days.set(
      key,
      dishes
        // Lunchbuffén visas inte i menyn och tas bort redan vid hämtningen.
        .filter((dish) => !BUFFET.test(dish.name ?? ''))
        .map((dish) => ({
          name: dish.name ?? '',
          description: dish.description ?? '',
          price: dish.price ?? '',
          tags: dish.tags ?? [],
        })),
    )
  }

  return { days, isDefaultMenu: entry.isDefaultMenu }
}

interface ApiDish {
  name?: string
  description?: string
  price?: string
  tags?: string[]
}

export interface BuildLunchMenuOptions {
  /** Restaurangens id. */
  restaurantId?: number
  /** Antal dagar i menyn (7 = hela veckan). */
  days?: number
  /** Antal veckor att förskjuta veckostarten (0 = denna vecka, 1 = nästa). */
  weekOffset?: number
  /** Dagens datum, används i tester. */
  today?: string
}

/**
 * Bygger menyn för de dagar som ska visas.
 */
export async function buildLunchMenu(
  options: BuildLunchMenuOptions = {},
): Promise<LunchMenu> {
  const {
    restaurantId = LUNCH_RESTAURANT_ID,
    days = 7,
    weekOffset = 0,
    today = todayInTimeZone(),
  } = options

  const monday = addDays(mondayOf(today), weekOffset * 7)
  const dates: string[] = []
  for (let offset = 0; offset < days; offset += 1) {
    dates.push(addDays(monday, offset))
  }

  const weeks = new Map<string, { week: number; year: number }>()
  for (const date of dates) {
    const { week, year } = isoWeekOf(date)
    weeks.set(`${week}-${year}`, { week, year })
  }

  const menus = new Map<string, WeekMenu>()
  for (const { week, year } of weeks.values()) {
    menus.set(`${week}-${year}`, await fetchWeekMenu(week, year, restaurantId))
  }

  const lunchDays: LunchDay[] = dates.map((date) => {
    const { week, year } = isoWeekOf(date)
    const weekdayIndex = (new Date(`${date}T00:00:00Z`).getUTCDay() + 6) % 7
    const key = WEEKDAY_KEYS[weekdayIndex]

    const weekMenu = menus.get(`${week}-${year}`)
    const dishes = weekMenu?.days.get(key) ?? []

    return {
      date,
      weekday: WEEKDAY_NAMES[weekdayIndex],
      isToday: date === today,
      isPublished: dishes.length > 0 && weekMenu?.isDefaultMenu === false,
      dishes,
    }
  })

  return {
    restaurant: {
      id: restaurantId,
      name: LUNCH_RESTAURANT_NAME,
      url: LUNCH_RESTAURANT_URL,
      address: LUNCH_RESTAURANT_ADDRESS,
    },
    timeZone: LUNCH_TIME_ZONE,
    generatedAt: new Date().toISOString(),
    days: lunchDays,
  }
}
