import React, { useEffect, useMemo, useState } from 'react'
import { createRoot } from 'react-dom/client'
import {
  Trophy,
  LogOut,
  Send,
  Search,
  Users,
  ShieldAlert
} from 'lucide-react'
import { api } from './api.js'
import MatchRoom from './Chess.jsx'
import CosmeticIcon, { AvatarGlyph, EquippedBadge, ProfileFrameBanner } from './cosmeticArt.jsx'
import './styles.css'

const FIXED_COIN_WAGER = 200
const MAX_PROFILE_FRAMES = 5

function isCosmeticEquipped(item, user) {
  if (item.item_type === 'avatar') return user.selected_avatar === item.image_key
  if (item.item_type === 'badge') return user.selected_badge === item.image_key
  if (item.item_type === 'profile_frame') {
    return (user.selected_profile_frames || []).includes(item.image_key)
  }
  return false
}

function App() {
  const [token, setToken] = useState(
    localStorage.getItem('gp_token') || ''
  )
  const [user, setUser] = useState(null)
  const [message, setMessage] = useState('')

  useEffect(() => {
    if (!token) return

    api('/me', { token })
      .then(setUser)
      .catch(() => {
        localStorage.removeItem('gp_token')
        setToken('')
        setUser(null)
      })
  }, [token])

  function saveSession(authResponse) {
    localStorage.setItem('gp_token', authResponse.token)
    setToken(authResponse.token)
    setUser(authResponse.user)
    setMessage('Signed in successfully.')
  }

  function logout() {
    localStorage.removeItem('gp_token')
    setToken('')
    setUser(null)
    setMessage('Logged out.')
  }

  return (
    <main>
      <header className="topbar">
        <div className="brand">
          <Trophy />
          <span>Gaming Platform</span>
        </div>

        {user && (
          <button className="ghost" onClick={logout}>
            <LogOut size={18} />
            Log out
          </button>
        )}
      </header>

      {message && <div className="toast">{message}</div>}

      {!user ? (
        <AuthScreen
          onAuth={saveSession}
          onMessage={setMessage}
        />
      ) : (
        <Dashboard
          user={user}
          token={token}
          onUser={setUser}
          onMessage={setMessage}
        />
      )}
    </main>
  )
}

function AuthScreen({ onAuth, onMessage }) {
  const [mode, setMode] = useState('login')
  const [form, setForm] = useState({
    username: '',
    email: '',
    password: ''
  })
  const [loading, setLoading] = useState(false)

  async function submit(event) {
    event.preventDefault()
    setLoading(true)

    try {
      const path = mode === 'login' ? '/login' : '/register'

      const body =
        mode === 'login'
          ? {
              username: form.username,
              password: form.password
            }
          : form

      const data = await api(path, {
        method: 'POST',
        body
      })

      onAuth(data)
    } catch (err) {
      onMessage(err.message)
    } finally {
      setLoading(false)
    }
  }

  return (
    <section className="auth-card">
      <h1>
        {mode === 'login'
          ? 'Welcome back'
          : 'Create account'}
      </h1>

      <p>
        Sign in to test your Go backend with a real browser UI.
      </p>

      <form onSubmit={submit}>
        <label>
          Username
          <input
            value={form.username}
            onChange={event =>
              setForm({
                ...form,
                username: event.target.value
              })
            }
          />
        </label>

        {mode === 'register' && (
          <label>
            Email
            <input
              value={form.email}
              onChange={event =>
                setForm({
                  ...form,
                  email: event.target.value
                })
              }
            />
          </label>
        )}

        <label>
          Password
          <input
            type="password"
            value={form.password}
            onChange={event =>
              setForm({
                ...form,
                password: event.target.value
              })
            }
          />
        </label>

        <button disabled={loading}>
          {loading
            ? 'Working...'
            : mode === 'login'
              ? 'Log in'
              : 'Register'}
        </button>
      </form>

      <button
        className="link"
        onClick={() =>
          setMode(
            mode === 'login'
              ? 'register'
              : 'login'
          )
        }
      >
        {mode === 'login'
          ? 'Need an account? Register'
          : 'Already have an account? Log in'}
      </button>
    </section>
  )
}

function Dashboard({
  user,
  token,
  onUser,
  onMessage
}) {
  const [queue, setQueue] = useState(null)
  const [activeMatch, setActiveMatch] = useState(null)

  const [gameMode, setGameMode] =
    useState('one_vs_one')

  const coinWager = FIXED_COIN_WAGER

  const [post, setPost] = useState({
    title: 'First post',
    body: 'Testing the gaming platform frontend.'
  })

  const [posts, setPosts] = useState([])

  const [cosmetics, setCosmetics] = useState([])

  const [lookup, setLookup] = useState({
    tournamentID: '',
    matchID: '',
    postID: ''
  })

  const [result, setResult] = useState(null)

  const record = useMemo(
    () => `${user.total_wins}-${user.total_losses}`,
    [user]
  )

  useEffect(() => {
    if (activeMatch) return

    let cancelled = false

    async function poll() {
      try {
        const match = await api('/matches/mine', {
          token
        })

        if (!cancelled) {
          setActiveMatch(match)
          setQueue(null)
        }
      } catch {
        // No active match yet.
      }
    }

    poll()

    const intervalID = setInterval(poll, 2500)

    return () => {
      cancelled = true
      clearInterval(intervalID)
    }
  }, [token, activeMatch])

  useEffect(() => {
    api('/cosmetics', { token })
      .then(setCosmetics)
      .catch(err => onMessage(err.message))
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [token])

  async function buyCosmetic(item) {
    try {
      await api(`/cosmetics/${item.cosmetic_item_id}/purchase`, {
        token,
        method: 'POST'
      })
      const [refreshedCosmetics, refreshedUser] = await Promise.all([
        api('/cosmetics', { token }),
        api('/me', { token })
      ])
      setCosmetics(refreshedCosmetics)
      onUser(refreshedUser)
      onMessage(`Purchased ${item.name}.`)
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function equipCosmetic(item) {
    const equipped = isCosmeticEquipped(item, user)

    try {
      if (item.item_type === 'avatar') {
        if (equipped) return
        onUser(
          await api('/me/avatar', {
            token,
            method: 'POST',
            body: { cosmetic_item_id: item.cosmetic_item_id }
          })
        )
      } else if (item.item_type === 'profile_frame') {
        onUser(
          await api('/me/profile-frames/toggle', {
            token,
            method: 'POST',
            body: { cosmetic_item_id: item.cosmetic_item_id }
          })
        )
      } else {
        onUser(
          await api('/me/badge', {
            token,
            method: 'POST',
            body: { cosmetic_item_id: equipped ? 0 : item.cosmetic_item_id }
          })
        )
      }
      onMessage(equipped ? `Unequipped ${item.name}.` : `Equipped ${item.name}.`)
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function refreshMe() {
    try {
      const data = await api('/me', { token })
      onUser(data)
      onMessage('Profile refreshed.')
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function claimDailyReward() {
    try {
      const data = await api('/me/daily-reward', {
        token,
        method: 'POST'
      })
      onUser(data)
      onMessage('Daily reward claimed!')
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function joinQueue() {
    try {
      const data = await api('/matchmaking/queue', {
        token,
        method: 'POST',
        body: {
          game_type: 'chess',
          game_mode: gameMode,
          coin_wager: Number(coinWager),
          min_elo: 800,
          max_elo: 2000
        }
      })

      setQueue(data)

      const modeName =
        gameMode === 'tournament'
          ? '4-player tournament'
          : '1v1 match'

      onMessage(`Joined the ${modeName} queue.`)
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function cancelQueue() {
    if (!queue?.queue_id) return

    try {
      await api(
        `/matchmaking/queue/${queue.queue_id}`,
        {
          token,
          method: 'DELETE'
        }
      )

      setQueue(null)
      onMessage('Queue entry cancelled.')
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function createPost(event) {
    event.preventDefault()

    try {
      const data = await api('/posts', {
        token,
        method: 'POST',
        body: post
      })

      setPosts(currentPosts => [
        data,
        ...currentPosts
      ])

      onMessage('Post created.')
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function loadMyPosts() {
    try {
      const data = await api(
        `/users/${user.user_id}/posts`,
        { token }
      )

      setPosts(data)
      onMessage('Posts loaded.')
    } catch (err) {
      onMessage(err.message)
    }
  }

  async function fetchThing(type) {
    try {
      const id = lookup[`${type}ID`]

      if (!id) {
        onMessage(`Enter a ${type} ID first.`)
        return
      }

      const data = await api(
        `/${type}s/${id}`,
        { token }
      )

      setResult(data)
    } catch (err) {
      onMessage(err.message)
    }
  }

  const coinBalance = Number(
    user.coin_balance ?? 0
  )

  const insufficientCoins =
    coinWager > coinBalance

  return (
    <section className="grid">
      <div className="card profile">
        <ProfileFrameBanner frameKeys={user.selected_profile_frames} />

        <div className="avatar">
          <AvatarGlyph avatarKey={user.selected_avatar} />
        </div>

        <div>
          <h2>
            {user.username}
            <EquippedBadge badgeKey={user.selected_badge} />
          </h2>
          <p>{user.email}</p>
        </div>

        <div className="stats">
          <Stat
            label="Elo"
            value={user.current_elo}
          />

          <Stat
            label="Record"
            value={record}
          />

          <Stat
            label="Coins"
            value={`${coinBalance} 🪙`}
          />

          <Stat
            label="Status"
            value={user.account_status}
          />
        </div>

        <div className="row">
          <button onClick={refreshMe}>
            Refresh Profile
          </button>

          <button onClick={claimDailyReward}>
            Claim Daily Reward
          </button>
        </div>
      </div>

      <div className="card matchmaking-card">
        <div className="section-heading">
          <div>
            <h2>
              <Users />
              Choose a Game Mode
            </h2>

            <p>
              Select how you want to compete before entering
              matchmaking.
            </p>
          </div>

          <div className="coin-balance">
            <span>Coin balance</span>
            <strong>{coinBalance} 🪙</strong>
          </div>
        </div>

        <div className="mode-grid">
          <button
            type="button"
            className={
              `mode-card ${
                gameMode === 'one_vs_one'
                  ? 'selected'
                  : ''
              }`
            }
            onClick={() =>
              setGameMode('one_vs_one')
            }
            disabled={Boolean(queue || activeMatch)}
          >
            <div className="mode-icon">♟</div>

            <div>
              <h3>1v1 Match</h3>
              <p>
                Get matched directly against one opponent.
              </p>
              <span className="mode-detail">
                Fastest matchmaking
              </span>
            </div>
          </button>

          <button
            type="button"
            className={
              `mode-card ${
                gameMode === 'tournament'
                  ? 'selected'
                  : ''
              }`
            }
            onClick={() =>
              setGameMode('tournament')
            }
            disabled={Boolean(queue || activeMatch)}
          >
            <div className="mode-icon">🏆</div>

            <div>
              <h3>4-Player Tournament</h3>
              <p>
                Win a semifinal and final to become champion.
              </p>
              <span className="mode-detail">
                Includes bonus coin rewards
              </span>
            </div>
          </button>
        </div>

        <div className="wager-field">
          Virtual coin wager
          <span className="wager-fixed">{FIXED_COIN_WAGER} coins</span>
        </div>

        {gameMode === 'tournament' && (
          <div className="reward-banner">
            <Trophy size={18} />
            Tournament champions receive the combined prize
            pool plus a bonus reward.
          </div>
        )}

        {queue ? (
          <div className="queue-status">
            <div className="queue-spinner" />

            <div>
              <b>Searching for players...</b>

              <p>
                {gameMode === 'tournament'
                  ? 'Waiting for four compatible players'
                  : 'Waiting for one compatible opponent'}
              </p>

              <span>
                Wager:{' '}
                {queue.coin_wager ?? coinWager} coins
                {' · '}
                Status: {queue.status}
              </span>
            </div>
          </div>
        ) : (
          <p className="muted">
            {activeMatch
              ? 'You are currently in a match.'
              : 'Choose a mode and wager to begin.'}
          </p>
        )}

        <div className="row">
          <button
            onClick={joinQueue}
            disabled={
              Boolean(queue || activeMatch) ||
              insufficientCoins
            }
          >
            {gameMode === 'tournament'
              ? 'Join Tournament Queue'
              : 'Find 1v1 Opponent'}
          </button>

          <button
            className="danger"
            onClick={cancelQueue}
            disabled={!queue}
          >
            Cancel Queue
          </button>
        </div>

        {insufficientCoins && (
          <p className="form-error">
            You do not have enough coins for this wager.
          </p>
        )}
      </div>

      {activeMatch && (
        <MatchRoom
          matchId={activeMatch.match_id}
          token={token}
          userId={user.user_id}
          onExit={() => setActiveMatch(null)}
        />
      )}

      <div className="card wide cosmetics-card">
        <h2>
          <Trophy />
          Cosmetic Shop
        </h2>

        <p>
          Spend coins on avatars, badges, and ridiculous profile frames.
          Equip anything you own — up to {MAX_PROFILE_FRAMES} profile frames at once.
        </p>

        <div className="cosmetics-grid">
          {cosmetics.map(item => {
            const equipped = isCosmeticEquipped(item, user)
            const framesAtCap =
              item.item_type === 'profile_frame' &&
              !equipped &&
              (user.selected_profile_frames || []).length >= MAX_PROFILE_FRAMES

            let equipLabel = 'Equip'
            if (equipped) {
              equipLabel = item.item_type === 'avatar' ? 'Equipped' : 'Unequip'
            }

            return (
              <div className="cosmetic-item" key={item.cosmetic_item_id}>
                <CosmeticIcon item={item} />
                <b>{item.name}</b>
                <span className="mode-detail">{item.item_type} · {item.coin_price} 🪙</span>
                <p>{item.description}</p>
                {item.owned ? (
                  <button
                    className={equipped ? 'equipped' : ''}
                    onClick={() => equipCosmetic(item)}
                    disabled={(equipped && item.item_type === 'avatar') || framesAtCap}
                    title={framesAtCap ? `You can only equip ${MAX_PROFILE_FRAMES} profile frames at once.` : undefined}
                  >
                    {equipLabel}
                  </button>
                ) : (
                  <button
                    onClick={() => buyCosmetic(item)}
                    disabled={coinBalance < item.coin_price}
                  >
                    Buy
                  </button>
                )}
              </div>
            )
          })}
        </div>
      </div>

      <div className="card">
        <h2>
          <Send />
          Social Post
        </h2>

        <form onSubmit={createPost}>
          <label>
            Title
            <input
              value={post.title}
              onChange={event =>
                setPost({
                  ...post,
                  title: event.target.value
                })
              }
            />
          </label>

          <label>
            Body
            <textarea
              value={post.body}
              onChange={event =>
                setPost({
                  ...post,
                  body: event.target.value
                })
              }
            />
          </label>

          <div className="row">
            <button>Create Post</button>

            <button
              type="button"
              onClick={loadMyPosts}
            >
              Load My Posts
            </button>
          </div>
        </form>

        {posts.map(item => (
          <article
            className="post"
            key={item.post_id}
          >
            <b>{item.title}</b>

            <span>
              #{item.post_id} · {item.status}
            </span>

            <p>{item.body}</p>
          </article>
        ))}
      </div>

      <div className="card">
        <h2>
          <Search />
          Lookups
        </h2>

        <div className="lookup">
          <input
            placeholder="Tournament ID"
            value={lookup.tournamentID}
            onChange={event =>
              setLookup({
                ...lookup,
                tournamentID: event.target.value
              })
            }
          />

          <button
            onClick={() =>
              fetchThing('tournament')
            }
          >
            Get
          </button>
        </div>

        <div className="lookup">
          <input
            placeholder="Match ID"
            value={lookup.matchID}
            onChange={event =>
              setLookup({
                ...lookup,
                matchID: event.target.value
              })
            }
          />

          <button
            onClick={() =>
              fetchThing('match')
            }
          >
            Get
          </button>
        </div>

        <div className="lookup">
          <input
            placeholder="Post ID"
            value={lookup.postID}
            onChange={event =>
              setLookup({
                ...lookup,
                postID: event.target.value
              })
            }
          />

          <button
            onClick={() =>
              fetchThing('post')
            }
          >
            Get
          </button>
        </div>

        {result && (
          <pre>
            {JSON.stringify(result, null, 2)}
          </pre>
        )}
      </div>

      <div className="card wide">
        <h2>
          <ShieldAlert />
          Notes
        </h2>

        <p>
          Select 1v1 or tournament mode, choose a virtual
          coin wager, and enter matchmaking. Once matched,
          the live chess board will appear and connect
          through WebSockets.
        </p>
      </div>
    </section>
  )
}

function Stat({ label, value }) {
  return (
    <div className="stat">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

createRoot(
  document.getElementById('root')
).render(<App />)
