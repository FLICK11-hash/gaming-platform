"""Study 3: matches between settings (resumable). Pairs given as weak:strong:games."""
import json, time, argparse, collections
from harness import Engine, play
ap = argparse.ArgumentParser()
ap.add_argument("--engine", required=True)
ap.add_argument("--pairs", default="1320:2000:200,2000:2500:200,1320:2500:400")
ap.add_argument("--movetime", type=int, default=10)
ap.add_argument("--out", required=True)
a = ap.parse_args()
done = collections.Counter()
try:
    for l in open(a.out):
        d = json.loads(l); done[(d["weak"], d["strong"])] += 1
except FileNotFoundError:
    pass
eA, eB = Engine(a.engine), Engine(a.engine)
with open(a.out, "a") as f:
    for spec in a.pairs.split(","):
        lo, hi, n = [int(x) for x in spec.split(":")]
        for g in range(done[(lo, hi)], n):
            weak_white = (g % 2 == 0)
            ew, eb = (eA, eB) if weak_white else (eB, eA)
            elw, elb = (lo, hi) if weak_white else (hi, lo)
            t0 = time.time()
            res, term, plies, moves = play(ew, eb, elw, elb, [], a.movetime)
            f.write(json.dumps(dict(study="calibration", weak=lo, strong=hi, weak_white=weak_white, result=res,
                                    termination=term, plies=plies, secs=round(time.time() - t0, 2), moves=moves)) + "\n")
            f.flush()
eA.close(); eB.close()
