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

/**
 * Hämtar bokningar för en period.
 *
 * API:t behandlar `to` som en exklusiv gräns, så periodens sista dag
 * skickas som dagen efter. Då ingår hela slutdagen i resultatet.
 */
export async function fetchAppointments(from: string, to: string): Promise<AppointmentListResponse> {
  return await $fetch<AppointmentListResponse>('/api/actorsmartbook/appointments', {
    query: {
      from,
      to: addDays(to, 1),
    },
  })
}
