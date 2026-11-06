// Package mapping kopplar bokningar i det kommunala bokningssystemet till
// banor och saler i MATCHi.
//
// Kopplingen ligger i databasen (tabellen court_mappings), eftersom samma
// kommunresurs kan peka på flera MATCHi-banor och vice versa.
package mapping

import (
	"strings"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/store"
)

// Courts returnerar de MATCHi-banor som en bokning i det kommunala
// bokningssystemet gäller.
//
// Det kommunala boknings-API:t levererar inte alltid resurs-id:t, utan bara
// resursens namn i bokningens ämne ("11:00 - 13:00 Baldershallen 1"). Därför
// matchas bokningen på namn när det finns, och på id som backup.
func Courts(appointment actorsmartbook.Appointment, mappings []store.CourtMapping) []store.CourtMapping {
	var courts []store.CourtMapping

	for _, court := range mappings {
		if MatchesResource(appointment, court) {
			courts = append(courts, court)
		}
	}

	return courts
}

// MatchesResource avgör om en bokning gäller en given kommunresurs.
func MatchesResource(appointment actorsmartbook.Appointment, court store.CourtMapping) bool {
	if name := strings.TrimSpace(appointment.Location); name != "" && court.KommunResourceName != "" {
		if strings.EqualFold(name, court.KommunResourceName) {
			return true
		}
	}

	for _, resource := range appointment.Resources {
		if resource.ID == court.KommunResourceID {
			return true
		}
	}

	return false
}
