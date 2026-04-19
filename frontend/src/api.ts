export interface ArcStarted {
  arc_id: string;
  genre: string;
  learner_id: string;
  session_id: string;
}

export interface Question {
  q: string;
  hint: string;
}

export interface QuestBadge {
  id: string;
  arc_id: string;
  learner_id: string;
  genre: string;
  earned_at: string;
}

export interface ChapterResponse {
  chapter_id: string;
  content: string;
  questions: Question[];
  arc_complete: boolean;
  badge: QuestBadge | null;
}

export interface AnswerResult {
  correct: boolean;
  hint: string;
  revealed: boolean;
  next_phase: "chapter" | "question" | "spelling" | "math";
}

export interface SpellingWord {
  id: string;
  learner_id: string;
  word: string;
  correct_count: number;
  hint_count: number;
  last_seen_at?: string;
  mastered_at?: string;
}

export interface SpellingResult {
  correct: boolean;
  mastered: boolean;
  hint?: string;
  hints_used?: number;
  reveal: boolean;
}

export interface MathProblem {
  id: string;
  narrative_hook: string;
  question: string;
  hint1: string;
  hint2: string;
  genre: string;
  difficulty: "easy" | "medium" | "hard";
}

export interface MathResult {
  correct: boolean;
  hint?: string;
  reveal: boolean;
  reveal_answer?: string;
}

export interface BrainCheckResult {
  acknowledged: boolean;
  next_difficulty: "normal" | "easier";
}

export interface WritingLogResult {
  saved: boolean;
  word_count: number;
}

export interface ProgressSessionDay {
  date: string;
  tasks_completed: number;
  engagement_seconds: number;
}

export interface ProgressSpellingWord {
  word: string;
  mastered: boolean;
  last_seen_at?: string;
}

export interface ProgressSpellingData {
  total: number;
  mastered: number;
  words: ProgressSpellingWord[];
}

export interface ProgressMathByType {
  problem_type: string;
  attempts: number;
  correct_rate: number;
}

export interface ProgressMathData {
  total_attempts: number;
  correct_rate: number;
  by_problem_type: ProgressMathByType[];
}

export interface ProgressQuestArc {
  arc_id: string;
  title: string;
  status: string;
  chapters_completed: number;
}

export interface ProgressBadge {
  badge_code: string;
  earned_at: string;
}

export interface ProgressBrainCheck {
  recorded_at: string;
  response: string;
  emoji: string;
}

export interface ProgressData {
  learner_id: string;
  sessions: ProgressSessionDay[];
  spelling: ProgressSpellingData;
  math: ProgressMathData;
  quest_arcs: ProgressQuestArc[];
  quest_badges: ProgressBadge[];
  weekly_words_typed: number;
  brain_checks: ProgressBrainCheck[];
}

interface ApiErrorBody {
  error?: string;
}

type JsonRecord = Record<string, unknown>;

const API_CONSTANTS = {
  contentTypeJson: "application/json",
  methods: {
    get: "GET",
    post: "POST",
  },
  paths: {
    startArc: "/api/story/arc/start",
    getChapter: "/api/story/chapter",
    submitAnswer: "/api/story/answer",
    getSpellingWord: "/api/spelling/word",
    checkSpelling: "/api/spelling/check",
    getMathProblem: "/api/math/problem",
    checkMath: "/api/math/check",
    submitBrainCheck: "/api/story/brain-check",
    logWriting: "/api/story/writing-log",
    getProgress: "/api/progress",
  },
  limits: {
    idMaxLength: 128,
    genreMaxLength: 64,
    answerMaxLength: 500,
    brainCheckResponseMaxLength: 64,
    mathSessionIdMaxLength: 128,
    mathProblemIdMaxLength: 64,
    writingMinLength: 20,
    writingMaxLength: 20000,
  },
} as const;

export class ApiError extends Error {
  readonly status: number;
  readonly path: string;
  readonly details?: unknown;

  constructor(params: {
    message: string;
    status: number;
    path: string;
    details?: unknown;
  }) {
    super(params.message);
    this.name = "ApiError";
    this.status = params.status;
    this.path = params.path;
    this.details = params.details;
  }
}

const isRecord = (value: unknown): value is JsonRecord =>
  typeof value === "object" && value !== null;

const isString = (value: unknown): value is string => typeof value === "string";

const isBoolean = (value: unknown): value is boolean =>
  typeof value === "boolean";

const isNumber = (value: unknown): value is number =>
  typeof value === "number" && Number.isFinite(value);

const validateTextInput = (
  value: string,
  fieldName: string,
  minLength: number,
  maxLength: number,
): string => {
  if (!isString(value)) {
    throw new ApiError({
      message: `${fieldName} must be a string`,
      status: 400,
      path: fieldName,
    });
  }

  const trimmedValue = value.trim();
  if (trimmedValue.length < minLength) {
    throw new ApiError({
      message: `${fieldName} is too short`,
      status: 400,
      path: fieldName,
    });
  }
  if (trimmedValue.length > maxLength) {
    throw new ApiError({
      message: `${fieldName} is too long`,
      status: 400,
      path: fieldName,
    });
  }
  return trimmedValue;
};

const validateUuidLike = (value: string, fieldName: string): string => {
  const trimmedValue = validateTextInput(
    value,
    fieldName,
    1,
    API_CONSTANTS.limits.idMaxLength,
  );
  const uuidLikePattern = /^[0-9a-fA-F-]{8,128}$/;
  if (!uuidLikePattern.test(trimmedValue)) {
    throw new ApiError({
      message: `${fieldName} has invalid format`,
      status: 400,
      path: fieldName,
    });
  }
  return trimmedValue;
};

const parseJsonSafe = async (response: Response): Promise<unknown> => {
  const rawText = await response.text();
  if (rawText.trim().length === 0) {
    return {};
  }
  try {
    return JSON.parse(rawText) as unknown;
  } catch {
    throw new ApiError({
      message: "Response is not valid JSON",
      status: response.status,
      path: response.url,
      details: rawText,
    });
  }
};

const getErrorMessage = (body: unknown, fallback: string): string => {
  if (!isRecord(body)) {
    return fallback;
  }
  const errorBody = body as ApiErrorBody;
  if (!isString(errorBody.error) || errorBody.error.trim().length === 0) {
    return fallback;
  }
  return errorBody.error;
};

const requestJson = async <T>(
  path: string,
  init: RequestInit,
  assertBody: (value: unknown) => value is T,
): Promise<T> => {
  const response = await fetch(path, init);
  const body = await parseJsonSafe(response);

  if (!response.ok) {
    const message = getErrorMessage(body, `Request failed with ${response.status}`);
    throw new ApiError({
      message,
      status: response.status,
      path,
      details: body,
    });
  }

  if (!assertBody(body)) {
    throw new ApiError({
      message: "Response JSON shape is invalid",
      status: response.status,
      path,
      details: body,
    });
  }
  return body;
};

const isQuestion = (value: unknown): value is Question => {
  if (!isRecord(value)) {
    return false;
  }
  return isString(value.q) && isString(value.hint);
};

const isQuestBadge = (value: unknown): value is QuestBadge => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.id) &&
    isString(value.arc_id) &&
    isString(value.learner_id) &&
    isString(value.genre) &&
    isString(value.earned_at)
  );
};

const isArcStarted = (value: unknown): value is ArcStarted => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.arc_id) &&
    isString(value.genre) &&
    isString(value.learner_id) &&
    isString(value.session_id)
  );
};

const isChapterResponse = (value: unknown): value is ChapterResponse => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.chapter_id) &&
    isString(value.content) &&
    Array.isArray(value.questions) &&
    value.questions.every(isQuestion) &&
    isBoolean(value.arc_complete) &&
    (value.badge === null || isQuestBadge(value.badge))
  );
};

const isAnswerResult = (value: unknown): value is AnswerResult => {
  if (!isRecord(value)) {
    return false;
  }
  const nextPhase = value.next_phase;
  return (
    isBoolean(value.correct) &&
    isString(value.hint) &&
    isBoolean(value.revealed) &&
    (nextPhase === "chapter" ||
      nextPhase === "question" ||
      nextPhase === "spelling" ||
      nextPhase === "math")
  );
};

const isSpellingWord = (value: unknown): value is SpellingWord => {
  if (!isRecord(value)) {
    return false;
  }
  const hasOptionalLastSeen =
    value.last_seen_at === undefined || isString(value.last_seen_at);
  const hasOptionalMastered =
    value.mastered_at === undefined || isString(value.mastered_at);
  return (
    isString(value.id) &&
    isString(value.learner_id) &&
    isString(value.word) &&
    isNumber(value.correct_count) &&
    isNumber(value.hint_count) &&
    hasOptionalLastSeen &&
    hasOptionalMastered
  );
};

const isSpellingResult = (value: unknown): value is SpellingResult => {
  if (!isRecord(value)) {
    return false;
  }
  const hasOptionalHint = value.hint === undefined || isString(value.hint);
  const hasOptionalHintsUsed =
    value.hints_used === undefined || isNumber(value.hints_used);
  return (
    isBoolean(value.correct) &&
    isBoolean(value.mastered) &&
    isBoolean(value.reveal) &&
    hasOptionalHint &&
    hasOptionalHintsUsed
  );
};

const isMathProblem = (value: unknown): value is MathProblem => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.id) &&
    isString(value.narrative_hook) &&
    isString(value.question) &&
    isString(value.hint1) &&
    isString(value.hint2) &&
    isString(value.genre) &&
    (value.difficulty === "easy" ||
      value.difficulty === "medium" ||
      value.difficulty === "hard")
  );
};

const isMathResult = (value: unknown): value is MathResult => {
  if (!isRecord(value)) {
    return false;
  }
  const hasOptionalHint = value.hint === undefined || isString(value.hint);
  const hasOptionalRevealAnswer =
    value.reveal_answer === undefined || isString(value.reveal_answer);
  return (
    isBoolean(value.correct) &&
    isBoolean(value.reveal) &&
    hasOptionalHint &&
    hasOptionalRevealAnswer
  );
};

const isBrainCheckResult = (value: unknown): value is BrainCheckResult => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isBoolean(value.acknowledged) &&
    (value.next_difficulty === "normal" || value.next_difficulty === "easier")
  );
};

const isWritingLogResult = (value: unknown): value is WritingLogResult => {
  if (!isRecord(value)) {
    return false;
  }
  return isBoolean(value.saved) && isNumber(value.word_count);
};

const isProgressSessionDay = (value: unknown): value is ProgressSessionDay => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.date) &&
    isNumber(value.tasks_completed) &&
    isNumber(value.engagement_seconds)
  );
};

const isProgressSpellingWord = (value: unknown): value is ProgressSpellingWord => {
  if (!isRecord(value)) {
    return false;
  }
  const optionalLastSeen =
    value.last_seen_at === undefined || isString(value.last_seen_at);
  return isString(value.word) && isBoolean(value.mastered) && optionalLastSeen;
};

const isProgressSpellingData = (value: unknown): value is ProgressSpellingData => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isNumber(value.total) &&
    isNumber(value.mastered) &&
    Array.isArray(value.words) &&
    value.words.every(isProgressSpellingWord)
  );
};

const isProgressMathByType = (value: unknown): value is ProgressMathByType => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.problem_type) &&
    isNumber(value.attempts) &&
    isNumber(value.correct_rate)
  );
};

const isProgressMathData = (value: unknown): value is ProgressMathData => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isNumber(value.total_attempts) &&
    isNumber(value.correct_rate) &&
    Array.isArray(value.by_problem_type) &&
    value.by_problem_type.every(isProgressMathByType)
  );
};

const isProgressQuestArc = (value: unknown): value is ProgressQuestArc => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.arc_id) &&
    isString(value.title) &&
    isString(value.status) &&
    isNumber(value.chapters_completed)
  );
};

const isProgressBadge = (value: unknown): value is ProgressBadge => {
  if (!isRecord(value)) {
    return false;
  }
  return isString(value.badge_code) && isString(value.earned_at);
};

const isProgressBrainCheck = (value: unknown): value is ProgressBrainCheck => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.recorded_at) &&
    isString(value.response) &&
    isString(value.emoji)
  );
};

const isProgressData = (value: unknown): value is ProgressData => {
  if (!isRecord(value)) {
    return false;
  }
  return (
    isString(value.learner_id) &&
    Array.isArray(value.sessions) &&
    value.sessions.every(isProgressSessionDay) &&
    isProgressSpellingData(value.spelling) &&
    isProgressMathData(value.math) &&
    Array.isArray(value.quest_arcs) &&
    value.quest_arcs.every(isProgressQuestArc) &&
    Array.isArray(value.quest_badges) &&
    value.quest_badges.every(isProgressBadge) &&
    isNumber(value.weekly_words_typed) &&
    Array.isArray(value.brain_checks) &&
    value.brain_checks.every(isProgressBrainCheck)
  );
};

export const startArc = async (genre: string): Promise<ArcStarted> => {
  const safeGenre = validateTextInput(
    genre,
    "genre",
    1,
    API_CONSTANTS.limits.genreMaxLength,
  );

  return requestJson<ArcStarted>(
    API_CONSTANTS.paths.startArc,
    {
      method: API_CONSTANTS.methods.post,
      headers: { "Content-Type": API_CONSTANTS.contentTypeJson },
      body: JSON.stringify({ genre: safeGenre }),
    },
    isArcStarted,
  );
};

export const getChapter = async (learnerId: string): Promise<ChapterResponse> => {
  const safeLearnerId = validateUuidLike(learnerId, "learnerId");
  const params = new URLSearchParams({ learner_id: safeLearnerId });
  const path = `${API_CONSTANTS.paths.getChapter}?${params.toString()}`;

  return requestJson<ChapterResponse>(
    path,
    { method: API_CONSTANTS.methods.get },
    isChapterResponse,
  );
};

export const submitAnswer = async (
  chapterId: string,
  qIdx: number,
  answer: string,
): Promise<AnswerResult> => {
  const safeChapterId = validateUuidLike(chapterId, "chapterId");
  if (!Number.isInteger(qIdx) || qIdx < 0) {
    throw new ApiError({
      message: "qIdx must be a non-negative integer",
      status: 400,
      path: "qIdx",
    });
  }
  const safeAnswer = validateTextInput(
    answer,
    "answer",
    1,
    API_CONSTANTS.limits.answerMaxLength,
  );

  return requestJson<AnswerResult>(
    API_CONSTANTS.paths.submitAnswer,
    {
      method: API_CONSTANTS.methods.post,
      headers: { "Content-Type": API_CONSTANTS.contentTypeJson },
      body: JSON.stringify({
        chapter_id: safeChapterId,
        question_index: qIdx,
        answer: safeAnswer,
      }),
    },
    isAnswerResult,
  );
};

export const getSpellingWord = async (
  learnerId: string,
): Promise<SpellingWord> => {
  void validateUuidLike(learnerId, "learnerId");

  return requestJson<SpellingWord>(
    API_CONSTANTS.paths.getSpellingWord,
    { method: API_CONSTANTS.methods.get },
    isSpellingWord,
  );
};

export const checkSpelling = async (
  learnerId: string,
  wordId: string,
  answer: string,
): Promise<SpellingResult> => {
  void validateUuidLike(learnerId, "learnerId");
  const safeWordId = validateUuidLike(wordId, "wordId");
  const safeAnswer = validateTextInput(
    answer,
    "answer",
    1,
    API_CONSTANTS.limits.answerMaxLength,
  );

  return requestJson<SpellingResult>(
    API_CONSTANTS.paths.checkSpelling,
    {
      method: API_CONSTANTS.methods.post,
      headers: { "Content-Type": API_CONSTANTS.contentTypeJson },
      body: JSON.stringify({
        word_id: safeWordId,
        answer: safeAnswer,
      }),
    },
    isSpellingResult,
  );
};

export const getMathProblem = async (
  learnerId: string,
  difficulty: string,
): Promise<MathProblem> => {
  void validateUuidLike(learnerId, "learnerId");
  const safeDifficulty = validateTextInput(
    difficulty,
    "difficulty",
    1,
    16,
  ).toLowerCase();
  if (
    safeDifficulty !== "easy" &&
    safeDifficulty !== "medium" &&
    safeDifficulty !== "hard"
  ) {
    throw new ApiError({
      message: "difficulty must be easy, medium, or hard",
      status: 400,
      path: "difficulty",
    });
  }

  const params = new URLSearchParams({ difficulty: safeDifficulty });
  const path = `${API_CONSTANTS.paths.getMathProblem}?${params.toString()}`;
  return requestJson<MathProblem>(
    path,
    { method: API_CONSTANTS.methods.get },
    isMathProblem,
  );
};

export const checkMath = async (
  learnerId: string,
  sessionId: string,
  problemId: string,
  answer: number,
): Promise<MathResult> => {
  void validateUuidLike(learnerId, "learnerId");
  const safeSessionId = validateTextInput(
    sessionId,
    "sessionId",
    1,
    API_CONSTANTS.limits.mathSessionIdMaxLength,
  );
  const safeProblemId = validateTextInput(
    problemId,
    "problemId",
    1,
    API_CONSTANTS.limits.mathProblemIdMaxLength,
  );
  if (!isNumber(answer) || !Number.isInteger(answer)) {
    throw new ApiError({
      message: "answer must be an integer",
      status: 400,
      path: "answer",
    });
  }

  return requestJson<MathResult>(
    API_CONSTANTS.paths.checkMath,
    {
      method: API_CONSTANTS.methods.post,
      headers: { "Content-Type": API_CONSTANTS.contentTypeJson },
      body: JSON.stringify({
        session_id: safeSessionId,
        problem_id: safeProblemId,
        answer,
      }),
    },
    isMathResult,
  );
};

export const submitBrainCheck = async (
  learnerId: string,
  response: string,
): Promise<BrainCheckResult> => {
  const safeLearnerId = validateUuidLike(learnerId, "learnerId");
  const safeResponse = validateTextInput(
    response,
    "response",
    1,
    API_CONSTANTS.limits.brainCheckResponseMaxLength,
  );

  return requestJson<BrainCheckResult>(
    API_CONSTANTS.paths.submitBrainCheck,
    {
      method: API_CONSTANTS.methods.post,
      headers: { "Content-Type": API_CONSTANTS.contentTypeJson },
      body: JSON.stringify({
        learner_id: safeLearnerId,
        response: safeResponse,
      }),
    },
    isBrainCheckResult,
  );
};

export const logWriting = async (
  learnerId: string,
  sessionId: string,
  text: string,
): Promise<WritingLogResult> => {
  const safeLearnerId = validateUuidLike(learnerId, "learnerId");
  const safeSessionId = validateUuidLike(sessionId, "sessionId");
  const safeText = validateTextInput(
    text,
    "text",
    API_CONSTANTS.limits.writingMinLength,
    API_CONSTANTS.limits.writingMaxLength,
  );

  return requestJson<WritingLogResult>(
    API_CONSTANTS.paths.logWriting,
    {
      method: API_CONSTANTS.methods.post,
      headers: { "Content-Type": API_CONSTANTS.contentTypeJson },
      body: JSON.stringify({
        learner_id: safeLearnerId,
        session_id: safeSessionId,
        text: safeText,
      }),
    },
    isWritingLogResult,
  );
};

export const getProgress = async (learnerId: string): Promise<ProgressData> => {
  const safeLearnerId = validateUuidLike(learnerId, "learnerId");
  const params = new URLSearchParams({ learner_id: safeLearnerId });
  const path = `${API_CONSTANTS.paths.getProgress}?${params.toString()}`;
  return requestJson<ProgressData>(path, { method: API_CONSTANTS.methods.get }, isProgressData);
};
