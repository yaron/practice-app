<script>
  import Wheel from "./components/Wheel.svelte";
  import SessionTracker from "./components/SessionTracker.svelte";
  import SuccessPanel from "./components/SuccessPanel.svelte";
  import { WHEEL_TASKS, WHEEL_TASKS_SHORT } from "./lib/wheelData.js";

  let sessionTasks = $state([]);
  let submitted = $state(false);

  function addTask(task) {
    sessionTasks = [...sessionTasks, task];
  }

  function handleSubmit() {
    submitted = true;
  }

  function handleReset() {
    sessionTasks = [];
    submitted = false;
  }
</script>

<main>
  <h1>🎻 Viool Quest</h1>

  {#if submitted}
    <SuccessPanel onreset={handleReset} />
  {:else}
    <Wheel
      tasks={WHEEL_TASKS}
      shorttasks={WHEEL_TASKS_SHORT}
      onresult={addTask}
    />
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
</style>
