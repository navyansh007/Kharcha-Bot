// Package category assigns spending categories to expense descriptions.
package category

import "strings"

// The built-in categories.
const (
	Food     = "food"
	Travel   = "travel"
	Shopping = "shopping"
	Bills    = "bills"
	Other    = "other"
)

var keywords = map[string]string{
	"chai": Food, "tea": Food, "coffee": Food, "lunch": Food, "dinner": Food,
	"breakfast": Food, "snacks": Food, "pizza": Food, "burger": Food, "biryani": Food,
	"maggi": Food, "samosa": Food, "swiggy": Food, "zomato": Food, "canteen": Food,
	"mess": Food, "juice": Food, "milk": Food, "groceries": Food,

	"uber": Travel, "ola": Travel, "rapido": Travel, "auto": Travel, "cab": Travel,
	"bus": Travel, "metro": Travel, "train": Travel, "flight": Travel, "petrol": Travel,
	"diesel": Travel, "fuel": Travel, "parking": Travel, "toll": Travel,

	"amazon": Shopping, "flipkart": Shopping, "myntra": Shopping, "clothes": Shopping,
	"shoes": Shopping, "shirt": Shopping, "jeans": Shopping, "shopping": Shopping,

	"rent": Bills, "electricity": Bills, "wifi": Bills, "internet": Bills,
	"recharge": Bills, "water": Bills, "gas": Bills, "subscription": Bills,
	"netflix": Bills, "spotify": Bills,
}

// Detect returns the category of the first known keyword in description, or Other.
func Detect(description string) string {
	for word := range strings.FieldsSeq(strings.ToLower(description)) {
		if c, ok := keywords[strings.Trim(word, ".,!?()")]; ok {
			return c
		}
	}
	return Other
}
