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
// player's name once equipped. Older frames use a plain emoji glyph; newer
// ones (see FRAME_ICONS below) use a hand-built SVG icon instead.
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
  // Greek mythology, rendered with custom SVG icons (see FRAME_ICONS) rather
  // than emoji.
  hercules: { label: 'Twelve Labors, Zero Losses' },
  achilles: { label: 'One Weak Spot, Zero Excuses' },
  icarus: { label: 'Flew Too Close to the W' },
  poseidon: { label: 'Making Waves, Taking Names' },
  hades: { label: 'Lord of the Underdogs' },
  pandora: { label: 'Opened the Box, Closed the Game' },
  midas: { label: 'Midas Touch, Golden Record' },
  atlas: { label: 'Carrying the Whole Team' },
  prometheus: { label: 'Stole Fire, Stole the Win' }
}

// Hand-built SVG icons for the newer profile frames - no emoji involved.
// Each is a simple, self-contained icon on a 24x24 grid, colored to match
// its theme (wood tones, bronze, flame, ocean, gold, etc.) with a thin dark
// outline for definition against any badge-tier background.
const FRAME_ICONS = {
  hercules: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <g transform="rotate(25 12 12)">
        <rect x="10.7" y="10" width="2.6" height="12" rx="1.3" fill="#92400e" stroke="#451a03" strokeWidth="0.5" />
        <path d="M7.5 9 C7 6 8.5 3 12 3 C15.5 3 17 6 16.5 9 C16 11.5 14 12.5 12 12.5 C10 12.5 8 11.5 7.5 9 Z" fill="#c2793a" stroke="#451a03" strokeWidth="0.6" />
      </g>
    </svg>
  ),
  achilles: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <path d="M12 2 L20 5.2 V11 C20 16.6 16.6 20.6 12 22 C7.4 20.6 4 16.6 4 11 V5.2 Z" fill="#b45309" stroke="#78350f" strokeWidth="0.8" />
      <path d="M12 6 L16 8.5 V12 C16 15 14.2 17.3 12 18.3 C9.8 17.3 8 15 8 12 V8.5 Z" fill="#dc2626" />
    </svg>
  ),
  icarus: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <path d="M12 21 C10 16 9 11 8 3 C11 6 12.5 10 12.5 14 C13.5 10 15.5 6.5 20 5 C17.5 9 15.5 12 14 15 C16.5 14 18.5 14.5 21 17 C16.5 17.5 14.5 18.5 12 21 Z" fill="#f97316" stroke="#9a3412" strokeWidth="0.6" />
      <path d="M12.5 14 C13.5 10 15.5 6.5 20 5 C17.5 9 15.5 12 14 15 Z" fill="#fde68a" opacity="0.9" />
    </svg>
  ),
  poseidon: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size} fill="none" stroke="#0e7490" strokeWidth="2.1" strokeLinecap="round" strokeLinejoin="round">
      <line x1="6" y1="6" x2="18" y2="6" />
      <line x1="12" y1="6" x2="12" y2="22" />
      <line x1="12" y1="6" x2="6" y2="1" />
      <line x1="12" y1="6" x2="18" y2="1" />
      <line x1="12" y1="6" x2="12" y2="1" />
    </svg>
  ),
  hades: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <path d="M4 12 C4 6.2 7.6 3 12 3 C16.4 3 20 6.2 20 12 V14 H15 V20 H9 V14 H4 Z" fill="#5b21b6" stroke="#2e1065" strokeWidth="0.7" />
      <path d="M4.3 12.5 H19.7" stroke="#c4b5fd" strokeWidth="0.9" opacity="0.7" />
    </svg>
  ),
  pandora: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <rect x="4" y="11.5" width="16" height="8.5" rx="1.5" fill="#92400e" stroke="#451a03" strokeWidth="0.6" />
      <path d="M4 11.5 L8 6.5 H16 L20 11.5 Z" fill="#c2793a" stroke="#451a03" strokeWidth="0.6" />
      <g stroke="#fbbf24" strokeWidth="1.5" strokeLinecap="round">
        <line x1="12" y1="5" x2="12" y2="1.3" />
        <line x1="8.3" y1="6" x2="6.3" y2="2.3" />
        <line x1="15.7" y1="6" x2="17.7" y2="2.3" />
      </g>
    </svg>
  ),
  midas: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <path d="M3 18 L4.6 8 L9 12.2 L12 5.5 L15 12.2 L19.4 8 L21 18 Z" fill="#eab308" stroke="#854d0e" strokeWidth="0.6" />
      <circle cx="4.6" cy="8" r="1.3" fill="#dc2626" />
      <circle cx="12" cy="5.5" r="1.3" fill="#2563eb" />
      <circle cx="19.4" cy="8" r="1.3" fill="#16a34a" />
    </svg>
  ),
  atlas: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <g fill="none" stroke="#64748b" strokeWidth="2" strokeLinecap="round">
        <line x1="4" y1="19" x2="20" y2="19" />
        <line x1="7.5" y1="19" x2="9.5" y2="14" />
        <line x1="16.5" y1="19" x2="14.5" y2="14" />
      </g>
      <circle cx="12" cy="9" r="6.2" fill="#1d4ed8" />
      <path d="M7.3 7.2 C8.8 6.2 10.3 8 9.2 9.6 C8.2 10.6 6.7 9.6 7.3 7.2 Z" fill="#16a34a" />
      <path d="M13.5 11.2 C15 10.2 17 11.7 16 13.2 C15 14.2 13 13.2 13.5 11.2 Z" fill="#16a34a" />
      <ellipse cx="12" cy="9" rx="6.2" ry="2.3" fill="none" stroke="#bfdbfe" strokeWidth="0.7" opacity="0.75" />
      <line x1="5.8" y1="9" x2="18.2" y2="9" stroke="#bfdbfe" strokeWidth="0.7" opacity="0.75" />
    </svg>
  ),
  prometheus: ({ size }) => (
    <svg viewBox="0 0 24 24" width={size} height={size}>
      <rect x="10.6" y="13" width="2.8" height="9" rx="1.2" fill="#78350f" />
      <path d="M12 1.5 C8.8 4.8 7.6 8 9.5 10.3 C9 8.7 9.8 7.8 10.7 8.6 C10 9.5 10.6 11 12 11 C13.4 11 14 9.5 13.3 8.6 C14.2 7.8 15 8.7 14.5 10.3 C16.4 8 15.2 4.8 12 1.5 Z" fill="#ea580c" />
      <path d="M12 4.2 C10.4 6.1 9.8 7.8 10.8 9.1 C10.6 8.3 11 7.8 11.5 8.2 C11.1 8.7 11.4 9.4 12 9.4 C12.6 9.4 12.9 8.7 12.5 8.2 C13 7.8 13.4 8.3 13.2 9.1 C14.2 7.8 13.6 6.1 12 4.2 Z" fill="#fde047" />
    </svg>
  )
}

// Renders whichever visual a profile frame uses: a custom SVG icon if one
// exists for its key, otherwise its emoji glyph.
function FrameVisual({ frameKey, size }) {
  const Icon = FRAME_ICONS[frameKey]
  if (Icon) {
    return (
      <span className="cosmetic-glyph">
        <Icon size={size} />
      </span>
    )
  }
  const emoji = PROFILE_FRAMES[frameKey]?.emoji
  return <span className="cosmetic-glyph" style={{ fontSize: size }}>{emoji || '🎉'}</span>
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
// top two tiers, with whatever glyph/icon passed as children centered on
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
// icon or emoji, all rendered on a tiered badge medallion so pricier items
// read as more prestigious at a glance.
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
    return (
      <BadgeMedallion tier={tier}>
        <FrameVisual frameKey={item.image_key} size={30} />
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

  const iconSize = size === 'small' ? 18 : 26

  return (
    <div className={`profile-frame-banner profile-frame-banner-${size}`}>
      {keys.map(key => (
        <div className="profile-frame-chip" key={key}>
          <span className="profile-frame-icon">
            <FrameVisual frameKey={key} size={iconSize} />
          </span>
          <span className="profile-frame-label">{PROFILE_FRAMES[key].label}</span>
        </div>
      ))}
    </div>
  )
}
