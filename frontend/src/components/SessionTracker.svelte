<script>
  let { tasks, onsubmit } = $props();
</script>

{#if tasks.length > 0}
  <div class="tracker">
    <h3>Jouw oefening ({tasks.length} {tasks.length === 1 ? 'taak' : 'taken'})</h3>
    <ul>
      {#each tasks as task}
        <li>✅ {task}</li>
      {/each}
    </ul>
    <button onclick={onsubmit}>Verstuur sessie 🚀</button>
  </div>
{/if}

<style>
  .tracker {
    margin-top: 2rem;
    background: rgba(255, 255, 255, 0.06);
    border-radius: 1rem;
    padding: 1.25rem;
    text-align: left;
    width: min(80vw, 320px);
  }

  h3 {
    margin: 0 0 0.75rem;
    color: #fee440;
    font-size: 1rem;
    font-weight: bold;
  }

  ul {
    list-style: none;
    padding: 0;
    margin: 0 0 1rem;
  }

  li {
    padding: 0.4rem 0;
    font-size: 0.95rem;
    border-bottom: 1px solid rgba(255, 255, 255, 0.07);
  }

  li:last-child {
    border-bottom: none;
  }

  button {
    width: 100%;
    padding: 0.75rem;
    background: #fee440;
    color: #1a1a2e;
    border: none;
    border-radius: 2rem;
    font-size: 1rem;
    font-weight: bold;
    cursor: pointer;
    transition: background 0.2s, transform 0.1s;
  }

  button:hover {
    background: #ffe84d;
    transform: scale(1.02);
  }
</style>
