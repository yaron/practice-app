<script>
  let { jwt, api } = $props();

  /** @type {any[]} */
  let children = $state([]);
  let selectedChildId = $state(/** @type {number|null} */ (null));
  /** @type {any[]} */
  let options = $state([]);
  let loadError = $state("");
  let optionsError = $state("");

  /** @type {Record<number, {text: string, short_text: string, is_bonus: boolean}>} */
  let editing = $state({});
  /** @type {Record<number, boolean>} */
  let busy = $state({});
  /** @type {Record<number, string>} */
  let rowError = $state({});

  let newText = $state("");
  let newShortText = $state("");
  let newIsBonus = $state(false);
  let createError = $state("");
  let createBusy = $state(false);

  async function loadChildren() {
    loadError = "";
    try {
      const res = await fetch(`${api}/api/admin/children`, {
        headers: { Authorization: `Bearer ${jwt}` },
      });
      if (res.ok) {
        children = await res.json();
        if (children.length > 0 && selectedChildId === null) {
          selectedChildId = children[0].id;
        }
      } else {
        loadError = "Kon kinderen niet laden";
      }
    } catch {
      loadError = "Kon kinderen niet laden";
    }
  }

  async function loadOptions(childId) {
    optionsError = "";
    options = [];
    try {
      const res = await fetch(`${api}/api/options?child_id=${childId}`);
      if (res.ok) {
        options = await res.json();
      } else {
        optionsError = "Kon opties niet laden";
      }
    } catch {
      optionsError = "Kon opties niet laden";
    }
  }

  $effect(() => {
    if (jwt) loadChildren();
  });

  $effect(() => {
    if (selectedChildId !== null) loadOptions(selectedChildId);
  });

  function startEdit(opt) {
    editing[opt.id] = { text: opt.text, short_text: opt.short_text, is_bonus: opt.is_bonus };
  }

  function cancelEdit(id) {
    delete editing[id];
    editing = { ...editing };
    rowError[id] = "";
  }

  async function saveEdit(id) {
    const e = editing[id];
    if (!e || !e.text.trim() || !e.short_text.trim()) return;
    busy[id] = true;
    rowError[id] = "";
    try {
      const res = await fetch(`${api}/api/admin/options/${id}`, {
        method: "PATCH",
        headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
        body: JSON.stringify({ text: e.text.trim(), short_text: e.short_text.trim(), is_bonus: e.is_bonus }),
      });
      if (res.ok) {
        const updated = await res.json();
        options = options.map((o) => (o.id === id ? updated : o));
        cancelEdit(id);
      } else {
        const d = await res.json().catch(() => ({}));
        rowError[id] = d.error ?? "Opslaan mislukt";
      }
    } catch {
      rowError[id] = "Opslaan mislukt";
    }
    busy[id] = false;
  }

  async function deleteOption(id) {
    const opt = options.find((o) => o.id === id);
    if (!confirm(`Optie "${opt?.text}" verwijderen?`)) return;
    busy[id] = true;
    rowError[id] = "";
    try {
      const res = await fetch(`${api}/api/admin/options/${id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${jwt}` },
      });
      if (res.ok) {
        options = options.filter((o) => o.id !== id);
      } else {
        const d = await res.json().catch(() => ({}));
        rowError[id] = d.error ?? "Verwijderen mislukt";
      }
    } catch {
      rowError[id] = "Verwijderen mislukt";
    }
    busy[id] = false;
  }

  async function createOption() {
    if (!newText.trim() || !newShortText.trim() || selectedChildId === null) return;
    createBusy = true;
    createError = "";
    try {
      const res = await fetch(`${api}/api/admin/options`, {
        method: "POST",
        headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
        body: JSON.stringify({
          child_id: selectedChildId,
          text: newText.trim(),
          short_text: newShortText.trim(),
          is_bonus: newIsBonus,
        }),
      });
      if (res.ok) {
        const created = await res.json();
        options = [...options, created];
        newText = "";
        newShortText = "";
        newIsBonus = false;
      } else {
        const d = await res.json().catch(() => ({}));
        createError = d.error ?? "Aanmaken mislukt";
      }
    } catch {
      createError = "Aanmaken mislukt";
    }
    createBusy = false;
  }
</script>

<div class="wo">
  {#if loadError}
    <p class="error">{loadError}</p>
  {/if}

  {#if children.length > 1}
    <div class="child-tabs">
      {#each children as child (child.id)}
        <button
          class="child-tab"
          class:active={selectedChildId === child.id}
          onclick={() => (selectedChildId = child.id)}
        >
          {child.name}
        </button>
      {/each}
    </div>
  {/if}

  {#if optionsError}
    <p class="error">{optionsError}</p>
  {/if}

  <div class="options-list">
    {#each options as opt (opt.id)}
      <div class="opt-row">
        {#if editing[opt.id]}
          <div class="edit-block">
            <label>
              Volledige tekst
              <input type="text" bind:value={editing[opt.id].text} />
            </label>
            <label>
              Wieltekst (kort)
              <input type="text" bind:value={editing[opt.id].short_text} maxlength="50" />
            </label>
            <label class="bonus-label">
              <input type="checkbox" bind:checked={editing[opt.id].is_bonus} />
              Bonus
            </label>
            <div class="row-actions">
              <button
                class="btn save"
                onclick={() => saveEdit(opt.id)}
                disabled={busy[opt.id] || !editing[opt.id].text.trim() || !editing[opt.id].short_text.trim()}
              >
                Opslaan
              </button>
              <button class="btn cancel" onclick={() => cancelEdit(opt.id)} disabled={busy[opt.id]}>
                Annuleren
              </button>
            </div>
          </div>
          {#if rowError[opt.id]}
            <p class="row-error">{rowError[opt.id]}</p>
          {/if}
        {:else}
          <div class="opt-info">
            <span class="opt-text">{opt.text}</span>
            <span class="opt-short">{opt.short_text}</span>
            {#if opt.is_bonus}<span class="bonus-badge">bonus</span>{/if}
          </div>
          <div class="row-actions">
            <button class="btn edit" onclick={() => startEdit(opt)} disabled={busy[opt.id]}>Bewerken</button>
            <button class="btn delete" onclick={() => deleteOption(opt.id)} disabled={busy[opt.id]}>Verwijderen</button>
          </div>
          {#if rowError[opt.id]}
            <p class="row-error">{rowError[opt.id]}</p>
          {/if}
        {/if}
      </div>
    {/each}

    {#if options.length === 0 && !optionsError}
      <p class="empty">Geen opties gevonden.</p>
    {/if}
  </div>

  <div class="create-form">
    <h3>Nieuwe optie</h3>
    <div class="create-fields">
      <input
        type="text"
        placeholder="Volledige tekst (bijv. Speel een liedje boos)"
        bind:value={newText}
        onkeydown={(e) => e.key === "Enter" && createOption()}
      />
      <input
        type="text"
        placeholder="Wieltekst (bijv. Boos 😠)"
        bind:value={newShortText}
        maxlength="50"
        onkeydown={(e) => e.key === "Enter" && createOption()}
      />
    </div>
    <div class="create-footer">
      <label class="bonus-label">
        <input type="checkbox" bind:checked={newIsBonus} />
        Bonus
      </label>
      <button
        class="btn save"
        onclick={createOption}
        disabled={createBusy || !newText.trim() || !newShortText.trim()}
      >
        Toevoegen
      </button>
    </div>
    {#if createError}
      <p class="row-error">{createError}</p>
    {/if}
  </div>
</div>

<style>
  .wo { display: flex; flex-direction: column; gap: 1rem; }

  .error { color: #fc8181; font-size: 0.9rem; margin: 0; }
  .empty { color: #a0aec0; font-size: 0.9rem; }
  .row-error { color: #fc8181; font-size: 0.8rem; margin: 0.25rem 0 0; }

  .child-tabs {
    display: flex;
    gap: 0.4rem;
    flex-wrap: wrap;
    margin-bottom: 0.25rem;
  }
  .child-tab {
    padding: 0.35rem 0.9rem;
    border-radius: 999px;
    border: 1px solid #2d3748;
    background: #16213e;
    color: #a0aec0;
    font-size: 0.85rem;
    font-weight: 600;
    cursor: pointer;
    transition: background 0.15s, color 0.15s;
  }
  .child-tab:hover { color: #e2e8f0; border-color: #4a5568; }
  .child-tab.active { background: #6c3ce1; border-color: #6c3ce1; color: white; }

  .options-list { display: flex; flex-direction: column; gap: 0.6rem; }

  .opt-row {
    background: #16213e;
    border-radius: 0.65rem;
    padding: 0.75rem 1rem;
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
    flex-direction: row;
  }
  .opt-row:has(.edit-block) {
    flex-direction: column;
    align-items: stretch;
  }

  .opt-info {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 0.6rem;
    flex-wrap: wrap;
    min-width: 0;
  }
  .opt-text { font-size: 0.9rem; color: #e2e8f0; flex: 1; min-width: 0; }
  .opt-short {
    font-size: 0.8rem;
    background: #2d3748;
    color: #a0aec0;
    padding: 0.1rem 0.5rem;
    border-radius: 999px;
    white-space: nowrap;
  }
  .bonus-badge {
    font-size: 0.7rem;
    background: #744210;
    color: #fbd38d;
    padding: 0.1rem 0.45rem;
    border-radius: 999px;
    font-weight: 700;
    white-space: nowrap;
  }

  .row-actions { display: flex; gap: 0.4rem; margin-left: auto; flex-shrink: 0; }

  .edit-block {
    display: flex;
    flex-direction: column;
    gap: 0.6rem;
  }
  .edit-block label {
    display: flex;
    flex-direction: column;
    gap: 0.25rem;
    font-size: 0.8rem;
    color: #a0aec0;
  }

  .bonus-label {
    display: flex !important;
    flex-direction: row !important;
    align-items: center;
    gap: 0.4rem;
    font-size: 0.85rem;
    color: #a0aec0;
    cursor: pointer;
  }
  .bonus-label input[type="checkbox"] { width: 1rem; height: 1rem; cursor: pointer; }

  input[type="text"] {
    padding: 0.4rem 0.7rem;
    border-radius: 0.45rem;
    border: 1px solid #2d3748;
    background: #1a1a2e;
    color: white;
    font-size: 0.875rem;
    width: 100%;
    box-sizing: border-box;
  }
  input[type="text"]:focus { outline: none; border-color: #6c3ce1; }

  .create-form {
    background: #16213e;
    border-radius: 0.65rem;
    padding: 1rem;
    margin-top: 0.25rem;
  }
  h3 { font-size: 0.9rem; font-weight: 700; color: #a0aec0; margin: 0 0 0.75rem; }

  .create-fields {
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
    margin-bottom: 0.6rem;
  }

  .create-footer {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .btn {
    padding: 0.4rem 0.85rem;
    border: none;
    border-radius: 0.45rem;
    cursor: pointer;
    font-size: 0.85rem;
    font-weight: 600;
    transition: opacity 0.15s;
    white-space: nowrap;
  }
  .btn:disabled { opacity: 0.45; cursor: not-allowed; }
  .save { background: #276749; color: white; }
  .save:hover:not(:disabled) { background: #38a169; }
  .cancel { background: #2d3748; color: #e2e8f0; }
  .cancel:hover:not(:disabled) { background: #4a5568; }
  .edit { background: #44337a; color: white; }
  .edit:hover:not(:disabled) { background: #6c3ce1; }
  .delete { background: #742a2a; color: white; }
  .delete:hover:not(:disabled) { background: #e53e3e; }
</style>
