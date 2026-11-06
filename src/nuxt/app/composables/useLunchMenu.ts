/**
 * Läsning av lunchmenyn som är hämtad vid byggtid.
 *
 * Menyn skrapas från matochmat.se av scripts/fetch-lunch.ts och skrivs till
 * app/data/lunch.json. Sidan importerar filen, vilket gör att lunchsidan
 * fungerar som en ren statisk sida (t.ex. på GitHub Pages).
 */

import rawMenu from '~/data/lunch.json'

/** En rätt i lunchmenyn. */
export interface LunchDish {
  name: string
  description: string
  price: string
  tags: string[]
}

/** En dag i lunchmenyn. */
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

/** Hela lunchmenyn. */
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

/**
 * Returnerar lunchmenyn.
 */
export function useLunchMenu(): LunchMenu {
  return rawMenu as unknown as LunchMenu
}

/**
 * Formaterar en dag som "Måndag 2/11".
 */
export function formatMenuDay(date: string, weekday: string): string {
  const [, month, day] = date.split('-')
  const name = weekday.charAt(0).toUpperCase() + weekday.slice(1)

  return `${name} ${Number(day)}/${Number(month)}`
}

/**
 * Räknar ut ISO-veckans nummer för ett datum.
 */
export function isoWeekNumber(date: string): number {
  const parsed = new Date(`${date}T00:00:00Z`)
  const thursday = new Date(parsed.getTime())
  const weekday = (thursday.getUTCDay() + 6) % 7

  thursday.setUTCDate(thursday.getUTCDate() - weekday + 3)

  const firstThursday = new Date(Date.UTC(thursday.getUTCFullYear(), 0, 4))
  const firstWeekday = (firstThursday.getUTCDay() + 6) % 7
  firstThursday.setUTCDate(firstThursday.getUTCDate() - firstWeekday + 3)

  return 1 + Math.round((thursday.getTime() - firstThursday.getTime()) / (7 * 86_400_000))
}

/**
 * Antal dagar som ryms på en skärm av storleken 1080x1920 px.
 *
 * Mockupen visar tre dagar på den första skärmen. Fler dagar pressar
 * innehållet för nära logotypen.
 */
export const DAYS_PER_SCREEN = 3

/**
 * Delar upp veckans dagar på skärmar.
 *
 * Dagarna sprids jämnt över skärmarna, så att en vecka med sju dagar
 * blir tre skärmar med tre, två och två dagar.
 */
export function groupDaysIntoScreens(days: LunchDay[]): LunchDay[][] {
  if (days.length === 0) {
    return []
  }

  const screenCount = Math.ceil(days.length / DAYS_PER_SCREEN)
  const perScreen = Math.floor(days.length / screenCount)
  const remainder = days.length % screenCount
  const screens: LunchDay[][] = []

  let index = 0
  for (let screen = 0; screen < screenCount; screen += 1) {
    const size = perScreen + (screen < remainder ? 1 : 0)
    screens.push(days.slice(index, index + size))
    index += size
  }

  return screens
}

/**
 * Formaterar tidpunkten menyn hämtades som "6 november 2026 kl. 13:45".
 */
export function formatGeneratedAt(value: string): string {
  return new Intl.DateTimeFormat('sv-SE', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Europe/Stockholm',
  }).format(new Date(value))
}
