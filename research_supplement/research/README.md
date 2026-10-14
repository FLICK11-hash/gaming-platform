# Opening rankings in engine self-play: data, game records, and analysis

Supplementary material for "Are Opening Rankings from Engine Self-Play Real? Sampling Error, Draws, and Strength Dependence in Strength-Limited Stockfish Games" (Flick & Camaro Paslar).

## Study 1 (platform runs, transcribed from screenshots)
- `screenshots_all_runs.pdf`: original simulator output for Runs 1-30 (Run 7 excluded as an error).
- `cells.csv`: designated-color wins out of 290 games for each of the 60 opening-by-setting cells, with 95% Wilson intervals.
- `run_summaries.csv`: run-level White win rate (White-opening set), Black win rate (Black-reply set), and draw rate (all 200 games) per run and setting. Run 12 at 2000 has no draw value.
- `data.py`: Study 1 chess data as Python structures; `c4data.py`: Connect Four per-column results (positive control).

## Studies 2-4 (replication program with full game records)
- `harness/harness.py`: reproduces the platform's game loop (Stockfish 15.1, UCI_LimitStrength, 1 MB hash, 1 thread, 10 ms per move, FEN-only input, automatic checkmate / stalemate / insufficient material / fivefold repetition / 75-move rule, 300-ply cap).
- `harness/complete.py`: plays a study to a target number of games per cell (resumable).
- `harness/calibrate.py`: cross-setting matches (Study 3).
- `harness/replication.jsonl` (Study 2), `harness/d3_confirm.jsonl` (Study 4), `harness/calibration*.jsonl` (Study 3): one JSON object per game with setting, forced move, result, termination, number of plies, and the full move list in UCI notation.

To rerun (about 2-3 hours on one laptop core): compile Stockfish 15.1 from https://github.com/official-stockfish/Stockfish (tag sf_15.1) with its default network, `pip install chess`, then
```
python harness/calibrate.py --engine /path/to/stockfish --pairs 1320:2000,2000:2500,1320:2500,1320:2500 --games 200 --out harness/calibration.jsonl
python harness/complete.py --engine /path/to/stockfish --study replication --target 30 --out harness/replication.jsonl
python harness/complete.py --engine /path/to/stockfish --study d3_confirm --target 60 --sets white --out harness/d3_confirm.jsonl
```

## Analysis
```
pip install numpy pandas scipy statsmodels matplotlib
python analysis.py && python a2.py && python a3.py && python a4.py && python a5_c4.py && python a6_new.py && python figs.py
```
Random seeds are fixed in the analysis scripts. Platform configuration for Study 1: commit 5872d85 of this repository.
