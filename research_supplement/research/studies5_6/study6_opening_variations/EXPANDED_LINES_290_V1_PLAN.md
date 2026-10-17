# Expanded Opening-Defense Experiment v1

Date: 2026-10-16

## Research question

How do alternative named Black defensive variations
perform against White engines at nominal Elo 1800 and 2000?

## Experimental design

- Engine: Stockfish 19
- White Elo: 1800 or 2000
- Black Elo: 2000
- Opening variations: 18
- Experimental conditions: 36
- Games per condition: 290
- Total scheduled games: 10,440
- Games randomized in 29 blocks

Opening families:
- Sicilian Defense: 4 variations
- Ruy Lopez: 4 variations
- Queen's Gambit: 4 variations
- French Defense: 2 variations
- Caro-Kann Defense: 2 variations
- English Opening: 2 variations

## Outcomes

Primary descriptive outcome:
Black expected score = (Black wins + 0.5 * draws)
                       / completed games

Also report:
- White wins, Black wins, and draws
- Censored games
- Duplicate move sequences
- Completed games per condition
- Uncertainty intervals

## Comparisons

Compare defensive variations within each opening family,
separately for White Elo 1800 and 2000.

Report differences in Black expected score.

Results are exploratory. Differences between forced opening
positions do not necessarily isolate a causal effect of one
defensive move.

## Research integrity

Do not change opening definitions after data collection starts.
Do not select variations based on interim outcomes.
Do not score censored games as draws.
Do not make significance claims without appropriate statistical
tests, multiple-comparison adjustment, and power assessment.

Keep all configurations, manifests, game records, PGNs,
engine information, and analysis results.
