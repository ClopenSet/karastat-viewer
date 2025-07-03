let svg = null;
let textMap = new Map(); // id -> <text> DOM
let initialized = false;
let showingCount = false;

let lastHoverText = null;

async function initOnce() {
  if (initialized) return;
  const svgText = await fetch('keyboard.svg').then(r => r.text());
  const doc = new DOMParser().parseFromString(svgText, 'image/svg+xml');
  svg = doc.documentElement;
  document.getElementById('svg-container').appendChild(svg);
  initialized = true;

  svg.addEventListener('mousemove', (e) => {
    const el = e.target.closest('[id$="-inner"]');
    if (!el) {
      if (lastHoverText) {
        svg.removeChild(lastHoverText);
        lastHoverText = null;
      }
      return;
    }

    const id = el.id;
    const count = el.getAttribute('data-count');
    if (!count) return;

    const box = el.getBBox();

    if (!lastHoverText) {
      lastHoverText = document.createElementNS("http://www.w3.org/2000/svg", "text");
      lastHoverText.setAttribute("font-size", "16");
      lastHoverText.setAttribute("fill", "#ff0");
      svg.appendChild(lastHoverText);
    }

    lastHoverText.textContent = count;
    lastHoverText.setAttribute("x", box.x + box.width / 2);
    lastHoverText.setAttribute("y", box.y - 6);
    lastHoverText.setAttribute("text-anchor", "middle");
  });
}

async function updateHeatmap() {
  const res = await fetch('/api/heatmap');
  const data = await res.json();

  for (const { id, fill, count } of data) {
    const el = svg.querySelector(`#${CSS.escape(id)}`);
    if (!el) continue;

    el.style.fill = fill;
    el.setAttribute('data-count', count);

    if (textMap.has(id)) {
      textMap.get(id).textContent = count;
    }
  }
}

document.getElementById('toggleCount').addEventListener('click', () => {
  showingCount = !showingCount;

  if (showingCount) {
    for (const el of svg.querySelectorAll('[id$="-inner"]')) {
      const id = el.id;
      const box = el.getBBox();
      const text = document.createElementNS("http://www.w3.org/2000/svg", "text");
      text.setAttribute("x", box.x + box.width / 2);
      text.setAttribute("y", box.y + box.height / 2 + 5);
      text.setAttribute("text-anchor", "middle");
      text.setAttribute("font-size", "14");
      text.setAttribute("fill", "#fff");
      text.textContent = el.getAttribute("data-count") || '?';
      svg.appendChild(text);
      textMap.set(id, text);
    }
  } else {
    for (const text of textMap.values()) {
      svg.removeChild(text);
    }
    textMap.clear();
  }
});

function setupSSE() {
  const evt = new EventSource("/events");
  evt.onmessage = (e) => {
    const data = JSON.parse(e.data);
    for (const { id, fill, count } of data) {
      const el = svg.querySelector(`#${CSS.escape(id)}`);
      if (!el) continue;
      el.style.fill = fill;
      el.setAttribute("data-count", count);
      if (textMap.has(id)) {
        textMap.get(id).textContent = count;
      }
    }
  };
}

async function start() {
  await initOnce();
  setupSSE();
}

start();
