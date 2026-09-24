package blackjack

// ParseCard returns the integer value of a card following blackjack ruleset.
func ParseCard(card string) int {
	switch card {
		case "ace":
			return 11
		case "two":
			return 2
		case "three":
			return 3
		case "four":
			return 4
		case "five":
			return 5
		case "six":
			return 6
		case "seven":
			return 7
		case "eight":
			return 8
		case "nine":
			return 9
		case "ten", "jack", "queen", "king":
			return 10
		default:
			return 0
	}
}

// FirstTurn returns the decision for the first turn, given two cards of the
// player and one card of the dealer.
func FirstTurn(card1, card2, dealerCard string) string {
	p1 := ParseCard(card1)
	p2 := ParseCard(card2)
	pSum := p1 + p2
	dScore := ParseCard(dealerCard)

	switch {
		case card1 == "ace" && card2 == "ace":
			return "P"

		case pSum == 21:
			if dScore >= 10 {
				return "S"
			}
			return "W"

		case pSum >= 17 && pSum <= 20:
			return "S"

		case pSum >= 12 && pSum <= 16:
			if dScore >= 2 && dScore <= 6 {
				return "S"
			}
			return "H"

		default:
			return "H"
	}
}
