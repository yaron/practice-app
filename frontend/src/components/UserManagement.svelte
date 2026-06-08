<script>
  let { jwt, api } = $props();

  /** @type {any[]} */
  let children = $state([]);
  /** @type {any[]} */
  let admins = $state([]);
  let loadError = $state("");

  /** @type {Record<number, string>} */
  let editingChildName = $state({});
  /** @type {Record<number, boolean>} */
  let childBusy = $state({});
  /** @type {Record<number, string>} */
  let childError = $state({});

  let newChildName = $state("");
  let createChildError = $state("");
  let createChildBusy = $state(false);

  let newAdminUsername = $state("");
  let newAdminPassword = $state("");
  let createError = $state("");
  let createBusy = $state(false);

  /** @type {Record<number, {username: string, password: string}|null>} */
  let editingAdmin = $state({});
  /** @type {Record<number, boolean>} */
  let adminBusy = $state({});
  /** @type {Record<number, string>} */
  let adminError = $state({});

  async function load() {
    loadError = "";
    try {
      const [cr, ar] = await Promise.all([
        fetch(`${api}/api/admin/children`, { headers: { Authorization: `Bearer ${jwt}` } }),
        fetch(`${api}/api/admin/admins`, { headers: { Authorization: `Bearer ${jwt}` } }),
      ]);
      if (cr.ok) children = await cr.json();
      if (ar.ok) admins = await ar.json();
      if (!cr.ok || !ar.ok) loadError = "Kon gebruikers niet laden";
    } catch {
      loadError = "Kon gebruikers niet laden";
    }
  }

  $effect(() => {
    if (jwt) load();
  });

  function startEditChild(child) {
    editingChildName[child.id] = child.name;
  }

  function cancelEditChild(id) {
    delete editingChildName[id];
    editingChildName = { ...editingChildName };
  }

  async function saveChildName(id) {
    const name = (editingChildName[id] ?? "").trim();
    if (!name) return;
    childBusy[id] = true;
    childError[id] = "";
    try {
      const res = await fetch(`${api}/api/admin/children/${id}`, {
        method: "PATCH",
        headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      if (res.ok) {
        children = children.map((c) => (c.id === id ? { ...c, name } : c));
        cancelEditChild(id);
      } else {
        const d = await res.json().catch(() => ({}));
        childError[id] = d.error ?? "Opslaan mislukt";
      }
    } catch {
      childError[id] = "Opslaan mislukt";
    }
    childBusy[id] = false;
  }

  async function createChild() {
    const name = newChildName.trim();
    if (!name) return;
    createChildBusy = true;
    createChildError = "";
    try {
      const res = await fetch(`${api}/api/admin/children`, {
        method: "POST",
        headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
        body: JSON.stringify({ name }),
      });
      if (res.ok) {
        const newChild = await res.json();
        children = [...children, newChild];
        newChildName = "";
      } else {
        const d = await res.json().catch(() => ({}));
        createChildError = d.error ?? "Aanmaken mislukt";
      }
    } catch {
      createChildError = "Aanmaken mislukt";
    }
    createChildBusy = false;
  }

  async function createAdmin() {
    const username = newAdminUsername.trim();
    const password = newAdminPassword.trim();
    if (!username || !password) return;
    createBusy = true;
    createError = "";
    try {
      const res = await fetch(`${api}/api/admin/admins`, {
        method: "POST",
        headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      if (res.ok) {
        const newAdmin = await res.json();
        admins = [...admins, newAdmin];
        newAdminUsername = "";
        newAdminPassword = "";
      } else {
        const d = await res.json().catch(() => ({}));
        createError = d.error ?? "Aanmaken mislukt";
      }
    } catch {
      createError = "Aanmaken mislukt";
    }
    createBusy = false;
  }

  async function deleteAdmin(id) {
    if (!confirm("Ben je zeker dat je deze beheerder wilt verwijderen?")) return;
    adminBusy[id] = true;
    adminError[id] = "";
    try {
      const res = await fetch(`${api}/api/admin/admins/${id}`, {
        method: "DELETE",
        headers: { Authorization: `Bearer ${jwt}` },
      });
      if (res.ok) {
        admins = admins.filter((a) => a.id !== id);
      } else {
        const d = await res.json().catch(() => ({}));
        adminError[id] = d.error ?? "Verwijderen mislukt";
      }
    } catch {
      adminError[id] = "Verwijderen mislukt";
    }
    adminBusy[id] = false;
  }

  function startEditAdmin(admin) {
    editingAdmin[admin.id] = { username: admin.username, password: "" };
  }

  function cancelEditAdmin(id) {
    delete editingAdmin[id];
    editingAdmin = { ...editingAdmin };
    adminError[id] = "";
  }

  async function saveAdmin(id) {
    const edit = editingAdmin[id];
    if (!edit) return;
    const username = edit.username.trim();
    const password = edit.password.trim();
    if (!username && !password) return;
    adminBusy[id] = true;
    adminError[id] = "";
    try {
      const body = {};
      if (username) body.username = username;
      if (password) body.password = password;
      const res = await fetch(`${api}/api/admin/admins/${id}`, {
        method: "PATCH",
        headers: { Authorization: `Bearer ${jwt}`, "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (res.ok) {
        if (username) admins = admins.map((a) => (a.id === id ? { ...a, username } : a));
        cancelEditAdmin(id);
      } else {
        const d = await res.json().catch(() => ({}));
        adminError[id] = d.error ?? "Opslaan mislukt";
      }
    } catch {
      adminError[id] = "Opslaan mislukt";
    }
    adminBusy[id] = false;
  }
</script>

<div class="um">
  {#if loadError}
    <p class="error">{loadError}</p>
  {/if}

  <section>
    <h2>Kinderen</h2>
    {#if children.length === 0}
      <p class="empty">Geen kinderen gevonden.</p>
    {:else}
      {#each children as child (child.id)}
        <div class="row">
          {#if editingChildName[child.id] !== undefined}
            <div class="edit-row">
              <input
                type="text"
                bind:value={editingChildName[child.id]}
                onkeydown={(e) => e.key === "Enter" && saveChildName(child.id)}
              />
              <button class="btn save" onclick={() => saveChildName(child.id)} disabled={childBusy[child.id]}>
                Opslaan
              </button>
              <button class="btn cancel" onclick={() => cancelEditChild(child.id)} disabled={childBusy[child.id]}>
                Annuleren
              </button>
            </div>
            {#if childError[child.id]}
              <p class="row-error">{childError[child.id]}</p>
            {/if}
          {:else}
            <span class="name">{child.name}</span>
            <span class="score">{child.total_points} pt</span>
            <button class="btn edit" onclick={() => startEditChild(child)}>Naam wijzigen</button>
          {/if}
        </div>
      {/each}
    {/if}

    <div class="create-form">
      <h3>Nieuw kind</h3>
      <div class="create-fields">
        <input
          type="text"
          placeholder="Naam"
          bind:value={newChildName}
          onkeydown={(e) => e.key === "Enter" && createChild()}
        />
        <button
          class="btn save"
          onclick={createChild}
          disabled={createChildBusy || !newChildName.trim()}
        >
          Aanmaken
        </button>
      </div>
      {#if createChildError}
        <p class="row-error">{createChildError}</p>
      {/if}
    </div>
  </section>

  <section>
    <h2>Beheerders</h2>
    {#each admins as admin (admin.id)}
      <div class="row">
        {#if editingAdmin[admin.id]}
          <div class="edit-block">
            <label>
              Gebruikersnaam
              <input type="text" bind:value={editingAdmin[admin.id].username} placeholder={admin.username} />
            </label>
            <label>
              Nieuw wachtwoord
              <input type="password" bind:value={editingAdmin[admin.id].password} placeholder="Ongewijzigd laten" />
            </label>
            <div class="edit-actions">
              <button class="btn save" onclick={() => saveAdmin(admin.id)} disabled={adminBusy[admin.id]}>
                Opslaan
              </button>
              <button class="btn cancel" onclick={() => cancelEditAdmin(admin.id)} disabled={adminBusy[admin.id]}>
                Annuleren
              </button>
            </div>
          </div>
        {:else}
          <span class="name">{admin.username}</span>
          <div class="admin-actions">
            <button class="btn edit" onclick={() => startEditAdmin(admin)} disabled={adminBusy[admin.id]}>
              Bewerken
            </button>
            <button class="btn delete" onclick={() => deleteAdmin(admin.id)} disabled={adminBusy[admin.id]}>
              Verwijderen
            </button>
          </div>
        {/if}
        {#if adminError[admin.id]}
          <p class="row-error">{adminError[admin.id]}</p>
        {/if}
      </div>
    {/each}

    <div class="create-form">
      <h3>Nieuwe beheerder</h3>
      <div class="create-fields">
        <input
          type="text"
          placeholder="Gebruikersnaam"
          bind:value={newAdminUsername}
        />
        <input
          type="password"
          placeholder="Wachtwoord"
          bind:value={newAdminPassword}
        />
        <button
          class="btn save"
          onclick={createAdmin}
          disabled={createBusy || !newAdminUsername.trim() || !newAdminPassword.trim()}
        >
          Aanmaken
        </button>
      </div>
      {#if createError}
        <p class="row-error">{createError}</p>
      {/if}
    </div>
  </section>
</div>

<style>
  .um { display: flex; flex-direction: column; gap: 2rem; }

  section { display: flex; flex-direction: column; gap: 0.75rem; }
  h2 { font-size: 1.05rem; font-weight: 700; color: #a78bfa; margin: 0 0 0.25rem; }
  h3 { font-size: 0.9rem; font-weight: 700; color: #a0aec0; margin: 0 0 0.5rem; }

  .empty { color: #a0aec0; font-size: 0.9rem; }
  .error { color: #fc8181; font-size: 0.9rem; }
  .row-error { color: #fc8181; font-size: 0.8rem; margin: 0; }

  .row {
    background: #16213e;
    border-radius: 0.65rem;
    padding: 0.9rem 1rem;
    display: flex;
    flex-direction: column;
    gap: 0.5rem;
  }

  .row > :first-child:not(.edit-row):not(.edit-block) {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  /* inline flex for the non-editing row */
  .row:not(:has(.edit-row)):not(:has(.edit-block)) {
    flex-direction: row;
    align-items: center;
    flex-wrap: wrap;
  }

  .name { font-weight: 600; flex: 1; }
  .score {
    background: #2d3748;
    padding: 0.15rem 0.55rem;
    border-radius: 999px;
    font-size: 0.8rem;
    color: #e2e8f0;
  }

  .admin-actions { display: flex; gap: 0.5rem; margin-left: auto; }

  .edit-row {
    display: flex;
    gap: 0.5rem;
    align-items: center;
    flex-wrap: wrap;
  }

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
  .edit-actions { display: flex; gap: 0.5rem; }

  input[type="text"],
  input[type="password"] {
    padding: 0.4rem 0.7rem;
    border-radius: 0.45rem;
    border: 1px solid #2d3748;
    background: #1a1a2e;
    color: white;
    font-size: 0.875rem;
    min-width: 150px;
  }
  input:focus { outline: none; border-color: #6c3ce1; }

  .create-form {
    background: #16213e;
    border-radius: 0.65rem;
    padding: 1rem;
    margin-top: 0.25rem;
  }
  .create-fields {
    display: flex;
    gap: 0.5rem;
    flex-wrap: wrap;
    align-items: flex-end;
  }
  .create-fields input { flex: 1; min-width: 130px; }

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
