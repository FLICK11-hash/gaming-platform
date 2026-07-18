import React, { useEffect, useRef, useState } from 'react'
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

function playerLabel(barColor, myColor) {
  const capitalized = barColor.charAt(0).toUpperCase() + barColor.slice(1)
  return barColor === myColor ? `You (${capitalized})` : `Opponent (${capitalized})`
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
            setOptimisticDrop(null)
            if (msg.board) setC4Board(msg.board)
            else setFen(msg.fen || START_FEN)
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

  const isConnectFour = matchInfo.game_type === 'connect_four'
  const isPlayer1 = userId === matchInfo.player_1_id
  const myColor = isConnectFour ? (isPlayer1 ? 'red' : 'green') : (isPlayer1 ? 'white' : 'black')
  const player1Color = isConnectFour ? 'red' : 'white'
  const player2Color = isConnectFour ? 'green' : 'black'
  const whiteId = matchInfo.player_1_id
  const blackId = matchInfo.player_2_id
  const board = withOptimisticMove(parseFEN(fen), optimisticMove)
  const displayC4Board = withOptimisticDrop(c4Board, optimisticDrop)
  const isMyTurn = !gameOver && !waiting && !optimisticMove && !optimisticDrop && turn === myColor
  const flip = !isConnectFour && myColor === 'black'

  function remainingFor(uid, colorForUid) {
    const base = clocks[String(uid)]
    if (base == null) return null
    if (turn !== colorForUid) return base
    const elapsed = Math.floor((Date.now() - clocksSyncedAt.current) / 1000)
    return Math.max(0, base - elapsed)
  }

  function sendMove(from, to, promotion) {
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'move', from, to, promotion }))
  }

  function sendDrop(column) {
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'move', column }))
  }

  function resign() {
    wsRef.current?.readyState === WebSocket.OPEN &&
      wsRef.current.send(JSON.stringify({ type: 'resign' }))
  }

  function handleColumnClick(column) {
    if (!isMyTurn) return
    const row = lowestEmptyRow(c4Board, column)
    if (row === -1) return
    setOptimisticDrop({ column, row, color: myColor === 'red' ? 1 : 2 })
    sendDrop(column)
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
        <h2>Match #{matchId} · {isConnectFour ? 'Connect Four' : 'Chess'} · {matchInfo.round}</h2>
        <span className={`conn-badge conn-${connStatus}`}>
          {connStatus === 'open' ? <Wifi size={14} /> : <WifiOff size={14} />} {connStatus}
        </span>
      </div>

      <div className="chess-layout">
        <div>
          <PlayerBar
            label={playerLabel(player1Color, myColor)}
            uid={whiteId}
            username={matchInfo.player_1_username}
            badgeKey={matchInfo.player_1_badge}
            isBot={matchInfo.player_1_is_bot}
            seconds={remainingFor(whiteId, player1Color)}
            active={turn === player1Color && !gameOver}
            frameKeys={matchInfo.player_1_profile_frames}
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
                      return (
                        <button
                          key={colIdx}
                          className={`c4-cell ${cell === 1 ? 'red' : cell === 2 ? 'green' : ''} ${isWinning ? 'winning' : ''}`}
                          onClick={() => handleColumnClick(colIdx)}
                          disabled={!isMyTurn || c4Board[0][colIdx] !== 0}
                          aria-label={`Drop in column ${colIdx + 1}`}
                        />
                      )
                    })}
                  </div>
                ))}
              </div>
            ) : (
              <div className="chessboard">
                {displayRanks.map(rankIdx => (
                  <div className="chess-row" key={rankIdx}>
                    {displayFiles.map(fileIdx => {
                      const piece = board[rankIdx][fileIdx]
                      const sq = squareName(rankIdx, fileIdx)
                      const dark = (rankIdx + fileIdx) % 2 === 1
                      const isSelected = selected === sq
                      const isWinning = winningSquare === sq
                      return (
                        <button
                          key={sq}
                          className={`chess-square ${dark ? 'dark' : 'light'} ${isSelected ? 'selected' : ''} ${isWinning ? 'winning' : ''}`}
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
            )}
          </div>
          <PlayerBar
            label={playerLabel(player2Color, myColor)}
            uid={blackId}
            username={matchInfo.player_2_username}
            badgeKey={matchInfo.player_2_badge}
            isBot={matchInfo.player_2_is_bot}
            seconds={remainingFor(blackId, player2Color)}
            active={turn === player2Color && !gameOver}
            frameKeys={matchInfo.player_2_profile_frames}
          />
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
            <div className={`post game-over ${gameOver.isDraw ? 'draw' : gameOver.isWinner ? 'win' : 'loss'}`}>
              <div className="game-over-emoji">
                {gameOver.isDraw ? '🤝' : gameOver.isWinner ? '🏆' : '💀'}
              </div>
              <b>{gameOver.message}</b>
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

function PlayerBar({ label, uid, username, badgeKey, isBot, seconds, active, frameKeys }) {
  const mm = seconds == null ? '--' : String(Math.floor(seconds / 60)).padStart(2, '0')
  const ss = seconds == null ? '--' : String(seconds % 60).padStart(2, '0')
  return (
    <div>
      <div className={`player-bar ${active ? 'active' : ''}`}>
        <span>
          {label} · {username || `#${uid}`}
          {isBot && <span className="bot-indicator" title="Computer opponent"> 🤖 Bot</span>}
          <EquippedBadge badgeKey={badgeKey} />
        </span>
        <strong className={seconds != null && seconds <= 10 ? 'clock-low' : ''}>{mm}:{ss}</strong>
      </div>
      <ProfileFrameBanner frameKeys={frameKeys} size="small" />
    </div>
  )
}
