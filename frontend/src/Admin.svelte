<script>
  import { onMount, onDestroy } from "svelte";
  import AdminLogin from "./components/AdminLogin.svelte";
  import PendingCards from "./components/PendingCards.svelte";
  import SessionHistory from "./components/SessionHistory.svelte";
  import UserManagement from "./components/UserManagement.svelte";

  const API = import.meta.env.VITE_API_URL ?? "http://localhost:8080";
  const STORAGE_KEY = "vq_refresh_token";

  /** @type {string|null} */
  let jwt = $state(null);
  let loading = $state(true);
  /** @type {any[]} */
  let sessions = $state([]);
  /** @type {any[]} */
  let history = $state([]);
  let historyLoaded = $state(false);
  let activeTab = $state("pending");
  let actionError = $state("");
  let pollInterval;

  onMount(async () => {
    const stored = sessionStorage.getItem(STORAGE_KEY);
    if (stored) {
      try {
        const res = await fetch(`${API}/api/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: stored }),
        });
        if (res.ok) {
          const data = await res.json();
          jwt = data.token;
          scheduleAutoRefresh(stored);
          await loadSessions();
          startPolling();
        } else {
          sessionStorage.removeItem(STORAGE_KEY);
        }
      } catch {
        /* silent — fall through to show login */
      }
    }
    loading = false;
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
  });

  function handleLogin({ token, refreshToken }) {
    jwt = token;
    sessionStorage.setItem(STORAGE_KEY, refreshToken);
    scheduleAutoRefresh(refreshToken);
    loadSessions();
    startPolling();
  }

  // Silently refresh JWT 1 min before expiry (JWT lifetime = 15 min → refresh at 14 min).
  function scheduleAutoRefresh(refreshToken) {
    setTimeout(async () => {
      try {
        const res = await fetch(`${API}/api/auth/refresh`, {
          method: "POST",
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ refresh_token: refreshToken }),
        });
        if (res.ok) {
          const data = await res.json();
          jwt = data.token;
          scheduleAutoRefresh(refreshToken);
        } else {
          jwt = null;
          sessionStorage.removeItem(STORAGE_KEY);
        }
      } catch {
        /* silent */
      }
    }, 14 * 60 * 1000);
  }

  async function loadSessions() {
    if (!jwt) return;
    try {
      const res = await fetch(`${API}/api/admin/sessions`, {
        headers: { Authorization: `Bearer ${jwt}` },
      });
      if (res.ok) {
        sessions = await res.json();
      } else if (res.status === 401) {
        jwt = null;
        sessionStorage.removeItem(STORAGE_KEY);
        if (pollInterval) clearInterval(pollInterval);
      }
    } catch {
      /* silent */
    }
  }

  async function loadHistory() {
    if (!jwt) return;
    try {
      const res = await fetch(`${API}/api/admin/history`, {
        headers: { Authorization: `Bearer ${jwt}` },
      });
      if (res.ok) {
        history = await res.json();
        historyLoaded = true;
      }
    } catch {
      /* silent */
    }
  }

  async function switchTab(tab) {
    activeTab = tab;
    if (tab === "history" && !historyLoaded) {
      await loadHistory();
    }
  }

  function startPolling() {
    if (pollInterval) clearInterval(pollInterval);
    pollInterval = setInterval(loadSessions, 30_000);
  }

  async function handleApprove(id) {
    actionError = "";
    const res = await fetch(`${API}/api/admin/approve/${id}`, {
      method: "POST",
      headers: { Authorization: `Bearer ${jwt}` },
    });
    if (res.ok) {
      sessions = sessions.filter((s) => s.id !== id);
      historyLoaded = false; // stale — reload next time history tab opens
    } else {
      const d = await res.json().catch(() => ({}));
      actionError = d.error ?? "Goedkeuren mislukt";
    }
  }

  async function handleReject(id, note) {
    actionError = "";
    const res = await fetch(`${API}/api/admin/reject/${id}`, {
      method: "POST",
      headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
      body: JSON.stringify({ note }),
    });
    if (res.ok) {
      sessions = sessions.filter((s) => s.id !== id);
      historyLoaded = false;
    } else {
      const d = await res.json().catch(() => ({}));
      actionError = d.error ?? "Afwijzen mislukt";
    }
  }
</script>

<main>
  {#if loading}
    <p class="hint">Laden…</p>
  {:else if !jwt}
    <AdminLogin onsuccess={handleLogin} />
  {:else}
    <div class="dashboard">
      <h1>🎻 Viool Quest — Ouder</h1>

      <div class="tabs" role="tablist">
        <button
          role="tab"
          class:active={activeTab === "pending"}
          onclick={() => switchTab("pending")}
        >
          Openstaand
          {#if sessions.length > 0}
            <span class="badge">{sessions.length}</span>
          {/if}
        </button>
        <button
          role="tab"
          class:active={activeTab === "history"}
          onclick={() => switchTab("history")}
        >
          Geschiedenis
        </button>
        <button
          role="tab"
          class:active={activeTab === "users"}
          onclick={() => switchTab("users")}
        >
          Gebruikers
        </button>
      </div>

      {#if actionError}
        <p class="action-error">{actionError}</p>
      {/if}

      {#if activeTab === "pending"}
        <PendingCards {sessions} onapprove={handleApprove} onreject={handleReject} />
      {:else if activeTab === "history"}
        <SessionHistory sessions={history} />
      {:else}
        <UserManagement {jwt} api={API} />
      {/if}
    </div>
  {/if}
</main>

<style>
  main { min-height: 100vh; padding: 1.5rem 1rem; }
  .hint { text-align: center; color: #a0aec0; padding: 4rem; }

  .dashboard { max-width: 640px; margin: 0 auto; }
  h1 { color: #fee440; font-size: 1.75rem; text-align: center; margin: 0 0 1.25rem; }

  .tabs {
    display: flex;
    gap: 0.5rem;
    margin-bottom: 1.25rem;
    border-bottom: 1px solid #2d3748;
    padding-bottom: 0;
  }
  .tabs button {
    background: none;
    border: none;
    color: #a0aec0;
    font-size: 0.95rem;
    font-weight: 600;
    padding: 0.5rem 1rem 0.65rem;
    cursor: pointer;
    border-bottom: 2px solid transparent;
    margin-bottom: -1px;
    display: flex;
    align-items: center;
    gap: 0.4rem;
    transition: color 0.15s;
  }
  .tabs button:hover { color: #e2e8f0; }
  .tabs button.active {
    color: #a78bfa;
    border-bottom-color: #6c3ce1;
  }
  .badge {
    background: #6c3ce1;
    color: white;
    font-size: 0.7rem;
    padding: 0.1rem 0.45rem;
    border-radius: 999px;
    font-weight: 700;
  }

  .action-error { color: #fc8181; text-align: center; margin-bottom: 1rem; font-size: 0.9rem; }
</style>
