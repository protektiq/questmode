export type RewardType = "correct" | "hint" | "chapter_complete";
export type AnimationType = "starburst" | "confetti" | "glow" | "character_cheer";
export type SoundType = "chime" | "cheer" | "sparkle";

const REWARD_STYLE_ID = "quest-reward-engine-styles";
const REWARD_LAYER_ID = "quest-reward-layer";
const CHAPTER_BANNER_ID = "quest-chapter-banner";

const ANIM_DURATION_MS = {
  starburst: 900,
  confetti: 1300,
  glow: 700,
  character_cheer: 1000,
  chapterBanner: 1200,
} as const;

const SOUND_DURATION_MS = {
  chimePart: 80,
  cheer: 200,
  sparkleTotal: 200,
} as const;

const ensureRewardStyles = (): void => {
  if (document.getElementById(REWARD_STYLE_ID)) {
    return;
  }

  const style = document.createElement("style");
  style.id = REWARD_STYLE_ID;
  style.textContent = `
    #${REWARD_LAYER_ID} {
      position: fixed;
      inset: 0;
      pointer-events: none;
      z-index: 9999;
      overflow: hidden;
    }

    .quest-reward-node {
      position: absolute;
      inset: 0;
      pointer-events: none;
    }

    .quest-starburst {
      width: 160px;
      height: 160px;
      position: absolute;
      left: 50%;
      top: 38%;
      transform: translate(-50%, -50%);
      opacity: 0;
      animation: quest-starburst-pop ${ANIM_DURATION_MS.starburst}ms ease-out forwards;
    }

    .quest-starburst-ray {
      transform-origin: center;
      animation: quest-starburst-ray ${ANIM_DURATION_MS.starburst}ms ease-out forwards;
    }

    .quest-confetti-piece {
      position: absolute;
      width: 10px;
      height: 10px;
      top: -12px;
      opacity: 0.95;
      animation-name: quest-confetti-drop;
      animation-timing-function: cubic-bezier(0.12, 0.82, 0.2, 1);
      animation-fill-mode: forwards;
    }

    .quest-glow-ring {
      position: absolute;
      inset: 0.6rem;
      border-radius: 1rem;
      border: 3px solid rgba(250, 204, 21, 0.85);
      box-shadow: 0 0 0 rgba(250, 204, 21, 0);
      opacity: 0;
      animation: quest-glow-pulse ${ANIM_DURATION_MS.glow}ms ease-out forwards;
    }

    .quest-character {
      width: 72px;
      height: 100px;
      position: absolute;
      right: 2rem;
      bottom: 1.5rem;
      opacity: 0;
      animation: quest-character-jump ${ANIM_DURATION_MS.character_cheer}ms ease-out forwards;
    }

    .quest-chapter-banner {
      position: fixed;
      left: 50%;
      top: 12%;
      transform: translateX(-50%);
      padding: 0.7rem 1rem;
      border-radius: 999px;
      background: rgba(22, 163, 74, 0.95);
      color: #ffffff;
      font-family: system-ui, sans-serif;
      font-weight: 700;
      letter-spacing: 0.02em;
      box-shadow: 0 12px 30px rgba(22, 163, 74, 0.3);
      pointer-events: none;
      opacity: 0;
      z-index: 10000;
      animation: quest-banner-slide ${ANIM_DURATION_MS.chapterBanner}ms ease-out forwards;
    }

    @keyframes quest-starburst-pop {
      0% { transform: translate(-50%, -50%) scale(0.2); opacity: 0; }
      20% { opacity: 1; }
      100% { transform: translate(-50%, -50%) scale(1.2); opacity: 0; }
    }

    @keyframes quest-starburst-ray {
      0% { stroke-dasharray: 0 100; opacity: 0.2; }
      25% { stroke-dasharray: 100 0; opacity: 1; }
      100% { stroke-dasharray: 100 0; opacity: 0; }
    }

    @keyframes quest-confetti-drop {
      0% { transform: translateY(0) rotate(0deg); opacity: 0.95; }
      100% { transform: translateY(110vh) rotate(800deg); opacity: 0; }
    }

    @keyframes quest-glow-pulse {
      0% { opacity: 0; box-shadow: 0 0 0 rgba(250, 204, 21, 0); }
      30% { opacity: 1; box-shadow: 0 0 20px rgba(250, 204, 21, 0.7); }
      100% { opacity: 0; box-shadow: 0 0 0 rgba(250, 204, 21, 0); }
    }

    @keyframes quest-character-jump {
      0% { transform: translateY(0) scale(0.9); opacity: 0; }
      25% { opacity: 1; }
      45% { transform: translateY(-30px) scale(1); }
      70% { transform: translateY(0) scale(1); opacity: 1; }
      100% { opacity: 0; }
    }

    @keyframes quest-banner-slide {
      0% { opacity: 0; transform: translate(-50%, -20px); }
      20% { opacity: 1; transform: translate(-50%, 0); }
      80% { opacity: 1; transform: translate(-50%, 0); }
      100% { opacity: 0; transform: translate(-50%, -8px); }
    }
  `;

  document.head.appendChild(style);
};

const getRewardLayer = (): HTMLDivElement => {
  const existing = document.getElementById(REWARD_LAYER_ID);
  if (existing instanceof HTMLDivElement) {
    return existing;
  }

  const layer = document.createElement("div");
  layer.id = REWARD_LAYER_ID;
  layer.setAttribute("aria-hidden", "true");
  document.body.appendChild(layer);
  return layer;
};

const safeRemoveNodeAfter = (node: Element, delayMs: number): void => {
  window.setTimeout(() => {
    node.remove();
  }, delayMs);
};

const isRewardType = (value: string): value is RewardType =>
  value === "correct" || value === "hint" || value === "chapter_complete";

const isAnimationType = (value: string): value is AnimationType =>
  value === "starburst" ||
  value === "confetti" ||
  value === "glow" ||
  value === "character_cheer";

const isSoundType = (value: string): value is SoundType =>
  value === "chime" || value === "cheer" || value === "sparkle";

export class RewardEngine {
  private animationIndex = 0;
  private soundIndex = 0;
  private audioCtx: AudioContext | null = null;
  private readonly animationCycle: AnimationType[] = [
    "starburst",
    "character_cheer",
    "confetti",
  ];
  private readonly soundCycle: SoundType[] = ["chime", "sparkle"];

  trigger(type: RewardType): void {
    if (!isRewardType(type)) {
      return;
    }

    const focusedElement = document.activeElement;
    const reducedMotion = window.matchMedia(
      "(prefers-reduced-motion: reduce)",
    ).matches;

    if (reducedMotion) {
      if (type !== "hint") {
        this.playSound(this.nextSound());
      }
      this.restoreFocus(focusedElement);
      return;
    }

    switch (type) {
      case "correct":
        this.animate(this.nextAnim());
        this.playSound(this.nextSound());
        break;
      case "hint":
        this.animate("glow");
        break;
      case "chapter_complete":
        this.animate("confetti");
        this.playSound("cheer");
        this.showChapterBanner();
        break;
    }

    this.restoreFocus(focusedElement);
  }

  private nextAnim(): AnimationType {
    const animation = this.animationCycle[this.animationIndex % this.animationCycle.length];
    this.animationIndex += 1;
    return animation;
  }

  private nextSound(): SoundType {
    const sound = this.soundCycle[this.soundIndex % this.soundCycle.length];
    this.soundIndex += 1;
    return sound;
  }

  private animate(type: AnimationType): void {
    if (!isAnimationType(type)) {
      return;
    }

    ensureRewardStyles();
    const layer = getRewardLayer();
    const wrapper = document.createElement("div");
    wrapper.className = "quest-reward-node";

    if (type === "starburst") {
      wrapper.innerHTML = `
        <svg class="quest-starburst" viewBox="0 0 100 100" aria-hidden="true">
          <circle cx="50" cy="50" r="7" fill="#facc15" />
          <line class="quest-starburst-ray" x1="50" y1="7" x2="50" y2="32" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="50" y1="68" x2="50" y2="93" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="7" y1="50" x2="32" y2="50" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="68" y1="50" x2="93" y2="50" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="19" y1="19" x2="35" y2="35" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="65" y1="65" x2="81" y2="81" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="19" y1="81" x2="35" y2="65" stroke="#f59e0b" stroke-width="4"/>
          <line class="quest-starburst-ray" x1="65" y1="35" x2="81" y2="19" stroke="#f59e0b" stroke-width="4"/>
        </svg>
      `;
      layer.appendChild(wrapper);
      safeRemoveNodeAfter(wrapper, ANIM_DURATION_MS.starburst + 120);
      return;
    }

    if (type === "confetti") {
      const confettiColors = ["#22c55e", "#3b82f6", "#f97316", "#eab308", "#ec4899"];
      const pieceCount = 26;
      for (let index = 0; index < pieceCount; index += 1) {
        const piece = document.createElement("span");
        const color = confettiColors[index % confettiColors.length];
        const left = Math.round((index / pieceCount) * 100);
        const duration = 850 + Math.round(Math.random() * 450);
        const delay = Math.round(Math.random() * 220);
        piece.className = "quest-confetti-piece";
        piece.style.left = `${left}%`;
        piece.style.backgroundColor = color;
        piece.style.animationDuration = `${duration}ms`;
        piece.style.animationDelay = `${delay}ms`;
        wrapper.appendChild(piece);
      }

      layer.appendChild(wrapper);
      safeRemoveNodeAfter(wrapper, ANIM_DURATION_MS.confetti + 220);
      return;
    }

    if (type === "glow") {
      const target = document.getElementById("quest-app");
      const targetRect = target?.getBoundingClientRect();
      const ring = document.createElement("div");
      ring.className = "quest-glow-ring";

      if (targetRect) {
        ring.style.left = `${targetRect.left}px`;
        ring.style.top = `${targetRect.top}px`;
        ring.style.width = `${targetRect.width}px`;
        ring.style.height = `${targetRect.height}px`;
        ring.style.inset = "auto";
      }

      wrapper.appendChild(ring);
      layer.appendChild(wrapper);
      safeRemoveNodeAfter(wrapper, ANIM_DURATION_MS.glow + 100);
      return;
    }

    const character = document.createElement("div");
    character.className = "quest-character";
    character.innerHTML = `
      <svg viewBox="0 0 72 100" aria-hidden="true">
        <circle cx="36" cy="16" r="10" fill="#fde68a" stroke="#0f172a" stroke-width="2" />
        <line x1="36" y1="26" x2="36" y2="62" stroke="#0f172a" stroke-width="4" />
        <line x1="36" y1="36" x2="18" y2="48" stroke="#0f172a" stroke-width="4" />
        <line x1="36" y1="36" x2="54" y2="48" stroke="#0f172a" stroke-width="4" />
        <line x1="36" y1="62" x2="22" y2="90" stroke="#0f172a" stroke-width="4" />
        <line x1="36" y1="62" x2="50" y2="90" stroke="#0f172a" stroke-width="4" />
      </svg>
    `;
    wrapper.appendChild(character);
    layer.appendChild(wrapper);
    safeRemoveNodeAfter(wrapper, ANIM_DURATION_MS.character_cheer + 120);
  }

  private showChapterBanner(): void {
    ensureRewardStyles();
    const existing = document.getElementById(CHAPTER_BANNER_ID);
    if (existing) {
      existing.remove();
    }

    const banner = document.createElement("div");
    banner.id = CHAPTER_BANNER_ID;
    banner.className = "quest-chapter-banner";
    banner.setAttribute("role", "status");
    banner.setAttribute("aria-live", "polite");
    banner.textContent = "Chapter Complete!";
    document.body.appendChild(banner);
    safeRemoveNodeAfter(banner, ANIM_DURATION_MS.chapterBanner + 60);
  }

  private playSound(type: SoundType): void {
    if (!isSoundType(type)) {
      return;
    }

    const audioCtx = this.getAudioContext();
    if (!audioCtx) {
      return;
    }

    const now = audioCtx.currentTime;
    if (type === "chime") {
      this.playTone(audioCtx, 523.25, now, SOUND_DURATION_MS.chimePart / 1000, 0.08);
      this.playTone(
        audioCtx,
        659.25,
        now + SOUND_DURATION_MS.chimePart / 1000,
        SOUND_DURATION_MS.chimePart / 1000,
        0.08,
      );
      return;
    }

    if (type === "cheer") {
      const osc = audioCtx.createOscillator();
      const gain = audioCtx.createGain();
      osc.type = "sine";
      osc.frequency.setValueAtTime(300, now);
      osc.frequency.linearRampToValueAtTime(
        800,
        now + SOUND_DURATION_MS.cheer / 1000,
      );
      gain.gain.setValueAtTime(0.0001, now);
      gain.gain.exponentialRampToValueAtTime(0.1, now + 0.015);
      gain.gain.exponentialRampToValueAtTime(
        0.0001,
        now + SOUND_DURATION_MS.cheer / 1000,
      );
      osc.connect(gain);
      gain.connect(audioCtx.destination);
      osc.start(now);
      osc.stop(now + SOUND_DURATION_MS.cheer / 1000);
      return;
    }

    const pingSpacing = SOUND_DURATION_MS.sparkleTotal / 3 / 1000;
    this.playTone(audioCtx, 880, now, 0.06, 0.06);
    this.playTone(audioCtx, 1046.5, now + pingSpacing, 0.06, 0.06);
    this.playTone(audioCtx, 1318.5, now + pingSpacing * 2, 0.06, 0.06);
  }

  private playTone(
    audioCtx: AudioContext,
    frequency: number,
    startTime: number,
    durationSeconds: number,
    maxGain: number,
  ): void {
    const oscillator = audioCtx.createOscillator();
    const gain = audioCtx.createGain();

    oscillator.type = "sine";
    oscillator.frequency.setValueAtTime(frequency, startTime);
    gain.gain.setValueAtTime(0.0001, startTime);
    gain.gain.exponentialRampToValueAtTime(maxGain, startTime + 0.01);
    gain.gain.exponentialRampToValueAtTime(
      0.0001,
      startTime + durationSeconds,
    );

    oscillator.connect(gain);
    gain.connect(audioCtx.destination);
    oscillator.start(startTime);
    oscillator.stop(startTime + durationSeconds);
  }

  private getAudioContext(): AudioContext | null {
    try {
      if (!this.audioCtx) {
        const AudioContextCtor =
          window.AudioContext ||
          (window as Window & { webkitAudioContext?: typeof AudioContext })
            .webkitAudioContext;
        if (!AudioContextCtor) {
          return null;
        }
        this.audioCtx = new AudioContextCtor();
      }

      if (this.audioCtx.state === "suspended") {
        void this.audioCtx.resume();
      }
      return this.audioCtx;
    } catch {
      return null;
    }
  }

  private restoreFocus(previouslyFocused: Element | null): void {
    if (!(previouslyFocused instanceof HTMLElement)) {
      return;
    }
    if (!document.contains(previouslyFocused)) {
      return;
    }
    if (previouslyFocused === document.activeElement) {
      return;
    }
    window.requestAnimationFrame(() => {
      previouslyFocused.focus({ preventScroll: true });
    });
  }
}

export const rewardEngine = new RewardEngine();
