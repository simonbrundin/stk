/**
 * Klient för MATCHi-endpointen i Go API:t.
 *
 * Endpointen jämför det kommunala bokningssystemet med MATCHi och returnerar
 * de bokningar som finns i båda systemen. Anropet är skrivläsande - det
 * bokar aldrig något i MATCHi.
 */

/** En bokning som finns i både det kommunala systemet och MATCHi. */
export interface MatchiBooking {
  /** Id:t för bokningen i det kommunala bokningssystemet. */
  appointment_id: number
  /** Starttid i lokal tid (Europe/Stockholm). */
  start: string
  /** Sluttid i lokal tid (Europe/Stockholm). */
  end: string
  /** Resursen i det kommunala bokningssystemet, t.ex. "Baldershallen 1". */
  location?: string
  /** MATCHi-banan eller salen, t.ex. "1. Idrottshall". */
  court_name: string
  /** Starttid för bokningen i MATCHi. */
  matchi_start: string
  /** Sluttid för bokningen i MATCHi. */
  matchi_end: string
  /** Information om bokningen i MATCHi, t.ex. kundnamn. */
  matchi_title?: string
  /** MATCHis boknings-id. */
  matchi_booking_id?: string
}

/** Svaret från jämförelsen. */
export interface MatchiBookingListResponse {
  from: string
  to: string
  count: number
  booked: MatchiBooking[]
  /** Meddelanden om dagar som inte kunde läsas från MATCHi. */
  errors?: string[]
}

/**
 * Hämtar de bokningar som även är bokade i MATCHi.
 *
 * API:t behandlar `to` som en exklusiv gräns, så periodens sista dag
 * skickas som dagen efter. Då ingår hela slutdagen i resultatet.
 */
export async function fetchMatchiBookings(from: string, to: string): Promise<MatchiBookingListResponse> {
  return await $fetch<MatchiBookingListResponse>('/api/matchi/bookings', {
    query: {
      from,
      to: addDays(to, 1),
    },
  })
}
