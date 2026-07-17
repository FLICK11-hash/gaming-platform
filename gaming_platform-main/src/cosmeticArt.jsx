import React from 'react'
import { Crown, Flame, Award } from 'lucide-react'

const AVATAR_GLYPHS = {
  rook: '♜',
  knight: '♞',
  king: '♚'
}

const BADGE_ICONS = {
  'first-victory': Award,
  'hot-streak': Flame,
  grandmaster: Crown
}

// Comedic, non-chess profile frames: puns rendered big and loud next to a
// player's name once equipped.
export const PROFILE_FRAMES = {
  'big-dill': { emoji: '🥒', label: 'Kind of a Big Dill' },
  'lettuce-celebrate': { emoji: '🥬', label: 'Lettuce Celebrate' },
  'nacho-average': { emoji: '🌮', label: 'Nacho Average Player' },
  'holy-guacamole': { emoji: '🥑', label: 'Holy Guacamole' },
  'donut-even': { emoji: '🍩', label: 'Donut Even Try It' },
  'legend-dairy': { emoji: '🧀', label: 'Legend-dairy' }
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
