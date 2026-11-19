// Package money represents Indian rupee amounts as integer paise.
package money

import (
	"errors"
	"strconv"
	"strings"
)

// Paise is an amount of money in paise. One rupee is 100 paise.
type Paise int64

const maxRupeeDigits = 12

// ErrInvalidAmount is returned when a string is not a valid amount.
var ErrInvalidAmount = errors.New("invalid amount")

// Parse converts text such as "120", "99.50" or "1.5k" to Paise without using floating point.
func Parse(s string) (Paise, error) {
	multiplier := int64(1)
	if rest, ok := strings.CutSuffix(strings.ToLower(s), "k"); ok {
		s, multiplier = rest, 1000
	}

	whole, frac, hasFrac := strings.Cut(s, ".")
	if !allDigits(whole) || len(whole) > maxRupeeDigits {
		return 0, ErrInvalidAmount
	}
	if hasFrac && (!allDigits(frac) || len(frac) > 2) {
		return 0, ErrInvalidAmount
	}

	rupees, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	paise, err := strconv.ParseInt((frac + "00")[:2], 10, 64)
	if err != nil {
		return 0, ErrInvalidAmount
	}
	return Paise((rupees*100 + paise) * multiplier), nil
}

func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// String formats the amount for display, for example "₹1,23,456.50".
func (p Paise) String() string {
	sign, rupees, paise := p.split()
	out := sign + "₹" + groupIndian(rupees)
	if paise != "00" {
		out += "." + paise
	}
	return out
}

// Rupees formats the amount as a plain decimal number, for example "1234.50".
func (p Paise) Rupees() string {
	sign, rupees, paise := p.split()
	return sign + rupees + "." + paise
}

func (p Paise) split() (sign, rupees, paise string) {
	v := int64(p)
	if v < 0 {
		sign, v = "-", -v
	}
	paise = strconv.FormatInt(v%100+100, 10)[1:]
	return sign, strconv.FormatInt(v/100, 10), paise
}

func groupIndian(digits string) string {
	if len(digits) <= 3 {
		return digits
	}
	head, tail := digits[:len(digits)-3], digits[len(digits)-3:]
	var groups []string
	for len(head) > 2 {
		groups = append([]string{head[len(head)-2:]}, groups...)
		head = head[:len(head)-2]
	}
	groups = append([]string{head}, groups...)
	return strings.Join(groups, ",") + "," + tail
}

// SplitEqual divides total into n shares that differ by at most one paisa.
// The earlier shares receive the remainder. It returns nil if n is not positive.
func SplitEqual(total Paise, n int) []Paise {
	if n <= 0 {
		return nil
	}
	base := total / Paise(n)
	remainder := int(total % Paise(n))
	shares := make([]Paise, n)
	for i := range shares {
		shares[i] = base
		if i < remainder {
			shares[i]++
		}
	}
	return shares
}

// Sum adds up the given amounts.
func Sum(amounts []Paise) Paise {
	var total Paise
	for _, a := range amounts {
		total += a
	}
	return total
}
