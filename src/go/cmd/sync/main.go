// Command sync speglar bokningar från det kommunala bokningssystemet till
// MATCHi och avslutas med status 0 om allt gick bra.
//
// Programmet används av cronjobbet i klustret:
//
//	sync
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	// Tidszonsdatabasen bäddas in i binären så att Europe/Stockholm alltid
	// finns, oberoende av vad basimageen innehåller.
	_ "time/tzdata"

	"github.com/simon/repo-template/internal/actorsmartbook"
	"github.com/simon/repo-template/internal/matchi"
	"github.com/simon/repo-template/internal/store"
	"github.com/simon/repo-template/internal/sync"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "sync: %v\n", err)
		os.Exit(1)
	}
}

// run utför en synkronisering.
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		return fmt.Errorf("DATABASE_URL saknas")
	}

	db, err := store.Open(dsn)
	if err != nil {
		return err
	}
	defer db.Close()

	schemaCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := db.EnsureSchema(schemaCtx); err != nil {
		return err
	}

	kommunClient := actorsmartbook.NewClient(
		os.Getenv("ACTORSMARTBOOK_BASE_URL"),
		os.Getenv("ACTORSMARTBOOK_RESOURCE_FILTER"),
		actorsmartbook.DefaultTimeout,
		nil,
	)

	matchiClient := matchi.NewClient(
		os.Getenv("MATCHI_BASE_URL"),
		os.Getenv("MATCHI_USERNAME"),
		os.Getenv("MATCHI_PASSWORD"),
		matchi.DefaultTimeout,
		nil,
	)

	syncer := sync.New(kommunClient, matchiClient, db, sync.Config{
		Days:               envInt("SYNC_DAYS", sync.DefaultDays),
		CustomerID:         int64(envInt("MATCHI_CUSTOMER_ID", 0)),
		OrganizationFilter: os.Getenv("SYNC_ORGANIZATION_FILTER"),
		RemoveOrphans:      os.Getenv("SYNC_REMOVE_ORPHANS") == "true",
	}, kommunClient.Location())

	report, err := syncer.Run(ctx)
	if err != nil {
		return err
	}

	encoded, err := json.Marshal(report)
	if err != nil {
		return err
	}

	fmt.Println(string(encoded))

	if len(report.Errors) > 0 {
		return fmt.Errorf("%d bokningar kunde inte synkroniseras", len(report.Errors))
	}

	return nil
}

// envInt läser ett heltal ur en miljövariabel, med fallback.
func envInt(name string, fallback int) int {
	value := os.Getenv(name)
	if value == "" {
		return fallback
	}

	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		fmt.Fprintf(os.Stderr, "sync: miljövariabelen %s har ogiltigt värde %q\n", name, value)
		return fallback
	}

	return parsed
}
