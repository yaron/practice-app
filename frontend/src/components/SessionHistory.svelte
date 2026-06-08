<script>
  /** @type {any[]} */
  let { sessions } = $props();

  const STATUS_LABEL = {
    APPROVED: "Goedgekeurd",
    REJECTED: "Afgewezen",
    PENDING:  "In behandeling",
  };

  const STATUS_CLASS = {
    APPROVED: "approved",
    REJECTED: "rejected",
    PENDING:  "pending",
  };
</script>

<div class="history">
  {#if sessions.length === 0}
    <p class="empty">Nog geen sessies 📋</p>
  {:else}
    {#each sessions as s (s.id)}
      <div class="card">
        <div class="card-header">
          <span class="child-name">{s.child_name}</span>
          <span class="date">{s.date}</span>
          <span class="tasks-count">{s.tasks_completed} {s.tasks_completed === 1 ? "taak" : "taken"}</span>
          <span class="status {STATUS_CLASS[s.status] ?? 'pending'}">
            {STATUS_LABEL[s.status] ?? s.status}
          </span>
        </div>

        {#if s.tasks && s.tasks.length > 0}
          <ul class="task-list">
            {#each s.tasks as task}
              <li>{task}</li>
            {/each}
          </ul>
        {:else}
          <p class="no-tasks">Geen taakdetails opgeslagen</p>
        {/if}

        {#if s.rejection_note}
          <p class="rejection-note">💬 {s.rejection_note}</p>
        {/if}
      </div>
    {/each}
  {/if}
</div>

<style>
  .history { display: flex; flex-direction: column; gap: 1rem; }
  .empty { text-align: center; color: #a0aec0; padding: 2.5rem 0; font-size: 1.05rem; }

  .card {
    background: #16213e;
    border-radius: 0.75rem;
    padding: 1.1rem 1.25rem;
    display: flex;
    flex-direction: column;
    gap: 0.65rem;
  }

  .card-header {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap;
  }

  .child-name { font-weight: 700; }
  .date { color: #a0aec0; font-size: 0.875rem; }
  .tasks-count {
    background: #2d3748;
    padding: 0.15rem 0.6rem;
    border-radius: 999px;
    font-size: 0.8rem;
  }

  .status {
    padding: 0.15rem 0.65rem;
    border-radius: 999px;
    font-size: 0.78rem;
    font-weight: 600;
    margin-left: auto;
  }
  .status.approved { background: #1c4532; color: #68d391; }
  .status.rejected { background: #742a2a; color: #fc8181; }
  .status.pending  { background: #2d3748; color: #a0aec0; }

  .task-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }
  .task-list li {
    padding: 0.3rem 0.75rem;
    background: #0f172a;
    border-radius: 0.4rem;
    font-size: 0.875rem;
    color: #cbd5e0;
  }
  .task-list li::before { content: "🎵 "; }

  .no-tasks { color: #4a5568; font-size: 0.8rem; margin: 0; font-style: italic; }

  .rejection-note {
    background: #2d1515;
    color: #fc8181;
    padding: 0.5rem 0.75rem;
    border-radius: 0.4rem;
    font-size: 0.85rem;
    margin: 0;
  }
</style>
