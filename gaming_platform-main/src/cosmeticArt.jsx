import React from 'react'
import { Crown, Flame, Award } from 'lucide-react'

export const AVATAR_GLYPHS = {
  rook: '♜',
  knight: '♞',
  king: '♚'
}

const BADGE_ICONS = {
  'first-victory': Award,
  'hot-streak': Flame,
  grandmaster: Crown
}

const BADGE_LABELS = {
  'first-victory': 'First Victory',
  'hot-streak': 'Hot Streak',
  grandmaster: 'Grandmaster'
}

// Comedic, non-chess profile frames: puns rendered big and loud next to a
// player's name once equipped.
export const PROFILE_FRAMES = {
  'big-dill': { emoji: '🥒', label: 'Kind of a Big Dill' },
  'lettuce-celebrate': { emoji: '🥬', label: 'Lettuce Celebrate' },
  'nacho-average': { emoji: '🌮', label: 'Nacho Average Player' },
  'holy-guacamole': { emoji: '🥑', label: 'Holy Guacamole' },
  'donut-even': { emoji: '🍩', label: 'Donut Even Try It' },
  'legend-dairy': { emoji: '🧀', label: 'Legend-dairy' },
  // Public-domain literary/folklore characters, played for laughs.
  dracula: { emoji: '🧛', label: 'Fangs for the Win' },
  'sherlock-holmes': { emoji: '🕵️', label: 'Elementary, My Dear Winner' },
  frankenstein: { emoji: '🧟', label: "It's Alive! (And Winning)" },
  medusa: { emoji: '🐍', label: 'Petrifying Performance' },
  'robin-hood': { emoji: '🏹', label: 'Robbing the Rich (of Their Coins)' },
  'king-arthur': { emoji: '⚔️', label: 'Sword in the Stone-Cold Killer' },
  'sun-wukong': { emoji: '🐒', label: 'Monkey Business Champion' },
  zeus: { emoji: '⚡', label: 'Thunderously Good' },
  'paul-bunyan': { emoji: '🪓', label: 'Larger Than Life W' },
  'uncle-sam': { emoji: '🎩', label: 'I Want YOU... To Lose' },
  'bat-guy': { emoji: '🏏', label: 'Bat Outta Nowhere' },
  // Public-domain character x modern-day-job/hobby mashups.
  'sherlock-holmes-gamer': { emoji: '🕵️🎮', label: 'Elementary, My Dear K/D Ratio' },
  'dracula-drivethru': { emoji: '🧛🍟', label: 'Sucks at Small Talk, Great at Fries' },
  'robin-hood-gig': { emoji: '🏹🛵', label: 'Steals From the Rich, Delivers to Everyone' },
  'winnie-pooh-hedgefund': { emoji: '🐻📈', label: 'Bearish Market, Bullish on Honey' },
  'frankenstein-gym': { emoji: '🧟💪', label: 'Built Different (Literally)' },
  'king-arthur-wholesale': { emoji: '⚔️🛒', label: 'Sword in the Bulk Aisle' },
  'alice-wonderland-ai': { emoji: '🐇🤖', label: 'Curiouser and Curiouser Prompts' },
  'medusa-salon': { emoji: '🐍💇', label: 'Snake Charmer Salon' }
}

// Tier palettes for the badge medallion artwork. Each tier gets its own
// metallic gradient, rim color, and (for the top two tiers) sparkle accents,
// so pricier cosmetics visibly read as more prestigious.
const TIER_COLORS = {
  bronze: { light: '#e8a86c', base: '#b87333', dark: '#7a4a1f', rim: '#f2c894' },
  silver: { light: '#f4f8fc', base: '#c7d2e0', dark: '#8b98ab', rim: '#ffffff' },
  gold: { light: '#ffe9a8', base: '#f5c542', dark: '#a8791b', rim: '#fff3c4' },
  legendary: { light: '#e9d5ff', base: '#a855f7', dark: '#6b21a8', rim: '#f3e8ff' }
}

function tierFor(coinPrice) {
  if (coinPrice >= 50000) return 'legendary'
  if (coinPrice >= 10000) return 'gold'
  if (coinPrice >= 2000) return 'silver'
  return 'bronze'
}

// Hexagonal "achievement badge" medallion: a metallic gradient shield with a
// bevelled inner rim, a diagonal shine highlight, and sparkle accents on the
// top two tiers, with whatever glyph/emoji passed as children centered on
// top. This is the actual artwork for every cosmetic icon, rather than a
// bare emoji sitting in a flat circle.
function BadgeMedallion({ tier, children }) {
  const c = TIER_COLORS[tier] || TIER_COLORS.bronze
  const gradId = `badge-grad-${tier}`

  return (
    <div className={`cosmetic-badge tier-${tier}`} aria-hidden="true">
      <svg viewBox="0 0 100 100" className="cosmetic-badge-svg">
        <defs>
          <linearGradient id={gradId} x1="10%" y1="0%" x2="90%" y2="100%">
            <stop offset="0%" stopColor={c.light} />
            <stop offset="55%" stopColor={c.base} />
            <stop offset="100%" stopColor={c.dark} />
          </linearGradient>
        </defs>
        <polygon
          points="50,96 12,73 12,27 50,4 88,27 88,73"
          fill={`url(#${gradId})`}
          stroke={c.dark}
          strokeWidth="3"
        />
        <polygon
          points="50,90 16,70 16,30 50,10 84,30 84,70"
          fill="none"
          stroke={c.rim}
          strokeWidth="1.5"
          opacity="0.85"
        />
        <ellipse cx="36" cy="26" rx="20" ry="8" fill="#ffffff" opacity="0.28" transform="rotate(-18 36 26)" />
        {(tier === 'gold' || tier === 'legendary') && (
          <g fill={c.rim}>
            <circle cx="50" cy="13" r="2.2" />
            <circle cx="19" cy="50" r="2.2" />
            <circle cx="81" cy="50" r="2.2" />
          </g>
        )}
      </svg>
      <div className="cosmetic-badge-content">{children}</div>
    </div>
  )
}

// A small visual "product photo" for a cosmetic shop item: an avatar shows
// its chess-piece glyph, a badge shows its icon, a profile frame shows its
// pun emoji, all rendered on a tiered badge medallion so pricier items read
// as more prestigious at a glance.
export default function CosmeticIcon({ item }) {
  const tier = tierFor(item.coin_price)

  if (item.item_type === 'avatar') {
    const glyph = AVATAR_GLYPHS[item.image_key] || '♟'
    return (
      <BadgeMedallion tier={tier}>
        <span className="cosmetic-glyph">{glyph}</span>
      </BadgeMedallion>
    )
  }

  if (item.item_type === 'profile_frame') {
    const frame = PROFILE_FRAMES[item.image_key]
    return (
      <BadgeMedallion tier={tier}>
        <span className="cosmetic-glyph">{frame?.emoji || '🎉'}</span>
      </BadgeMedallion>
    )
  }

  const Icon = BADGE_ICONS[item.image_key] || Award

  return (
    <BadgeMedallion tier={tier}>
      <Icon size={26} strokeWidth={2.25} color="#1e293b" />
    </BadgeMedallion>
  )
}

// Renders the chess-piece glyph for a player's equipped avatar, so the
// profile picture actually changes when a different avatar is equipped
// instead of always showing the same placeholder icon.
export function AvatarGlyph({ avatarKey }) {
  return <span>{AVATAR_GLYPHS[avatarKey] || '♟'}</span>
}

// Renders a player's equipped badge as a small icon + label chip, or nothing
// if no badge is equipped.
export function EquippedBadge({ badgeKey }) {
  if (!badgeKey) return null
  const Icon = BADGE_ICONS[badgeKey] || Award
  const label = BADGE_LABELS[badgeKey] || badgeKey

  return (
    <span className="equipped-badge">
      <Icon size={14} strokeWidth={2.5} />
      {label}
    </span>
  )
}

// Renders every equipped profile frame (up to 5) as a row of large pun
// chips. Used on a player's own profile card and, given an opponent's
// frame keys, in the match room so both players can see each other's pins.
export function ProfileFrameBanner({ frameKeys, size = 'large' }) {
  const keys = (frameKeys || []).filter(key => PROFILE_FRAMES[key])
  if (keys.length === 0) return null

  return (
    <div className={`profile-frame-banner profile-frame-banner-${size}`}>
      {keys.map(key => (
        <div className="profile-frame-chip" key={key}>
          <span className="profile-frame-emoji">{PROFILE_FRAMES[key].emoji}</span>
          <span className="profile-frame-label">{PROFILE_FRAMES[key].label}</span>
        </div>
      ))}
    </div>
  )
}
