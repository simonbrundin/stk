/**
 * Läsning av lunchmenyn som är hämtad vid byggtid.
 *
 * Menyn skrapas från matochmat.se av scripts/fetch-lunch.ts och skrivs till
 * app/data/lunch.json. Sidan importerar filen, vilket gör att lunchsidan
 * fungerar som en ren statisk sida (t.ex. på GitHub Pages).
 */

import rawMenu from '~/data/lunch.json'
import type { LunchDay, LunchDish, LunchMenu } from '~/utils/lunchSource'

export type { LunchDay, LunchDish, LunchMenu }

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
 * En veckas layout på en skärm av storleken 1080x1920 px.
 */
export interface WeekLayout {
  /** Måndag–torsdag (vänster kolumn). */
  left: LunchDay[]
  /** Fredag–söndag (höger kolumn). */
  right: LunchDay[]
  /** Veckans soppa, om den finns. */
  soup: LunchDish | null
  /** Veckans vegetariska rätt, om den finns. */
  vegetarian: LunchDish | null
}

/** Antal dagar i vänster kolumn (måndag–torsdag). */
export const DAYS_IN_LEFT_COLUMN = 4

/** Rätter som hör till hela veckan och därför visas längst ner. */
const WEEKLY_HIGHLIGHT = /soppa|vegetarisk/i

/** Lunchbuffén visas inte i menyn. */
const BUFFET = /lunchbuff/i

/**
 * Returnerar den första rätten i veckan vars namn matchar mönstret.
 */
function findWeeklyDish(days: LunchDay[], pattern: RegExp): LunchDish | null {
  for (const day of days) {
    const dish = day.dishes.find((candidate) => pattern.test(candidate.name))
    if (dish) {
      return dish
    }
  }

  return null
}

/**
 * Bygger veckans layout.
 *
 * Veckans dagar delas i två kolumner (måndag–torsdag till vänster,
 * fredag–söndag till höger). Soppa och vegetarisk rätt plockas bort ur
 * dagarna och visas i stället längst ner på sidan.
 */
export function buildWeekLayout(days: LunchDay[]): WeekLayout {
  const soup = findWeeklyDish(days, /soppa/i)
  const vegetarian = findWeeklyDish(days, /vegetarisk/i)

  const cleaned = days.map((day) => ({
    ...day,
    dishes: day.dishes.filter(
      (dish) => !BUFFET.test(dish.name) && !WEEKLY_HIGHLIGHT.test(dish.name),
    ),
  }))

  return {
    left: cleaned.slice(0, DAYS_IN_LEFT_COLUMN),
    right: cleaned.slice(DAYS_IN_LEFT_COLUMN, 7),
    soup,
    vegetarian,
  }
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
