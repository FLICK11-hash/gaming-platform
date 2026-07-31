import React, { useEffect, useLayoutEffect, useRef, useState } from 'react'
import { Flag, Wifi, WifiOff } from 'lucide-react'
import { api } from './api.js'
import { EquippedBadge, ProfileFrameBanner } from './cosmeticArt.jsx'

const START_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'
const C4_ROWS = 6
const C4_COLUMNS = 7

const PIECE_GLYPHS = {
  K: '♔', Q: '♕', R: '♖', B: '♗', N: '♘', P: '♙',
  k: '♚', q: '♛', r: '♜', b: '♝', n: '♞', p: '♟',
}

function emptyC4Board() {
  return Array.from({ length: C4_ROWS }, () => Array(C4_COLUMNS).fill(0))
}

function parseFEN(fen) {
  const board = Array.from({ length: 8 }, () => Array(8).fill(null))
  const placement = (fen || START_FEN).split(' ')[0]
  placement.split('/').forEach((rankStr, rankIdx) => {
    let file = 0
    for (const ch of rankStr) {
      if (/\d/.test(ch)) {
        file += Number(ch)
      } else {
        board[rankIdx][file] = ch
        file += 1
      }
    }
  })
  return board // board[0] = rank 8 ... board[7] = rank 1
}

function squareName(rankIdx, fileIdx) {
  const file = String.fromCharCode('a'.charCodeAt(0) + fileIdx)
  const rank = 8 - rankIdx
  return `${file}${rank}`
}

function isOwnPiece(piece, color) {
  if (!piece) return false
  return color === 'white' ? piece === piece.toUpperCase() : piece === piece.toLowerCase()
}

// Renders a move immediately, before the server confirms it, so the board
// feels responsive instead of freezing for a round trip. The server's next
// authoritative 'state' broadcast silently overwrites this preview.
function withOptimisticMove(board, move) {
  if (!move) return board
  const next = board.map(row => row.slice())
  const [fromRank, fromFile] = squareToIndices(move.from)
  const [toRank, toFile] = squareToIndices(move.to)
  let piece = next[fromRank][fromFile]
  if (piece && move.promotion) {
    piece = piece === piece.toUpperCase() ? move.promotion.toUpperCase() : move.promotion.toLowerCase()
  }
  next[fromRank][fromFile] = null
  next[toRank][toFile] = piece
  return next
}

function squareToIndices(square) {
  const file = square.charCodeAt(0) - 'a'.charCodeAt(0)
  const rank = 8 - Number(square[1])
  return [rank, file]
}

// Same idea as withOptimisticMove but for a dropped Connect Four chip.
function withOptimisticDrop(board, drop) {
  if (!drop) return board
  const next = board.map(row => row.slice())
  next[drop.row][drop.column] = drop.color
  return next
}

function lowestEmptyRow(board, column) {
  for (let r = C4_ROWS - 1; r >= 0; r--) {
    if (board[r][column] === 0) return r
  }
  return -1
}

// Best-effort diff between two board snapshots, used to figure out what an
// incoming server 'state' broadcast changed (so we know which sound/highlight
// to show for a move we didn't initiate ourselves). Not a full move parser —
// multi-square moves like castling just yield *a* plausible from/to pair.
function diffBoards(oldBoard, newBoard) {
  let from = null
  let to = null
  let captured = false
  for (let r = 0; r < 8; r++) {
    for (let f = 0; f < 8; f++) {
      const o = oldBoard[r][f]
      const n = newBoard[r][f]
      if (o === n) continue
      if (n == null && o != null) from = squareName(r, f)
      if (n != null) {
        to = squareName(r, f)
        if (o != null) captured = true
      }
    }
  }
  return from && to ? { from, to, captured } : null
}

const WIN_MESSAGES = [
  '🏆 Victory! You crushed it!',
  '🎉 GG! You won!',
  '🔥 Dominant win!',
  '👑 Champion status achieved!',
  '💪 Flawless victory!',
]

const LOSS_MESSAGES = [
  '😵 Tough break — you lost.',
  '💀 Defeated... run it back?',
  '😤 So close! Better luck next time.',
  '🥲 Not your day. Try again!',
  '😬 Ouch, that one hurt.',
]

const DRAW_MESSAGES = [
  "🤝 It's a draw!",
  '😐 Stalemate. Nobody wins this one.',
]

function randomOf(list) {
  return list[Math.floor(Math.random() * list.length)]
}

function wsURL(matchId, token) {
  const apiUrl = import.meta.env.VITE_API_URL
  let proto, host
  if (apiUrl) {
    const u = new URL(apiUrl)
    proto = u.protocol === 'https:' ? 'wss:' : 'ws:'
    host = u.host
  } else {
    proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
    host = window.location.host
  }
  return `${proto}//${host}/matches/${matchId}/ws?token=${encodeURIComponent(token)}`
}

// --- Lightweight synthesized sound effects (no audio assets, zero network
// latency). One shared AudioContext, lazily created on the first user
// gesture so the browser's autoplay policy doesn't block it. ---
let sharedAudioCtx = null
function getAudioCtx() {
  if (typeof window === 'undefined') return null
  const Ctx = window.AudioContext || window.webkitAudioContext
  if (!Ctx) return null
  if (!sharedAudioCtx) sharedAudioCtx = new Ctx()
  if (sharedAudioCtx.state === 'suspended') sharedAudioCtx.resume()
  return sharedAudioCtx
}

function tone(ctx, freq, start, duration, type = 'sine', gain = 0.16) {
  const osc = ctx.createOscillator()
  const g = ctx.createGain()
  osc.type = type
  osc.frequency.value = freq
  const t0 = ctx.currentTime + start
  g.gain.setValueAtTime(0, t0)
  g.gain.linearRampToValueAtTime(gain, t0 + 0.008)
  g.gain.exponentialRampToValueAtTime(0.0001, t0 + duration)
  osc.connect(g)
  g.connect(ctx.destination)
  osc.start(t0)
  osc.stop(t0 + duration + 0.02)
}

const SOUND_PLAYERS = {
  move: ctx => tone(ctx, 720, 0, 0.09, 'sine', 0.16),
  capture: ctx => { tone(ctx, 480, 0, 0.07, 'square', 0.14); tone(ctx, 260, 0.03, 0.12, 'square', 0.12) },
  drop: ctx => tone(ctx, 620, 0, 0.1, 'triangle', 0.16),
  illegal: ctx => tone(ctx, 160, 0, 0.16, 'sawtooth', 0.12),
  start: ctx => { tone(ctx, 523, 0, 0.09); tone(ctx, 784, 0.09, 0.16) },
  win: ctx => { tone(ctx, 523, 0, 0.1); tone(ctx, 659, 0.1, 0.1); tone(ctx, 784, 0.2, 0.24) },
  loss: ctx => { tone(ctx, 392, 0, 0.13, 'sine', 0.14); tone(ctx, 293, 0.12, 0.18, 'sine', 0.12) },
  draw: ctx => { tone(ctx, 440, 0, 0.12); tone(ctx, 440, 0.15, 0.14) },
}

function playSound(kind) {
  const ctx = getAudioCtx()
  if (!ctx) return
  const player = SOUND_PLAYERS[kind]
  if (player) player(ctx)
}

export default function MatchRoom({ matchId, token, userId, onExit }) {
  const [matchInfo, setMatchInfo] = useState(null)
  const [fen, setFen] = useState(START_FEN)
  const [c4Board, setC4Board] = useState(emptyC4Board)
  const [turn, setTurn] = useState('white')
  const [clocks, setClocks] = useState({})
  const [waiting, setWaiting] = useState(true)
  const [connStatus, setConnStatus] = useState('connecting')
  const [gameOver, setGameOver] = useState(null)
  const [selected, setSelected] = useState(null)
  const [pendingPromo, setPendingPromo] = useState(null)
  const [notice, setNotice] = useState('')
  const [countdown, setCountdown] = useState(null)
  const [optimisticMove, setOptimisticMove] = useState(null)
  const [optimisticDrop, setOptimisticDrop] = useState(null)
  const [winningSquare, setWinningSquare] = useState(null)
  const [winningCells, setWinningCells] = useState(null)
  const [premoveSelected, setPremoveSelected] = useState(null)
  const [premove, setPremove] = useState(null)
  const [premoveColumn, setPremoveColumn] = useState(null)
  const [lastMove, setLastMove] = useState(null)
  const [draggingFrom, setDraggingFrom] = useState(null)
  const [dragOverSquare, setDragOverSquare] = useState(null)
  const [, forceTick] = useState(0)

  const wsRef = useRef(null)
  const clocksSyncedAt = useRef(Date.now())
  const mountedRef = useRef(true)
  const gameOverRef = useRef(false)
  const lastMoveWasPremoveRef = useRef(false)
  const prevFenRef = useRef(START_FEN)
  const lastSentMoveRef = useRef(null)
  const boardRef = useRef(null)
  const dragInfoRef = useRef(null)
  const dragGhostRef = useRef(null)

  useEffect(() => {
    api(`/matches/${matchId}`, { token }).then(setMatchInfo).catch(err => setNotice(err.message))
  }, [matchId, token])

  useEffect(() => {
    gameOverRef.current = !!gameOver
    if (gameOver) {
      setPremove(null)
      setPremoveColumn(null)
      setPremoveSelected(null)
      playSound(gameOver.isDraw ? 'draw' : gameOver.isWinner ? 'win' : 'loss')
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [gameOver])

  useEffect(() => {
    const id = setInterval(() => forceTick(t => t + 1), 1000)
    return () => clearInterval(id)
  }, [])

  useEffect(() => {
    if (countdown == null) return
    if (countdown <= 0) {
      playSound('start')
      setCountdown(null)
      return
    }
    const id = setTimeout(() => setCountdown(c => (c == null ? null : c - 1)), 1000)
    return () => clearTimeout(id)
  }, [countdown])

  useEffect(() => {
    mountedRef.current = true
    let reconnectTimer = null
    prevFenRef.current = START_FEN
    lastSentMoveRef.current = null

    function connect() {
      if (!mountedRef.current) return
      setConnStatus('connecting')
      const ws = new WebSocket(wsURL(matchId, token))
      wsRef.current = ws

      ws.onopen = () => mountedRef.current && setConnStatus('open')
      ws.onclose = () => {
        if (!mountedRef.current) return
        setConnStatus('closed')
        if (!gameOverRef.current) reconnectTimer = setTimeout(connect, 600)
      }
      ws.onerror = () => {}
      ws.onmessage = evt => {
        const msg = JSON.parse(evt.data)
        if (!mountedRef.current) return
        switch (msg.type) {
          case 'waiting':
            setWaiting(true)
            break
          case 'countdown':
            setCountdown(msg.seconds || 5)
            break
          case 'state':
            setWaiting(false)
            setCountdown(null)
            setOptimisticMove(null)
            setOptimisticDrop(null)
            if (msg.board) {
              setC4Board(msg.board)
            } else {
              const newFen = msg.fen || START_FEN
              if (newFen !== prevFenRef.current) {
                const diff = diffBoards(parseFEN(prevFenRef.current), parseFEN(newFen))
                const sent = lastSentMoveRef.current
                const isOwn = diff && sent && diff.from === sent.from && diff.to === sent.to
                if (diff && !isOwn) {
                  playSound(diff.captured ? 'capture' : 'move')
                  setLastMove({ from: diff.from, to: diff.to })
                }
                lastSentMoveRef.current = null
                prevFenRef.current = newFen
              }
              setFen(newFen)
            }
            setTurn(msg.turn || 'white')
            setWinningSquare(msg.winning_square || null)
            setWinningCells(msg.winning_cells || null)
            if (msg.clocks) {
              setClocks(msg.clocks)
              clocksSyncedAt.current = Date.now()
            }
            break
          case 'match_over': {
            const winnerUserId = msg.winner_user_id || 0
            const isDraw = winnerUserId === 0
            const isWinner = winnerUserId === userId
            setGameOver({
              winnerUserId,
              isDraw,
              isWinner,
              message: randomOf(isDraw ? DRAW_MESSAGES : isWinner ? WIN_MESSAGES : LOSS_MESSAGES),
              isTimeTiebreak: msg.reason === 'c4_time_tiebreak',
            })
            break
          }
          case 'opponent_disconnected':
            setNotice('Opponent disconnected. They forfeit in 5 seconds if they don’t reconnect.')
            break
          case 'opponent_reconnected':
            setNotice('')
            break
          case 'error':
            setOptimisticMove(null)
            setOptimisticDrop(null)
            if (!lastMoveWasPremoveRef.current) playSound('illegal')
            setNotice((lastMoveWasPremoveRef.current ? 'Premove failed: ' : '') + (msg.error || 'Error'))
            break
          default:
            break
        }
      }
    }

    connect()

    return () => {
      mountedRef.current = false
      if (reconnectTimer) clearTimeout(reconnectTimer)
      wsRef.current?.close()
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [matchId, token])

  const isConnectFour = matchInfo?.game_type === 'connect_four'
  const isPlayer1 = !!matchInfo && userId === matchInfo.player_1_id
  const myColor = isConnectFour ? (isPlayer1 ? 'red' : 'green') : (isPlayer1 ? 'white' : 'black')
  const isMyTurn = !!matchInfo && !gameOver && !waiting && !optimisticMove && !optimisticDrop && turn === myColor

  // Fire any queued pre-move the instant it becomes this player's turn, so
  // it lands as close to the turn flip as possible.
  useEffect(() => {
    if (!isMyTurn) return
    if (premove) {
      setOptimisticMove({ from: premove.from, to: premove.to, promotion: premove.promotion })
      setLastMove({ from: premove.from, to: premove.to })
      sendMove(premove.from, premove.to, premove.promotion, true)
      setPremove(null)
    } else if (premoveColumn != null) {
      const row = lowestEmptyRow(c4Board, premoveColumn)
      if (row !== -1) {
        setOptimisticDrop({ column: premoveColumn, row, color: myColor === 'red' ? 1 : 2 })
        playSound('drop')
        sendDrop(premoveColumn, true)
      }
      setPremoveColumn(null)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [isMyTurn])

  // Keeps the dragged-piece ghost glued to the pointer even when a drag
  // starts moving before React re-renders with the new draggingFrom state.
  useLayoutEffect(() => {
    if (draggingFrom && dragInfoRef.current) {
      positionGhost(dragInfoRef.current.lastX, dragInfoRef.current.lastY)
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [draggingFrom])

  if (!matchInfo) return <div className="card wide chessroom"><p className="muted">Loading match...</p></div>

  // The viewer's own bar always renders at the bottom (below the board) and
  // the opponent's always at the top, regardless of which color/player slot
  // either of them actually occupies.
  const me = {
    color: myColor,
    uid: userId,
    username: isPlayer1 ? matchInfo.player_1_username : matchInfo.player_2_username,
    badgeKey: isPlayer1 ? matchInfo.player_1_badge : matchInfo.player_2_badge,
    isBot: isPlayer1 ? matchInfo.player_1_is_bot : matchInfo.player_2_is_bot,
    frameKeys: isPlayer1 ? matchInfo.player_1_profile_frames : matchInfo.player_2_profile_frames,
  }
  const opponent = {
    color: isConnectFour ? (isPlayer1 ? 'green' : 'red') : (isPlayer1 ? 'black' : 'white'),
    uid: isPlayer1 ? matchInfo.player_2_id : matchInfo.player_1_id,
    username: isPlayer1 ? matchInfo.player_2_username : matchInfo.player_1_username,
    badgeKey: isPlayer1 ? matchInfo.player_2_badge : matchInfo.player_1_badge,
    isBot: isPlayer1 ? matchInfo.player_2_is_bot : matchInfo.player_1_is_bot,
    frameKeys: isPlayer1 ? matchInfo.player_2_profile_frames : matchInfo.player_1_profile_frames,
  }
  const board = withOptimisticMove(parseFEN(fen), optimisticMove)
  const displayC4Board = withOptimisticDrop(c4Board, optimisticDrop)
  const flip = !isConnectFour && myColor === 'black'
  const interactive = !gameOver && !waiting

  function remainingFor(uid, colorForUid) {
    const base = clocks[String(uid)]
    if (base == null) return null
    // Once the match is over, freeze the display at whatever the clocks
    // last reported instead of continuing to count down against a turn
    // that no longer means anything.
    if (gameOver || turn !== colorForUid) return base
    const elapsed = Math.floor((Date.now() - clocksSyncedAt.current) / 1000)
    return Math.max(0, base - elapsed)
  }

  function sendMove(from, to, promotion, premove) {
    lastMoveWasPremoveRef.current = !!premove
    lastSentMoveRef.current = { from, to }
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'move', from, to, promotion, premove: !!premove }))
  }

  function sendDrop(column, premove) {
    lastMoveWasPremoveRef.current = !!premove
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'move', column, premove: !!premove }))
  }

  function resign() {
    if (!window.confirm('Resign this match? This counts as a loss.')) return
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'resign' }))
  }

  function handleColumnClick(column) {
    if (gameOver || waiting) return
    const row = lowestEmptyRow(c4Board, column)
    if (row === -1) return
    if (!isMyTurn) {
      setPremoveColumn(prev => (prev === column ? null : column))
      return
    }
    setOptimisticDrop({ column, row, color: myColor === 'red' ? 1 : 2 })
    playSound('drop')
    sendDrop(column)
  }

  function handlePremoveSquareClick(piece, sq) {
    if (isConnectFour) return
    if (!premoveSelected) {
      if (isOwnPiece(piece, myColor)) setPremoveSelected(sq)
      return
    }
    if (sq === premoveSelected) {
      setPremoveSelected(null)
      return
    }
    if (isOwnPiece(piece, myColor)) {
      setPremoveSelected(sq)
      return
    }

    const [selRankI, selFileI] = squareToIndices(premoveSelected)
    const movingPiece = board[selRankI][selFileI]
    const destRank = sq[1]
    const isPromotion = movingPiece && movingPiece.toLowerCase() === 'p' && (destRank === '8' || destRank === '1')

    setPremove({ from: premoveSelected, to: sq, promotion: isPromotion ? 'q' : '' })
    setPremoveSelected(null)
  }

  function handleSquareClick(rankIdx, fileIdx) {
    if (pendingPromo) return
    const piece = board[rankIdx][fileIdx]
    const sq = squareName(rankIdx, fileIdx)

    if (!isMyTurn) {
      if (!gameOver && !waiting) handlePremoveSquareClick(piece, sq)
      return
    }

    if (!selected) {
      if (isOwnPiece(piece, myColor)) setSelected(sq)
      return
    }
    if (sq === selected) {
      setSelected(null)
      return
    }
    if (isOwnPiece(piece, myColor)) {
      setSelected(sq)
      return
    }

    const selRank = board[8 - Number(selected[1])]
    const selFile = selected.charCodeAt(0) - 'a'.charCodeAt(0)
    const movingPiece = selRank[selFile]
    const destRank = sq[1]
    const isPromotion = movingPiece && movingPiece.toLowerCase() === 'p' && (destRank === '8' || destRank === '1')

    if (isPromotion) {
      setPendingPromo({ from: selected, to: sq })
      setSelected(null)
      return
    }

    const captured = !!piece
    setOptimisticMove({ from: selected, to: sq, promotion: '' })
    setLastMove({ from: selected, to: sq })
    playSound(captured ? 'capture' : 'move')
    sendMove(selected, sq, '')
    setSelected(null)
  }

  function cancelPremove() {
    setPremove(null)
    setPremoveColumn(null)
    setPremoveSelected(null)
  }

  function choosePromotion(letter) {
    if (!pendingPromo) return
    const [toRank, toFile] = squareToIndices(pendingPromo.to)
    const captured = !!board[toRank][toFile]
    setOptimisticMove({ from: pendingPromo.from, to: pendingPromo.to, promotion: letter })
    setLastMove({ from: pendingPromo.from, to: pendingPromo.to })
    playSound(captured ? 'capture' : 'move')
    sendMove(pendingPromo.from, pendingPromo.to, letter)
    setPendingPromo(null)
  }

  // --- Drag & drop (chess.com-style: click-to-move still works unchanged;
  // dragging a piece is layered on top and funnels into the exact same
  // handleSquareClick logic a two-click move would use, so there's only one
  // source of truth for move legality/UX). ---
  function squareFromPoint(clientX, clientY, rect) {
    const x = clientX - rect.left
    const y = clientY - rect.top
    if (x < 0 || y < 0 || x >= rect.width || y >= rect.height) return null
    let fileIdx = Math.min(7, Math.floor((x / rect.width) * 8))
    let rankIdx = Math.min(7, Math.floor((y / rect.height) * 8))
    if (flip) {
      fileIdx = 7 - fileIdx
      rankIdx = 7 - rankIdx
    }
    return { rank: rankIdx, file: fileIdx }
  }

  function positionGhost(clientX, clientY) {
    if (dragGhostRef.current) {
      dragGhostRef.current.style.transform = `translate(${clientX}px, ${clientY}px) translate(-50%, -50%)`
    }
  }

  function onSquarePointerDown(e, rankIdx, fileIdx) {
    if (isConnectFour || !interactive || pendingPromo || dragInfoRef.current) return
    const rect = boardRef.current.getBoundingClientRect()
    dragInfoRef.current = {
      fromRank: rankIdx,
      fromFile: fileIdx,
      startX: e.clientX,
      startY: e.clientY,
      lastX: e.clientX,
      lastY: e.clientY,
      rect,
      squareSize: rect.width / 8,
      pointerId: e.pointerId,
      moved: false,
      draggable: isOwnPiece(board[rankIdx][fileIdx], myColor),
    }
    e.currentTarget.setPointerCapture(e.pointerId)
  }

  function onSquarePointerMove(e) {
    const info = dragInfoRef.current
    if (!info || info.pointerId !== e.pointerId) return
    info.lastX = e.clientX
    info.lastY = e.clientY
    if (!info.draggable) return
    if (!info.moved) {
      const dx = e.clientX - info.startX
      const dy = e.clientY - info.startY
      if (Math.hypot(dx, dy) > 4) {
        info.moved = true
        setDraggingFrom(squareName(info.fromRank, info.fromFile))
        handleSquareClick(info.fromRank, info.fromFile)
      }
    }
    if (info.moved) {
      positionGhost(e.clientX, e.clientY)
      const dest = squareFromPoint(e.clientX, e.clientY, info.rect)
      const destSq = dest ? squareName(dest.rank, dest.file) : null
      setDragOverSquare(prev => (prev === destSq ? prev : destSq))
    }
  }

  function onSquarePointerUp(e) {
    const info = dragInfoRef.current
    if (!info || info.pointerId !== e.pointerId) return
    dragInfoRef.current = null
    if (info.moved) {
      setDraggingFrom(null)
      setDragOverSquare(null)
      const dest = squareFromPoint(e.clientX, e.clientY, info.rect)
      if (!dest || (dest.rank === info.fromRank && dest.file === info.fromFile)) {
        if (isMyTurn) setSelected(null)
        else setPremoveSelected(null)
      } else {
        handleSquareClick(dest.rank, dest.file)
      }
    } else {
      handleSquareClick(info.fromRank, info.fromFile)
    }
  }

  const displayRanks = flip ? [...Array(8).keys()].reverse() : [...Array(8).keys()]
  const displayFiles = flip ? [...Array(8).keys()].reverse() : [...Array(8).keys()]
  const bottomRank = displayRanks[displayRanks.length - 1]
  const leftFile = displayFiles[0]
  const draggingPiece = draggingFrom ? board[squareToIndices(draggingFrom)[0]][squareToIndices(draggingFrom)[1]] : null
  const dragSquareSize = dragInfoRef.current?.squareSize || 0

  return (
    <div className="card wide chessroom">
      <div className="chess-header">
        <h2>Match #{matchId} · {isConnectFour ? 'Connect Four' : 'Chess'} · {matchInfo.round}</h2>
        <span className={`conn-badge conn-${connStatus}`}>
          {connStatus === 'open' ? <Wifi size={14} /> : <WifiOff size={14} />} {connStatus}
        </span>
      </div>

      <div className="chess-layout">
        <div className="board-column">
          <PlayerBar
            isYou={false}
            uid={opponent.uid}
            username={opponent.username}
            badgeKey={opponent.badgeKey}
            isBot={opponent.isBot}
            color={opponent.color}
            seconds={remainingFor(opponent.uid, opponent.color)}
            active={turn === opponent.color && !gameOver}
            frameKeys={opponent.frameKeys}
          />
          <div className="board-wrap">
            {countdown != null && (
              <div className="countdown-overlay">
                <span>Get ready</span>
                <strong>{countdown > 0 ? countdown : 'Go!'}</strong>
              </div>
            )}
            {isConnectFour ? (
              <div className="connect-four-board">
                {displayC4Board.map((row, rowIdx) => (
                  <div className="c4-row" key={rowIdx}>
                    {row.map((cell, colIdx) => {
                      const isWinning = winningCells?.some(([r, c]) => r === rowIdx && c === colIdx)
                      const isPremoveTarget = premoveColumn === colIdx && rowIdx === lowestEmptyRow(displayC4Board, colIdx)
                      return (
                        <button
                          key={colIdx}
                          className={`c4-cell ${cell === 1 ? 'red' : cell === 2 ? 'green' : ''} ${isWinning ? 'winning' : ''} ${isPremoveTarget ? 'premove-target' : ''}`}
                          onClick={() => handleColumnClick(colIdx)}
                          disabled={gameOver || waiting || c4Board[0][colIdx] !== 0}
                          aria-label={`Drop in column ${colIdx + 1}`}
                        />
                      )
                    })}
                  </div>
                ))}
              </div>
            ) : (
              <div className={`chessboard ${draggingFrom ? 'dragging' : ''}`} ref={boardRef}>
                {displayRanks.map(rankIdx => (
                  <div className="chess-row" key={rankIdx}>
                    {displayFiles.map(fileIdx => {
                      const piece = board[rankIdx][fileIdx]
                      const sq = squareName(rankIdx, fileIdx)
                      const dark = (rankIdx + fileIdx) % 2 === 1
                      const isSelected = selected === sq
                      const isWinning = winningSquare === sq
                      const isPremoveFrom = premoveSelected === sq || premove?.from === sq
                      const isPremoveTo = premove?.to === sq
                      const isLastMove = lastMove && (lastMove.from === sq || lastMove.to === sq)
                      const isDragOver = dragOverSquare === sq
                      const isDraggingPiece = draggingFrom === sq
                      const hasOwnPiece = interactive && isOwnPiece(piece, myColor)
                      return (
                        <div
                          key={sq}
                          role="button"
                          tabIndex={interactive ? 0 : -1}
                          data-file={rankIdx === bottomRank ? String.fromCharCode(97 + fileIdx) : undefined}
                          data-rank={fileIdx === leftFile ? String(8 - rankIdx) : undefined}
                          className={`chess-square ${dark ? 'dark' : 'light'} ${isSelected ? 'selected' : ''} ${isWinning ? 'winning' : ''} ${isPremoveFrom ? 'premove-selected' : ''} ${isPremoveTo ? 'premove-target' : ''} ${isLastMove ? 'last-move' : ''} ${isDragOver ? 'drag-over' : ''} ${hasOwnPiece ? 'has-own-piece' : ''}`}
                          onPointerDown={e => onSquarePointerDown(e, rankIdx, fileIdx)}
                          onPointerMove={onSquarePointerMove}
                          onPointerUp={onSquarePointerUp}
                          onKeyDown={e => {
                            if ((e.key === 'Enter' || e.key === ' ') && interactive) {
                              e.preventDefault()
                              handleSquareClick(rankIdx, fileIdx)
                            }
                          }}
                        >
                          {piece && !isDraggingPiece && (
                            <span className={piece === piece.toUpperCase() ? 'piece-white' : 'piece-black'}>
                              {PIECE_GLYPHS[piece]}
                            </span>
                          )}
                        </div>
                      )
                    })}
                  </div>
                ))}
                {draggingFrom && draggingPiece && (
                  <div
                    ref={dragGhostRef}
                    className="drag-ghost"
                    style={{ width: dragSquareSize, height: dragSquareSize, fontSize: dragSquareSize * 0.72 }}
                  >
                    <span className={draggingPiece === draggingPiece.toUpperCase() ? 'piece-white' : 'piece-black'}>
                      {PIECE_GLYPHS[draggingPiece]}
                    </span>
                  </div>
                )}
              </div>
            )}
          </div>
          <PlayerBar
            isYou
            uid={me.uid}
            username={me.username}
            badgeKey={me.badgeKey}
            isBot={me.isBot}
            color={me.color}
            seconds={remainingFor(me.uid, me.color)}
            active={turn === me.color && !gameOver}
            frameKeys={me.frameKeys}
          />
        </div>

        <div className="chess-side">
          {waiting && countdown == null && !gameOver && <p className="muted">Waiting for your opponent to connect...</p>}
          {countdown != null && !gameOver && <p className="muted">Both players connected. Match starts in a moment...</p>}
          {!waiting && !gameOver && <p className={isMyTurn ? 'turn-yours' : 'muted'}>{isMyTurn ? 'Your move' : "Opponent's move"}</p>}
          {!isMyTurn && !gameOver && !waiting && (premove || premoveColumn != null) && (
            <p className="notice premove-notice">
              ⏩ Premove queued: {premove ? `${premove.from} → ${premove.to}` : `column ${premoveColumn + 1}`} (costs 0.2s){' '}
              <button type="button" className="link-button" onClick={cancelPremove}>Cancel</button>
            </p>
          )}
          {notice && <p className="notice">{notice}</p>}

          {pendingPromo && (
            <div className="promo-picker">
              <p>Promote to:</p>
              <div className="row">
                {['q', 'r', 'b', 'n'].map(l => (
                  <button key={l} onClick={() => choosePromotion(l)}>
                    {PIECE_GLYPHS[myColor === 'white' ? l.toUpperCase() : l]}
                  </button>
                ))}
              </div>
            </div>
          )}

          {gameOver && (
            <div className={`post game-over ${gameOver.isDraw ? 'draw' : gameOver.isWinner ? 'win' : 'loss'}`}>
              <div className="game-over-emoji">
                {gameOver.isDraw ? '🤝' : gameOver.isWinner ? '🏆' : '💀'}
              </div>
              <b>{gameOver.message}</b>
              {gameOver.isTimeTiebreak && (
                <p className="tiebreak-note">
                  Board full with no winner — player with more time remaining wins.
                </p>
              )}
              <button onClick={onExit}>Back to Lobby</button>
            </div>
          )}

          {!gameOver && (
            <button className="danger" onClick={resign}><Flag size={16} /> Resign</button>
          )}
        </div>
      </div>
    </div>
  )
}

function PlayerBar({ isYou, uid, username, badgeKey, isBot, color, seconds, active, frameKeys }) {
  const mm = seconds == null ? '--' : String(Math.floor(seconds / 60)).padStart(2, '0')
  const ss = seconds == null ? '--' : String(seconds % 60).padStart(2, '0')
  return (
    <div>
      <div className={`player-bar ${active ? 'active' : ''}`}>
        <span className="player-bar-id">
          <span className={`side-dot side-${color}`} title={color} />
          <span className="player-bar-name">
            {isYou ? 'You' : 'Opponent'} · {username || `#${uid}`}
          </span>
          {isBot && <span className="bot-indicator" title="Computer opponent"> 🤖 Bot</span>}
          <EquippedBadge badgeKey={badgeKey} />
        </span>
        <strong className={seconds != null && seconds <= 10 ? 'clock-low' : ''}>{mm}:{ss}</strong>
      </div>
      <ProfileFrameBanner frameKeys={frameKeys} size="small" />
    </div>
  )
}
