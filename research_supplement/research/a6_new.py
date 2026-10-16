import json, numpy as np, pandas as pd, warnings
from scipy import stats
import statsmodels.api as sm, patsy
from statsmodels.stats.proportion import proportion_confint
from statsmodels.stats.contingency_tables import StratifiedTable
warnings.filterwarnings("ignore")
from data import WHITE, BLACK, WHITE_MOVES, BLACK_MOVES, LEVELS, N
R=json.load(open("results.json"))
UCI2NAME_W={"g1f3":"Reti","c2c4":"English","b1c3":"Van Geet","e2e4":"King's Pawn","d2d4":"Queen's Pawn","e2e3":"Van 't Kruijs","d2d3":"King's Indian Attack","b2b3":"Nimzo-Larsen","f2f4":"Bird's Opening","g2g3":"King's Fianchetto"}
UCI2NAME_B={"g7g6":"Modern","e7e6":"French","c7c6":"Caro-Kann","e7e5":"Open Game","g8f6":"Alekhine's","c7c5":"Sicilian","b8c6":"Nimzowitsch","a7a6":"St. George","d7d5":"Scandinavian","d7d6":"Pirc-style"}
def load(f):
    try: return pd.DataFrame([json.loads(l) for l in open(f)])
    except FileNotFoundError: return pd.DataFrame()
import sys
SRC=sys.argv[1] if len(sys.argv)>1 else "conrad"; KEY=sys.argv[2] if len(sys.argv)>2 else "new"
suf="_conrad" if SRC=="conrad" else ""
rep=load(f"data_{SRC}/replication{suf}.jsonl"); d3=load(f"data_{SRC}/d3_confirm{suf}.jsonl"); cal=load(f"data_{SRC}/calibration{suf}.jsonl")
NEW={}
def outcome_cols(df):
    df=df.copy()
    df["ww"]=(df.result=="1-0").astype(int); df["bw"]=(df.result=="0-1").astype(int); df["dr"]=(df.result=="1/2-1/2").astype(int)
    df["name"]=[UCI2NAME_W[m] if s=="white" else UCI2NAME_B[m] for m,s in zip(df.forced,df.set)]
    df["des_win"]=np.where(df.set=="white",df.ww,df.bw); df["des_loss"]=np.where(df.set=="white",df.bw,df.ww)
    df["des_score"]=df.des_win+0.5*df.dr
    return df
if len(rep):
    rep=outcome_cols(rep)
    NEW["rep_n"]=len(rep)
    # setting-level
    lev={}
    for L in LEVELS:
        a=rep[rep.tier==L]; w=a[a.set=="white"]; b=a[a.set=="black"]
        o=R["runs"][str(L)]
        def cmp(x,orig_mean,orig_n):
            p1=x.mean(); p0=orig_mean/100; se=np.sqrt(p1*(1-p1)/len(x)+p0*(1-p0)/orig_n)
            return dict(rep=100*p1,orig=orig_mean,diff=100*(p1-p0),lo=100*(p1-p0-1.96*se),hi=100*(p1-p0+1.96*se),p=2*stats.norm.sf(abs((p1-p0)/se)) if se>0 else 1.0,n=len(x))
        lev[L]=dict(white=cmp(w.ww,o["white"]["mean"],2900),black=cmp(b.bw,o["black"]["mean"],2900),draw=cmp(a.dr,o["draw"]["mean"],5800 if L!=2000 else 5600),
                    draw_w=100*w.dr.mean(),draw_b=100*b.dr.mean(),
                    fma=100*(a.ww.mean()-a.bw.mean()),white_score=100*(a.ww.mean()+0.5*a.dr.mean()),
                    plies=float(a.plies.mean()),plies_sd=float(a.plies.std()),
                    term={k:float(v) for k,v in (100*a.termination.value_counts(normalize=True)).items()},
                    secs=float(a.secs.mean()))
        # fma CI via per-game: pooled White win - Black win; se from multinomial
        pw,pb=a.ww.mean(),a.bw.mean(); se=np.sqrt((pw+pb-(pw-pb)**2)/len(a)); lev[L]["fma_lo"]=lev[L]["fma"]-196*se; lev[L]["fma_hi"]=lev[L]["fma"]+196*se
        # draw heterogeneity across openings within set
        for s in ["white","black"]:
            x=a[a.set==s].groupby("forced").dr.agg(["sum","count"])
            tab=np.vstack([x["sum"],x["count"]-x["sum"]])
            lev[L][f"drawhet_{s}"]=float(stats.chi2_contingency(tab)[1]) if tab[0].sum()>0 else 1.0
    NEW["levels"]={str(k):v for k,v in lev.items()}
    # duplicates
    NEW["dup_full"]=int(rep.moves.duplicated().sum())
    pref=rep.moves.str.split().str[:20].str.join(" ")
    NEW["dup_20"]=int(pref.duplicated().sum()); NEW["n_games_ge20"]=int((rep.plies>=20).sum())
    # per opening table
    tab=rep.groupby(["set","name","tier"]).agg(n=("des_win","size"),win=("des_win","sum"),draw=("dr","sum"),loss=("des_loss","sum"),score=("des_score","mean")).reset_index()
    tab.to_csv(f"rep_cells_{SRC}.csv",index=False)
    # consistency of opening effects across studies (designated wins), per color
    cons={}
    for col,D,s in [("White",WHITE,"white"),("Black",BLACK,"black")]:
        rows=[]
        for o,ks in D.items():
            for L,k in zip(LEVELS,ks): rows.append(dict(opening=o,lev=str(L),study="orig",wins=k,n=N))
        x=rep[rep.set==s].groupby(["name","tier"]).des_win.agg(["sum","count"]).reset_index()
        for _,r in x.iterrows(): rows.append(dict(opening=r["name"],lev=str(r.tier),study="rep",wins=int(r["sum"]),n=int(r["count"])))
        df=pd.DataFrame(rows); endog=np.c_[df.wins,df.n-df.wins]
        f=lambda form: sm.GLM(endog,patsy.dmatrix(form,df,return_type="dataframe"),family=sm.families.Binomial()).fit()
        m1=f("C(opening)+C(lev)*C(study)"); m2=f("C(opening)*C(study)+C(lev)*C(study)")
        ma=f("C(opening)+C(lev)+C(study)+C(lev):C(study)"); mi=f("C(opening)*C(lev)+C(lev):C(study)+C(study)")
        rr=df[df.study=="rep"]; er=np.c_[rr.wins,rr.n-rr.wins]
        g=lambda form: sm.GLM(er,patsy.dmatrix(form,rr,return_type="dataframe"),family=sm.families.Binomial()).fit()
        ra,ri=g("C(opening)+C(lev)"),g("C(opening)*C(lev)"); r0=g("C(lev)")
        cons[col]=dict(lr_study_open=m1.deviance-m2.deviance,df_study_open=9,p_study_open=stats.chi2.sf(m1.deviance-m2.deviance,9),
                       lr_int_pooled=ma.deviance-mi.deviance,p_int_pooled=stats.chi2.sf(ma.deviance-mi.deviance,18),
                       lr_int_rep=ra.deviance-ri.deviance,p_int_rep=stats.chi2.sf(ra.deviance-ri.deviance,18),
                       lr_open_rep=r0.deviance-ra.deviance,p_open_rep=stats.chi2.sf(r0.deviance-ra.deviance,9))
        # correlation of opening main effects (pooled over levels) between studies
        o_orig=np.array([sum(D[o])/(3*N) for o in D]); o_rep=np.array([rr[rr.opening==o].wins.sum()/rr[rr.opening==o].n.sum() for o in D])
        cons[col]["r_open"]=float(stats.pearsonr(o_orig,o_rep)[0]); cons[col]["p_r_open"]=float(stats.pearsonr(o_orig,o_rep)[1])
    NEW["consistency"]=cons
    # score-based interaction in replication (quasi-binomial on score)
    sc={}
    for s in ["white","black"]:
        x=rep[rep.set==s].copy(); x["lev"]=x.tier.astype(str)
        y=x.des_score.values
        fa=sm.GLM(y,patsy.dmatrix("C(name)+C(lev)",x,return_type="dataframe"),family=sm.families.Binomial()).fit(scale="X2")
        fi=sm.GLM(y,patsy.dmatrix("C(name)*C(lev)",x,return_type="dataframe"),family=sm.families.Binomial()).fit(scale="X2")
        F=((fa.deviance-fi.deviance)/18)/fi.scale; sc[s]=dict(F=F,p=float(stats.f.sf(F,18,fi.df_resid)))
    NEW["score_int"]=sc
# d3 confirmation: fresh White-set games from replication + d3_confirm
if len(d3) and len(rep):
    d3=outcome_cols(d3); fresh=pd.concat([rep[rep.set=="white"],d3]); NEW["fresh_white_n"]=len(fresh)
    tabs=[];out={}
    for L in LEVELS:
        a=fresh[fresh.tier==L]; x=a[a.forced=="d2d3"]; y=a[a.forced!="d2d3"]
        tabs.append(np.array([[x.ww.sum(),len(x)-x.ww.sum()],[y.ww.sum(),len(y)-y.ww.sum()]]))
        out[str(L)]=dict(d3=100*x.ww.mean(),others=100*y.ww.mean(),n_d3=len(x),d3_score=100*x.des_score.mean(),others_score=100*y.des_score.mean())
    st=StratifiedTable(tabs); z=(np.log(st.oddsratio_pooled))/np.sqrt(st.logodds_pooled_se**2)
    out["or"]=st.oddsratio_pooled; out["ci"]=list(st.oddsratio_pooled_confint()); out["p_two"]=st.test_null_odds().pvalue
    out["p_one"]=out["p_two"]/2 if st.oddsratio_pooled<1 else 1-out["p_two"]/2
    out["p_homog"]=st.test_equal_odds().pvalue
    out["d3_all"]=100*fresh[fresh.forced=="d2d3"].ww.mean(); out["others_all"]=100*fresh[fresh.forced!="d2d3"].ww.mean()
    out["n_d3"]=int((fresh.forced=="d2d3").sum()); out["n_others"]=int((fresh.forced!="d2d3").sum())
    # score-based
    xs=fresh[fresh.forced=="d2d3"].des_score; ys=fresh[fresh.forced!="d2d3"].des_score
    out["score_d3"]=100*xs.mean(); out["score_others"]=100*ys.mean(); t=stats.ttest_ind(xs,ys,equal_var=False); out["p_score_one"]=t.pvalue/2 if t.statistic<0 else 1-t.pvalue/2
    # all White moves in fresh data (pooled over tiers)
    out["by_move"]={UCI2NAME_W[m]:100*g.ww.mean() for m,g in fresh.groupby("forced")}
    NEW["d3"]=out
# calibration
if len(cal):
    cal["score_strong"]=np.where(cal.result=="1/2-1/2",0.5,np.where((cal.result=="1-0")&(~cal.weak_white),1.0,np.where((cal.result=="0-1")&(cal.weak_white),1.0,0.0)))
    cc={}
    for (lo,hi),g in cal.groupby(["weak","strong"]):
        s=g.score_strong.mean(); n=len(g); se=g.score_strong.std(ddof=1)/np.sqrt(n)
        elo=lambda p: -400*np.log10(1/min(max(p,1e-4),1-1e-4)-1)
        cc[f"{lo}:{hi}"]=dict(n=n,score=100*s,elo=elo(s),elo_lo=elo(s-1.96*se),elo_hi=elo(s+1.96*se),draws=100*(g.result=="1/2-1/2").mean(),
                              wins=int((g.score_strong==1).sum()),losses=int((g.score_strong==0).sum()),nominal=(max(lo,1350) if False else hi-max(lo,1350)))
    NEW["calibration"]=cc
R[KEY]=NEW; json.dump(R,open("results.json","w"),indent=1,default=lambda o: float(o) if hasattr(o,"__float__") else (int(o) if hasattr(o,"__int__") else str(o)))
print(json.dumps(NEW,indent=0,default=lambda o: float(o) if hasattr(o,"__float__") else str(o))[:3000])
