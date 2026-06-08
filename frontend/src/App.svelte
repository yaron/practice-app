<script>
  import { onMount, onDestroy } from "svelte";
  import Wheel from "./components/Wheel.svelte";
  import SessionTracker from "./components/SessionTracker.svelte";
  import SuccessPanel from "./components/SuccessPanel.svelte";
  import Hud from "./components/Hud.svelte";
  import Admin from "./Admin.svelte";
  const API = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

  const path = window.location.pathname;
  const isAdmin = path === "/admin";
  const childMatch = !isAdmin && path.match(/^\/child\/(\d+)/);
  const CHILD_ID = childMatch ? parseInt(childMatch[1], 10) : null;
  const isRoot = !isAdmin && !childMatch;

  /** @type {{ id: number, name: string }[]} */
  let children = $state([]);
  let childrenLoading = $state(isRoot);

  /** @type {string[]} */
  let wheelTasks = $state([]);
  /** @type {string[]} */
  let wheelTasksShort = $state([]);
  /** @type {boolean[]} */
  let wheelBonus = $state([]);
  /** @type {string[]} */
  let sessionTasks = $state([]);
  let submitted = $state(false);
  let isSubmitting = $state(false);
  let loading = $state(true);
  let submitError = $state("");
  /** @type {{ current_level: number, experience_points: number, total_points: number, shield_count: number, week_session_count: number, milestone_reached: boolean, year_week: string }|null} */
  let stats = $state(null);
  /** @type {ReturnType<typeof setInterval>|null} */
  let statsInterval = $state(null);
  onMount(async () => {
    if (isRoot) {
      try {
        const res = await fetch(`${API}/api/children`);
        if (res.ok) children = await res.json();
      } finally {
        childrenLoading = false;
      }
      return;
    }
    if (!CHILD_ID) {
      loading = false;
      return;
    }
    try {
      const [optRes, statsRes] = await Promise.all([
        fetch(`${API}/api/options?child_id=${CHILD_ID}`),
        fetch(`${API}/api/stats?child_id=${CHILD_ID}`),
      ]);
      if (!optRes.ok) throw new Error(`HTTP ${optRes.status}`);
      const options = await optRes.json();
      wheelTasks = options.map((/** @type {{ text: string }} */ o) => o.text);
      wheelTasksShort = options.map((/** @type {{ short_text: string }} */ o) => o.short_text);
      wheelBonus = options.map((/** @type {{ is_bonus: boolean }} */ o) => !!o.is_bonus);
      if (statsRes.ok) stats = await statsRes.json();
    } catch {
      wheelTasks = [];
    } finally {
      loading = false;
    }

    // Poll stats every 10 s so XP/level update when parent approves
    statsInterval = setInterval(async () => {
      try {
        const res = await fetch(`${API}/api/stats?child_id=${CHILD_ID}`);
        if (res.ok) stats = await res.json();
      } catch {
        /* ignore */
      }
    }, 10_000);
  });

  onDestroy(() => {
    if (statsInterval) clearInterval(statsInterval);
  });

  /** @param {string} task */
  function addTask(task) {
    sessionTasks = [...sessionTasks, task];
  }

  async function registerChildNotifications() {
    try {
      const { messaging, VAPID_KEY } = await import("./firebase.js");
      if (!messaging) return;
      const { getToken, onMessage } = await import("firebase/messaging");
      const permission = await Notification.requestPermission();
      if (permission !== "granted") return;
      const fcmToken = await getToken(messaging, { vapidKey: VAPID_KEY });
      await fetch(`${API}/api/fcm/register`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ token: fcmToken, role: "CHILD", child_id: CHILD_ID }),
      });
      // Show notifications while the tab is in the foreground (FCM suppresses them otherwise).
      onMessage(messaging, ({ notification }) => {
        new Notification(notification?.title ?? 'Viool Quest', { body: notification?.body ?? '' })
      })
    } catch (err) {
      console.warn('FCM registration failed:', err)
    }
  }

  async function handleSubmit() {
    if (isSubmitting) return;
    submitError = "";
    isSubmitting = true;
    registerChildNotifications();
    try {
      const res = await fetch(`${API}/api/session`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          child_id: CHILD_ID,
          tasks: sessionTasks,
        }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      submitted = true;
    } catch {
      submitError = "Kon de sessie niet versturen. Probeer opnieuw.";
    } finally {
      isSubmitting = false;
    }
  }

  function handleReset() {
    sessionTasks = [];
    submitted = false;
    submitError = "";
  }
</script>

{#if isAdmin}
  <Admin />
{:else if isRoot}
  <main>
    <h1>🎻 Viool Quest</h1>
    {#if childrenLoading}
      <p class="hint">Laden…</p>
    {:else if children.length === 0}
      <p class="hint error">Geen kinderen gevonden.</p>
    {:else}
      <p class="hint">Kies een speler:</p>
      <ul class="child-list">
        {#each children as child}
          <li>
            <a href="/child/{child.id}" class="child-btn">{child.name}</a>
          </li>
        {/each}
      </ul>
    {/if}
  </main>
{:else}
  <main>
    <h1>🎻 Viool Quest</h1>

    {#if !CHILD_ID}
      <p class="hint error">Ongeldig adres. Ga naar <code>/child/1</code></p>
    {:else if submitted}
      <SuccessPanel onreset={handleReset} />
    {:else if loading}
      <p class="hint">Laden…</p>
    {:else if wheelTasks.length === 0}
      <p class="hint error">Kon de wiel-opties niet laden.</p>
      <button class="retry" onclick={() => location.reload()}>Opnieuw proberen</button>
    {:else}
      {#if stats}
        <Hud {stats} />
      {/if}
      <Wheel tasks={wheelTasks} shorttasks={wheelTasksShort} bonusflags={wheelBonus} onresult={addTask} />
      {#if submitError}
        <p class="hint error">{submitError}</p>
      {/if}
      <SessionTracker tasks={sessionTasks} onsubmit={handleSubmit} {isSubmitting} />
    {/if}
  </main>
{/if}

<style>
  main {
    max-width: 500px;
    margin: 0 auto;
    padding: 1.5rem 1rem 3rem;
    text-align: center;
  }

  h1 {
    color: #fee440;
    font-size: 2rem;
    margin: 0 0 1.25rem;
    letter-spacing: 0.02em;
  }

  .hint {
    color: #aaa;
    font-size: 1rem;
    margin: 2rem 0 0;
  }

  .error {
    color: #f15bb5;
  }

  .retry {
    margin-top: 1rem;
    padding: 0.75rem 2rem;
    background: #6c3ce1;
    color: #fff;
    border: none;
    border-radius: 2rem;
    font-size: 1rem;
    font-weight: bold;
    cursor: pointer;
  }

  .child-list {
    list-style: none;
    padding: 0;
    margin: 1.5rem auto 0;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    max-width: 280px;
  }

  .child-btn {
    display: block;
    padding: 0.9rem 1.5rem;
    background: #6c3ce1;
    color: #fff;
    border-radius: 2rem;
    font-size: 1.15rem;
    font-weight: bold;
    text-decoration: none;
    transition: background 0.2s, transform 0.1s;
  }

  .child-btn:hover {
    background: #7d50ef;
    transform: scale(1.03);
  }
</style>
