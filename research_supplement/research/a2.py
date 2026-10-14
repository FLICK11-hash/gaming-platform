from data import *
import numpy as np, json
from scipy import stats
from statsmodels.stats.proportion import proportion_confint
R=json.load(open("results.json"))
R["pooled"]={}; R["shrink"]={}
for col,D in [("White",WHITE),("Black",BLACK)]:
    K=np.array(list(D.values())); names=list(D)
    tot=K.sum(1); n=3*N; out={}
    lev=K.sum(0)/(10*N)
    for nm,k,row in zip(names,tot,K):
        lo,hi=proportion_confint(k,n,method="wilson")
        # level-standardised deviation: mean over levels of (p_ol - p_l)
        dev=np.mean(row/N-lev)*100
        out[nm]=dict(wins=int(k),pct=100*k/n,lo=100*lo,hi=100*hi,dev=dev)
    R["pooled"][col]=out
    # method-of-moments between-opening SD per level and shrinkage factor
    sh={}
    for j,L in enumerate(LEVELS):
        p=K[:,j]/N; v=np.var(p,ddof=1); s2=np.mean(p*(1-p)/N); tau2=max(0,v-s2)
        sh[L]=dict(obs_sd=100*np.sqrt(v),noise_sd=100*np.sqrt(s2),tau_sd=100*np.sqrt(tau2),B=tau2/(tau2+s2) if tau2>0 else 0.0)
    R["shrink"][col]=sh
# KIA vs other nine, pooled stratified (Mantel-Haenszel)
from statsmodels.stats.contingency_tables import StratifiedTable
tabs=[]
for j in range(3):
    a=WHITE["King's Indian Attack"][j]; others=sum(v[j] for k,v in WHITE.items() if k!="King's Indian Attack")
    tabs.append(np.array([[a,N-a],[others,9*N-others]]))
st=StratifiedTable(tabs)
R["kia"]=dict(or_mh=st.oddsratio_pooled,ci=list(st.oddsratio_pooled_confint()),p=st.test_null_odds().pvalue,
              p_homog=st.test_equal_odds().pvalue,
              pct=100*sum(WHITE["King's Indian Attack"])/(3*N),
              pct_other=100*sum(sum(v) for k,v in WHITE.items() if k!="King's Indian Attack")/(27*N))
json.dump(R,open("results.json","w"),indent=1)
print(json.dumps({k:R[k] for k in ["pooled","shrink","kia"]},indent=1))
