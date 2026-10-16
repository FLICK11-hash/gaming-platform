from data import *
import numpy as np, json
R=json.load(open("results.json")); rng=np.random.default_rng(7)
eb={}
for col,D in [("White",WHITE),("Black",BLACK)]:
    K=np.array(list(D.values())); names=list(D); eb[col]={}
    for j,L in enumerate(LEVELS):
        p=K[:,j]/N; m=p.mean(); s2=p*(1-p)/N; tau2=max(0,np.var(p,ddof=1)-s2.mean())
        if tau2==0: probs=np.full(10,0.1)
        else:
            Bi=tau2/(tau2+s2); post_m=m+Bi*(p-m); post_sd=np.sqrt(Bi*s2)
            sims=rng.normal(post_m,post_sd,size=(40000,10)); probs=np.bincount(sims.argmax(1),minlength=10)/40000
        eb[col][str(L)]=dict(zip(names,probs.round(3).tolist()))
R["eb_top"]=eb; json.dump(R,open("results.json","w"),indent=1)
for c in eb:
    for L in eb[c]: print(c,L,sorted(eb[c][L].items(),key=lambda x:-x[1])[:3])
