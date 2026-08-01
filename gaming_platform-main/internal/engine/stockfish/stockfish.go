// Package stockfish drives one or more real Stockfish chess engine
// processes over the UCI protocol, so bot opponents can play at a genuine,
// configurable playing strength instead of the platform's own hand-rolled
// move-choosing logic.
package stockfish

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// RealMatchMoveTime is how long each Stockfish search is given per move in
// a live match. Kept short deliberately: this powers a bullet-chess bot
// opponent, and per the platform's real-time fairness goals a bot's move
// needs to land well under a second, not run a multi-second search.
const RealMatchMoveTime = 200 * time.Millisecond

// MinUCIElo and MaxUCIElo are Stockfish's own supported range for the
// UCI_Elo strength-limiting option (as of Stockfish 16). Anything outside
// this range gets clamped by Stockfish itself rather than rejected, so
// asking for, say, 1200 quietly plays at 1320 instead - there is no way to
// make stock Stockfish play weaker than MinUCIElo through this option
// alone.
const (
	MinUCIElo = 1320
	MaxUCIElo = 3190
)

// Engine manages a fixed-size pool of persistent Stockfish processes, all
// configured to the same strength limit and per-move search budget.
// Reusing long-lived processes (rather than spawning one per move) avoids
// paying process-startup latency on every bot move, and the pool avoids
// serializing every concurrent match behind a single process.
type Engine struct {
	procs    chan *process
	moveTime time.Duration
}

// NewEngine starts poolSize Stockfish processes at path, each limited to
// elo playing strength via UCI_LimitStrength/UCI_Elo, searching for
// moveTime per move. If starting any process fails, every process already
// started is torn down and an error is returned - there's no
// partially-usable Engine.
func NewEngine(path string, elo, poolSize int, moveTime time.Duration) (*Engine, error) {
	if poolSize < 1 {
		return nil, errors.New("stockfish: poolSize must be at least 1")
	}

	e := &Engine{procs: make(chan *process, poolSize), moveTime: moveTime}
	for i := 0; i < poolSize; i++ {
		p, err := startProcess(path, elo)
		if err != nil {
			e.Close()
			return nil, fmt.Errorf("stockfish: start process %d/%d: %w", i+1, poolSize, err)
		}
		e.procs <- p
	}
	return e, nil
}

type process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
}

func startProcess(path string, elo int) (*process, error) {
	cmd := exec.Command(path)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	p := &process{cmd: cmd, stdin: stdin, stdout: scanner}

	for _, line := range []string{
		"uci",
	} {
		if err := p.send(line); err != nil {
			return nil, err
		}
	}
	if err := p.waitFor("uciok"); err != nil {
		return nil, err
	}
	for _, line := range []string{
		"setoption name UCI_LimitStrength value true",
		"setoption name UCI_Elo value " + strconv.Itoa(elo),
		// Minimum transposition table size: these bots search shallow,
		// short-movetime lines anyway, so a large hash table buys nothing,
		// and every process's memory footprint matters on a free-tier
		// deploy target with very little RAM to spare.
		"setoption name Hash value 1",
		"isready",
	} {
		if err := p.send(line); err != nil {
			return nil, err
		}
	}
	if err := p.waitFor("readyok"); err != nil {
		return nil, err
	}
	return p, nil
}

func (p *process) send(cmd string) error {
	_, err := io.WriteString(p.stdin, cmd+"\n")
	return err
}

func (p *process) waitFor(token string) error {
	for p.stdout.Scan() {
		if strings.Contains(p.stdout.Text(), token) {
			return nil
		}
	}
	if err := p.stdout.Err(); err != nil {
		return err
	}
	return fmt.Errorf("stockfish: process exited before %q", token)
}

// BestMove asks for the best move in fen, returning it in from/to/promo
// square form (e.g. "e2", "e4", and "" or "q"/"r"/"b"/"n") - the same shape
// bot.ChooseChessMove returns, so this can be used as a drop-in move
// source for it.
//
// Every call checks out one process from the pool (blocking if all are
// currently busy) and returns it when done, so concurrent callers never
// share a process mid-conversation.
func (e *Engine) BestMove(fen string) (from, to, promo string, err error) {
	p := <-e.procs
	defer func() { e.procs <- p }()

	if err := p.send("position fen " + fen); err != nil {
		return "", "", "", err
	}
	if err := p.send(fmt.Sprintf("go movetime %d", e.moveTime.Milliseconds())); err != nil {
		return "", "", "", err
	}

	for p.stdout.Scan() {
		line := p.stdout.Text()
		if !strings.HasPrefix(line, "bestmove") {
			continue
		}
		return parseBestMoveLine(line)
	}
	if err := p.stdout.Err(); err != nil {
		return "", "", "", err
	}
	return "", "", "", errors.New("stockfish: process exited before bestmove")
}

// parseBestMoveLine parses a UCI "bestmove <move> [ponder <move>]" line into
// from/to square form plus an optional promotion letter, e.g. "bestmove
// e7e8q ponder ..." becomes ("e7", "e8", "q"). Split out from BestMove so
// tests can exercise the parsing itself without depending on which move a
// real (strength-limited, and therefore not always optimal by design)
// engine actually chooses.
func parseBestMoveLine(line string) (from, to, promo string, err error) {
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", "", fmt.Errorf("stockfish: malformed bestmove line: %q", line)
	}
	move := fields[1]
	if move == "(none)" {
		return "", "", "", errors.New("stockfish: no legal move available")
	}
	if len(move) < 4 {
		return "", "", "", fmt.Errorf("stockfish: malformed move: %q", move)
	}
	from, to = move[0:2], move[2:4]
	if len(move) >= 5 {
		promo = move[4:5]
	}
	return from, to, promo, nil
}

// Close quits every process in the pool. It must only be called once no
// BestMove call is in flight and no further calls will be made - a
// concurrent BestMove would try to return its process to a channel that's
// already been closed, which panics. In practice that means: tests tearing
// down after they're done, not a live server's request path.
func (e *Engine) Close() {
	close(e.procs)
	for p := range e.procs {
		_ = p.send("quit")
		_ = p.cmd.Wait()
	}
}
