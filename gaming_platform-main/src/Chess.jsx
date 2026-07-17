import React, { useEffect, useRef, useState } from 'react'
import { Flag, Wifi, WifiOff } from 'lucide-react'
import { api } from './api.js'
import { ProfileFrameBanner } from './cosmeticArt.jsx'

const START_FEN = 'rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1'

const PIECE_GLYPHS = {
  K: '♔', Q: '♕', R: '♖', B: '♗', N: '♘', P: '♙',
  k: '♚', q: '♛', r: '♜', b: '♝', n: '♞', p: '♟',
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

export default function MatchRoom({ matchId, token, userId, onExit }) {
  const [matchInfo, setMatchInfo] = useState(null)
  const [fen, setFen] = useState(START_FEN)
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
  const [, forceTick] = useState(0)

  const wsRef = useRef(null)
  const clocksSyncedAt = useRef(Date.now())
  const mountedRef = useRef(true)
  const gameOverRef = useRef(false)

  useEffect(() => {
    api(`/matches/${matchId}`, { token }).then(setMatchInfo).catch(err => setNotice(err.message))
  }, [matchId, token])

  useEffect(() => {
    gameOverRef.current = !!gameOver
  }, [gameOver])

  useEffect(() => {
    const id = setInterval(() => forceTick(t => t + 1), 1000)
    return () => clearInterval(id)
  }, [])

  useEffect(() => {
    if (countdown == null) return
    if (countdown <= 0) {
      setCountdown(null)
      return
    }
    const id = setTimeout(() => setCountdown(c => (c == null ? null : c - 1)), 1000)
    return () => clearTimeout(id)
  }, [countdown])

  useEffect(() => {
    mountedRef.current = true
    let reconnectTimer = null

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
            setFen(msg.fen || START_FEN)
            setTurn(msg.turn || 'white')
            if (msg.clocks) {
              setClocks(msg.clocks)
              clocksSyncedAt.current = Date.now()
            }
            break
          case 'match_over':
            setGameOver({ winnerUserId: msg.winner_user_id || 0 })
            break
          case 'opponent_disconnected':
            setNotice('Opponent disconnected. They forfeit in 5 seconds if they don’t reconnect.')
            break
          case 'opponent_reconnected':
            setNotice('')
            break
          case 'error':
            setOptimisticMove(null)
            setNotice(msg.error || 'Error')
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

  if (!matchInfo) return <div className="card wide chessroom"><p className="muted">Loading match...</p></div>

  const myColor = userId === matchInfo.player_1_id ? 'white' : 'black'
  const whiteId = matchInfo.player_1_id
  const blackId = matchInfo.player_2_id
  const board = withOptimisticMove(parseFEN(fen), optimisticMove)
  const isMyTurn = !gameOver && !waiting && !optimisticMove && turn === myColor
  const flip = myColor === 'black'

  function remainingFor(uid) {
    const base = clocks[String(uid)]
    if (base == null) return null
    const theirTurn = (uid === whiteId && turn === 'white') || (uid === blackId && turn === 'black')
    if (!theirTurn) return base
    const elapsed = Math.floor((Date.now() - clocksSyncedAt.current) / 1000)
    return Math.max(0, base - elapsed)
  }

  function sendMove(from, to, promotion) {
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'move', from, to, promotion }))
  }

  function resign() {
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'resign' }))
  }

  function handleSquareClick(rankIdx, fileIdx) {
    if (!isMyTurn || pendingPromo) return
    const piece = board[rankIdx][fileIdx]
    const sq = squareName(rankIdx, fileIdx)

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

    setOptimisticMove({ from: selected, to: sq, promotion: '' })
    sendMove(selected, sq, '')
    setSelected(null)
  }

  function choosePromotion(letter) {
    if (!pendingPromo) return
    setOptimisticMove({ from: pendingPromo.from, to: pendingPromo.to, promotion: letter })
    sendMove(pendingPromo.from, pendingPromo.to, letter)
    setPendingPromo(null)
  }

  const displayRanks = flip ? [...Array(8).keys()].reverse() : [...Array(8).keys()]
  const displayFiles = flip ? [...Array(8).keys()].reverse() : [...Array(8).keys()]

  return (
    <div className="card wide chessroom">
      <div className="chess-header">
        <h2>Match #{matchId} · {matchInfo.round}</h2>
        <span className={`conn-badge conn-${connStatus}`}>
          {connStatus === 'open' ? <Wifi size={14} /> : <WifiOff size={14} />} {connStatus}
        </span>
      </div>

      <div className="chess-layout">
        <div>
          <PlayerBar label={myColor === 'black' ? 'Opponent (White)' : 'You (White)'} uid={whiteId} seconds={remainingFor(whiteId)} active={turn === 'white' && !gameOver} frameKeys={matchInfo.player_1_profile_frames} />
          <div className="board-wrap">
            {countdown != null && (
              <div className="countdown-overlay">
                <span>Get ready</span>
                <strong>{countdown > 0 ? countdown : 'Go!'}</strong>
              </div>
            )}
            <div className="chessboard">
            {displayRanks.map(rankIdx => (
              <div className="chess-row" key={rankIdx}>
                {displayFiles.map(fileIdx => {
                  const piece = board[rankIdx][fileIdx]
                  const sq = squareName(rankIdx, fileIdx)
                  const dark = (rankIdx + fileIdx) % 2 === 1
                  const isSelected = selected === sq
                  return (
                    <button
                      key={sq}
                      className={`chess-square ${dark ? 'dark' : 'light'} ${isSelected ? 'selected' : ''}`}
                      onClick={() => handleSquareClick(rankIdx, fileIdx)}
                      disabled={!isMyTurn}
                    >
                      {piece && (
                        <span className={piece === piece.toUpperCase() ? 'piece-white' : 'piece-black'}>
                          {PIECE_GLYPHS[piece]}
                        </span>
                      )}
                    </button>
                  )
                })}
              </div>
            ))}
            </div>
          </div>
          <PlayerBar label={myColor === 'white' ? 'Opponent (Black)' : 'You (Black)'} uid={blackId} seconds={remainingFor(blackId)} active={turn === 'black' && !gameOver} frameKeys={matchInfo.player_2_profile_frames} />
        </div>

        <div className="chess-side">
          {waiting && countdown == null && !gameOver && <p className="muted">Waiting for your opponent to connect...</p>}
          {countdown != null && !gameOver && <p className="muted">Both players connected. Match starts in a moment...</p>}
          {!waiting && !gameOver && <p className={isMyTurn ? 'turn-yours' : 'muted'}>{isMyTurn ? 'Your move' : "Opponent's move"}</p>}
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
            <div className="post game-over">
              <b>
                {gameOver.winnerUserId === 0
                  ? 'Draw'
                  : gameOver.winnerUserId === userId
                    ? 'You won!'
                    : 'You lost.'}
              </b>
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

function PlayerBar({ label, uid, seconds, active, frameKeys }) {
  const mm = seconds == null ? '--' : String(Math.floor(seconds / 60)).padStart(2, '0')
  const ss = seconds == null ? '--' : String(seconds % 60).padStart(2, '0')
  return (
    <div>
      <div className={`player-bar ${active ? 'active' : ''}`}>
        <span>{label} · #{uid}</span>
        <strong className={seconds != null && seconds <= 10 ? 'clock-low' : ''}>{mm}:{ss}</strong>
      </div>
      <ProfileFrameBanner frameKeys={frameKeys} size="small" />
    </div>
  )
}
