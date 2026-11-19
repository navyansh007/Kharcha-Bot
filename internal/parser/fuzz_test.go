package parser

import "testing"

func FuzzParse(f *testing.F) {
	seeds := []string{
		"120 chai",
		"chai 120 #food",
		"/split 900 dinner @a:300 @b:600",
		"/settle @a 300",
		"/budget food 3000",
		"1.5k rent",
		"/split 900 🍕 @a",
	}
	for _, s := range seeds {
		f.Add(s, 18, 2)
	}

	f.Fuzz(func(t *testing.T, text string, offset, length int) {
		entities := []Entity{{Type: EntityMention, Offset: offset, Length: length}}
		cmd, err := testParser.Parse(text, entities)
		if err != nil {
			return
		}
		switch cmd.Kind {
		case KindExpense, KindSplit, KindSettle, KindBudget:
			if cmd.Amount <= 0 || cmd.Amount > testParser.MaxAmount {
				t.Fatalf("Parse(%q) accepted amount %d", text, cmd.Amount)
			}
		}
		for _, p := range cmd.Participants {
			if p.Share < 0 || p.Share > testParser.MaxAmount {
				t.Fatalf("Parse(%q) accepted share %d", text, p.Share)
			}
		}
	})
}
