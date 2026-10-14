from data import *
import numpy as np, pandas as pd, json, matplotlib
matplotlib.use("Agg"); import matplotlib.pyplot as plt
from scipy import stats
plt.rcParams.update({"font.family":"DejaVu Serif","font.size":9,"axes.spines.top":False,"axes.spines.right":False,"savefig.dpi":220})
R=json.load(open("results.json")); cells=pd.read_csv("cells.csv"); cards=pd.read_csv("cards.csv")
LC={1320:"#4C72B0",2000:"#DD8452",2500:"#55A868"}
def save(n): plt.tight_layout(); plt.savefig(f"figures/{n}.png",bbox_inches="tight"); plt.close()

# Fig1 design schematic
fig,ax=plt.subplots(figsize=(7.2,3.0)); ax.axis("off")
def box(x,y,w,h,t,fc="#EEF2F7"):
    ax.add_patch(plt.Rectangle((x,y),w,h,fc=fc,ec="#333",lw=.8)); ax.text(x+w/2,y+h/2,t,ha="center",va="center",fontsize=7.2)
box(0.0,0.38,0.17,0.24,"One run\n(600 games)")
for i,(L,nm) in enumerate([(1320,"RookieBot"),(2000,"ExpertBot"),(2500,"GrandmasterBot")]):
    y=0.70-i*0.32; box(0.25,y,0.22,0.22,f"{nm}\nnominal {L}\n200 games")
    ax.annotate("",xy=(0.25,y+0.11),xytext=(0.17,0.5),arrowprops=dict(arrowstyle="->",lw=.7))
    box(0.55,y+0.115,0.45,0.10,"White set: 10 forced first moves × 10 games",fc="#DCE8F7")
    box(0.55,y,0.45,0.10,"Black set: 10 forced first replies × 10 games",fc="#FBEFD5")
    ax.annotate("",xy=(0.55,y+0.165),xytext=(0.47,y+0.11),arrowprops=dict(arrowstyle="->",lw=.6))
    ax.annotate("",xy=(0.55,y+0.05),xytext=(0.47,y+0.11),arrowprops=dict(arrowstyle="->",lw=.6))
ax.set_xlim(0,1); ax.set_ylim(0,1)
ax.text(0.5,-0.06,"30 runs planned; Run 7 excluded → 29 valid runs, 17,400 games, 290 games per opening × level cell",ha="center",fontsize=8,style="italic")
save("fig1_design")

# Fig2 run-level outcome rates
fig,axs=plt.subplots(1,3,figsize=(6.5,2.6),sharey=False)
for ax,(v,t) in zip(axs,[("white","White wins, White-opening set"),("black","Black wins, Black-defense set"),("draw","Draw rate")]):
    for i,L in enumerate(LEVELS):
        x=cards[cards.level==L][v].dropna().values; j=np.random.default_rng(i).uniform(-.12,.12,len(x))
        ax.scatter(i+j,x,s=9,alpha=.55,color=LC[L],lw=0)
        m=R["runs"][str(L)][v]; ax.errorbar(i+.28,m["mean"],yerr=[[m["mean"]-m["lo"]],[m["hi"]-m["mean"]]],fmt="o",color="k",ms=3.5,capsize=2,lw=.9)
    ax.set_xticks(range(3)); ax.set_xticklabels(LEVELS); ax.set_title(t,fontsize=8.5); ax.set_xlabel("Nominal strength setting")
axs[0].set_ylabel("Percent of games in run")
save("fig2_runlevel")

# Fig3/4 forest
for col,fn in [("White","fig3_white_forest"),("Black","fig4_black_forest")]:
    order=sorted(R["pooled"][col],key=lambda k:R["pooled"][col][k]["pct"])
    mv=WHITE_MOVES if col=="White" else BLACK_MOVES
    fig,ax=plt.subplots(figsize=(6.5,4.2))
    for i,o in enumerate(order):
        for j,L in enumerate(LEVELS):
            r=cells[(cells.color==col)&(cells.opening==o)&(cells.level==L)].iloc[0]
            y=i+(j-1)*0.22; ax.plot([r.lo,r.hi],[y,y],color=LC[L],lw=1.1); ax.plot(r.pct,y,"o",color=LC[L],ms=3.8,label=f"{L}" if i==0 else None)
    for j,L in enumerate(LEVELS):
        ax.axvline(R["homog"][f"{col}_{L}"]["pooled"],color=LC[L],ls=":",lw=.8)
    ax.set_yticks(range(10)); ax.set_yticklabels([f"{o} ({mv[o]})" for o in order])
    ax.set_xlabel(f"{col} win percentage with 95% Wilson interval (n = 290 per point)")
    ax.legend(title="Strength",frameon=False,loc="lower right",fontsize=8)
    save(fn)

# Fig5 trajectories
fig,axs=plt.subplots(1,2,figsize=(6.5,4.4),sharey=True)
for ax,(col,D) in zip(axs,[("White",WHITE),("Black",BLACK)]):
    cmap=plt.get_cmap("tab10")
    for i,(o,ks) in enumerate(D.items()):
        ax.plot(range(3),np.array(ks)/N*100,"-o",ms=3,lw=1,color=cmap(i),label=o)
    m=[R["homog"][f"{col}_{L}"]["pooled"] for L in LEVELS]; ax.plot(range(3),m,"k--",lw=2,label="Set mean")
    ax.set_xticks(range(3)); ax.set_xticklabels(LEVELS); ax.set_title(f"{col} set"); ax.set_xlabel("Nominal strength setting")
    ax.legend(fontsize=6.5,frameon=False,ncol=2,loc="upper center",bbox_to_anchor=(0.5,-0.2))
axs[0].set_ylabel("Designated-color win percentage")
save("fig5_trajectories")

# Fig6 DiD
fig,axs=plt.subplots(1,2,figsize=(6.5,3.4),sharex=True)
for ax,col in zip(axs,["White","Black"]):
    t=pd.DataFrame(R["change"][col]).sort_values("did")
    y=np.arange(len(t)); ax.errorbar(t.did,y,xerr=[t.did-t.did_lo,t.did_hi-t.did],fmt="o",color="#333",ms=3.5,capsize=2,lw=.9)
    ax.axvline(0,color="grey",lw=.8); ax.set_yticks(y); ax.set_yticklabels(t.opening,fontsize=7.5); ax.set_title(f"{col} set")
fig.supxlabel("Change from 1320 to 2500 relative to the mean change of the other nine openings (pp, 95% CI, unadjusted)",fontsize=8.5)
save("fig6_did")

# Fig7 null range
fig,axs=plt.subplots(2,3,figsize=(6.5,3.6),sharex=True)
for r,col in enumerate(["White","Black"]):
    for c,L in enumerate(LEVELS):
        ax=axs[r,c]; x=np.load(f"nullrange_{col}_{L}.npy"); h=R["homog"][f"{col}_{L}"]
        ax.hist(x,bins=np.arange(0,25,1.0345),color="#BBBBBB",edgecolor="white",density=True)
        ax.axvline(h["range"],color="crimson",lw=1.5); ax.set_title(f"{col}, {L}: P = {h['p_range']:.3f}",fontsize=8)
        ax.set_yticks([])
fig.supxlabel("Range (max − min) of win percentage across the 10 openings, percentage points",fontsize=8.5)
save("fig7_nullrange")

# Fig8 top-rank heatmap
fig,axs=plt.subplots(1,2,figsize=(6.5,3.4))
for ax,col in zip(axs,["White","Black"]):
    order=sorted(R["pooled"][col],key=lambda k:-R["pooled"][col][k]["pct"])
    M=np.array([[R["rank"][col][str(L)]["top"][o] for L in LEVELS] for o in order])
    im=ax.imshow(M,cmap="Greys",vmin=0,vmax=1,aspect="auto")
    for i in range(10):
        for j in range(3): ax.text(j,i,f"{M[i,j]:.2f}",ha="center",va="center",fontsize=7,color="white" if M[i,j]>.5 else "black")
    ax.set_xticks(range(3)); ax.set_xticklabels(LEVELS); ax.set_yticks(range(10)); ax.set_yticklabels(order,fontsize=7.5); ax.set_title(f"{col}: P(ranked first)")
save("fig8_toprank")

# Fig9 power
fig,ax=plt.subplots(figsize=(6.0,3.0))
n=np.arange(100,6001,10); p=.45; zb=stats.norm.ppf(.8)
for a,ls,lab in [(0.05,"-","α = .05"),(0.005,"--","α = .005")]:
    za=stats.norm.ppf(1-a/2)
    ax.plot(n,100*(za+zb)*np.sqrt(2*p*(1-p)/n),ls=ls,color="#4C72B0",label=f"Pairwise difference, {lab}")
    ax.plot(n,100*(za+zb)*np.sqrt(4*p*(1-p)/n),ls=ls,color="#C44E52",label=f"Interaction contrast, {lab}")
ax.axvline(290,color="grey",lw=.8); ax.text(310,24,"current n = 290",fontsize=7.5,color="grey")
ax.set_xlabel("Games per opening × level cell"); ax.set_ylabel("Min. detectable effect (pp)"); ax.set_ylim(0,26)
ax.legend(frameon=False,fontsize=7.5)
save("fig9_power")
print("ok")

# Fig10 rank-identification simulation
fig,ax=plt.subplots(figsize=(6.0,3.0))
for tau,ls in [("0.02",":"),("0.03","-"),("0.04","--")]:
    r=R["ranksim"][tau]; ax.plot([x["n"] for x in r],[x["p_top"] for x in r],ls,marker="o",ms=3,color="#333",label=f"True between-opening SD = {int(float(tau)*100)} pp")
ax.axvline(290,color="grey",lw=.8); ax.text(310,0.05,"current n = 290",fontsize=7.5,color="grey")
ax.axhline(0.8,color="#C44E52",lw=.7,ls="-."); ax.set_xscale("log"); ax.set_ylim(0,1)
ax.set_xlabel("Games per opening (log scale)"); ax.set_ylabel("P(observed leader is true best)")
ax.legend(frameon=False,fontsize=7.5,loc="upper left")
save("fig10_ranksim")

# Fig11 Connect Four positive control
c=R["c4"]; fig,ax=plt.subplots(figsize=(6.0,2.8))
x=np.arange(1,8); p=np.array(c["pct"]); lo=np.array([a for a,b in c["ci"]]); hi=np.array([b for a,b in c["ci"]])
ax.errorbar(x,p,yerr=[p-lo,hi-p],fmt="o",color="#333",capsize=3,ms=5)
ax.scatter([4],[p[3]],color="#C44E52",zorder=3,s=40)
ax.axhline(np.mean(p),ls=":",color="grey",lw=.8)
ax.set_xticks(x); ax.set_xlabel("Forced opening column (4 = center)"); ax.set_ylabel("First-player win % (95% CI)")
ax.set_ylim(20,75)
save("fig11_c4")

# Fig12 Study 1 vs Study 2 per-cell designated-color win %
import os
if os.path.exists("rep_cells.csv"):
    rc=pd.read_csv("rep_cells.csv")
    fig,axs=plt.subplots(1,2,figsize=(6.5,3.2),sharex=True,sharey=True)
    for ax,(col,s) in zip(axs,[("White","white"),("Black","black")]):
        for L in LEVELS:
            a=cells[(cells.color==col)&(cells.level==L)].set_index("opening").pct
            b=rc[(rc.set==s)&(rc.tier==L)].set_index("name"); b=100*b.win/b.n
            ax.scatter(a.values,b.reindex(a.index).values,color=LC[L],s=16,label=str(L))
        ax.plot([20,70],[20,70],color="grey",lw=.8,ls=":"); ax.set_title(f"{col} set"); ax.set_xlabel("Study 1 win % (n = 290)")
    axs[0].set_ylabel("Study 2 win % (n = 30)"); axs[1].legend(frameon=False,fontsize=7.5,title="Setting")
    save("fig12_replication")
