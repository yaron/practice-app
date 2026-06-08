<script>
  import { onDestroy } from 'svelte';

  let { stats } = $props();

  let xpPercent = $derived(
    stats ? Math.min(100, (stats.experience_points / (stats.current_level * 100)) * 100) : 0
  );

  let showBurst = $state(false);
  let prevMilestone = null;
  let burstTimer = null;

  $effect(() => {
    const cur = stats?.milestone_reached ?? false;
    if (prevMilestone === false && cur === true) {
      showBurst = true;
      if (burstTimer) clearTimeout(burstTimer);
      burstTimer = setTimeout(() => { showBurst = false; }, 600);
    }
    prevMilestone = cur;
  });

  onDestroy(() => { if (burstTimer) clearTimeout(burstTimer); });
</script>

<div class="hud">
  <div class="level-badge">Lvl {stats?.current_level ?? 1}</div>
  <div class="xp-bar-wrap">
    <div class="xp-fill" style="width: {xpPercent}%"></div>
  </div>
  <div class="xp-label">{stats?.experience_points ?? 0}/{(stats?.current_level ?? 1) * 100}</div>
  <div class="streak-icons" class:burst={showBurst}>
    {#each [0, 1, 2] as i}
      <span class="streak-icon" class:filled={stats && i < stats.week_session_count}>🎵</span>
    {/each}
  </div>
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
  .streak-icons {
    display: flex;
    gap: 0.15rem;
  }
  .streak-icon {
    font-size: 1.1rem;
    filter: grayscale(1) opacity(0.35);
    transition: filter 0.3s ease;
  }
  .streak-icon.filled {
    filter: none;
  }
  @keyframes streak-burst {
    0%   { transform: scale(1); }
    40%  { transform: scale(1.5); }
    65%  { transform: scale(0.85); }
    100% { transform: scale(1); }
  }
  .streak-icons.burst .streak-icon.filled {
    animation: streak-burst 0.6s cubic-bezier(0.36, 0.07, 0.19, 0.97) both;
  }
</style>
