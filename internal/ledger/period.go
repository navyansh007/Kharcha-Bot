package ledger

import "time"

// Period is the half-open time range [From, To).
type Period struct {
	From time.Time
	To   time.Time
}

// Day returns the calendar day containing now in loc.
func Day(now time.Time, loc *time.Location) Period {
	y, m, d := now.In(loc).Date()
	from := time.Date(y, m, d, 0, 0, 0, 0, loc)
	return Period{From: from, To: from.AddDate(0, 0, 1)}
}

// Week returns the week containing now in loc. Weeks start on Monday.
func Week(now time.Time, loc *time.Location) Period {
	day := Day(now, loc)
	sinceMonday := (int(day.From.Weekday()) + 6) % 7
	from := day.From.AddDate(0, 0, -sinceMonday)
	return Period{From: from, To: from.AddDate(0, 0, 7)}
}

// Month returns the calendar month containing now in loc.
func Month(now time.Time, loc *time.Location) Period {
	y, m, _ := now.In(loc).Date()
	from := time.Date(y, m, 1, 0, 0, 0, 0, loc)
	return Period{From: from, To: from.AddDate(0, 1, 0)}
}
