import React, { useEffect, useState } from 'react'
import { Zap, Users, Swords, Trophy, Crown, Sparkles } from 'lucide-react'
import { api } from './api.js'
import './landing.css'

const CAROUSEL_SLIDES = [
  { key: 'chess', title: 'Bullet Chess' },
  { key: 'connect-four', title: 'Connect Four' },
  { key: 'shop', title: 'Cosmetic Shop' }
]

// Hand-drawn inline SVG chessboard mockup, in the same spirit as the
// cosmetic badge art in cosmeticArt.jsx - no screenshots, just illustration.
function ChessMockup() {
  const squares = []
  for (let row = 0; row < 6; row++) {
    for (let col = 0; col < 6; col++) {
      const dark = (row + col) % 2 === 1
      squares.push(
        <rect
          key={`${row}-${col}`}
          x={col * 30}
          y={row * 30}
          width={30}
          height={30}
          fill={dark ? '#1a2436' : '#2b3a55'}
        />
      )
    }
  }
  return (
    <svg viewBox="0 0 180 180" className="mockup-art" role="img" aria-label="Chess board illustration">
      {squares}
      <text x="45" y="55" fontSize="26" textAnchor="middle" fill="#f8fafc">♜</text>
      <text x="105" y="55" fontSize="26" textAnchor="middle" fill="#f8fafc">♞</text>
      <text x="75" y="115" fontSize="26" textAnchor="middle" fill="#0b1020">♚</text>
      <text x="135" y="145" fontSize="26" textAnchor="middle" fill="#0b1020">♟</text>
    </svg>
  )
}

function ConnectFourMockup() {
  const cells = []
  const filled = {
    '0-3': 'red', '1-3': 'green', '2-3': 'red',
    '0-4': 'green', '1-4': 'red', '0-2': 'green'
  }
  for (let row = 0; row < 5; row++) {
    for (let col = 0; col < 6; col++) {
      const key = `${col}-${row}`
      const color = filled[key]
      cells.push(
        <circle
          key={key}
          cx={col * 30 + 15}
          cy={row * 30 + 15}
          r={12}
          fill={color === 'red' ? '#ef4444' : color === 'green' ? '#22c55e' : '#0b1020'}
        />
      )
    }
  }
  return (
    <svg viewBox="0 0 180 150" className="mockup-art" role="img" aria-label="Connect Four board illustration">
      <rect x="0" y="0" width="180" height="150" fill="#1d4ed8" rx="10" />
      {cells}
    </svg>
  )
}

function ShopMockup() {
  const tiers = [
    { color: '#b87333', label: 'Bronze' },
    { color: '#c7d2e0', label: 'Silver' },
    { color: '#f5c542', label: 'Gold' }
  ]
  return (
    <svg viewBox="0 0 180 150" className="mockup-art" role="img" aria-label="Cosmetic shop illustration">
      <rect x="0" y="0" width="180" height="150" fill="#111827" rx="10" />
      {tiers.map((tier, i) => (
        <g key={tier.label} transform={`translate(${20 + i * 55},30)`}>
          <polygon
            points="30,86 6,66 6,24 30,4 54,24 54,66"
            fill={tier.color}
            stroke="#0b1020"
            strokeWidth="2"
          />
          <text x="30" y="50" fontSize="20" textAnchor="middle" fill="#0b1020">★</text>
        </g>
      ))}
    </svg>
  )
}

const MOCKUP_COMPONENTS = {
  chess: ChessMockup,
  'connect-four': ConnectFourMockup,
  shop: ShopMockup
}

// Purely decorative, client-side matchmaking preview: loops between
// "searching" and "matched" states so logged-out visitors get a feel for
// the queue flow without needing any real backend call.
function MatchmakingPreview() {
  const [phase, setPhase] = useState('searching')

  useEffect(() => {
    const timer = setInterval(() => {
      setPhase(p => (p === 'searching' ? 'matched' : 'searching'))
    }, 2600)
    return () => clearInterval(timer)
  }, [])

  return (
    <div className="matchmaking-preview">
      <div className={`mm-avatar mm-you ${phase === 'matched' ? 'mm-pulse-done' : 'mm-pulse'}`}>
        <Swords size={22} />
      </div>
      <div className="mm-status">
        <span className={phase === 'matched' ? 'mm-matched-text' : 'mm-searching-text'}>
          {phase === 'searching' ? 'Searching for opponent...' : 'Matched!'}
        </span>
        <div className="mm-dots">
          <span />
          <span />
          <span />
        </div>
      </div>
      <div className={`mm-avatar mm-opponent ${phase === 'matched' ? 'mm-pulse-done' : 'mm-pulse'}`}>
        <Swords size={22} />
      </div>
    </div>
  )
}

function StatTile({ icon, label, value }) {
  return (
    <div className="landing-stat">
      {icon}
      <div>
        <strong>{value}</strong>
        <span>{label}</span>
      </div>
    </div>
  )
}

function PlayerPreviewRow({ entry, rank, unit }) {
  return (
    <div className="landing-player-row">
      <span className="landing-player-rank">#{rank}</span>
      <span className="landing-player-name">{entry.username}</span>
      <span className="landing-player-value">
        {entry.value} {unit}
      </span>
    </div>
  )
}

export default function Landing({ onGetStarted }) {
  const [stats, setStats] = useState(null)
  const [slide, setSlide] = useState(0)

  useEffect(() => {
    api('/public/landing-stats')
      .then(setStats)
      .catch(() => setStats(null))
  }, [])

  useEffect(() => {
    const timer = setInterval(() => {
      setSlide(s => (s + 1) % CAROUSEL_SLIDES.length)
    }, 3500)
    return () => clearInterval(timer)
  }, [])

  const topPlayers = stats?.top_players || []
  const champions = stats?.tournament_champions || []
  const ActiveMockup = MOCKUP_COMPONENTS[CAROUSEL_SLIDES[slide].key]

  return (
    <div className="landing">
      <section className="landing-hero">
        <Sparkles className="landing-hero-sparkle" size={28} />
        <h1>Play Bullet Chess in Seconds</h1>
        <p>
          Jump into fast-paced chess and Connect Four matches, climb the Elo
          leaderboard, and earn cosmetics along the way.
        </p>
        <button className="landing-cta" onClick={onGetStarted}>
          <Zap size={18} />
          Play Now
        </button>
      </section>

      <section className="landing-stats-row">
        <StatTile
          icon={<Users size={22} />}
          label="Registered Players"
          value={stats ? stats.total_players.toLocaleString() : '...'}
        />
        <StatTile
          icon={<Swords size={22} />}
          label="Matches Played"
          value={stats ? stats.matches_played.toLocaleString() : '...'}
        />
      </section>

      <section className="landing-card">
        <h2>See It In Action</h2>
        <div className="landing-carousel">
          <div className="landing-carousel-frame">
            <ActiveMockup />
            <p className="landing-carousel-caption">{CAROUSEL_SLIDES[slide].title}</p>
          </div>
          <div className="landing-carousel-dots">
            {CAROUSEL_SLIDES.map((s, i) => (
              <button
                key={s.key}
                className={i === slide ? 'dot dot-active' : 'dot'}
                aria-label={`Show ${s.title}`}
                onClick={() => setSlide(i)}
              />
            ))}
          </div>
        </div>
      </section>

      <section className="landing-two-col">
        <div className="landing-card">
          <h2>
            <Trophy size={20} /> Top Players
          </h2>
          {topPlayers.length === 0 ? (
            <p className="muted">Leaderboard is warming up - be the first champion.</p>
          ) : (
            topPlayers.map((entry, i) => (
              <PlayerPreviewRow key={entry.username} entry={entry} rank={i + 1} unit="Elo" />
            ))
          )}
        </div>

        <div className="landing-card">
          <h2>
            <Crown size={20} /> Tournament Winners
          </h2>
          {champions.length === 0 ? (
            <p className="muted">No champions yet - tournament mode is coming soon.</p>
          ) : (
            champions.map((entry, i) => (
              <PlayerPreviewRow key={entry.username} entry={entry} rank={i + 1} unit="titles" />
            ))
          )}
        </div>
      </section>

      <section className="landing-card landing-matchmaking-section">
        <h2>Matchmaking Made Easy</h2>
        <p className="muted">Queue up and get matched with an opponent near your skill level.</p>
        <MatchmakingPreview />
      </section>

      <section className="landing-final-cta">
        <h2>Ready to make your first move?</h2>
        <button className="landing-cta" onClick={onGetStarted}>
          <Zap size={18} />
          Get Started
        </button>
      </section>
    </div>
  )
}
