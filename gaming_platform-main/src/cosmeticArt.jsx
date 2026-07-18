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
  'bat-guy': { emoji: '🏏', label: 'Bat Outta Nowhere' }
}

function tierFor(coinPrice) {
  if (coinPrice >= 1500) return 'gold'
  if (coinPrice >= 500) return 'silver'
  return 'bronze'
}

// A small visual "product photo" for a cosmetic shop item: an avatar shows
// its chess-piece glyph, a badge shows its icon, a profile frame shows its
// pun emoji, each tinted by price tier so pricier items read as more
// prestigious at a glance.
export default function CosmeticIcon({ item }) {
  const tier = tierFor(item.coin_price)

  if (item.item_type === 'avatar') {
    const glyph = AVATAR_GLYPHS[item.image_key] || '♟'
    return (
      <div
        className={`cosmetic-icon cosmetic-icon-avatar tier-${tier}`}
        aria-hidden="true"
      >
        <span>{glyph}</span>
      </div>
    )
  }

  if (item.item_type === 'profile_frame') {
    const frame = PROFILE_FRAMES[item.image_key]
    return (
      <div
        className={`cosmetic-icon cosmetic-icon-avatar tier-${tier}`}
        aria-hidden="true"
      >
        <span>{frame?.emoji || '🎉'}</span>
      </div>
    )
  }

  const Icon = BADGE_ICONS[item.image_key] || Award

  return (
    <div
      className={`cosmetic-icon cosmetic-icon-badge tier-${tier}`}
      aria-hidden="true"
    >
      <Icon size={28} strokeWidth={2.25} />
    </div>
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
