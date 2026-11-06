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
  /** Veckedagen på svenska, t.ex. "fredag". */
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
 * Formaterar ett datum som "fredag 6 november".
 */
export function formatMenuDate(date: string): string {
  const formatted = new Intl.DateTimeFormat('sv-SE', {
    weekday: 'long',
    day: 'numeric',
    month: 'long',
    timeZone: 'UTC',
  }).format(new Date(`${date}T00:00:00Z`))

  return formatted.charAt(0).toUpperCase() + formatted.slice(1)
}

/**
 * Formaterar tidpunkten menyn hämtades som "6 november 2026 kl. 13:45".
 */
export function formatGeneratedAt(value: string): string {
  const formatted = new Intl.DateTimeFormat('sv-SE', {
    day: 'numeric',
    month: 'long',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    timeZone: 'Europe/Stockholm',
  }).format(new Date(value))

  return formatted
}
