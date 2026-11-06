/**
 * Klient för kopplingen mellan kommunresurser och MATCHi-banor.
 *
 * Kopplingen ligger i databasen och används för att veta vilken MATCHi-bana
 * en bokning i det kommunala systemet ska bokas på.
 */

/** En koppling mellan en resurs i det kommunala systemet och en MATCHi-bana. */
export interface CourtMapping {
  /** Resursens id i det kommunala bokningssystemet. */
  kommun_resource_id: number
  /** Resursens namn i det kommunala bokningssystemet. */
  kommun_resource_name: string
  /** MATCHis id för banan eller salen. */
  matchi_court_id: number
  /** MATCHis namn för banan eller salen. */
  matchi_court_name: string
}

/** Hämtar alla kopplingar mellan kommunresurser och MATCHi-banor. */
export async function fetchCourtMappings(): Promise<CourtMapping[]> {
  return await $fetch<CourtMapping[]>('/api/court-mappings')
}
