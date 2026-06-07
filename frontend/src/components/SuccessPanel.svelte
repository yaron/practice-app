<script>
  let { onreset } = $props();
</script>

<div class="success">
  <div class="check">✅</div>
  <h2>Goed gedaan! 🎻</h2>
  <p>Je sessie is verstuurd!</p>
  <button onclick={onreset}>Nieuwe sessie</button>
</div>

<style>
  .success {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1.25rem;
    padding: 3rem 1rem;
    animation: pop 0.4s cubic-bezier(0.34, 1.56, 0.64, 1);
  }

  @keyframes pop {
    from { transform: scale(0.7); opacity: 0; }
    to   { transform: scale(1);   opacity: 1; }
  }

  .check {
    font-size: 5rem;
    line-height: 1;
  }

  h2 {
    color: #fee440;
    margin: 0;
    font-size: 2rem;
  }

  p {
    color: #ccc;
    margin: 0;
    font-size: 1.1rem;
  }

  button {
    padding: 0.75rem 2.5rem;
    background: #6c3ce1;
    color: #fff;
    border: none;
    border-radius: 2rem;
    font-size: 1rem;
    font-weight: bold;
    cursor: pointer;
    transition: background 0.2s;
  }

  button:hover {
    background: #7d50ef;
  }
</style>
