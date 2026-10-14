"""Cross-tier matches to estimate the effective strength gaps between settings."""
import json, sys, time, argparse
from harness import Engine, play
ap = argparse.ArgumentParser()
ap.add_argument("--engine", required=True); ap.add_argument("--pairs", default="1320:2000,2000:2500,1320:2500")
ap.add_argument("--games", type=int, default=200); ap.add_argument("--movetime", type=int, default=10)
ap.add_argument("--out", required=True)
a = ap.parse_args()
eA, eB = Engine(a.engine), Engine(a.engine)       # one process per side so each keeps its own setting
with open(a.out, "a") as f:
    for pair in a.pairs.split(","):
        lo, hi = [int(x) for x in pair.split(":")]
        for g in range(a.games):
            weak_white = (g % 2 == 0)
            ew, eb = (eA, eB) if weak_white else (eB, eA)
            elw, elb = (lo, hi) if weak_white else (hi, lo)
            t0 = time.time()
            res, term, plies, moves = play(ew, eb, elw, elb, [], a.movetime)
            f.write(json.dumps(dict(study="calibration", weak=lo, strong=hi, weak_white=weak_white, result=res,
                                    termination=term, plies=plies, secs=round(time.time()-t0, 2), moves=moves)) + "\n"); f.flush()
eA.close(); eB.close()
