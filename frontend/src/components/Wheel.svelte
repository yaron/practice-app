<script>
  import confetti from "canvas-confetti";

  let { tasks, shorttasks, bonusflags = [], onresult } = $props();

  let canvas;
  let spinning = $state(false);
  let currentRotation = 0;

  const CANVAS_SIZE = 320;
  const COLORS = [
    "#6C3CE1",
    "#9B5DE5",
    "#F15BB5",
    "#FEE440",
    "#00BBF9",
    "#00F5D4",
    "#FB5607",
  ];

  $effect(() => {
    tasks; // track prop changes so wheel redraws if tasks ever update
    draw();
  });

  function draw() {
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    const n = tasks.length;
    const segAngle = (2 * Math.PI) / n;
    const r = CANVAS_SIZE / 2 - 2;
    const cx = CANVAS_SIZE / 2;
    const cy = CANVAS_SIZE / 2;

    ctx.clearRect(0, 0, CANVAS_SIZE, CANVAS_SIZE);

    for (let i = 0; i < n; i++) {
      const startAngle = i * segAngle - Math.PI / 2;
      const endAngle = startAngle + segAngle;

      ctx.beginPath();
      ctx.moveTo(cx, cy);
      ctx.arc(cx, cy, r, startAngle, endAngle);
      ctx.closePath();
      ctx.fillStyle = COLORS[i % COLORS.length];
      ctx.fill();
      ctx.strokeStyle = "#1a1a2e";
      ctx.lineWidth = 2;
      ctx.stroke();

      ctx.save();
      ctx.translate(cx, cy);
      ctx.rotate(startAngle + segAngle / 2);
      ctx.textAlign = "right";
      ctx.fillStyle = "#fff";
      ctx.font = "bold 11px sans-serif";

      const maxWidth = r - 28;
      const lines = wrapText(ctx, shorttasks[i], maxWidth);
      const lineHeight = 13;
      const totalH = lines.length * lineHeight;
      lines.forEach((line, j) => {
        ctx.fillText(
          line,
          r - 14,
          j * lineHeight - totalH / 2 + lineHeight / 2,
        );
      });
      ctx.restore();
    }

    // Center knob
    ctx.beginPath();
    ctx.arc(cx, cy, 14, 0, 2 * Math.PI);
    ctx.fillStyle = "#1a1a2e";
    ctx.fill();
    ctx.strokeStyle = "#fff";
    ctx.lineWidth = 2;
    ctx.stroke();
  }

  function wrapText(ctx, text, maxWidth) {
    const words = text.split(" ");
    const lines = [];
    let line = "";
    for (const word of words) {
      const test = line ? `${line} ${word}` : word;
      if (ctx.measureText(test).width <= maxWidth) {
        line = test;
      } else {
        if (line) lines.push(line);
        line = word;
      }
    }
    if (line) lines.push(line);
    return lines;
  }

  function spin() {
    if (spinning) return;
    spinning = true;
    const extra = 5 * 360;
    const random = Math.floor(Math.random() * 360);
    currentRotation += extra + random;
    canvas.style.transition =
      "transform 4s cubic-bezier(0.17, 0.67, 0.12, 0.99)";
    canvas.style.transform = `rotate(${currentRotation}deg)`;
  }

  function handleTransitionEnd(e) {
    if (e.propertyName !== "transform") return;
    spinning = false;
    const n = tasks.length;
    const segDeg = 360 / n;
    const norm = ((currentRotation % 360) + 360) % 360;
    const index = Math.floor(((360 - norm) % 360) / segDeg) % n;
    if (bonusflags[index]) {
      confetti({
        particleCount: 200,
        spread: 120,
        origin: { y: 0.4 },
        colors: ["#FFD700", "#FF69B4", "#00CED1"],
      });
    }
    onresult(tasks[index]);
  }
</script>

<div class="wheel-wrap">
  <div class="pointer">▼</div>
  <canvas
    bind:this={canvas}
    width={CANVAS_SIZE}
    height={CANVAS_SIZE}
    ontransitionend={handleTransitionEnd}
  ></canvas>
  <button onclick={spin} disabled={spinning}>
    {spinning ? "Draaien…" : "Draaien! 🎰"}
  </button>
</div>

<style>
  .wheel-wrap {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
  }

  .pointer {
    font-size: 1.8rem;
    color: #fee440;
    line-height: 1;
    margin-bottom: -0.5rem;
  }

  canvas {
    width: min(80vw, 320px);
    height: min(80vw, 320px);
    border-radius: 50%;
    box-shadow: 0 0 32px rgba(108, 60, 225, 0.5);
  }

  button {
    padding: 0.75rem 2.5rem;
    background: #6c3ce1;
    color: #fff;
    border: none;
    border-radius: 2rem;
    font-size: 1.1rem;
    font-weight: bold;
    cursor: pointer;
    transition:
      background 0.2s,
      transform 0.1s;
  }

  button:hover:not(:disabled) {
    background: #7d50ef;
    transform: scale(1.03);
  }

  button:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }
</style>
