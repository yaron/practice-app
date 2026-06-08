<script>
  let { sessions, onapprove, onreject } = $props();

  /** @type {Record<number, string>} */
  let rejectNotes = $state({});
  /** @type {Record<number, boolean>} */
  let showReject = $state({});
  /** @type {Record<number, boolean>} */
  let busy = $state({});

  function toggleReject(id) {
    showReject[id] = !showReject[id];
  }

  async function approve(id) {
    busy[id] = true;
    await onapprove(id);
    busy[id] = false;
  }

  async function reject(id) {
    busy[id] = true;
    await onreject(id, rejectNotes[id] ?? "");
    busy[id] = false;
    showReject[id] = false;
  }
</script>

<div class="cards">
  {#if sessions.length === 0}
    <p class="empty">Geen openstaande sessies 🎉</p>
  {:else}
    {#each sessions as s (s.id)}
      <div class="card">
        <div class="card-header">
          <span class="child-name">{s.child_name}</span>
          <span class="date">{s.date}</span>
          <span class="tasks-badge">{s.tasks_completed} {s.tasks_completed === 1 ? "taak" : "taken"}</span>
        </div>
        <div class="card-actions">
          <button class="btn approve" onclick={() => approve(s.id)} disabled={busy[s.id]}>
            ✅ Goedkeuren
          </button>
          <button class="btn reject-toggle" onclick={() => toggleReject(s.id)} disabled={busy[s.id]}>
            ❌ Afwijzen
          </button>
        </div>
        {#if showReject[s.id]}
          <div class="reject-form">
            <input
              type="text"
              placeholder="Optioneel: berichtje voor het kind…"
              bind:value={rejectNotes[s.id]}
            />
            <button class="btn confirm-reject" onclick={() => reject(s.id)} disabled={busy[s.id]}>
              Bevestigen
            </button>
          </div>
        {/if}
      </div>
    {/each}
  {/if}
</div>

<style>
  .cards { display: flex; flex-direction: column; gap: 1rem; }
  .empty { text-align: center; color: #a0aec0; padding: 2.5rem 0; font-size: 1.05rem; }
  .card {
    background: #16213e;
    border-radius: 0.75rem;
    padding: 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }
  .card-header {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }
  .child-name { font-weight: 700; font-size: 1.05rem; }
  .date { color: #a0aec0; font-size: 0.9rem; }
  .tasks-badge {
    background: #2d3748;
    padding: 0.2rem 0.65rem;
    border-radius: 999px;
    font-size: 0.8rem;
  }
  .card-actions { display: flex; gap: 0.6rem; }
  .btn {
    padding: 0.45rem 0.9rem;
    border: none;
    border-radius: 0.5rem;
    cursor: pointer;
    font-size: 0.875rem;
    font-weight: 600;
    transition: opacity 0.15s;
  }
  .btn:disabled { opacity: 0.5; cursor: not-allowed; }
  .approve { background: #276749; color: white; }
  .approve:hover:not(:disabled) { background: #38a169; }
  .reject-toggle { background: #742a2a; color: white; }
  .reject-toggle:hover:not(:disabled) { background: #e53e3e; }
  .reject-form {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
  }
  .reject-form input {
    flex: 1;
    min-width: 180px;
    padding: 0.45rem 0.75rem;
    border-radius: 0.5rem;
    border: 1px solid #2d3748;
    background: #1a1a2e;
    color: white;
    font-size: 0.875rem;
  }
  .reject-form input:focus { outline: none; border-color: #e53e3e; }
  .confirm-reject { background: #744210; color: white; }
  .confirm-reject:hover:not(:disabled) { background: #b7791f; }
</style>
