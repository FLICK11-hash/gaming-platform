# Opening rankings in engine self-play: data, game records, and analysis

Supplementary material for "Are Opening Rankings from Engine Self-Play Real? Sampling Error, Draws, and Strength Dependence in Strength-Limited Stockfish Games" (Flick & Camaro Paslar).

## Study 1 (platform runs, transcribed from screenshots)
- `screenshots_all_runs.pdf`: original simulator output for Runs 1-30 (Run 7 excluded as an error).
- `cells.csv`: designated-color wins out of 290 games for each of the 60 opening-by-setting cells, with 95% Wilson intervals.
- `run_summaries.csv`: run-level White win rate (White-opening set), Black win rate (Black-reply set), and draw rate (all 200 games) per run and setting. Run 12 at 2000 has no draw value.
- `data.py`: Study 1 chess data as Python structures; `c4data.py`: Connect Four per-column results (positive control).

## Studies 2-4 (replication program with full game records)
- `harness/harness.py`: reproduces the platform's game loop (Stockfish 15.1, UCI_LimitStrength, 1 MB hash, 1 thread, 10 ms per move, FEN-only input, automatic checkmate / stalemate / insufficient material / fivefold repetition / 75-move rule, 300-ply cap).
- `harness/complete.py`: plays a study to a target number of games per cell (resumable). `harness/calibrate.py`: cross-setting matches (resumable).
- `data_conrad/`: primary Studies 2-4, run by the first author on his Windows laptop.
- `data_ai/`: the earlier robustness run of the same program on a single-core cloud processor, executed by the AI system (Section 4.16).
- Each line is one game: setting, forced move, result, termination, number of plies, seconds, and the full move list in UCI notation.

To rerun: build Stockfish 15.1 from https://github.com/official-stockfish/Stockfish (tag sf_15.1), `pip install chess`, then
```
python harness/calibrate.py --engine /path/to/stockfish --out calibration.jsonl
python harness/complete.py --engine /path/to/stockfish --study replication --target 30 --out replication.jsonl
python harness/complete.py --engine /path/to/stockfish --study d3_confirm --target 60 --sets white --out d3_confirm.jsonl
```

## Studies 5-6 (Stockfish 19, 3,000 nodes per move; J. Huang)
- `studies5_6/`: game-level CSV and PGN records, locked protocols, game schedules (manifests), and analysis outputs for the opening-response policy study (Study 5: 6,960 survey and 5,220 evaluation games) and the multi-move variation study (Study 6: 10,440 games), with the configuration, policy, and engine SHA-256 fingerprints reported in the paper.
- `jh_data.py`: the summary values used for the Study 5 and 6 tables and figures.

## Analysis
```
pip install numpy pandas scipy statsmodels matplotlib
python analysis.py && python a2.py && python a3.py && python a4.py && python a5_c4.py
python a6_new.py ai new_ai && python a6_new.py conrad new && python a7_cross.py && python figs.py
```
Random seeds are fixed in the analysis scripts. Platform configuration for Study 1: commit 5872d85 of this repository; permanent archive https://doi.org/10.5281/zenodo.23357406.
