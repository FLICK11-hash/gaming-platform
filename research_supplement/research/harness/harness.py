"""Replication program for the opening study.

Reproduces the platform's game loop (internal/research/simulate and
internal/engine/stockfish at commit 5872d85) with full per-game logging:
  * Stockfish with UCI_LimitStrength=true and a target UCI_Elo per side
  * Hash 1 MB, 1 thread, fixed movetime per move
  * the current position is sent as FEN on every move (no move history)
  * automatic game end on checkmate, stalemate, insufficient material,
    fivefold repetition and the 75-move rule; 300 plies scored as a draw
"""
import json, subprocess, sys, time, argparse, random
import chess

WHITE_MOVES = ["g1f3", "c2c4", "b1c3", "e2e4", "d2d4", "e2e3", "d2d3", "b2b3", "f2f4", "g2g3"]
BLACK_MOVES = ["g7g6", "e7e6", "c7c6", "e7e5", "g8f6", "c7c5", "b8c6", "a7a6", "d7d5", "d7d6"]
MAX_PLIES = 300


class Engine:
    def __init__(self, path, hash_mb=1):
        self.p = subprocess.Popen([path], stdin=subprocess.PIPE, stdout=subprocess.PIPE, text=True, bufsize=1)
        self.send("uci"); self.wait("uciok")
        self.send(f"setoption name Hash value {hash_mb}")
        self.send("setoption name Threads value 1")
        self.send("isready"); self.wait("readyok")
        self.elo = None

    def send(self, s):
        self.p.stdin.write(s + "\n")

    def wait(self, token):
        while True:
            line = self.p.stdout.readline()
            if not line:
                raise RuntimeError("engine died")
            if line.startswith(token):
                return line

    def set_elo(self, elo):
        if elo == self.elo:
            return
        if elo is None:
            self.send("setoption name UCI_LimitStrength value false")
        else:
            self.send("setoption name UCI_LimitStrength value true")
            self.send(f"setoption name UCI_Elo value {elo}")
        self.elo = elo

    def best(self, board, elo, movetime_ms, history):
        self.set_elo(elo)
        if history:
            moves = " ".join(m.uci() for m in board.move_stack)
            self.send("position startpos" + (" moves " + moves if moves else ""))
        else:
            self.send("position fen " + board.fen())
        self.send(f"go movetime {movetime_ms}")
        return self.wait("bestmove").split()[1]

    def close(self):
        try:
            self.send("quit"); self.p.wait(timeout=5)
        except Exception:
            self.p.kill()


def play(eng_w, eng_b, elo_w, elo_b, forced, movetime, history=False, claim=False):
    board = chess.Board()
    for m in forced:
        board.push_uci(m)
    term = None
    while True:
        if claim:
            over = board.is_game_over(claim_draw=True)
        else:
            over = board.is_game_over(claim_draw=False)
        if over:
            o = board.outcome(claim_draw=claim)
            term = o.termination.name.lower(); res = o.result(); break
        if board.ply() >= MAX_PLIES:
            term = "ply_cap"; res = "1/2-1/2"; break
        if board.turn == chess.WHITE:
            mv = eng_w.best(board, elo_w, movetime, history)
        else:
            mv = eng_b.best(board, elo_b, movetime, history)
        board.push_uci(mv)
    return res, term, board.ply(), " ".join(m.uci() for m in board.move_stack)


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--engine", required=True)
    ap.add_argument("--study", required=True)
    ap.add_argument("--tiers", default="1320,2000,2500")
    ap.add_argument("--games", type=int, default=10, help="games per forced move per tier")
    ap.add_argument("--sets", default="white,black")
    ap.add_argument("--white-moves", default=",".join(WHITE_MOVES))
    ap.add_argument("--black-moves", default=",".join(BLACK_MOVES))
    ap.add_argument("--movetime", type=int, default=10)
    ap.add_argument("--history", action="store_true")
    ap.add_argument("--claim", action="store_true")
    ap.add_argument("--hash", type=int, default=1)
    ap.add_argument("--out", required=True)
    ap.add_argument("--seed", type=int, default=1)
    a = ap.parse_args()
    tiers = [int(t) for t in a.tiers.split(",")]
    jobs = []
    for t in tiers:
        for g in range(a.games):
            if "white" in a.sets:
                for m in a.white_moves.split(","):
                    jobs.append((t, "white", m, [m]))
            if "black" in a.sets:
                for m in a.black_moves.split(","):
                    jobs.append((t, "black", m, ["e2e4", m]))
    random.Random(a.seed).shuffle(jobs)          # interleave cells so drift cannot track treatment
    eng = Engine(a.engine, a.hash)
    with open(a.out, "a") as f:
        for i, (t, s, m, forced) in enumerate(jobs):
            t0 = time.time()
            res, term, plies, moves = play(eng, eng, t, t, forced, a.movetime, a.history, a.claim)
            f.write(json.dumps(dict(study=a.study, tier=t, set=s, forced=m, result=res, termination=term,
                                    plies=plies, secs=round(time.time() - t0, 2), moves=moves)) + "\n")
            f.flush()
    eng.close()


if __name__ == "__main__":
    main()
