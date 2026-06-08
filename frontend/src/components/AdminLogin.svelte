<script>
  const API = import.meta.env.VITE_API_URL ?? "http://localhost:8080";

  let { onsuccess } = $props();

  let username = $state("");
  let password = $state("");
  let error = $state("");
  let loading = $state(false);

  async function handleLogin(e) {
    e.preventDefault();
    error = "";
    loading = true;
    try {
      const res = await fetch(`${API}/api/auth/login`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ username, password }),
      });
      const data = await res.json();
      if (!res.ok) {
        error = data.error ?? "Inloggen mislukt";
        return;
      }
      onsuccess({ token: data.token, refreshToken: data.refresh_token });
    } catch {
      error = "Verbinding mislukt. Probeer opnieuw.";
    } finally {
      loading = false;
    }
  }
</script>

<div class="login-box">
  <h1>🎻 Viool Quest</h1>
  <h2>Ouder inloggen</h2>
  <form onsubmit={handleLogin}>
    <input
      type="text"
      bind:value={username}
      placeholder="Gebruikersnaam"
      required
      autocomplete="username"
    />
    <input
      type="password"
      bind:value={password}
      placeholder="Wachtwoord"
      required
      autocomplete="current-password"
    />
    {#if error}
      <p class="error">{error}</p>
    {/if}
    <button type="submit" disabled={loading}>
      {loading ? "Bezig…" : "Inloggen"}
    </button>
  </form>
</div>

<style>
  .login-box {
    max-width: 380px;
    margin: 4rem auto;
    padding: 2rem;
    background: #16213e;
    border-radius: 1rem;
    text-align: center;
  }
  h1 { color: #fee440; font-size: 1.8rem; margin: 0 0 0.25rem; }
  h2 { color: #a0aec0; font-size: 1rem; margin: 0 0 1.5rem; font-weight: 400; }
  form { display: flex; flex-direction: column; gap: 0.75rem; }
  input {
    padding: 0.75rem 1rem;
    border-radius: 0.5rem;
    border: 1px solid #2d3748;
    background: #1a1a2e;
    color: white;
    font-size: 1rem;
  }
  input:focus { outline: none; border-color: #6c3ce1; }
  button {
    padding: 0.75rem;
    background: #6c3ce1;
    border: none;
    border-radius: 0.5rem;
    color: white;
    font-size: 1rem;
    font-weight: 700;
    cursor: pointer;
    transition: background 0.2s;
  }
  button:hover:not(:disabled) { background: #7d50ef; }
  button:disabled { opacity: 0.6; cursor: not-allowed; }
  .error { color: #fc8181; font-size: 0.875rem; margin: 0; }
</style>
