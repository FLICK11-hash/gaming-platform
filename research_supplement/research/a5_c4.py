from c4data import *
import numpy as np, json
from scipy import stats
from statsmodels.stats.proportion import proportion_confint
R=json.load(open("results.json")); rng=np.random.default_rng(5)
runs=sorted(C4); M=np.array([[v//10 for v in C4[r]] for r in runs]); K=M.sum(0); n=290
chi2,p,_,_=stats.chi2_contingency(np.vstack([K,n-K]))
ci=[proportion_confint(k,n,method="wilson") for k in K]
others=K.sum()-K[3]; d=K[3]/n-others/(6*n)
se=np.sqrt((K[3]/n)*(1-K[3]/n)/n+(others/(6*n))*(1-others/(6*n))/(6*n))
sims=rng.binomial(n,K/n,size=(20000,7)).astype(float)+rng.uniform(0,1e-3,(20000,7))
top=np.bincount(sims.argmax(1),minlength=7)/20000
v=np.var(K/n,ddof=1); s2=np.mean((K/n)*(1-K/n)/n); tau=np.sqrt(max(0,v-s2))
runs_top=int((M.argmax(1)==3).sum())  # ties: argmax picks first
runs_top_strict=int(sum(1 for row in M if row[3]==row.max()))
cards=np.array([C4_CARDS[r] for r in runs])
R["c4"]=dict(wins=K.tolist(),pct=(100*K/n).tolist(),ci=[[100*a,100*b] for a,b in ci],chi2=chi2,p=p,
   center_diff=100*d,center_lo=100*(d-1.96*se),center_hi=100*(d+1.96*se),p_top_center=float(top[3]),tau=100*tau,
   rel=float(tau**2/(tau**2+s2)),runs_center_top=runs_top_strict,nruns=len(runs),
   red=float(cards[:,0].mean()),green=float(cards[:,1].mean()),draw=float(cards[:,2].mean()))
json.dump(R,open("results.json","w"),indent=1)
print({k:(round(v,4) if isinstance(v,float) else v) for k,v in R["c4"].items() if k not in("ci",)})
