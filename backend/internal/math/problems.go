package math

type Difficulty string

const (
	DiffEasy   Difficulty = "easy"
	DiffMedium Difficulty = "medium"
	DiffHard   Difficulty = "hard"
)

func Downgrade(d Difficulty) Difficulty {
	switch d {
	case DiffHard:
		return DiffMedium
	case DiffMedium:
		return DiffEasy
	default:
		return DiffEasy
	}
}

type MathProblem struct {
	ID            string     `json:"id"`
	NarrativeHook string     `json:"narrative_hook"`
	Question      string     `json:"question"`
	Hint1         string     `json:"hint1"`
	Hint2         string     `json:"hint2"`
	Genre         string     `json:"genre"`
	Answer        int        `json:"answer"`
	Difficulty    Difficulty `json:"difficulty"`
}

type PublicMathProblem struct {
	ID            string     `json:"id"`
	NarrativeHook string     `json:"narrative_hook"`
	Question      string     `json:"question"`
	Hint1         string     `json:"hint1"`
	Hint2         string     `json:"hint2"`
	Genre         string     `json:"genre"`
	Difficulty    Difficulty `json:"difficulty"`
}

func PublicProblem(problem MathProblem) PublicMathProblem {
	return PublicMathProblem{
		ID:            problem.ID,
		NarrativeHook: problem.NarrativeHook,
		Question:      problem.Question,
		Hint1:         problem.Hint1,
		Hint2:         problem.Hint2,
		Genre:         problem.Genre,
		Difficulty:    problem.Difficulty,
	}
}

var ProblemBank = []MathProblem{
	{ID: "adv-001", NarrativeHook: "The bridge has 3 sections of 12 steps each.", Question: "How many steps total?", Hint1: "Multiply sections by steps per section.", Hint2: "Compute 3 x 12.", Genre: "adventure", Answer: 36, Difficulty: DiffMedium},
	{ID: "adv-002", NarrativeHook: "You found 7 coins and then 5 more near the camp.", Question: "How many coins do you have now?", Hint1: "Add the two groups.", Hint2: "Compute 7 + 5.", Genre: "adventure", Answer: 12, Difficulty: DiffEasy},
	{ID: "adv-003", NarrativeHook: "A raft moves 48 miles in 6 hours.", Question: "What is the speed in miles per hour?", Hint1: "Use distance divided by time.", Hint2: "Compute 48 / 6.", Genre: "adventure", Answer: 8, Difficulty: DiffEasy},
	{ID: "adv-004", NarrativeHook: "The map shows 9 caves with 14 crystals each.", Question: "How many crystals are there in all caves?", Hint1: "Multiply cave count by crystals per cave.", Hint2: "Compute 9 x 14.", Genre: "adventure", Answer: 126, Difficulty: DiffHard},
	{ID: "adv-005", NarrativeHook: "You spend 19 arrows and had 45 at dawn.", Question: "How many arrows remain?", Hint1: "Subtract spent arrows from starting arrows.", Hint2: "Compute 45 - 19.", Genre: "adventure", Answer: 26, Difficulty: DiffEasy},
	{ID: "adv-006", NarrativeHook: "A trail has 5 checkpoints, each 18 minutes apart.", Question: "How many minutes for all checkpoint intervals?", Hint1: "Multiply checkpoint intervals by minutes each.", Hint2: "Compute 5 x 18.", Genre: "adventure", Answer: 90, Difficulty: DiffMedium},
	{ID: "adv-007", NarrativeHook: "Three teams carry 27 packs each to base.", Question: "How many packs total?", Hint1: "Multiply teams by packs each.", Hint2: "Compute 3 x 27.", Genre: "adventure", Answer: 81, Difficulty: DiffMedium},
	{ID: "adv-008", NarrativeHook: "The final gate code is the sum of 68 and 47.", Question: "What code opens the gate?", Hint1: "Add both numbers.", Hint2: "Compute 68 + 47.", Genre: "adventure", Answer: 115, Difficulty: DiffHard},
	{ID: "adv-009", NarrativeHook: "You split 96 rations equally among 8 explorers.", Question: "How many rations per explorer?", Hint1: "Divide total by number of explorers.", Hint2: "Compute 96 / 8.", Genre: "adventure", Answer: 12, Difficulty: DiffMedium},
	{ID: "adv-010", NarrativeHook: "A canyon rope is 150 feet. You use 87 feet climbing down.", Question: "How many feet of rope are left?", Hint1: "Subtract used length from total length.", Hint2: "Compute 150 - 87.", Genre: "adventure", Answer: 63, Difficulty: DiffHard},
	{ID: "mys-001", NarrativeHook: "A detective finds 4 clues in each of 6 rooms.", Question: "How many clues were found?", Hint1: "Multiply clues per room by rooms.", Hint2: "Compute 4 x 6.", Genre: "mystery", Answer: 24, Difficulty: DiffEasy},
	{ID: "mys-002", NarrativeHook: "The suspect list had 23 names, and 9 were cleared.", Question: "How many suspects remain?", Hint1: "Subtract cleared names from total names.", Hint2: "Compute 23 - 9.", Genre: "mystery", Answer: 14, Difficulty: DiffEasy},
	{ID: "mys-003", NarrativeHook: "A coded note repeats every 15 letters for 7 cycles.", Question: "How many letters appear in total?", Hint1: "Multiply letters per cycle by cycles.", Hint2: "Compute 15 x 7.", Genre: "mystery", Answer: 105, Difficulty: DiffMedium},
	{ID: "mys-004", NarrativeHook: "Three witnesses each saw the clock 17 minutes apart.", Question: "How many minutes from first to last sighting over 3 gaps?", Hint1: "Multiply number of gaps by minutes per gap.", Hint2: "Compute 3 x 17.", Genre: "mystery", Answer: 51, Difficulty: DiffMedium},
	{ID: "mys-005", NarrativeHook: "A vault has 12 dials, each with 9 symbols.", Question: "How many symbol positions exist overall?", Hint1: "Multiply dials by symbols each.", Hint2: "Compute 12 x 9.", Genre: "mystery", Answer: 108, Difficulty: DiffHard},
	{ID: "mys-006", NarrativeHook: "An alibi timeline spans 180 minutes with events every 20 minutes.", Question: "How many event slots are in the timeline?", Hint1: "Divide total minutes by interval.", Hint2: "Compute 180 / 20.", Genre: "mystery", Answer: 9, Difficulty: DiffMedium},
	{ID: "mys-007", NarrativeHook: "A file drawer has 56 folders and 28 are reviewed.", Question: "How many folders are still unchecked?", Hint1: "Subtract reviewed from total.", Hint2: "Compute 56 - 28.", Genre: "mystery", Answer: 28, Difficulty: DiffEasy},
	{ID: "mys-008", NarrativeHook: "A cipher wheel rotates 13 steps each turn for 8 turns.", Question: "How many steps total?", Hint1: "Multiply steps per turn by number of turns.", Hint2: "Compute 13 x 8.", Genre: "mystery", Answer: 104, Difficulty: DiffHard},
	{ID: "mys-009", NarrativeHook: "The safe opens when 432 is split into 9 equal parts.", Question: "What is each part?", Hint1: "Divide 432 by 9.", Hint2: "Compute 432 / 9.", Genre: "mystery", Answer: 48, Difficulty: DiffHard},
	{ID: "mys-010", NarrativeHook: "A clue score is the sum of 39, 27, and 14.", Question: "What is the final clue score?", Hint1: "Add the three numbers step by step.", Hint2: "Compute 39 + 27 + 14.", Genre: "mystery", Answer: 80, Difficulty: DiffMedium},
	{ID: "fan-001", NarrativeHook: "A wizard brews 5 potions each hour for 4 hours.", Question: "How many potions are brewed?", Hint1: "Multiply potions per hour by hours.", Hint2: "Compute 5 x 4.", Genre: "fantasy", Answer: 20, Difficulty: DiffEasy},
	{ID: "fan-002", NarrativeHook: "A dragon hoard had 70 gems and gave away 18.", Question: "How many gems remain?", Hint1: "Subtract given gems from total gems.", Hint2: "Compute 70 - 18.", Genre: "fantasy", Answer: 52, Difficulty: DiffEasy},
	{ID: "fan-003", NarrativeHook: "A rune circle has 16 runes on each of 6 rings.", Question: "How many runes in all?", Hint1: "Multiply runes per ring by number of rings.", Hint2: "Compute 16 x 6.", Genre: "fantasy", Answer: 96, Difficulty: DiffMedium},
	{ID: "fan-004", NarrativeHook: "The portal opens every 45 minutes during a 6-hour watch.", Question: "How many openings occur?", Hint1: "Convert 6 hours to minutes, then divide by 45.", Hint2: "Compute 360 / 45.", Genre: "fantasy", Answer: 8, Difficulty: DiffHard},
	{ID: "fan-005", NarrativeHook: "A knight rides 14 miles per day for 9 days.", Question: "How many miles total?", Hint1: "Multiply miles per day by days.", Hint2: "Compute 14 x 9.", Genre: "fantasy", Answer: 126, Difficulty: DiffMedium},
	{ID: "fan-006", NarrativeHook: "An enchanted tower has 11 floors with 13 windows each.", Question: "How many windows are there?", Hint1: "Multiply floors by windows per floor.", Hint2: "Compute 11 x 13.", Genre: "fantasy", Answer: 143, Difficulty: DiffHard},
	{ID: "fan-007", NarrativeHook: "A sorcerer divides 84 mana stones among 7 apprentices.", Question: "How many stones per apprentice?", Hint1: "Divide total stones by apprentices.", Hint2: "Compute 84 / 7.", Genre: "fantasy", Answer: 12, Difficulty: DiffMedium},
	{ID: "fan-008", NarrativeHook: "The elf library adds 22 scrolls to 19 existing scrolls.", Question: "How many scrolls are on the shelf?", Hint1: "Add both counts.", Hint2: "Compute 22 + 19.", Genre: "fantasy", Answer: 41, Difficulty: DiffEasy},
	{ID: "fan-009", NarrativeHook: "A giant puzzle requires 250 pieces, and 137 are already placed.", Question: "How many pieces are left?", Hint1: "Subtract placed pieces from total required pieces.", Hint2: "Compute 250 - 137.", Genre: "fantasy", Answer: 113, Difficulty: DiffHard},
	{ID: "fan-010", NarrativeHook: "A phoenix cycle lasts 9 days and repeats 12 times.", Question: "How many days in 12 cycles?", Hint1: "Multiply days per cycle by cycle count.", Hint2: "Compute 9 x 12.", Genre: "fantasy", Answer: 108, Difficulty: DiffMedium},
}
