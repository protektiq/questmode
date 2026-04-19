import {
  ApiError,
  type ChapterResponse,
  type QuestBadge,
  type Question,
  checkMath,
  checkSpelling,
  getChapter,
  getMathProblem,
  getSpellingWord,
  logWriting,
  startArc,
  submitAnswer,
  submitBrainCheck,
} from "./api";
import { rewardEngine } from "./rewards";

type QuestPhase = "chapter" | "question" | "spelling" | "math";
type Difficulty = "easy" | "medium" | "hard";
type BrainCheckChoice = "focused" | "frustrated";

interface QuestSessionState {
  learnerId: string;
  arcId: string;
  sessionId: string;
  genre: string;
  chapterId: string;
  chapterIndex: number;
  questionIndex: number;
  activePhase: QuestPhase;
  completedTasks: number;
  progressPercent: number;
  engagementSeconds: number;
  breakOffered: boolean;
  difficulty: Difficulty;
  lastActivityMs: number;
}

interface PromptResult {
  value: string;
  usedHint: boolean;
}

const QUEST_CONSTANTS = {
  dom: {
    rootId: "quest-app",
    statusId: "quest-status",
    progressId: "quest-progress",
    chapterId: "quest-chapter",
    taskId: "quest-task",
    feedbackId: "quest-feedback",
  },
  progress: {
    keydownIncrement: 2,
    correctIncrement: 10,
    hintIncrement: 5,
    min: 0,
    max: 100,
  },
  loop: {
    brainCheckEveryTasks: 5,
    breakAfterSeconds: 12 * 60,
  },
  difficulty: {
    normal: "medium" as Difficulty,
    easier: "easy" as Difficulty,
  },
  genre: {
    defaultValue: "fantasy",
    maxLength: 64,
  },
  input: {
    minLength: 1,
    writingMinLength: 20,
    maxAnswerLength: 500,
    maxWordLength: 256,
    maxWritingLength: 20000,
  },
} as const;

const UI_TEXT = {
  documentTitle: "Quest Mode - Active Session",
  heading: "Quest Session",
  loading: "Starting your quest...",
  statusReady: "Quest ready.",
  statusChapter: "New chapter loaded.",
  statusQuestion: "Answer the current chapter question.",
  statusSpelling: "Complete the spelling challenge.",
  statusMath: "Solve the math puzzle.",
  statusBrainCheck: "Brain check submitted.",
  statusBreak: "Power-Up break is ready.",
  statusErrorPrefix: "Error:",
  progressLabelPrefix: "Progress",
  chapterTitlePrefix: "Chapter",
  questionLabel: "Question",
  answerLabel: "Your answer",
  answerPlaceholder: "Type your answer",
  submitButton: "Submit",
  hintButton: "Use hint",
  hintPrefix: "Hint:",
  responseCorrect: "Correct! Great job.",
  responseIncorrect: "Not quite, try again.",
  spellingPromptPrefix: "Spell this word:",
  spellingAnswerPlaceholder: "Type the spelling",
  spellingCorrect: "Spelling challenge complete.",
  mathPromptPrefix: "Puzzle:",
  mathAnswerPlaceholder: "Type a whole number",
  mathCorrect: "Math puzzle complete.",
  brainCheckPrompt: "How are you feeling?",
  brainCheckFocused: "Focused",
  brainCheckFrustrated: "Frustrated",
  brainCheckNextNormal: "Next challenge stays at normal difficulty.",
  brainCheckNextEasier: "Next challenge will be easier.",
  breakPrompt:
    "You have been actively engaged for 12 minutes. Take a power-up break.",
  initErrorMissingRoot: "Missing quest root element.",
  validationErrorEmptyInput: "Input cannot be empty.",
  validationErrorLongInput: "Input is too long.",
  validationErrorNumber: "Please enter a valid whole number.",
  apiErrorDefault: "Request failed.",
  arcCompleteHeading: "Quest complete!",
  arcCompleteNewQuest: "New quest",
  arcCompleteViewProgress: "View progress",
  arcCompleteBadgePrefix: "Genre",
  writingPromptHeading: "What do you think happens next?",
  writingPromptHint: "Share your prediction in at least 20 characters.",
  writingPromptPlaceholder: "Write your idea for what happens next...",
  writingPromptSubmit: "Save and continue",
  writingPromptTooShort: "Please write at least 20 characters.",
  writingPromptSaved: "Great idea saved!",
} as const;

const sessionState: QuestSessionState = {
  learnerId: "",
  arcId: "",
  sessionId: "",
  genre: QUEST_CONSTANTS.genre.defaultValue,
  chapterId: "",
  chapterIndex: 0,
  questionIndex: 0,
  activePhase: "chapter",
  completedTasks: 0,
  progressPercent: 0,
  engagementSeconds: 0,
  breakOffered: false,
  difficulty: QUEST_CONSTANTS.difficulty.normal,
  lastActivityMs: Date.now(),
};

const getRequiredElement = <T extends HTMLElement>(id: string): T => {
  const element = document.getElementById(id);
  if (!element) {
    throw new Error(UI_TEXT.initErrorMissingRoot);
  }
  return element as T;
};

const escapeHtml = (value: string): string =>
  value
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;")
    .replaceAll("'", "&#39;");

const clamp = (value: number, minValue: number, maxValue: number): number =>
  Math.min(maxValue, Math.max(minValue, value));

const setStatus = (message: string): void => {
  const statusElement = getRequiredElement<HTMLParagraphElement>(
    QUEST_CONSTANTS.dom.statusId,
  );
  statusElement.textContent = message;
};

const setFeedback = (message: string): void => {
  const feedbackElement = getRequiredElement<HTMLDivElement>(
    QUEST_CONSTANTS.dom.feedbackId,
  );
  feedbackElement.textContent = message;
};

const renderProgress = (): void => {
  const progressElement = getRequiredElement<HTMLProgressElement>(
    QUEST_CONSTANTS.dom.progressId,
  );
  progressElement.value = sessionState.progressPercent;
  progressElement.max = QUEST_CONSTANTS.progress.max;

  const label = `${UI_TEXT.progressLabelPrefix}: ${sessionState.progressPercent}%`;
  progressElement.setAttribute("aria-label", label);
};

const incrementProgress = (delta: number): void => {
  sessionState.progressPercent = clamp(
    sessionState.progressPercent + delta,
    QUEST_CONSTANTS.progress.min,
    QUEST_CONSTANTS.progress.max,
  );
  renderProgress();
};

const updateEngagementSeconds = (): void => {
  const nowMs = Date.now();
  const elapsedSeconds = Math.max(
    0,
    Math.floor((nowMs - sessionState.lastActivityMs) / 1000),
  );
  sessionState.lastActivityMs = nowMs;
  sessionState.engagementSeconds += elapsedSeconds;
};

const maybeOfferBreak = (): void => {
  if (sessionState.breakOffered) {
    return;
  }
  if (sessionState.engagementSeconds < QUEST_CONSTANTS.loop.breakAfterSeconds) {
    return;
  }
  sessionState.breakOffered = true;
  setStatus(UI_TEXT.statusBreak);
  setFeedback(UI_TEXT.breakPrompt);
};

const trackUserInteraction = (): void => {
  updateEngagementSeconds();
  incrementProgress(QUEST_CONSTANTS.progress.keydownIncrement);
  maybeOfferBreak();
};

const markTaskCompleted = async (): Promise<void> => {
  sessionState.completedTasks += 1;
  if (
    sessionState.completedTasks % QUEST_CONSTANTS.loop.brainCheckEveryTasks !==
    0
  ) {
    return;
  }

  const choice = await askBrainCheckChoice();
  const result = await submitBrainCheck(sessionState.learnerId, choice);
  sessionState.difficulty =
    result.next_difficulty === "easier"
      ? QUEST_CONSTANTS.difficulty.easier
      : QUEST_CONSTANTS.difficulty.normal;
  setStatus(UI_TEXT.statusBrainCheck);
  setFeedback(
    result.next_difficulty === "easier"
      ? UI_TEXT.brainCheckNextEasier
      : UI_TEXT.brainCheckNextNormal,
  );
};

const validateFreeText = (
  rawValue: string,
  maxLength: number,
  minLength = QUEST_CONSTANTS.input.minLength,
): string => {
  const value = rawValue.trim();
  if (value.length < minLength) {
    throw new Error(UI_TEXT.validationErrorEmptyInput);
  }
  if (value.length > maxLength) {
    throw new Error(UI_TEXT.validationErrorLongInput);
  }
  return value;
};

const renderShell = (): void => {
  document.title = UI_TEXT.documentTitle;
  const app = getRequiredElement<HTMLDivElement>(QUEST_CONSTANTS.dom.rootId);
  app.innerHTML = `<style>
    @keyframes questBadgePulse {
      0%, 100% { transform: scale(1) rotate(0deg); }
      50% { transform: scale(1.08) rotate(5deg); }
    }
    .questOverlay {
      position: fixed;
      inset: 0;
      background: rgba(15, 23, 42, 0.95);
      display: grid;
      place-items: center;
      z-index: 999;
      padding: 1.5rem;
    }
    .questOverlayCard {
      width: min(34rem, 100%);
      text-align: center;
      background: #111827;
      border: 1px solid #334155;
      border-radius: 1rem;
      padding: 2rem 1.5rem;
      color: #f8fafc;
    }
    .questBadgeStar {
      width: 10rem;
      height: 10rem;
      margin: 0 auto 1rem;
      background: linear-gradient(135deg, #fbbf24, #f59e0b);
      clip-path: polygon(50% 0%, 61% 35%, 98% 35%, 69% 57%, 79% 91%, 50% 70%, 21% 91%, 31% 57%, 2% 35%, 39% 35%);
      display: grid;
      place-items: center;
      font-weight: 700;
      color: #1f2937;
      text-transform: uppercase;
      animation: questBadgePulse 1.8s ease-in-out infinite;
      padding: 1rem;
    }
    .questOverlayActions {
      display: flex;
      gap: 0.75rem;
      justify-content: center;
      flex-wrap: wrap;
      margin-top: 1rem;
    }
    .questOverlayActions button,
    .questOverlayActions a {
      border-radius: 0.6rem;
      border: 1px solid #475569;
      padding: 0.6rem 1rem;
      text-decoration: none;
      color: #f8fafc;
      background: #1e293b;
      cursor: pointer;
    }
  </style>
  <main style="font-family: system-ui, sans-serif; max-width: 50rem; margin: 2rem auto; padding: 0 1rem;">
    <h1>${escapeHtml(UI_TEXT.heading)}</h1>
    <p id="${QUEST_CONSTANTS.dom.statusId}" role="status" aria-live="polite">${escapeHtml(UI_TEXT.loading)}</p>
    <label for="${QUEST_CONSTANTS.dom.progressId}">${escapeHtml(UI_TEXT.progressLabelPrefix)}</label>
    <progress id="${QUEST_CONSTANTS.dom.progressId}" value="0" max="${QUEST_CONSTANTS.progress.max}" aria-label="${escapeHtml(UI_TEXT.progressLabelPrefix)}"></progress>
    <section id="${QUEST_CONSTANTS.dom.chapterId}" aria-live="polite"></section>
    <section id="${QUEST_CONSTANTS.dom.taskId}" aria-live="polite"></section>
    <div id="${QUEST_CONSTANTS.dom.feedbackId}" aria-live="polite"></div>
  </main>`;
};

const renderChapter = (chapter: ChapterResponse): void => {
  const chapterElement = getRequiredElement<HTMLElement>(QUEST_CONSTANTS.dom.chapterId);
  const chapterTitle = `${UI_TEXT.chapterTitlePrefix} ${sessionState.chapterIndex + 1}`;
  chapterElement.innerHTML = `<h2>${escapeHtml(chapterTitle)}</h2><p>${escapeHtml(chapter.content)}</p>`;
  setStatus(UI_TEXT.statusChapter);
};

const waitForFormSubmit = (
  formId: string,
  inputId: string,
  hintButtonId: string,
  hintText: string,
): Promise<PromptResult> =>
  new Promise<PromptResult>((resolve) => {
    const form = getRequiredElement<HTMLFormElement>(formId);
    const input = getRequiredElement<HTMLInputElement>(inputId);
    const hintButton = getRequiredElement<HTMLButtonElement>(hintButtonId);

    let usedHint = false;
    hintButton.addEventListener("click", () => {
      usedHint = true;
      incrementProgress(QUEST_CONSTANTS.progress.hintIncrement);
      setFeedback(`${UI_TEXT.hintPrefix} ${hintText}`);
      rewardEngine.trigger("hint");
    });

    form.addEventListener("submit", (event) => {
      event.preventDefault();
      resolve({
        value: input.value,
        usedHint,
      });
    });
  });

const renderTextPrompt = async (params: {
  title: string;
  prompt: string;
  placeholder: string;
  hintText: string;
}): Promise<PromptResult> => {
  const taskElement = getRequiredElement<HTMLElement>(QUEST_CONSTANTS.dom.taskId);
  const formId = "quest-form";
  const inputId = "quest-input";
  const hintButtonId = "quest-hint";

  taskElement.innerHTML = `
    <h3>${escapeHtml(params.title)}</h3>
    <p>${escapeHtml(params.prompt)}</p>
    <button type="button" id="${hintButtonId}">${escapeHtml(UI_TEXT.hintButton)}</button>
    <form id="${formId}">
      <label for="${inputId}">${escapeHtml(UI_TEXT.answerLabel)}</label>
      <input id="${inputId}" type="text" placeholder="${escapeHtml(params.placeholder)}" aria-label="${escapeHtml(params.title)}" required />
      <button type="submit">${escapeHtml(UI_TEXT.submitButton)}</button>
    </form>
  `;

  return waitForFormSubmit(formId, inputId, hintButtonId, params.hintText);
};

const showWritingPromptIfNeeded = async (): Promise<void> => {
  if (sessionState.chapterIndex % 3 !== 0) {
    return;
  }
  const taskElement = getRequiredElement<HTMLElement>(QUEST_CONSTANTS.dom.taskId);
  const formId = "writing-form";
  const textAreaId = "writing-input";
  const validationId = "writing-validation";
  taskElement.innerHTML = `
    <h3>${escapeHtml(UI_TEXT.writingPromptHeading)}</h3>
    <p>${escapeHtml(UI_TEXT.writingPromptHint)}</p>
    <form id="${formId}">
      <label for="${textAreaId}">${escapeHtml(UI_TEXT.writingPromptHeading)}</label>
      <textarea id="${textAreaId}" rows="5" placeholder="${escapeHtml(UI_TEXT.writingPromptPlaceholder)}" aria-label="${escapeHtml(UI_TEXT.writingPromptHeading)}"></textarea>
      <p id="${validationId}" role="status" aria-live="polite"></p>
      <button type="submit">${escapeHtml(UI_TEXT.writingPromptSubmit)}</button>
    </form>
  `;

  await new Promise<void>((resolve) => {
    const form = getRequiredElement<HTMLFormElement>(formId);
    const textArea = getRequiredElement<HTMLTextAreaElement>(textAreaId);
    const validation = getRequiredElement<HTMLParagraphElement>(validationId);

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      try {
        const safeText = validateFreeText(
          textArea.value,
          QUEST_CONSTANTS.input.maxWritingLength,
          QUEST_CONSTANTS.input.writingMinLength,
        );
        await logWriting(sessionState.learnerId, sessionState.sessionId, safeText);
        setFeedback(UI_TEXT.writingPromptSaved);
        resolve();
      } catch (error) {
        if (error instanceof ApiError || error instanceof Error) {
          validation.textContent =
            textArea.value.trim().length < QUEST_CONSTANTS.input.writingMinLength
              ? UI_TEXT.writingPromptTooShort
              : error.message;
          return;
        }
        validation.textContent = UI_TEXT.apiErrorDefault;
      }
    });
  });
};

const showArcCompletionOverlay = async (badge: QuestBadge | null): Promise<void> => {
  const root = getRequiredElement<HTMLDivElement>(QUEST_CONSTANTS.dom.rootId);
  const overlay = document.createElement("section");
  overlay.className = "questOverlay";
  overlay.setAttribute("role", "dialog");
  overlay.setAttribute("aria-label", UI_TEXT.arcCompleteHeading);
  const genreLabel = badge?.genre?.trim() || sessionState.genre;
  overlay.innerHTML = `
    <div class="questOverlayCard">
      <div class="questBadgeStar">${escapeHtml(genreLabel)}</div>
      <h2>${escapeHtml(UI_TEXT.arcCompleteHeading)}</h2>
      <p>${escapeHtml(`${UI_TEXT.arcCompleteBadgePrefix}: ${genreLabel}`)}</p>
      <div class="questOverlayActions">
        <button type="button" id="quest-new-arc">${escapeHtml(UI_TEXT.arcCompleteNewQuest)}</button>
        <a href="/progress.html" id="quest-view-progress">${escapeHtml(UI_TEXT.arcCompleteViewProgress)}</a>
      </div>
    </div>
  `;
  root.appendChild(overlay);
  rewardEngine.trigger("chapter_complete");

  await new Promise<void>((resolve) => {
    const newQuestButton = getRequiredElement<HTMLButtonElement>("quest-new-arc");
    newQuestButton.addEventListener("click", async () => {
      overlay.remove();
      await initializeSession();
      resolve();
    });
  });
};

const askBrainCheckChoice = async (): Promise<BrainCheckChoice> => {
  const taskElement = getRequiredElement<HTMLElement>(QUEST_CONSTANTS.dom.taskId);
  const focusedButtonId = "brain-focused";
  const frustratedButtonId = "brain-frustrated";
  taskElement.innerHTML = `
    <h3>${escapeHtml(UI_TEXT.brainCheckPrompt)}</h3>
    <button type="button" id="${focusedButtonId}">${escapeHtml(UI_TEXT.brainCheckFocused)}</button>
    <button type="button" id="${frustratedButtonId}">${escapeHtml(UI_TEXT.brainCheckFrustrated)}</button>
  `;

  return new Promise<BrainCheckChoice>((resolve) => {
    const focusedButton =
      getRequiredElement<HTMLButtonElement>(focusedButtonId);
    const frustratedButton =
      getRequiredElement<HTMLButtonElement>(frustratedButtonId);
    focusedButton.addEventListener("click", () => resolve("focused"));
    frustratedButton.addEventListener("click", () => resolve("frustrated"));
  });
};

const runChapterQuestions = async (chapter: ChapterResponse): Promise<void> => {
  sessionState.activePhase = "question";
  setStatus(UI_TEXT.statusQuestion);

  for (
    sessionState.questionIndex = 0;
    sessionState.questionIndex < chapter.questions.length;
    sessionState.questionIndex += 1
  ) {
    const question: Question = chapter.questions[sessionState.questionIndex];
    let isCorrect = false;

    while (!isCorrect) {
      const prompt = await renderTextPrompt({
        title: `${UI_TEXT.questionLabel} ${sessionState.questionIndex + 1}`,
        prompt: question.q,
        placeholder: UI_TEXT.answerPlaceholder,
        hintText: question.hint,
      });
      trackUserInteraction();
      const safeAnswer = validateFreeText(
        prompt.value,
        QUEST_CONSTANTS.input.maxAnswerLength,
      );

      const result = await submitAnswer(
        sessionState.chapterId,
        sessionState.questionIndex,
        safeAnswer,
      );

      if (prompt.usedHint || result.hint.length > 0 || result.revealed) {
        incrementProgress(QUEST_CONSTANTS.progress.hintIncrement);
      }

      if (!result.correct) {
        setFeedback(
          result.hint.length > 0
            ? `${UI_TEXT.responseIncorrect} ${result.hint}`
            : UI_TEXT.responseIncorrect,
        );
        continue;
      }

      rewardEngine.trigger("correct");
      incrementProgress(QUEST_CONSTANTS.progress.correctIncrement);
      setFeedback(UI_TEXT.responseCorrect);
      await markTaskCompleted();
      isCorrect = true;
    }
  }
};

const runSpellingTask = async (): Promise<void> => {
  sessionState.activePhase = "spelling";
  setStatus(UI_TEXT.statusSpelling);

  const word = await getSpellingWord(sessionState.learnerId);
  let solved = false;

  while (!solved) {
    const prompt = await renderTextPrompt({
      title: UI_TEXT.statusSpelling,
      prompt: `${UI_TEXT.spellingPromptPrefix} ${word.word}`,
      placeholder: UI_TEXT.spellingAnswerPlaceholder,
      hintText: word.word,
    });
    trackUserInteraction();
    const safeAnswer = validateFreeText(
      prompt.value,
      QUEST_CONSTANTS.input.maxWordLength,
    );

    const result = await checkSpelling(sessionState.learnerId, word.id, safeAnswer);
    if (prompt.usedHint || Boolean(result.hint) || result.reveal) {
      incrementProgress(QUEST_CONSTANTS.progress.hintIncrement);
    }

    if (!result.correct) {
      const hintText = result.hint ?? "";
      setFeedback(
        hintText.length > 0
          ? `${UI_TEXT.responseIncorrect} ${hintText}`
          : UI_TEXT.responseIncorrect,
      );
      continue;
    }

    rewardEngine.trigger("correct");
    incrementProgress(QUEST_CONSTANTS.progress.correctIncrement);
    setFeedback(UI_TEXT.spellingCorrect);
    await markTaskCompleted();
    solved = true;
  }
};

const parseWholeNumber = (value: string): number => {
  const trimmed = value.trim();
  if (!/^-?\d+$/.test(trimmed)) {
    throw new Error(UI_TEXT.validationErrorNumber);
  }
  const parsed = Number.parseInt(trimmed, 10);
  if (!Number.isInteger(parsed)) {
    throw new Error(UI_TEXT.validationErrorNumber);
  }
  return parsed;
};

const runMathTask = async (): Promise<void> => {
  sessionState.activePhase = "math";
  setStatus(UI_TEXT.statusMath);

  const problem = await getMathProblem(sessionState.learnerId, sessionState.difficulty);
  let solved = false;

  while (!solved) {
    const prompt = await renderTextPrompt({
      title: UI_TEXT.statusMath,
      prompt: `${UI_TEXT.mathPromptPrefix} ${problem.question}`,
      placeholder: UI_TEXT.mathAnswerPlaceholder,
      hintText: problem.hint1,
    });
    trackUserInteraction();
    const numericAnswer = parseWholeNumber(prompt.value);

    const result = await checkMath(
      sessionState.learnerId,
      sessionState.sessionId,
      problem.id,
      numericAnswer,
    );
    if (prompt.usedHint || Boolean(result.hint) || result.reveal) {
      incrementProgress(QUEST_CONSTANTS.progress.hintIncrement);
    }

    if (!result.correct) {
      const hintText = result.reveal_answer ?? result.hint ?? "";
      setFeedback(
        hintText.length > 0
          ? `${UI_TEXT.responseIncorrect} ${hintText}`
          : UI_TEXT.responseIncorrect,
      );
      continue;
    }

    rewardEngine.trigger("correct");
    incrementProgress(QUEST_CONSTANTS.progress.correctIncrement);
    setFeedback(UI_TEXT.mathCorrect);
    await markTaskCompleted();
    solved = true;
  }
};

const parseGenreFromQuery = (): string => {
  const params = new URLSearchParams(window.location.search);
  const genreParam = params.get("genre");
  if (!genreParam) {
    return QUEST_CONSTANTS.genre.defaultValue;
  }
  const safeGenre = genreParam.trim();
  if (safeGenre.length === 0 || safeGenre.length > QUEST_CONSTANTS.genre.maxLength) {
    return QUEST_CONSTANTS.genre.defaultValue;
  }
  return safeGenre;
};

const initializeSession = async (): Promise<void> => {
  sessionState.genre = parseGenreFromQuery();
  const arc = await startArc(sessionState.genre);
  sessionState.learnerId = arc.learner_id;
  sessionState.arcId = arc.arc_id;
  sessionState.sessionId = arc.session_id;
  sessionState.lastActivityMs = Date.now();
  setStatus(UI_TEXT.statusReady);
};

const formatErrorMessage = (error: unknown): string => {
  if (error instanceof ApiError) {
    return `${UI_TEXT.statusErrorPrefix} ${error.message}`;
  }
  if (error instanceof Error) {
    return `${UI_TEXT.statusErrorPrefix} ${error.message}`;
  }
  return `${UI_TEXT.statusErrorPrefix} ${UI_TEXT.apiErrorDefault}`;
};

const runQuestLoop = async (): Promise<void> => {
  renderShell();
  renderProgress();

  document.addEventListener("keydown", () => {
    trackUserInteraction();
  });

  await initializeSession();

  while (true) {
    sessionState.activePhase = "chapter";
    const chapter = await getChapter(sessionState.learnerId);
    sessionState.chapterId = chapter.chapter_id;
    sessionState.questionIndex = 0;
    sessionState.chapterIndex += 1;
    renderChapter(chapter);

    await runChapterQuestions(chapter);
    if (chapter.arc_complete) {
      await showArcCompletionOverlay(chapter.badge);
      setStatus(UI_TEXT.statusReady);
      setFeedback("");
      continue;
    }
    await runSpellingTask();
    await showWritingPromptIfNeeded();
    await runMathTask();
  }
};

void runQuestLoop().catch((error: unknown) => {
  const message = formatErrorMessage(error);
  setStatus(message);
  setFeedback(message);
});
