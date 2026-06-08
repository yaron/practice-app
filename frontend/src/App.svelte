<script>
  import { onMount } from "svelte";
  import Wheel from "./components/Wheel.svelte";
  import SessionTracker from "./components/SessionTracker.svelte";
  import SuccessPanel from "./components/SuccessPanel.svelte";
  import { WHEEL_TASKS_SHORT } from "./lib/wheelData.js";

  const API = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

  const match = window.location.pathname.match(/^\/child\/(\d+)/);
  const CHILD_ID = match ? parseInt(match[1], 10) : null;

  /** @type {string[]} */
  let wheelTasks = $state([]);
  let sessionTasks = $state([]);
  let submitted = $state(false);
  let loading = $state(true);
  let submitError = $state("");

  onMount(async () => {
    if (!CHILD_ID) {
      loading = false;
      return;
    }
    try {
      const res = await fetch(`${API}/api/options?child_id=${CHILD_ID}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const options = await res.json();
      wheelTasks = options.map((/** @type {{ text: string }} */ o) => o.text);
    } catch {
      wheelTasks = []; // error state handled in template
    } finally {
      loading = false;
    }
  });

  function addTask(task) {
    sessionTasks = [...sessionTasks, task];
  }

  async function handleSubmit() {
    submitError = "";
    try {
      const res = await fetch(`${API}/api/session`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          child_id: CHILD_ID,
          tasks_completed: sessionTasks.length,
        }),
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      submitted = true;
    } catch {
      submitError = "Kon de sessie niet versturen. Probeer opnieuw.";
    }
  }

  function handleReset() {
    sessionTasks = [];
    submitted = false;
    submitError = "";
  }
</script>

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
    <Wheel tasks={wheelTasks} shorttasks={WHEEL_TASKS_SHORT} onresult={addTask} />
    {#if submitError}
      <p class="hint error">{submitError}</p>
    {/if}
    <SessionTracker tasks={sessionTasks} onsubmit={handleSubmit} />
  {/if}
</main>

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
    margin: 0 0 1.75rem;
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
</style>
