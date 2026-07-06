package gebiz

import (
	"context"
	"log"

	"github.com/midtone/tender-worker/internal/store"
)

// FetchAll pulls every GeBIZ feed and maps each opportunity to a store.Record.
// Per-feed errors are logged and skipped (partial success), matching the worker's
// previous behaviour. DetailsFetched stays false — api-server-go scrapes the
// opportunity-page detail on demand.
func FetchAll(ctx context.Context) ([]store.Record, error) {
	var out []store.Record
	for _, f := range Feeds {
		tenders, err := Fetch(ctx, f)
		if err != nil {
			log.Printf("gebiz: fetch %s: %v", CategoryFromURL(f), err)
			continue
		}
		for _, t := range tenders {
			out = append(out, store.Record{
				Source:        "gebiz",
				SourceID:      t.OpportunityID,
				ReferenceNo:   t.ReferenceNo,
				Title:         t.Title,
				Link:          t.Link,
				Category:      t.Category,
				Agency:        t.CallingEntity,
				TenderType:    t.TenderType,
				PublishedDate: t.PublishedDate,
				ClosingDate:   t.ClosingDate,
				Status:        t.Status,
				Description:   t.Description,
			})
		}
	}
	return out, nil
}
