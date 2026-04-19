# QuestMode Data Flow Diagram

```mermaid
flowchart TD
    questHtml[quest_html] --> questLoop[quest_ts_session_loop]
    questLoop --> apiClient[api_ts_typed_client]
    questLoop --> progressState[quest_session_state]
    progressState --> brainCheckGate[every_5th_task_completion]
    progressState --> breakGate[after_720_engagement_seconds]

    apiClient --> startArc
    apiClient --> getChapter
    apiClient --> submitAnswer
    apiClient --> brainCheck
    apiClient --> spellingWord[GET_api_spelling_word]
    apiClient --> spellingCheck[POST_api_spelling_check]
    apiClient --> mathProblem[GET_api_math_problem]
    apiClient --> mathCheck[POST_api_math_check]

    startArc[POST_api_story_arc_start] --> learnerProfiles[(learner_profiles)]
    startArc --> questArcs[(quest_arcs)]
    startArc --> storyStateRedis[(Redis_story_state)]
    startArc --> questSessions[(quest_sessions)]

    getChapter[GET_api_story_chapter] --> storyStateRedis
    getChapter --> claudeGenerate[Claude_GenerateChapter]
    claudeGenerate -->|"error_or_missing_key"| fallbackChapter[FALLBACK_CHAPTER]
    claudeGenerate --> chapterPayload[Chapter_and_Questions]
    fallbackChapter --> chapterPayload
    chapterPayload --> questChapters[(quest_chapters)]
    chapterPayload --> storyStateRedis

    submitAnswer[POST_api_story_answer] --> questChapters
    submitAnswer --> answerMetaRedis[(Redis_answer_meta)]
    submitAnswer --> storyStateRedis
    submitAnswer --> phaseTransition[Phase_question_to_spelling]

    brainCheck[POST_api_story_brain_check] --> storyStateRedis
    brainCheck --> applyBrainCheck[story_ApplyBrainCheck]
    applyBrainCheck --> storyStateRedis
    brainCheck --> questSessions

    spellingWord --> questSessions
    spellingWord --> spellingMastery[(spelling_mastery)]
    spellingCheck --> spellingMastery

    mathProblem --> mathLibrary[(math_problem_library)]
    mathCheck --> mathAttempts[(math_attempts)]
    mathCheck --> questSessions

    statusGet[GET_api_story_status] --> storyStateRedis
```
