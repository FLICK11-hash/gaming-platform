import numpy as np, json
rng=np.random.default_rng(11)
R=json.load(open("results.json"))
ns=[100,290,500,1000,2000,3000,5000,10000,20000]
out={}
for tau in [0.02,0.03,0.04]:
    res=[]
    for n in ns:
        S=20000
        p=np.clip(rng.normal(0.45,tau,(S,10)),0.01,0.99)
        k=rng.binomial(n,p)+rng.uniform(0,1e-3,(S,10))
        t=p.argmax(1); o=k.argmax(1)
        top3=np.argsort(-k,1)[:,:3]
        res.append(dict(n=n,p_top=float((t==o).mean()),p_top3=float((top3==t[:,None]).any(1).mean())))
    out[str(tau)]=res
R["ranksim"]=out; json.dump(R,open("results.json","w"),indent=1)
for t in out: print(t,[(r["n"],round(r["p_top"],2)) for r in out[t]])
