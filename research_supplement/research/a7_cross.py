import json, numpy as np, pandas as pd
from scipy import stats
from statsmodels.stats.contingency_tables import StratifiedTable
R=json.load(open("results.json"))
def load(f): return pd.DataFrame([json.loads(l) for l in open(f)])
cr={}
# setting-level comparison laptop vs cloud
for L in ["1320","2000","2500"]:
    a,b=R["new"]["levels"][L],R["new_ai"]["levels"][L]
    cr[L]={k:dict(lap=a[k]["rep"],ai=b[k]["rep"],diff=a[k]["rep"]-b[k]["rep"]) for k in ["white","black","draw"]}
cr["maxabs"]=max(abs(cr[L][k]["diff"]) for L in ["1320","2000","2500"] for k in ["white","black","draw"])
# pooled d3 across both runs, strata = run x tier
tabs=[]
for f1,f2 in [("data_conrad/replication_conrad.jsonl","data_conrad/d3_confirm_conrad.jsonl"),("data_ai/replication.jsonl","data_ai/d3_confirm.jsonl")]:
    df=pd.concat([load(f1).query("set=='white'"),load(f2)])
    df["ww"]=(df.result=="1-0").astype(int)
    for L in [1320,2000,2500]:
        a=df[df.tier==L]; x=a[a.forced=="d2d3"]; y=a[a.forced!="d2d3"]
        tabs.append(np.array([[x.ww.sum(),len(x)-x.ww.sum()],[y.ww.sum(),len(y)-y.ww.sum()]]))
st=StratifiedTable(tabs); p2=st.test_null_odds().pvalue
cr["d3_pooled"]=dict(or_=st.oddsratio_pooled,ci=list(st.oddsratio_pooled_confint()),p_one=p2/2 if st.oddsratio_pooled<1 else 1-p2/2,n=int(sum(t.sum() for t in tabs)),
                     n_d3=int(sum(t[0].sum() for t in tabs)))
# replication cell correlation between the two runs (60 cells)
ca=pd.read_csv("rep_cells_conrad.csv"); cb=pd.read_csv("rep_cells_ai.csv")
m=ca.merge(cb,on=["set","name","tier"],suffixes=("_a","_b"))
cr["cell_r"]=float(stats.pearsonr(m.win_a/m.n_a,m.win_b/m.n_b)[0])
R["cross"]=cr; json.dump(R,open("results.json","w"),indent=1)
print(json.dumps(cr,indent=0)[:1500])
