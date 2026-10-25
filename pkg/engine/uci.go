package engine

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

// UCIEngine controls a local Stockfish process via the Universal Chess Interface (UCI) protocol.
type UCIEngine struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Scanner
	mu     sync.Mutex
}

// NewUCIEngine launches and initializes the local Stockfish 19 engine.
func NewUCIEngine(binaryPath string) (*UCIEngine, error) {
	absPath, err := filepath.Abs(binaryPath)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	cmd := exec.Command(absPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start stockfish: %w", err)
	}

	eng := &UCIEngine{
		cmd:    cmd,
		stdin:  stdin,
		stdout: bufio.NewScanner(stdoutPipe),
	}

	if err := eng.init(); err != nil {
		_ = eng.Close()
		return nil, err
	}

	return eng, nil
}

func (e *UCIEngine) send(cmd string) error {
	_, err := fmt.Fprintf(e.stdin, "%s\n", cmd)
	return err
}

func (e *UCIEngine) init() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if err := e.send("uci"); err != nil {
		return err
	}

	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if line == "uciok" {
			break
		}
	}

	// Optimize for minimum memory footprint (~16MB total VPS RAM target) and fast Berserk blitz response
	_ = e.send("setoption name Threads value 1")
	_ = e.send("setoption name Hash value 4")
	_ = e.send("setoption name MultiPV value 3")
	_ = e.send("isready")

	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if line == "readyok" {
			break
		}
	}

	return nil
}

// SetSkillLevel sets engine strength (0 to 20, 20 = GM ~3500+ ELO, 5 = ~1500 ELO).
func (e *UCIEngine) SetSkillLevel(skill int) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if skill < 0 {
		skill = 0
	} else if skill > 20 {
		skill = 20
	}
	return e.send(fmt.Sprintf("setoption name Skill Level value %d", skill))
}

// MoveCandidate represents an evaluated candidate line from Stockfish MultiPV.
type MoveCandidate struct {
	Move    string
	ScoreCP int
	IsMate  bool
	MateVal int
	Depth   int
}

// GetBestMoveFromPosition evaluates the position given initialFen (or "startpos") and a space-separated moves string.
func (e *UCIEngine) GetBestMoveFromPosition(initialFen string, moves string, moveTimeMs int, mode string) (bestMove, eval string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if moveTimeMs <= 0 {
		moveTimeMs = 150
	}

	var posCmd string
	if initialFen == "" || initialFen == "startpos" {
		posCmd = "position startpos"
	} else {
		posCmd = fmt.Sprintf("position fen %s", initialFen)
	}

	trimmedMoves := strings.TrimSpace(moves)
	if trimmedMoves != "" {
		posCmd = fmt.Sprintf("%s moves %s", posCmd, trimmedMoves)
	}

	if err := e.send(posCmd); err != nil {
		return "", "", fmt.Errorf("send position: %w", err)
	}

	if err := e.send(fmt.Sprintf("go movetime %d", moveTimeMs)); err != nil {
		return "", "", fmt.Errorf("send go: %w", err)
	}

	candidates := make(map[int]MoveCandidate)
	defaultBestMove := ""
	defaultEval := "0.00"
	defaultDepth := 0

	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if strings.Contains(line, "CRITICAL ERROR") || strings.Contains(line, "Unsupported position") {
			return "", "", fmt.Errorf("stockfish error: %s", line)
		}

		if strings.HasPrefix(line, "info ") && strings.Contains(line, " score ") {
			fields := strings.Fields(line)
			var mpvIdx, depth, scoreCP, mateVal int
			var isMate bool
			var pvMove string

			for i := 0; i < len(fields); i++ {
				if fields[i] == "multipv" && i+1 < len(fields) {
					mpvIdx, _ = strconv.Atoi(fields[i+1])
				}
				if fields[i] == "depth" && i+1 < len(fields) {
					depth, _ = strconv.Atoi(fields[i+1])
				}
				if fields[i] == "score" && i+2 < len(fields) {
					if fields[i+1] == "cp" {
						scoreCP, _ = strconv.Atoi(fields[i+2])
					} else if fields[i+1] == "mate" {
						isMate = true
						mateVal, _ = strconv.Atoi(fields[i+2])
					}
				}
				if fields[i] == "pv" && i+1 < len(fields) {
					pvMove = fields[i+1]
				}
			}

			if mpvIdx == 0 {
				mpvIdx = 1
			}

			if pvMove != "" {
				candidates[mpvIdx] = MoveCandidate{
					Move:    pvMove,
					ScoreCP: scoreCP,
					IsMate:  isMate,
					MateVal: mateVal,
					Depth:   depth,
				}
			}

			if mpvIdx == 1 {
				defaultDepth = depth
				if isMate {
					defaultEval = fmt.Sprintf("M%d", mateVal)
				} else {
					defaultEval = fmt.Sprintf("%+.2f", float64(scoreCP)/100.0)
				}
			}
		}

		if strings.HasPrefix(line, "bestmove ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				defaultBestMove = parts[1]
			}
			break
		}
	}

	c1, hasC1 := candidates[1]
	if !hasC1 && defaultBestMove != "" {
		c1 = MoveCandidate{Move: defaultBestMove, Depth: defaultDepth}
	}

	if c1.Move != "" {
		if c1.IsMate {
			eval = fmt.Sprintf("M%d (depth %d)", c1.MateVal, c1.Depth)
		} else {
			eval = fmt.Sprintf("%+.2f (depth %d)", float64(c1.ScoreCP)/100.0, c1.Depth)
		}
		return c1.Move, eval, nil
	}

	if defaultBestMove != "" {
		eval = fmt.Sprintf("%s (depth %d)", defaultEval, defaultDepth)
		return defaultBestMove, eval, nil
	}

	return "", "", fmt.Errorf("stockfish stdout closed unexpectedly")
}

// GetBestMove evaluates the position and returns bestmove (e.g. "e2e4"), eval string, and error.
func (e *UCIEngine) GetBestMove(fen string, moveTimeMs int) (bestMove, eval string, err error) {
	return e.GetBestMoveWithMode(fen, moveTimeMs, "attack")
}

// GetBestMoveTal evaluates with Mikhail Tal style bias (high risk, tactical complications, sacrifices).
func (e *UCIEngine) GetBestMoveTal(fen string, moveTimeMs int) (bestMove, eval string, err error) {
	return e.GetBestMoveWithMode(fen, moveTimeMs, "tal")
}

// GetBestMoveBerserk evaluates with extreme aggression (altíssimo risco: aceita sacrifício de peça, recusa troca de Damas, caça o Rei).
func (e *UCIEngine) GetBestMoveBerserk(fen string, moveTimeMs int) (bestMove, eval string, err error) {
	return e.GetBestMoveWithMode(fen, moveTimeMs, "berserk")
}

// GetBestMoveKingside evaluates prioritizing Kingside attack and fortress (Bobby Fischer & Garry Kasparov).
func (e *UCIEngine) GetBestMoveKingside(fen string, moveTimeMs int) (bestMove, eval string, err error) {
	return e.GetBestMoveWithMode(fen, moveTimeMs, "kingside")
}

// GetBestMoveWithMode evaluates the position using the specified mode ("attack", "tal", or "solid").
func (e *UCIEngine) GetBestMoveWithMode(fen string, moveTimeMs int, mode string) (bestMove, eval string, err error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if moveTimeMs <= 0 {
		moveTimeMs = 150 // 150ms default for ultra-low latency GM calculation
	}

	if fen == "" {
		fen = "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	}

	if err := e.send(fmt.Sprintf("position fen %s", fen)); err != nil {
		return "", "", fmt.Errorf("send position: %w", err)
	}

	if err := e.send(fmt.Sprintf("go movetime %d", moveTimeMs)); err != nil {
		return "", "", fmt.Errorf("send go: %w", err)
	}

	candidates := make(map[int]MoveCandidate)
	defaultBestMove := ""
	defaultEval := "0.00"
	defaultDepth := 0

	for e.stdout.Scan() {
		line := strings.TrimSpace(e.stdout.Text())
		if strings.Contains(line, "CRITICAL ERROR") || strings.Contains(line, "Unsupported position") {
			return "", "", fmt.Errorf("stockfish rejected fen: %s", line)
		}

		// Parse info lines for multipv score, depth and pv moves
		if strings.HasPrefix(line, "info ") && strings.Contains(line, " score ") {
			fields := strings.Fields(line)
			var mpvIdx, depth, scoreCP, mateVal int
			var isMate bool
			var pvMove string

			for i := 0; i < len(fields); i++ {
				if fields[i] == "multipv" && i+1 < len(fields) {
					mpvIdx, _ = strconv.Atoi(fields[i+1])
				}
				if fields[i] == "depth" && i+1 < len(fields) {
					depth, _ = strconv.Atoi(fields[i+1])
				}
				if fields[i] == "score" && i+2 < len(fields) {
					if fields[i+1] == "cp" {
						scoreCP, _ = strconv.Atoi(fields[i+2])
					} else if fields[i+1] == "mate" {
						isMate = true
						mateVal, _ = strconv.Atoi(fields[i+2])
					}
				}
				if fields[i] == "pv" && i+1 < len(fields) {
					pvMove = fields[i+1]
				}
			}

			if mpvIdx == 0 {
				mpvIdx = 1
			}

			if pvMove != "" {
				candidates[mpvIdx] = MoveCandidate{
					Move:    pvMove,
					ScoreCP: scoreCP,
					IsMate:  isMate,
					MateVal: mateVal,
					Depth:   depth,
				}
			}

			if mpvIdx == 1 {
				defaultDepth = depth
				if isMate {
					defaultEval = fmt.Sprintf("M%d", mateVal)
				} else {
					defaultEval = fmt.Sprintf("%+.2f", float64(scoreCP)/100.0)
				}
			}
		}

		if strings.HasPrefix(line, "bestmove ") {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				defaultBestMove = parts[1]
			}
			break
		}
	}

	c1, hasC1 := candidates[1]
	if !hasC1 && defaultBestMove != "" {
		c1 = MoveCandidate{Move: defaultBestMove, Depth: defaultDepth}
	}

	// Se o modo for "kingside" ou "kia" ou "kid" (Ataque pela Ala do Rei)
	if (mode == "kingside" || mode == "kia" || mode == "kid") && len(candidates) > 1 && !c1.IsMate {
		selected := c1
		bestScore := calculateKingsideScore(fen, c1.Move)

		for i := 2; i <= 4; i++ {
			c, ok := candidates[i]
			if !ok || len(c.Move) < 4 {
				continue
			}

			if c.IsMate && c.MateVal > 0 {
				selected = c
				break
			}

			// Margem aceitável: até 60 centipeões do melhor lance técnico
			if !c.IsMate && c.ScoreCP >= c1.ScoreCP-60 {
				ks := calculateKingsideScore(fen, c.Move)
				if ks > bestScore+10 {
					selected = c
					bestScore = ks
				}
			}
		}

		if selected.Move != c1.Move {
			evalStr := fmt.Sprintf("%+.2f (depth %d) [👑 Ala do Rei: Fischer/Kasparov]", float64(selected.ScoreCP)/100.0, selected.Depth)
			return selected.Move, evalStr, nil
		}
	}

	// Se o modo for "berserk" ou "ultra" (Altíssimo Risco / Kamikaze)
	if (mode == "berserk" || mode == "ultra") && len(candidates) > 1 && !c1.IsMate {
		selected := c1
		bestAgg := calculateAggression(fen, c1.Move)

		for i := 2; i <= 5; i++ {
			c, ok := candidates[i]
			if !ok || len(c.Move) < 4 {
				continue
			}

			if c.IsMate && c.MateVal > 0 {
				selected = c
				break
			}

			// Margem de risco Berserk: tolera sacrifício de até 1.8 peões (-180cp) por iniciativa e ataque ao Rei
			if !c.IsMate && c.ScoreCP >= c1.ScoreCP-180 {
				agg := calculateAggression(fen, c.Move)

				// Penalidade para troca de Damas: Berserk quer as Damas no tabuleiro para atacar!
				fromP := strings.ToLower(string(getPieceAt(fen, c.Move[0:2])))
				toP := strings.ToLower(string(getPieceAt(fen, c.Move[2:4])))
				if fromP == "q" && toP == "q" {
					agg -= 60
				}

				if agg > bestAgg+10 {
					selected = c
					bestAgg = agg
				}
			}
		}

		if selected.Move != c1.Move {
			evalStr := fmt.Sprintf("%+.2f (depth %d) [💀 Modo Berserk: Altíssimo Risco & Caos]", float64(selected.ScoreCP)/100.0, selected.Depth)
			return selected.Move, evalStr, nil
		}
	}

	// Se o modo for "tal" ou "risk", avalia os candidatos em busca do lance de ataque/risco moderado
	if (mode == "tal" || mode == "risk") && len(candidates) > 1 && !c1.IsMate {
		selected := c1
		bestAgg := calculateAggression(fen, c1.Move)

		for i := 2; i <= 3; i++ {
			c, ok := candidates[i]
			if !ok || len(c.Move) < 4 {
				continue
			}

			if c.IsMate && c.MateVal > 0 {
				selected = c
				break
			}

			// Margem de risco tolerada: até 50 centipeões do melhor lance frio
			if !c.IsMate && c.ScoreCP >= c1.ScoreCP-50 {
				agg := calculateAggression(fen, c.Move)
				if agg > bestAgg+12 {
					selected = c
					bestAgg = agg
				}
			}
		}

		if selected.Move != c1.Move {
			evalStr := fmt.Sprintf("%+.2f (depth %d) [🔥 Modo Mikhail Tal: Risco & Ataque]", float64(selected.ScoreCP)/100.0, selected.Depth)
			return selected.Move, evalStr, nil
		}
	}

	if c1.Move != "" {
		if c1.IsMate {
			eval = fmt.Sprintf("M%d (depth %d)", c1.MateVal, c1.Depth)
		} else {
			eval = fmt.Sprintf("%+.2f (depth %d)", float64(c1.ScoreCP)/100.0, c1.Depth)
		}
		return c1.Move, eval, nil
	}

	if defaultBestMove != "" {
		eval = fmt.Sprintf("%s (depth %d)", defaultEval, defaultDepth)
		return defaultBestMove, eval, nil
	}

	return "", "", fmt.Errorf("stockfish stdout closed unexpectedly")
}

func calculateAggression(fen string, move string) int {
	if len(move) < 4 {
		return 0
	}
	from := move[0:2]
	to := move[2:4]
	score := 0

	piece := getPieceAt(fen, from)
	target := getPieceAt(fen, to)
	isWhite := piece >= 'A' && piece <= 'Z'

	// 1. Captura com ganho tático
	if target != '.' {
		score += 35
		pLower := strings.ToLower(string(piece))
		tLower := strings.ToLower(string(target))
		if pLower == "p" && tLower != "p" {
			score += 30
		}
	}

	// 2. Avanço para território inimigo
	toRow := int(to[1] - '0')
	if isWhite {
		if toRow >= 5 {
			score += 15
		}
		if toRow >= 6 {
			score += 25
		}
	} else {
		if toRow <= 4 {
			score += 15
		}
		if toRow <= 3 {
			score += 25
		}
	}

	// 3. Casas críticas de infiltração e ataque ao Rei
	critSquares := map[string]int{
		"f7": 45, "f2": 45,
		"g7": 35, "g2": 35,
		"h7": 30, "h2": 30,
		"e6": 30, "e3": 30,
		"d5": 20, "e5": 20, "d4": 20, "e4": 20,
		"c4": 15, "c5": 15, "f4": 30, "f5": 30, "g4": 30, "g5": 30, "h4": 20, "h5": 20,
		"b4": 25, "b5": 25,
	}
	if pts, ok := critSquares[to]; ok {
		score += pts
	}

	// 4. Peças ativas em ataque / sacrifício
	pLower := strings.ToLower(string(piece))
	if pLower == "n" || pLower == "b" {
		score += 20 // Salto de peça menor em direção ao Rei
	} else if pLower == "q" {
		score += 25 // Invasão de Dama
	} else if pLower == "p" {
		// Rupturas de gambito clássicas
		if to == "b4" || to == "f4" || to == "c4" || to == "d4" || to == "e4" || to == "g4" ||
			to == "b5" || to == "f5" || to == "c5" || to == "d5" || to == "e5" || to == "g5" {
			score += 30
		}
	}

	return score
}

func calculateKingsideScore(fen string, move string) int {
	if len(move) < 4 {
		return 0
	}
	from := move[0:2]
	to := move[2:4]
	score := 0

	// 1. Roque pequeno (Rei 100% blindado na Ala do Rei)
	if (from == "e1" && to == "g1") || (from == "e8" && to == "g8") {
		return 65
	}

	piece := getPieceAt(fen, from)
	target := getPieceAt(fen, to)
	isWhite := piece >= 'A' && piece <= 'Z'

	// 2. Destino nas colunas da Ala do Rei (f, g, h)
	toFile := to[0]
	if toFile == 'f' || toFile == 'g' || toFile == 'h' {
		score += 30
	} else if toFile == 'e' {
		score += 15
	}

	// 3. Casas de ataque decisivas na Ala do Rei
	critKingside := map[string]int{
		"f7": 45, "g7": 45, "h7": 45,
		"f2": 45, "g2": 45, "h2": 45,
		"f6": 35, "g6": 35, "h6": 35,
		"f3": 35, "g3": 35, "h3": 35,
		"f4": 30, "g4": 30, "h4": 30,
		"f5": 30, "g5": 30, "h5": 30,
	}
	if pts, ok := critKingside[to]; ok {
		score += pts
	}

	// 4. Capturas
	if target != '.' {
		score += 25
	}

	// 5. Avanço de peões na Ala do Rei
	pLower := strings.ToLower(string(piece))
	if pLower == "p" && (toFile == 'f' || toFile == 'g' || toFile == 'h') {
		score += 35
	}

	// 6. Avanço para território inimigo
	toRow := int(to[1] - '0')
	if isWhite && toRow >= 5 {
		score += 15
	} else if !isWhite && toRow <= 4 {
		score += 15
	}

	return score
}

// Close gracefully terminates the Stockfish engine.
func (e *UCIEngine) Close() error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_ = e.send("quit")
	time.Sleep(50 * time.Millisecond)
	if e.cmd != nil && e.cmd.Process != nil {
		_ = e.cmd.Process.Kill()
	}
	return nil
}
