"""Complete a study to a target number of games per cell (resumable)."""
import json, random, sys, time, collections, argparse
from harness import Engine, play, WHITE_MOVES, BLACK_MOVES
ap=argparse.ArgumentParser(); ap.add_argument("--engine",required=True); ap.add_argument("--study",required=True)
ap.add_argument("--target",type=int,required=True); ap.add_argument("--sets",default="white,black"); ap.add_argument("--out",required=True)
ap.add_argument("--seed",type=int,default=7); ap.add_argument("--movetime",type=int,default=10); a=ap.parse_args()
done=collections.Counter()
try:
    for l in open(a.out): d=json.loads(l); done[(d["tier"],d["set"],d["forced"])]+=1
except FileNotFoundError: pass
jobs=[]
for t in (1320,2000,2500):
    if "white" in a.sets:
        for m in WHITE_MOVES: jobs+=[(t,"white",m,[m])]*max(0,a.target-done[(t,"white",m)])
    if "black" in a.sets:
        for m in BLACK_MOVES: jobs+=[(t,"black",m,["e2e4",m])]*max(0,a.target-done[(t,"black",m)])
random.Random(a.seed).shuffle(jobs); print(len(jobs),"jobs",flush=True)
eng=Engine(a.engine)
with open(a.out,"a") as f:
    for t,s,m,forced in jobs:
        t0=time.time(); res,term,plies,moves=play(eng,eng,t,t,forced,a.movetime)
        f.write(json.dumps(dict(study=a.study,tier=t,set=s,forced=m,result=res,termination=term,plies=plies,secs=round(time.time()-t0,2),moves=moves))+"\n"); f.flush()
eng.close()
