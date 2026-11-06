/**
 * Klient för bokningsendpointen i Go API:t.
 *
 * Endpointen svarar med normaliserade bokningar där tiderna är i
 * lokal tid (Europe/Stockholm), serialiserade som RFC3339.
 */

/** En resurs (sal eller bana) som en bokning gäller. */
export interface AppointmentResource {
  id: number
  name: string
}

/** En bokning i det kommunala bokningssystemet. */
export interface Appointment {
  id: number
  encoded_id: string
  subject: string
  location: string
  description: string
  start: string
  end: string
  duration_minutes: number
  visible: boolean
  recurring: boolean
  css_class?: string
  resources: AppointmentResource[]
}

/** Svaret från listendpointen. */
export interface AppointmentListResponse {
  from: string
  to: string
  count: number
  appointments: Appointment[]
}

/** Periodens längd i dagar, räknat inklusive båda dagarna. */
export function periodLengthDays(from: string, to: string): number {
  const start = new Date(`${from}T00:00:00`)
  const end = new Date(`${to}T00:00:00`)
  return Math.round((end.getTime() - start.getTime()) / 86_400_000) + 1
}

/**
 * Lägger till en dag på ett datum i formatet YYYY-MM-DD.
 *
 * Datumet räknas som väggklocka i Europe/Stockholm, precis som
 * bokningssystemet gör.
 */
export function addDays(date: string, days: number): string {
  const parsed = new Date(`${date}T00:00:00`)
  parsed.setDate(parsed.getDate() + days)
  const year = parsed.getFullYear()
  const month = `${parsed.getMonth() + 1}`.padStart(2, '0')
  const day = `${parsed.getDate()}`.padStart(2, '0')
  return `${year}-${month}-${day}`
}

/** Sökvägen till bokningsendpointen, relativt så att nginx kan proxy:a den. */
export const APPOINTMENTS_PATH = '/api/actorsmartbook/appointments'

/** Antal försök när API:t svarar med ett tillfälligt fel. */
const MAX_ATTEMPTS = 3

/** Ett fel från API:t med information som går att visa för användaren. */
export class AppointmentsApiError extends Error {
  /** HTTP-statuskod, eller null om API:t aldrig svarade. */
  readonly status: number | null

  /** URL som anropades. */
  readonly url: string

  constructor(message: string, status: number | null, url: string) {
    super(message)
    this.name = 'AppointmentsApiError'
    this.status = status
    this.url = url
  }
}

function delay(milliseconds: number): Promise<void> {
  return new Promise((resolve) => setTimeout(resolve, milliseconds))
}

function statusOf(cause: unknown): number | null {
  if (cause !== null && typeof cause === 'object' && 'status' in cause) {
    const status = (cause as { status?: unknown }).status
    if (typeof status === 'number') {
      return status
    }
  }
  return null
}

/** Ett fel är tillfälligt om API:t inte svarade alls eller svarade med 5xx. */
function isRetryable(cause: unknown): boolean {
  const status = statusOf(cause)
  return status === null || status >= 500
}

/** Bygger ett läsbart felmeddelande ur ett $fetch-fel. */
function describe(cause: unknown, url: string): AppointmentsApiError {
  const status = statusOf(cause)
  const payload =
    cause !== null && typeof cause === 'object' && 'data' in cause
      ? (cause as { data?: unknown }).data
      : undefined

  if (payload !== null && typeof payload === 'object' && 'error' in payload) {
    const message = String((payload as { error: unknown }).error)
    if (message !== '') {
      return new AppointmentsApiError(message, status, url)
    }
  }

  if (status !== null) {
    return new AppointmentsApiError(`API:t svarade med HTTP ${status} (${url}).`, status, url)
  }

  const reason =
    cause !== null && typeof cause === 'object' && 'message' in cause
      ? String((cause as { message: unknown }).message)
      : 'okänt fel'
  return new AppointmentsApiError(`Kunde inte nå API:t på ${url} (${reason}).`, null, url)
}

/**
 * Hämtar bokningar för en period.
 *
 * API:t behandlar `to` som en exklusiv gräns, så periodens sista dag
 * skickas som dagen efter. Då ingår hela slutdagen i resultatet.
 *
 * Tillfälliga fel (nätverksproblem eller 5xx, till exempel medan en podd
 * startar om i klustret) provas om innan felet rapporteras.
 */
export async function fetchAppointments(from: string, to: string): Promise<AppointmentListResponse> {
  const query = { from, to: addDays(to, 1) }

  let lastError: unknown
  for (let attempt = 1; attempt <= MAX_ATTEMPTS; attempt++) {
    try {
      return await $fetch<AppointmentListResponse>(APPOINTMENTS_PATH, { query })
    } catch (cause: unknown) {
      lastError = cause
      if (!isRetryable(cause) || attempt === MAX_ATTEMPTS) {
        break
      }
      await delay(1000 * attempt)
    }
  }

  throw describe(lastError, APPOINTMENTS_PATH)
}
