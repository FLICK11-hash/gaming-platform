from data import *
import numpy as np, pandas as pd, json, warnings
warnings.filterwarnings('ignore')
from scipy import stats
import statsmodels.api as sm, statsmodels.formula.api as smf
from statsmodels.stats.proportion import proportion_confint
from statsmodels.stats.multitest import multipletests
rng=np.random.default_rng(20261012)
R={}
def wilson(k,n=N):
    lo,hi=proportion_confint(k,n,method="wilson"); return lo*100,hi*100
rows=[]
for col,D in [("White",WHITE),("Black",BLACK)]:
    for o,ks in D.items():
        for L,k in zip(LEVELS,ks):
            lo,hi=wilson(k); rows.append(dict(color=col,opening=o,level=L,wins=k,n=N,pct=100*k/N,lo=lo,hi=hi))
df=pd.DataFrame(rows); df.to_csv("cells.csv",index=False)

# homogeneity chi-square per level/color + null range sim
R["homog"]={}
for col in ["White","Black"]:
    for L in LEVELS:
        d=df[(df.color==col)&(df.level==L)]
        k=d.wins.values; tab=np.vstack([k,N-k])
        chi2,p,dof,_=stats.chi2_contingency(tab)
        p0=k.sum()/(10*N); obs=(k.max()-k.min())/N*100
        sims=rng.binomial(N,p0,size=(100000,10)); rng_=(sims.max(1)-sims.min(1))/N*100
        R["homog"][f"{col}_{L}"]=dict(chi2=chi2,df=dof,p=p,range=obs,null_med=float(np.median(rng_)),null_95=float(np.percentile(rng_,95)),p_range=float((rng_>=obs-1e-9).mean()),pooled=100*p0)
        np.save(f"nullrange_{col}_{L}.npy",rng_[:20000])

# GLM interaction tests
R["glm"]={}
for col in ["White","Black"]:
    d=df[df.color==col].copy(); d["losses"]=d.n-d.wins; d["lev"]=d.level.astype(str)
    f=lambda form: smf.glm(form,data=d,family=sm.families.Binomial()).fit()
    m0=smf.glm("wins + losses ~ C(lev)",data=d,family=sm.families.Binomial()).fit() if False else None
    endog=d[["wins","losses"]].values
    def fit(X): return sm.GLM(endog,X,family=sm.families.Binomial()).fit()
    import patsy
    Xl=patsy.dmatrix("C(lev)",d,return_type="dataframe")
    Xa=patsy.dmatrix("C(opening)+C(lev)",d,return_type="dataframe")
    Xi=patsy.dmatrix("C(opening)*C(lev)",d,return_type="dataframe")
    ml,ma,mi=fit(Xl),fit(Xa),fit(Xi)
    lr_open=ml.deviance-ma.deviance; df_open=Xa.shape[1]-Xl.shape[1]
    lr_int=ma.deviance-mi.deviance; df_int=Xi.shape[1]-Xa.shape[1]
    # linear-in-level interaction (1 df per opening -> 9 df)
    d["s"]=d.level.map({1320:-1,2000:0,2500:1})
    Xt=patsy.dmatrix("C(opening)+C(lev)+C(opening):s",d,return_type="dataframe")
    mt=fit(Xt); lr_t=ma.deviance-mt.deviance; df_t=np.linalg.matrix_rank(Xt.values)-np.linalg.matrix_rank(Xa.values)
    R["glm"][col]=dict(lr_level=ml.null_deviance-ml.deviance,df_level=2,p_level=stats.chi2.sf(ml.null_deviance-ml.deviance,2),
        lr_open=lr_open,df_open=df_open,p_open=stats.chi2.sf(lr_open,df_open),
        lr_int=lr_int,df_int=df_int,p_int=stats.chi2.sf(lr_int,df_int),
        lr_trend=lr_t,df_trend=df_t,p_trend=stats.chi2.sf(lr_t,df_t),
        add_dev=ma.deviance,add_pearson=float(ma.pearson_chi2),add_df=ma.df_resid)

# per-opening change 1320->2500 and difference-in-differences vs level mean
R["change"]={}
for col,D in [("White",WHITE),("Black",BLACK)]:
    tot=np.array(list(D.values())).sum(0)/(10*N)
    out=[]
    for o,ks in D.items():
        p1,p3=ks[0]/N,ks[2]/N
        diff=p3-p1; se=np.sqrt(p1*(1-p1)/N+p3*(1-p3)/N); z=diff/se
        # DiD: opening change minus mean change of the other nine
        oth=[v for kk,v in D.items() if kk!=o]
        q1=sum(v[0] for v in oth)/(9*N); q3=sum(v[2] for v in oth)/(9*N)
        did=diff-(q3-q1); se_d=np.sqrt(se**2+q1*(1-q1)/(9*N)+q3*(1-q3)/(9*N))
        out.append(dict(opening=o,p1320=100*p1,p2000=100*ks[1]/N,p2500=100*p3,diff=100*diff,lo=100*(diff-1.96*se),hi=100*(diff+1.96*se),p=2*stats.norm.sf(abs(z)),
                        did=100*did,did_lo=100*(did-1.96*se_d),did_hi=100*(did+1.96*se_d),p_did=2*stats.norm.sf(abs(did/se_d))))
    t=pd.DataFrame(out)
    t["p_holm"]=multipletests(t.p,method="holm")[1]; t["p_did_holm"]=multipletests(t.p_did,method="holm")[1]
    t["p_did_bh"]=multipletests(t.p_did,method="fdr_bh")[1]
    R["change"][col]=t.to_dict("records")

# rank probabilities (parametric bootstrap, binomial resampling of cells)
R["rank"]={}
B=20000
for col,D in [("White",WHITE),("Black",BLACK)]:
    names=list(D); K=np.array(list(D.values()))
    res={}
    for j,L in enumerate(LEVELS):
        p=K[:,j]/N; sims=rng.binomial(N,p,size=(B,10)).astype(float)+rng.uniform(0,1e-3,(B,10))
        top=np.bincount(sims.argmax(1),minlength=10)/B
        bot=np.bincount(sims.argmin(1),minlength=10)/B
        ranks=(-sims).argsort(1).argsort(1)+1
        res[L]=dict(top=dict(zip(names,top.round(4).tolist())),bottom=dict(zip(names,bot.round(4).tolist())),
                    rank_lo=dict(zip(names,np.percentile(ranks,2.5,0).tolist())),rank_hi=dict(zip(names,np.percentile(ranks,97.5,0).tolist())))
    R["rank"][col]=res
# rank correlation across levels
R["spearman"]={}
for col,D in [("White",WHITE),("Black",BLACK)]:
    K=np.array(list(D.values()))
    R["spearman"][col]={f"{a}-{b}":stats.spearmanr(K[:,i],K[:,j]).statistic for (i,a),(j,b) in [((0,1320),(1,2000)),((0,1320),(2,2500)),((1,2000),(2,2500))]}

# run-level cards
cr=[]
for r,v in CARDS.items():
    for L,(w,b,d) in v.items(): cr.append(dict(run=r,level=L,white=w,black=b,draw=d))
cr=pd.DataFrame(cr); cr.to_csv("cards.csv",index=False)
R["runs"]={}
for L in LEVELS:
    c=cr[cr.level==L]; out={}
    for v in ["white","black","draw"]:
        x=c[v].dropna().values.astype(float); m=x.mean(); s=x.std(ddof=1); n=len(x)
        tq=stats.t.ppf(.975,n-1)
        p=m/100; exp_var=p*(1-p)/100*1e4 if v!="draw" else p*(1-p)/200*1e4
        disp=s**2/exp_var; chi=(n-1)*disp; pdisp=stats.chi2.sf(chi,n-1); plow=stats.chi2.cdf(chi,n-1)
        out[v]=dict(mean=m,sd=s,n=n,lo=m-tq*s/np.sqrt(n),hi=m+tq*s/np.sqrt(n),min=x.min(),max=x.max(),disp=disp,p_over=pdisp,p_under=plow)
    # FMA bounds: W - B +/- D
    W=out["white"]["mean"]; Bm=out["black"]["mean"]; Dm=out["draw"]["mean"]
    out["fma"]=dict(point=W-Bm,lo=W-Bm-Dm,hi=W-Bm+Dm)
    # implied opposite-colour wins if draws split evenly
    out["implied"]=dict(black_in_white_set=100-W-Dm,white_in_black_set=100-Bm-Dm,
                        white_pooled=(W+100-Bm-Dm)/2,black_pooled=(Bm+100-W-Dm)/2,
                        white_score=((W+100-Bm-Dm)/2+Dm/2), )
    # per-run FMA
    f=(c.white-c.black).values.astype(float)
    out["fma_run"]=dict(mean=f.mean(),sd=f.std(ddof=1),pos=int((f>0).sum()),n=len(f),t=stats.ttest_1samp(f,0).statistic,p=stats.ttest_1samp(f,0).pvalue)
    R["runs"][L]=out
# draw trend tests
d=cr.dropna(subset=["draw"])
R["draw_tests"]=dict(kw=stats.kruskal(*[d[d.level==L].draw for L in LEVELS]).pvalue,
   t_2000_2500=stats.ttest_ind(d[d.level==2000].draw,d[d.level==2500].draw).pvalue)
# paired across runs: 1320 vs 2500 white set
tt=stats.ttest_rel(cr[cr.level==1320].sort_values("run").white,cr[cr.level==2500].sort_values("run").white); R["paired_white_1320_2500"]=dict(t=float(tt.statistic),p=float(tt.pvalue))

# power
from statsmodels.stats.power import NormalIndPower
from statsmodels.stats.proportion import proportion_effectsize
R["power"]={}
for p0 in [0.40,0.45,0.50]:
    # MDE two cells n=290 at 80% power, alpha .05 two-sided; and with Bonferroni over 45 pairs
    for a,lab in [(0.05,"raw"),(0.05/45,"bonf45")]:
        for dlt in np.arange(0.005,0.3,0.0005):
            h=proportion_effectsize(p0+dlt,p0)
            if NormalIndPower().power(h,N,a)>=0.8: R["power"][f"{p0}_{lab}"]=dlt*100; break
# DiD n per cell for 5pp/8pp/10pp at p=.45, alpha .05 and alpha .05/10
pw={}
for dd in [0.05,0.08,0.10]:
    for a,lab in [(0.05,"raw"),(0.005,"holm10")]:
        za=stats.norm.ppf(1-a/2); zb=stats.norm.ppf(.8)
        n=4*0.45*0.55*((za+zb)/dd)**2
        pw[f"{int(dd*100)}_{lab}"]=int(np.ceil(n))
R["did_n"]=pw
json.dump(R,open("results.json","w"),indent=1,default=lambda o: float(o) if hasattr(o,'__float__') else str(o))
print(json.dumps({k:R[k] for k in ["homog","glm","spearman","draw_tests","power","did_n"]},indent=1,default=lambda o: float(o) if hasattr(o,'__float__') else str(o)))
for L in LEVELS: print(L,{k:{kk:round(vv,3) for kk,vv in v.items()} for k,v in R["runs"][L].items()})
