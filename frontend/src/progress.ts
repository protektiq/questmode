import {
  ApiError,
  getProgress,
  type ProgressBadge,
  type ProgressBrainCheck,
  type ProgressData,
  type ProgressSessionDay,
  type ProgressSpellingWord,
} from "./api";

const APP_ID = "progress-app";
const MAX_QUERY_LENGTH = 128;

const getSafeLearnerId = (): string | null => {
  const queryValue = new URLSearchParams(window.location.search).get("learner_id");
  if (!queryValue) {
    return null;
  }
  const safeValue = queryValue.trim();
  if (safeValue.length === 0 || safeValue.length > MAX_QUERY_LENGTH) {
    return null;
  }
  return safeValue;
};

const createBaseStyles = (): void => {
  const style = document.createElement("style");
  style.textContent = `
    :root { font-family: Inter, system-ui, sans-serif; color-scheme: light dark; }
    body { margin: 0; background: #0f172a; color: #e2e8f0; }
    #${APP_ID} { max-width: 1080px; margin: 0 auto; padding: 1.25rem; }
    .panel { background: rgba(15, 23, 42, 0.9); border: 1px solid #334155; border-radius: 12px; padding: 1rem; margin-bottom: 1rem; }
    .metricRow { display: flex; gap: 1rem; align-items: baseline; flex-wrap: wrap; }
    .metricValue { font-size: 1.5rem; font-weight: 700; color: #f8fafc; }
    .muted { color: #94a3b8; }
    table { width: 100%; border-collapse: collapse; }
    th, td { text-align: left; padding: 0.5rem; border-bottom: 1px solid #334155; }
    .timeline { display: flex; gap: 0.6rem; flex-wrap: wrap; }
    .timelineItem { background: #1e293b; border: 1px solid #334155; border-radius: 8px; padding: 0.35rem 0.6rem; }
    .badgeRow { display: flex; gap: 0.4rem; flex-wrap: wrap; }
    .badgePill { background: #1d4ed8; border-radius: 999px; padding: 0.25rem 0.65rem; color: #eff6ff; }
    .error { color: #fecaca; }
  `;
  document.head.appendChild(style);
};

const renderBarChart = (sessions: ProgressSessionDay[]): SVGSVGElement => {
  const width = 720;
  const height = 220;
  const padding = 24;
  const maxTasks = Math.max(1, ...sessions.map((item) => item.tasks_completed));
  const barWidth = Math.floor((width - padding * 2) / Math.max(1, sessions.length)) - 8;

  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", `0 0 ${width} ${height}`);
  svg.setAttribute("width", "100%");
  svg.setAttribute("height", "220");
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", "Weekly completed tasks bar chart");

  sessions.forEach((item, index) => {
    const x = padding + index * (barWidth + 8);
    const barHeight = Math.round((item.tasks_completed / maxTasks) * (height - padding * 2));
    const y = height - padding - barHeight;

    const bar = document.createElementNS("http://www.w3.org/2000/svg", "rect");
    bar.setAttribute("x", String(x));
    bar.setAttribute("y", String(y));
    bar.setAttribute("width", String(Math.max(8, barWidth)));
    bar.setAttribute("height", String(barHeight));
    bar.setAttribute("fill", "#38bdf8");
    svg.appendChild(bar);

    const label = document.createElementNS("http://www.w3.org/2000/svg", "text");
    label.setAttribute("x", String(x + Math.max(8, barWidth) / 2));
    label.setAttribute("y", String(height - 8));
    label.setAttribute("text-anchor", "middle");
    label.setAttribute("font-size", "10");
    label.setAttribute("fill", "#cbd5e1");
    label.textContent = item.date.slice(5);
    svg.appendChild(label);
  });

  return svg;
};

const renderDonutChart = (correctRate: number): SVGSVGElement => {
  const size = 180;
  const radius = 64;
  const center = size / 2;
  const circumference = 2 * Math.PI * radius;
  const clampedRate = Math.max(0, Math.min(1, correctRate));
  const dash = clampedRate * circumference;

  const svg = document.createElementNS("http://www.w3.org/2000/svg", "svg");
  svg.setAttribute("viewBox", `0 0 ${size} ${size}`);
  svg.setAttribute("width", "180");
  svg.setAttribute("height", "180");
  svg.setAttribute("role", "img");
  svg.setAttribute("aria-label", "Math accuracy donut chart");

  const track = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  track.setAttribute("cx", String(center));
  track.setAttribute("cy", String(center));
  track.setAttribute("r", String(radius));
  track.setAttribute("fill", "none");
  track.setAttribute("stroke", "#334155");
  track.setAttribute("stroke-width", "16");
  svg.appendChild(track);

  const ring = document.createElementNS("http://www.w3.org/2000/svg", "circle");
  ring.setAttribute("cx", String(center));
  ring.setAttribute("cy", String(center));
  ring.setAttribute("r", String(radius));
  ring.setAttribute("fill", "none");
  ring.setAttribute("stroke", "#22c55e");
  ring.setAttribute("stroke-width", "16");
  ring.setAttribute("stroke-linecap", "round");
  ring.setAttribute("stroke-dasharray", `${dash} ${circumference}`);
  ring.setAttribute("transform", `rotate(-90 ${center} ${center})`);
  svg.appendChild(ring);

  const text = document.createElementNS("http://www.w3.org/2000/svg", "text");
  text.setAttribute("x", String(center));
  text.setAttribute("y", String(center + 6));
  text.setAttribute("text-anchor", "middle");
  text.setAttribute("font-size", "22");
  text.setAttribute("fill", "#f8fafc");
  text.textContent = `${Math.round(clampedRate * 100)}%`;
  svg.appendChild(text);

  return svg;
};

const buildSpellingTable = (words: ProgressSpellingWord[]): HTMLTableElement => {
  const table = document.createElement("table");
  table.setAttribute("aria-label", "Spelling mastery table");

  const thead = document.createElement("thead");
  thead.innerHTML = "<tr><th>Word</th><th>Status</th><th>Last Seen</th></tr>";
  table.appendChild(thead);

  const tbody = document.createElement("tbody");
  words.forEach((word) => {
    const row = document.createElement("tr");
    const status = word.mastered ? "Mastered" : "In progress";
    row.innerHTML = `<td>${word.word}</td><td>${status}</td><td>${word.last_seen_at ?? "-"}</td>`;
    tbody.appendChild(row);
  });
  table.appendChild(tbody);
  return table;
};

const buildBrainTimeline = (events: ProgressBrainCheck[]): HTMLDivElement => {
  const wrapper = document.createElement("div");
  wrapper.className = "timeline";
  if (events.length === 0) {
    wrapper.innerHTML = `<span class="muted">No brain check events yet.</span>`;
    return wrapper;
  }
  events.forEach((event) => {
    const item = document.createElement("div");
    item.className = "timelineItem";
    item.textContent = `${event.emoji} ${event.response}`;
    item.setAttribute("title", event.recorded_at);
    wrapper.appendChild(item);
  });
  return wrapper;
};

const buildBadgeRow = (badges: ProgressBadge[]): HTMLDivElement => {
  const row = document.createElement("div");
  row.className = "badgeRow";
  if (badges.length === 0) {
    row.innerHTML = `<span class="muted">No badges earned yet.</span>`;
    return row;
  }
  badges.forEach((badge) => {
    const badgeNode = document.createElement("span");
    badgeNode.className = "badgePill";
    badgeNode.textContent = `⭐ ${badge.badge_code}`;
    badgeNode.title = badge.earned_at;
    row.appendChild(badgeNode);
  });
  return row;
};

const renderDashboard = (container: HTMLElement, data: ProgressData): void => {
  const root = document.createElement("main");
  root.innerHTML = `
    <h1>Progress Dashboard</h1>
    <p class="muted">Learner: ${data.learner_id}</p>
  `;

  const weeklyWords = document.createElement("section");
  weeklyWords.className = "panel";
  weeklyWords.innerHTML = `
    <h2>Weekly Words Typed</h2>
    <div class="metricRow">
      <span class="metricValue">${data.weekly_words_typed}</span>
      <span class="muted">words this week</span>
    </div>
  `;
  root.appendChild(weeklyWords);

  const sessionsPanel = document.createElement("section");
  sessionsPanel.className = "panel";
  sessionsPanel.innerHTML = "<h2>Weekly Sessions</h2>";
  sessionsPanel.appendChild(renderBarChart(data.sessions));
  root.appendChild(sessionsPanel);

  const spellingPanel = document.createElement("section");
  spellingPanel.className = "panel";
  spellingPanel.innerHTML = `
    <h2>Spelling Mastery</h2>
    <p class="muted">${data.spelling.mastered}/${data.spelling.total} mastered</p>
  `;
  spellingPanel.appendChild(buildSpellingTable(data.spelling.words));
  root.appendChild(spellingPanel);

  const mathPanel = document.createElement("section");
  mathPanel.className = "panel";
  mathPanel.innerHTML = `
    <h2>Math Accuracy</h2>
    <p class="muted">${data.math.total_attempts} attempts</p>
  `;
  mathPanel.appendChild(renderDonutChart(data.math.correct_rate));
  root.appendChild(mathPanel);

  const brainPanel = document.createElement("section");
  brainPanel.className = "panel";
  brainPanel.innerHTML = "<h2>Brain Check Timeline</h2>";
  brainPanel.appendChild(buildBrainTimeline(data.brain_checks));
  root.appendChild(brainPanel);

  const badgesPanel = document.createElement("section");
  badgesPanel.className = "panel";
  badgesPanel.innerHTML = "<h2>Quest Badges</h2>";
  badgesPanel.appendChild(buildBadgeRow(data.quest_badges));
  root.appendChild(badgesPanel);

  container.innerHTML = "";
  container.appendChild(root);
};

const renderError = (container: HTMLElement, message: string): void => {
  container.innerHTML = `<p class="error">${message}</p>`;
};

const bootstrap = async (): Promise<void> => {
  createBaseStyles();
  const container = document.getElementById(APP_ID);
  if (!(container instanceof HTMLElement)) {
    return;
  }

  const learnerId = getSafeLearnerId();
  if (!learnerId) {
    renderError(container, "Missing or invalid learner_id query parameter.");
    return;
  }

  try {
    const data = await getProgress(learnerId);
    renderDashboard(container, data);
  } catch (error) {
    if (error instanceof ApiError) {
      renderError(container, error.message);
      return;
    }
    renderError(container, "Unable to load dashboard data.");
  }
};

void bootstrap();
