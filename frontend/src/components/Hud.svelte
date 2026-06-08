<script>
  let { stats } = $props();

  let xpPercent = $derived(
    stats ? Math.min(100, (stats.experience_points / (stats.current_level * 100)) * 100) : 0
  );
</script>

<div class="hud">
  <div class="level-badge">Lvl {stats?.current_level ?? 1}</div>
  <div class="xp-bar-wrap">
    <div class="xp-fill" style="width: {xpPercent}%"></div>
  </div>
  <div class="xp-label">{stats?.experience_points ?? 0}/{(stats?.current_level ?? 1) * 100}</div>
</div>

<style>
  .hud {
    display: flex;
    align-items: center;
    gap: 0.6rem;
    padding: 0.6rem 1rem;
    background: #16213e;
    border-radius: 0.75rem;
    margin-bottom: 1rem;
  }
  .level-badge {
    background: #6c3ce1;
    color: white;
    padding: 0.2rem 0.65rem;
    border-radius: 999px;
    font-weight: 700;
    font-size: 0.85rem;
    white-space: nowrap;
  }
  .xp-bar-wrap {
    flex: 1;
    height: 8px;
    background: #2d3748;
    border-radius: 999px;
    overflow: hidden;
  }
  .xp-fill {
    height: 100%;
    background: linear-gradient(90deg, #6c3ce1, #a855f7);
    border-radius: 999px;
    transition: width 0.5s ease;
  }
  .xp-label {
    font-size: 0.75rem;
    color: #a0aec0;
    white-space: nowrap;
  }
</style>
