/**
 * Hämtar lunchmenyn direkt från matochmat.se i webbläsaren.
 *
 * API:t skickar "access-control-allow-origin: *", så hämtningen fungerar
 * utan backend - även på den statiska sajten på GitHub Pages. Det gör att
 * startsidan alltid kan visa den senast publicerade menyn, utan att den
 * statiska sajten behöver byggas om.
 *
 * Samma hämtningslogik används av byggskriptet (scripts/fetch-lunch.ts).
 */

import type { ComputedRef } from 'vue'

import { buildLunchMenu, isoWeekNumber, isoWeekOf } from '~/utils/lunchSource'
import type { LunchDay } from '~/utils/lunchSource'

interface LiveLunchState {
  loading: boolean
  error: string | null
  days: LunchDay[]
  week: number
  year: number
  weekOffset: number
  fetchedAt: string | null
}

export interface LiveLunchMenu {
  /** Laddar just nu. */
  loading: ComputedRef<boolean>
  /** Felmeddelande om hämtningen misslyckades. */
  error: ComputedRef<string | null>
  /** Dagarna i den hämtade veckan. */
  days: ComputedRef<LunchDay[]>
  /** ISO-veckans nummer. */
  week: ComputedRef<number>
  /** ISO-året. */
  year: ComputedRef<number>
  /** Antal veckor förskjutning från den aktuella veckan. */
  weekOffset: ComputedRef<number>
  /** När menyn senast hämtades, ISO 8601. */
  fetchedAt: ComputedRef<string | null>
  /** Hämtar menyn för den valda veckan. */
  refresh: (weekOffset?: number) => Promise<void>
  /** Visar föregående vecka. */
  previousWeek: () => Promise<void>
  /** Visar nästa vecka. */
  nextWeek: () => Promise<void>
}

/**
 * Returnerar en reaktiv meny som kan hämtas om från webbläsaren.
 */
export function useLiveLunchMenu(): LiveLunchMenu {
  const state = useState<LiveLunchState>('live-lunch-menu', () => ({
    loading: false,
    error: null,
    days: [],
    week: 0,
    year: 0,
    weekOffset: 0,
    fetchedAt: null,
  }))

  /**
   * Hämtar menyn för den valda veckan.
   */
  async function refresh(weekOffset = state.value.weekOffset): Promise<void> {
    state.value.loading = true
    state.value.error = null

    try {
      const result = await buildLunchMenu({ weekOffset })
      const first = result.days[0]

      state.value.days = result.days
      state.value.week = first ? isoWeekNumber(first.date) : 0
      state.value.year = first ? isoWeekOf(first.date).year : 0
      state.value.weekOffset = weekOffset
      state.value.fetchedAt = result.generatedAt
    } catch (error: unknown) {
      state.value.error =
        error instanceof Error ? error.message : 'Kunde inte hämta menyn'
    } finally {
      state.value.loading = false
    }
  }

  return {
    loading: computed(() => state.value.loading),
    error: computed(() => state.value.error),
    days: computed(() => state.value.days),
    week: computed(() => state.value.week),
    year: computed(() => state.value.year),
    weekOffset: computed(() => state.value.weekOffset),
    fetchedAt: computed(() => state.value.fetchedAt),
    refresh,
    previousWeek: () => refresh(state.value.weekOffset - 1),
    nextWeek: () => refresh(state.value.weekOffset + 1),
  }
}
