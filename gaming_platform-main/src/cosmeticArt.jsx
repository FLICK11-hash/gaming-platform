import React from 'react'
import { Crown, Flame, Award, Frame, PaintBucket } from 'lucide-react'

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

const TYPE_ICONS = {
  profile_frame: Frame,
  board_theme: PaintBucket
}

function tierFor(coinPrice) {
  if (coinPrice >= 1500) return 'gold'
  if (coinPrice >= 500) return 'silver'
  return 'bronze'
}

// A small visual "product photo" for a cosmetic shop item: an avatar shows
// its chess-piece glyph, a badge shows its icon, each tinted by price tier
// so pricier items read as more prestigious at a glance.
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

  const Icon =
    BADGE_ICONS[item.image_key] ||
    TYPE_ICONS[item.item_type] ||
    Award

  return (
    <div
      className={`cosmetic-icon cosmetic-icon-badge tier-${tier}`}
      aria-hidden="true"
    >
      <Icon size={28} strokeWidth={2.25} />
    </div>
  )
}
