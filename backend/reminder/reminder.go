package reminder

import (
	"log"
	"time"

	"github.com/robfig/cron/v3"

	"violin-quest-api/db"
	"violin-quest-api/fcm"
)

// scheduler is the single shared cron instance for all children.
var scheduler *cron.Cron

// StartReminders schedules a daily 16:00 reminder for each child.
// Uses a single cron instance with per-child jobs keyed to each child's timezone.
// Fires an FCM notification if the child has no session (PENDING or APPROVED) today.
func StartReminders() {
	rows, err := db.DB.Query(`SELECT id, name, timezone FROM children`)
	if err != nil {
		log.Printf("reminder: failed to load children: %v", err)
		return
	}
	defer rows.Close()

	type child struct {
		id   int64
		name string
		tz   string
	}
	var children []child
	for rows.Next() {
		var ch child
		rows.Scan(&ch.id, &ch.name, &ch.tz) //nolint:errcheck
		children = append(children, ch)
	}

	// A single cron runner; each job carries its own location via the spec.
	scheduler = cron.New()
	for _, ch := range children {
		loc, err := time.LoadLocation(ch.tz)
		if err != nil {
			loc = time.UTC
		}
		childID := ch.id
		childName := ch.name
		// TZ= prefix makes the cron library interpret the schedule in that timezone.
		spec := "TZ=" + loc.String() + " 0 16 * * *"
		scheduler.AddFunc(spec, func() { //nolint:errcheck
			sendReminderIfNeeded(childID, childName)
		})
		log.Printf("reminder: scheduled for child %d (%s) at 16:00 %s", childID, childName, loc)
	}
	scheduler.Start()
}

func sendReminderIfNeeded(childID int64, childName string) {
	today := todayForChild(childID)

	var count int
	db.DB.QueryRow( //nolint:errcheck
		`SELECT COUNT(*) FROM sessions WHERE child_id = ? AND date = ? AND status IN ('PENDING', 'APPROVED')`,
		childID, today,
	).Scan(&count)

	if count > 0 {
		return
	}

	tokens := db.GetChildTokens(childID)
	if len(tokens) == 0 {
		return
	}
	fcm.Send(tokens, "Tijd voor viool! 🎻", "Het magische wiel wacht op je.")
	log.Printf("reminder: sent to child %d (%s)", childID, childName)
}

func todayForChild(childID int64) string {
	var tz string
	db.DB.QueryRow(`SELECT timezone FROM children WHERE id = ?`, childID).Scan(&tz) //nolint:errcheck
	loc, err := time.LoadLocation(tz)
	if err != nil {
		loc = time.UTC
	}
	return time.Now().In(loc).Format("2006-01-02")
}
