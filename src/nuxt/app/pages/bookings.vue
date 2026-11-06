<script setup lang="ts">
import type { Appointment } from '~/composables/useAppointments'
import type { MatchiBooking } from '~/composables/useMatchiBookings'

const today = new Date().toISOString().slice(0, 10)

const from = ref(today)
const to = ref(addDays(today, 6))

const appointments = ref<Appointment[]>([])
const loading = ref(false)
const error = ref<string | null>(null)
const loaded = ref(false)
const search = ref('')

/** Bokningar som även är bokade i MATCHi, grupperade per kommunapptid. */
const matchiBookings = ref<Map<number, MatchiBooking[]>>(new Map())
const matchiLoading = ref(false)
const matchiError = ref<string | null>(null)

const timeZone = 'Europe/Stockholm'

const timeFormatter = new Intl.DateTimeFormat('sv-SE', {
  timeZone,
  hour: '2-digit',
  minute: '2-digit',
})

const dateFormatter = new Intl.DateTimeFormat('sv-SE', {
  timeZone,
  weekday: 'long',
  day: 'numeric',
  month: 'long',
})

function formatTime(value: string): string {
  return timeFormatter.format(new Date(value))
}

function formatDay(value: string): string {
  return dateFormatter.format(new Date(value))
}

/** Antal dagar i den valda perioden, inklusive båda dagarna. */
const selectedDays = computed(() => periodLengthDays(from.value, to.value))

/** API:t tillåter högst 31 dagars perioder. */
const maxPeriodDays = 31

const periodError = computed(() => {
  if (to.value < from.value) {
    return 'Slutdatumet måste vara efter startdatumet.'
  }
  if (selectedDays.value > maxPeriodDays) {
    return `Perioden får vara högst ${maxPeriodDays} dagar.`
  }
  return null
})

/** Bokningar grupperade per dag, med dagar utan bokningar utelämnade. */
const groupedAppointments = computed(() => {
  const groups = new Map<string, Appointment[]>()

  for (const appointment of filteredAppointments.value) {
    const day = appointment.start.slice(0, 10)
    const bucket = groups.get(day)
    if (bucket) {
      bucket.push(appointment)
    } else {
      groups.set(day, [appointment])
    }
  }

  return [...groups.entries()]
    .map(([day, items]) => ({
      day,
      label: formatDay(`${day}T12:00:00`),
      items: items.sort((a, b) => a.start.localeCompare(b.start)),
    }))
    .sort((a, b) => a.day.localeCompare(b.day))
})

/** Bokningar som matchar sökningen. */
const filteredAppointments = computed(() => {
  const term = search.value.trim().toLowerCase()
  if (term === '') {
    return appointments.value
  }

  return appointments.value.filter((appointment) =>
    [appointment.location, appointment.description, appointment.subject]
      .join(' ')
      .toLowerCase()
      .includes(term),
  )
})

const totalMinutes = computed(() =>
  filteredAppointments.value.reduce((sum, appointment) => sum + appointment.duration_minutes, 0),
)

/** Antal bokningar i urvalet som även är bokade i MATCHi. */
const matchiBookedCount = computed(
  () => filteredAppointments.value.filter((appointment) => matchiBookings.value.has(appointment.id)).length,
)

/** MATCHi-bokningarna som hör till en bokning i det kommunala systemet. */
function matchiBookingsFor(appointment: Appointment): MatchiBooking[] {
  return matchiBookings.value.get(appointment.id) ?? []
}

/** Om en bokning i det kommunala systemet även är bokad i MATCHi. */
function isBookedInMatchi(appointment: Appointment): boolean {
  return matchiBookings.value.has(appointment.id)
}

/** Tidsintervallet i MATCHi som hör till en bokning. */
function matchiRange(appointment: Appointment): string {
  return matchiBookingsFor(appointment)
    .map((booking) => `${formatTime(booking.matchi_start)}–${formatTime(booking.matchi_end)}`)
    .join(', ')
}

function selectPreset(days: number): void {
  from.value = today
  to.value = addDays(today, days - 1)
}

async function load(): Promise<void> {
  if (periodError.value !== null) {
    return
  }

  loading.value = true
  error.value = null

  try {
    const response = await fetchAppointments(from.value, to.value)
    appointments.value = response.appointments
    loaded.value = true
  } catch (cause: unknown) {
    error.value =
      cause instanceof Error
        ? cause.message
        : 'Kunde inte hämta bokningarna från det kommunala systemet.'
    appointments.value = []
  } finally {
    loading.value = false
  }

  await loadMatchi()
}

/**
 * Hämtar de bokningar som även är bokade i MATCHi.
 *
 * Anropet är skrivläsende - ingenting bokas i MATCHi. Ett fel här ska inte
 * stoppa listan från det kommunala systemet, så felet visas bara som en
 * notis.
 */
async function loadMatchi(): Promise<void> {
  matchiLoading.value = true
  matchiError.value = null

  if (appointments.value.length === 0) {
    matchiBookings.value = new Map()
    matchiLoading.value = false
    return
  }

  try {
    const response = await fetchMatchiBookings(from.value, to.value)
    const grouped = new Map<number, MatchiBooking[]>()

    for (const booking of response.booked) {
      const existing = grouped.get(booking.appointment_id)
      if (existing) {
        existing.push(booking)
      } else {
        grouped.set(booking.appointment_id, [booking])
      }
    }

    matchiBookings.value = grouped

    if (response.errors && response.errors.length > 0) {
      matchiError.value = `Vissa dagar kunde inte läsas från MATCHi: ${response.errors.join('; ')}`
    }
  } catch {
    matchiBookings.value = new Map()
    matchiError.value = 'Kunde inte hämta bokningarna från MATCHi.'
  } finally {
    matchiLoading.value = false
  }
}

onMounted(() => {
  void load()
})
</script>

<template>
  <main class="page">
    <header class="page__header">
      <h1>Bokningar</h1>
      <p class="page__lead">
        Välj en period och hämta alla bokningar från kommunens bokningssystem.
      </p>
      <p class="page__hint">
        Tider som också är bokade i MATCHi (Baldershallen) markeras grönt.
      </p>
    </header>

    <section class="panel">
      <form class="filters" @submit.prevent="load">
        <div class="filters__dates">
          <label class="field">
            <span class="field__label">Från</span>
            <input v-model="from" type="date" class="field__input" />
          </label>

          <label class="field">
            <span class="field__label">Till</span>
            <input v-model="to" type="date" class="field__input" />
          </label>
        </div>

        <div class="presets">
          <button type="button" class="chip" @click="selectPreset(1)">Idag</button>
          <button type="button" class="chip" @click="selectPreset(7)">7 dagar</button>
          <button type="button" class="chip" @click="selectPreset(14)">14 dagar</button>
          <button type="button" class="chip" @click="selectPreset(31)">31 dagar</button>
        </div>

        <div class="filters__actions">
          <span class="filters__hint">{{ selectedDays }} dag{{ selectedDays === 1 ? '' : 'ar' }}</span>
          <button type="submit" class="button" :disabled="loading || periodError !== null">
            {{ loading ? 'Hämtar…' : 'Hämta bokningar' }}
          </button>
        </div>
      </form>

      <p v-if="periodError" class="notice notice--error">{{ periodError }}</p>
      <p v-else-if="error" class="notice notice--error">
        {{ error }}
        <button type="button" class="notice__retry" :disabled="loading" @click="load">
          Försök igen
        </button>
      </p>
      <p v-else-if="matchiError" class="notice notice--warning">{{ matchiError }}</p>
      <p v-else-if="matchiLoading" class="notice">Jämför bokningarna med MATCHi…</p>
    </section>

    <section v-if="loaded || appointments.length > 0" class="panel">
      <div class="summary">
        <div class="summary__item">
          <span class="summary__value">{{ filteredAppointments.length }}</span>
          <span class="summary__label">bokningar</span>
        </div>
        <div class="summary__item">
          <span class="summary__value">{{ Math.round(totalMinutes / 60) }}</span>
          <span class="summary__label">bokade timmar</span>
        </div>
        <div class="summary__item">
          <span class="summary__value">{{ matchiBookedCount }}</span>
          <span class="summary__label">bokade i MATCHi</span>
        </div>
        <div class="summary__item summary__item--grow">
          <input
            v-model="search"
            type="search"
            class="field__input"
            placeholder="Filtrera på plats eller organisation"
          />
        </div>
      </div>

      <p v-if="filteredAppointments.length === 0" class="notice">
        Inga bokningar matchade filtret.
      </p>

      <article v-for="group in groupedAppointments" :key="group.day" class="day">
        <h2 class="day__title">{{ group.label }}</h2>
        <ul class="day__list">
          <li
            v-for="appointment in group.items"
            :key="appointment.id"
            class="booking"
            :class="{ 'booking--booked': isBookedInMatchi(appointment) }"
          >
            <span class="booking__time">
              {{ formatTime(appointment.start) }}–{{ formatTime(appointment.end) }}
            </span>
            <span class="booking__place">{{ appointment.location }}</span>
            <span class="booking__who">{{ appointment.description || '—' }}</span>
            <span class="booking__meta">
              <span
                v-if="isBookedInMatchi(appointment)"
                class="badge badge--matchi"
                :title="`Bokad i MATCHi ${matchiRange(appointment)}`"
              >
                Bokad i MATCHi
              </span>
              <span v-if="appointment.recurring" class="badge">Återkommande</span>
              <span v-if="!appointment.visible" class="badge badge--muted">Dold</span>
              <span v-if="appointment.css_class" class="badge badge--muted">{{ appointment.css_class }}</span>
            </span>
          </li>
        </ul>
      </article>
    </section>
  </main>
</template>

<style scoped>
.page {
  max-width: 60rem;
  margin: 0 auto;
  padding: 2.5rem 1.25rem 4rem;
  color: var(--color-text, #1f2933);
}

.page__header h1 {
  margin: 0 0 0.25rem;
  font-size: 1.75rem;
  letter-spacing: -0.01em;
}

.page__lead {
  margin: 0 0 0.25rem;
  color: var(--color-muted, #64748b);
}

.page__hint {
  margin: 0 0 1.5rem;
  color: var(--color-muted, #64748b);
  font-size: 0.875rem;
}

.panel {
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 0.75rem;
  padding: 1.25rem;
  margin-bottom: 1.25rem;
}

.filters {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.filters__dates {
  display: flex;
  flex-wrap: wrap;
  gap: 1rem;
}

.filters__actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
}

.filters__hint {
  color: var(--color-muted, #64748b);
  font-size: 0.875rem;
}

.field {
  display: flex;
  flex-direction: column;
  gap: 0.35rem;
}

.field__label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--color-muted, #64748b);
}

.field__input {
  padding: 0.5rem 0.65rem;
  border: 1px solid var(--color-border, #cbd5e1);
  border-radius: 0.5rem;
  background: var(--color-surface, #fff);
  color: inherit;
  font: inherit;
  min-width: 10rem;
}

.field__input:focus-visible {
  outline: 2px solid var(--color-accent, #2563eb);
  outline-offset: 1px;
}

.presets {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.chip {
  padding: 0.35rem 0.75rem;
  border: 1px solid var(--color-border, #cbd5e1);
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 0.875rem;
  cursor: pointer;
}

.chip:hover {
  border-color: var(--color-accent, #2563eb);
  color: var(--color-accent, #2563eb);
}

.button {
  padding: 0.55rem 1.1rem;
  border: 0;
  border-radius: 0.5rem;
  background: var(--color-accent, #2563eb);
  color: #fff;
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}

.button:disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.notice {
  margin: 1rem 0 0;
  padding: 0.65rem 0.85rem;
  border-radius: 0.5rem;
  background: var(--color-muted-bg, #f1f5f9);
  color: var(--color-muted, #475569);
  font-size: 0.9rem;
}

.notice--error {
  background: #fef2f2;
  color: #b91c1c;
}

.notice--warning {
  background: #fffbeb;
  color: #b45309;
}

.notice__retry {
  margin-left: 0.75rem;
  padding: 0.2rem 0.6rem;
  border: 1px solid currentColor;
  border-radius: 0.4rem;
  background: transparent;
  color: inherit;
  font: inherit;
  font-size: 0.85rem;
  cursor: pointer;
}

.notice__retry:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}

.summary {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 1.5rem;
  margin-bottom: 1.25rem;
}

.summary__item {
  display: flex;
  flex-direction: column;
}

.summary__item--grow {
  flex: 1 1 14rem;
}

.summary__value {
  font-size: 1.35rem;
  font-weight: 700;
}

.summary__label {
  font-size: 0.8125rem;
  color: var(--color-muted, #64748b);
}

.day {
  margin-top: 1.5rem;
}

.day__title {
  margin: 0 0 0.5rem;
  font-size: 1rem;
  text-transform: capitalize;
}

.day__list {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.booking {
  display: grid;
  grid-template-columns: 9.5rem 1fr 1fr auto;
  align-items: center;
  gap: 0.75rem;
  padding: 0.6rem 0.75rem;
  border: 1px solid var(--color-border, #e2e8f0);
  border-radius: 0.6rem;
  background: var(--color-surface, #fff);
}

.booking__time {
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

/* Tider som också är bokade i MATCHi. */
.booking--booked {
  border-color: #86efac;
  background: #f0fdf4;
}

.booking__place {
  font-weight: 500;
}

.booking__who {
  color: var(--color-muted, #64748b);
  font-size: 0.9rem;
}

.booking__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.35rem;
  justify-content: flex-end;
}

.badge {
  padding: 0.1rem 0.5rem;
  border-radius: 999px;
  background: #dbeafe;
  color: #1d4ed8;
  font-size: 0.75rem;
  white-space: nowrap;
}

.badge--muted {
  background: #f1f5f9;
  color: #64748b;
}

.badge--matchi {
  background: #dcfce7;
  color: #15803d;
}

@media (max-width: 40rem) {
  .booking {
    grid-template-columns: 1fr;
    gap: 0.25rem;
  }

  .booking__meta {
    justify-content: flex-start;
  }
}
</style>
