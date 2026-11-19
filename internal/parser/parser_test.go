package parser

import (
	"errors"
	"reflect"
	"strings"
	"testing"
	"unicode/utf16"
)

var testParser = Parser{MaxAmount: 10_000_000, BotUsername: "KharchaBot"}

func mentionsIn(text string) []Entity {
	var entities []Entity
	offset := 0
	for _, r := range text {
		if r == '@' && offset > 0 {
			rest := string(utf16.Decode(utf16.Encode([]rune(text))[offset:]))
			end := strings.IndexAny(rest, " :")
			if end < 0 {
				end = len(rest)
			}
			if !strings.HasPrefix(text, "/setupi") {
				entities = append(entities, Entity{Type: EntityMention, Offset: offset, Length: end})
			}
		}
		offset += utf16.RuneLen(r)
	}
	return entities
}

func TestParse(t *testing.T) {
	user := func(name string) Mention { return Mention{Username: name} }

	tests := []struct {
		in   string
		want Command
	}{
		{"120 chai", Command{Kind: KindExpense, Amount: 12000, Description: "chai"}},
		{"chai 120", Command{Kind: KindExpense, Amount: 12000, Description: "chai"}},
		{"450 uber to airport", Command{Kind: KindExpense, Amount: 45000, Description: "uber to airport"}},
		{"1.5k rent #Bills", Command{Kind: KindExpense, Amount: 150000, Description: "rent", Tag: "bills"}},
		{"/today", Command{Kind: KindToday}},
		{"/week@kharchabot", Command{Kind: KindWeek}},
		{"/budget food 3000", Command{Kind: KindBudget, Category: "food", Amount: 300000}},
		{"/split 900 dinner", Command{Kind: KindSplit, Amount: 90000, Description: "dinner"}},
		{"/split 900 team dinner @a @b", Command{
			Kind: KindSplit, Amount: 90000, Description: "team dinner",
			Participants: []Participant{{Mention: user("a")}, {Mention: user("b")}},
		}},
		{"/split 900 🍕 night @a:300 @b:600", Command{
			Kind: KindSplit, Amount: 90000, Description: "🍕 night",
			Participants: []Participant{{Mention: user("a"), Share: 30000}, {Mention: user("b"), Share: 60000}},
		}},
		{"/settle @a 300", Command{
			Kind: KindSettle, Amount: 30000, Participants: []Participant{{Mention: user("a")}},
		}},
		{"/setupi name@okaxis", Command{Kind: KindSetUPI, VPA: "name@okaxis"}},
		{"/summary on", Command{Kind: KindSummary, Enabled: true}},
		{"/summary OFF", Command{Kind: KindSummary}},
		{"/deleteme", Command{Kind: KindDeleteMe}},
		{"/deleteme confirm", Command{Kind: KindDeleteMe, Confirmed: true}},
	}

	for _, tt := range tests {
		got, err := testParser.Parse(tt.in, mentionsIn(tt.in))
		if err != nil {
			t.Errorf("Parse(%q) error = %v", tt.in, err)
			continue
		}
		if !reflect.DeepEqual(got, tt.want) {
			t.Errorf("Parse(%q) = %+v, want %+v", tt.in, got, tt.want)
		}
	}
}

func TestParseTextMention(t *testing.T) {
	text := "/split 900 dinner Aman Kumar:900"
	entities := []Entity{{Type: EntityTextMention, Offset: 18, Length: 10, UserID: 42}}

	got, err := testParser.Parse(text, entities)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	want := []Participant{{Mention: Mention{UserID: 42, Name: "Aman Kumar"}, Share: 90000}}
	if !reflect.DeepEqual(got.Participants, want) {
		t.Errorf("participants = %+v, want %+v", got.Participants, want)
	}
}

func TestParseErrors(t *testing.T) {
	usageErrors := []string{
		"0 chai",
		"200000 laptop",
		"/budget food",
		"/split 900",
		"/split dinner 900",
		"/split 900 dinner @a:300 @b",
		"/split 900 dinner @a and @b",
		"/settle @a",
		"/settle 300",
		"/setupi nothandle",
		"/summary maybe",
	}
	for _, in := range usageErrors {
		var usageErr *UsageError
		if _, err := testParser.Parse(in, mentionsIn(in)); !errors.As(err, &usageErr) {
			t.Errorf("Parse(%q) error = %v, want a UsageError", in, err)
		}
	}

	sentinels := map[string]error{
		"":               ErrNotUnderstood,
		"hello":          ErrNotUnderstood,
		"hello there":    ErrNotUnderstood,
		"120":            ErrNotUnderstood,
		"/dance":         ErrUnknownCommand,
		"/help@otherbot": ErrOtherBot,
	}
	for in, want := range sentinels {
		if _, err := testParser.Parse(in, nil); !errors.Is(err, want) {
			t.Errorf("Parse(%q) error = %v, want %v", in, err, want)
		}
	}
}
