package simulate

// WhiteChessOpenings are ten well-known first moves for White, in from+to
// square form. Testing one always seats the bot under study as White,
// since these are moves that open the game - not replies.
var WhiteChessOpenings = []string{
	"e2e4", // King's Pawn
	"d2d4", // Queen's Pawn
	"g1f3", // Reti
	"c2c4", // English
	"e2e3", // Van 't Kruijs
	"d2d3", // King's Indian Attack setup
	"b1c3", // Van Geet
	"g2g3", // King's Fianchetto
	"f2f4", // Bird's Opening
	"b2b3", // Nimzo-Larsen
}

// BlackOpeningReplyTo is the White first move every entry in
// BlackChessOpenings is a reply to. Named "defenses" don't exist as an
// opening move in their own right - they're only meaningful as a response
// to something - so testing them holds White's opening fixed at the
// overwhelmingly most common first move and varies Black's reply instead.
const BlackOpeningReplyTo = "e2e4"

// BlackChessOpenings are ten well-known replies to 1.e4, in from+to square
// form. Testing one always seats the bot under study as Black.
var BlackChessOpenings = []string{
	"e7e5", // Open Game (King's Pawn reply)
	"c7c5", // Sicilian Defense
	"e7e6", // French Defense
	"c7c6", // Caro-Kann Defense
	"d7d5", // Scandinavian Defense
	"d7d6", // Pirc-style setup
	"g7g6", // Modern Defense
	"g8f6", // Alekhine's Defense
	"b8c6", // Nimzowitsch Defense
	"a7a6", // St. George Defense
}
