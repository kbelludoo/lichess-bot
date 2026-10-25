package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/kbelludoo/chess-mbvmdb/pkg/chess"
	"github.com/kbelludoo/chess-mbvmdb/pkg/engine"
	"github.com/kbelludoo/mbvm-db/pkg/mbvmdb"
)

type MoveRequest struct {
	FEN        string   `json:"fen"`
	Moves      []string `json:"moves"`
	Turn       string   `json:"turn"`
	PlayingAs  string   `json:"playing_as"`
	LegalMoves []string `json:"legal_moves"`
	Mode       string   `json:"mode"`
}

type MoveResponse struct {
	BestMove     string  `json:"best_move"`
	From         string  `json:"from"`
	To           string  `json:"to"`
	SAN          string  `json:"san"`
	Source       string  `json:"source"`
	Explanation  string  `json:"explanation"`
	TacticalPlan string  `json:"tactical_plan"`
	Threat       string  `json:"threat"`
	EvalScore    float64 `json:"eval_score"`
	HumanDelay   float64 `json:"human_delay"`
}

var (
	db          *mbvmdb.DB
	localEngine *engine.UCIEngine
	clientHTTP  = &http.Client{Timeout: 4 * time.Second}
)

func main() {
	dbPath := filepath.Join("..", "..", "chess.mbvmdb")
	if _, err := os.Stat("chess.mbvmdb"); err == nil {
		dbPath = "chess.mbvmdb"
	}

	var err error
	db, err = mbvmdb.LoadFromFile(dbPath)
	if err != nil {
		log.Printf("⚠️ Aviso: não foi possível carregar %s (%v). Criando banco em memória...", dbPath, err)
		db = mbvmdb.New()
	} else {
		log.Printf("✅ MBVM-DB carregado com sucesso (%d símbolos indexados)!", db.TotalSymbols())
	}

	// Inicializar motor Stockfish 19 local offline
	stockfishBin := filepath.Join("bin", "stockfish.exe")
	if _, err := os.Stat(stockfishBin); err == nil {
		localEngine, err = engine.NewUCIEngine(stockfishBin)
		if err == nil {
			log.Printf("🚀 Stockfish 19 Nativo Local INICIALIZADO com Sucesso (NNUE Ativo, Latência Zero)!")
			defer localEngine.Close()
		} else {
			log.Printf("⚠️ Aviso ao iniciar Stockfish local: %v (usando Cloud fallback)", err)
		}
	} else {
		log.Printf("⚠️ Stockfish binário não encontrado em %s (usando Cloud fallback)", stockfishBin)
	}

	http.HandleFunc("/bestmove", handleBestMove)
	http.HandleFunc("/health", handleHealth)
	http.HandleFunc("/script", handleScript)
	http.HandleFunc("/learn", handleLearn)
	http.HandleFunc("/dashboard", handleDashboard)

	port := "8080"
	fmt.Printf(`
===================================================================
⚡ CHESS MBVM-DB LIVE BRIDGE SERVER RODANDO EM http://127.0.0.1:%s
===================================================================
1. Abra no seu navegador: https://www.chess.com/play/computer/Komodo25
2. Pressione F12 para abrir o Console do Desenvolvedor
3. Cole o script disponível em: http://127.0.0.1:%s/script
   (ou use o código fornecido no chat)
===================================================================
Aguardando lances da partida...
`, port, port)

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Erro no servidor: %v", err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"symbols": db.TotalSymbols(),
	})
}

func handleDashboard(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>MBVM-DB Chess Brain Dashboard</title>
  <style>
    body { font-family: system-ui, sans-serif; background: #0e0e13; color: #fff; margin: 0; padding: 24px; }
    .card { background: #16161f; border: 1px solid #232330; border-radius: 12px; padding: 20px; margin-bottom: 20px; box-shadow: 0 4px 20px rgba(0,0,0,0.5); }
    h1, h2 { color: #00ff44; margin-top: 0; }
    .stat-val { font-size: 32px; font-weight: bold; color: #00ff44; }
    .stat-lbl { color: #888; font-size: 13px; text-transform: uppercase; letter-spacing: 1px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 20px; }
    table { width: 100%%; border-collapse: collapse; margin-top: 10px; font-size: 14px; }
    th, td { text-align: left; padding: 10px; border-bottom: 1px solid #232330; }
    th { color: #888; }
    .badge { padding: 3px 8px; border-radius: 4px; font-size: 11px; font-weight: bold; }
    .badge-win { background: rgba(0,255,68,0.15); color: #00ff44; border: 1px solid #00ff44; }
  </style>
</head>
<body>
  <h1>⚡ MBVM-DB Chess Brain Dashboard</h1>
  <p style="color: #aaa;">Status em tempo real do banco de dados em grafo e motor neural local.</p>
  
  <div class="grid">
    <div class="card">
      <div class="stat-lbl">Símbolos no Grafo</div>
      <div class="stat-val">%d</div>
    </div>
    <div class="card">
      <div class="stat-lbl">Motor Local</div>
      <div class="stat-val" style="font-size: 24px; color: #00d0ff;">Stockfish 19 NNUE</div>
    </div>
    <div class="card">
      <div class="stat-lbl">Auto-Aprendizado</div>
      <div class="stat-val" style="font-size: 24px; color: #ffd700;">Ativo (Disco)</div>
    </div>
  </div>

  <div class="card">
    <h2>🧠 Últimas Aberturas e Partidas Memorizadas</h2>
    <table>
      <thead>
        <tr>
          <th>Código</th>
          <th>Nome / Abertura</th>
          <th>Sequência de Lances</th>
          <th>Status</th>
        </tr>
      </thead>
      <tbody>`, db.TotalSymbols())

	symbols := db.Symbols()
	count := 0
	for i := len(symbols) - 1; i >= 0 && count < 15; i-- {
		s := symbols[i]
		if strings.HasPrefix(s.Kind, "chess_") || s.Kind == "learned_move" {
			count++
			firstField := ""
			if len(s.Fields) > 0 {
				firstField = s.Fields[0]
			}
			html += fmt.Sprintf(`<tr>
          <td><span class="badge badge-win">%s</span></td>
          <td><b>%s</b></td>
          <td style="color: #aaa; font-family: monospace;">%s</td>
          <td style="color: #00ff44;">Memorizado</td>
        </tr>`, s.Package, s.Name, firstField)
		}
	}

	html += `</tbody>
    </table>
  </div>
</body>
</html>`
	_, _ = w.Write([]byte(html))
}

type LearnRequest struct {
	Moves    []string `json:"moves"`
	Result   string   `json:"result"`   // "1-0", "0-1", or "win"
	Winner   string   `json:"winner"`   // "w" or "b"
	MyColor  string   `json:"my_color"` // "w" or "b"
	Opponent string   `json:"opponent"`
	Reason   string   `json:"reason"`
}

func handleLearn(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req LearnRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	learnedCount := autoLearnGame(req)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":        "learned",
		"learned_moves": learnedCount,
		"total_symbols": db.TotalSymbols(),
		"message":       fmt.Sprintf("Vitória contra %s memorizada no grafo MBVM-DB!", req.Opponent),
	})
}

func autoLearnGame(req LearnRequest) int {
	if len(req.Moves) == 0 {
		return 0
	}
	isWin := (req.Winner == req.MyColor) ||
		(strings.Contains(req.Result, "1-0") && req.MyColor == "w") ||
		(strings.Contains(req.Result, "0-1") && req.MyColor == "b") ||
		strings.EqualFold(req.Result, "win")

	if !isWin {
		return 0
	}

	// 1. Ingerir como ChessGameItem
	gameItem := mbvmdb.ChessGameItem{
		ID:        fmt.Sprintf("auto_%d", time.Now().Unix()),
		White:     "Você",
		Black:     req.Opponent,
		Result:    req.Result,
		Date:      time.Now().Format("2006.01.02"),
		Moves:     req.Moves,
		TimeClass: "live",
	}
	if req.MyColor == "b" {
		gameItem.White = req.Opponent
		gameItem.Black = "Você"
	}
	db.IngestChessGames([]mbvmdb.ChessGameItem{gameItem})

	// 2. Indexar cada transição da sequência da vitória para busca imediata com 0ms
	learned := 0
	for i := 0; i < len(req.Moves); i++ {
		isOurTurn := (req.MyColor == "w" && i%2 == 0) || (req.MyColor == "b" && i%2 == 1)
		if isOurTurn {
			prefix := strings.Join(req.Moves[:i], "_")
			winningMove := req.Moves[i]
			cand := fmt.Sprintf("%s -> %s", prefix, winningMove)
			title := fmt.Sprintf("Auto-Aprendizado: %s", winningMove)
			if prefix == "" {
				prefix = "start"
			}

			db.IngestChessOpenings([]mbvmdb.ChessOpeningItem{
				{
					ECO:           "LRN",
					Name:          title,
					Moves:         prefix,
					WhiteWin:      100.0,
					Continuations: []string{winningMove, cand},
					KeyThemes:     []string{fmt.Sprintf("Vitória contra %s (%s)", req.Opponent, req.Reason)},
				},
			})
			learned++
		}
	}

	db.CompileChessBytecode()

	// Persistir de volta no arquivo chess.mbvmdb
	dbPath := "chess.mbvmdb"
	if _, err := os.Stat("chess.mbvmdb"); err != nil {
		dbPath = filepath.Join("..", "..", "chess.mbvmdb")
	}
	if err := db.SaveToFile(dbPath); err != nil {
		log.Printf("⚠️ Erro ao salvar auto-aprendizado em %s: %v", dbPath, err)
	} else {
		log.Printf("🧠 [AUTO-APRENDIZADO] %d lances vitoriosos memorizados em %s! Total de símbolos: %d",
			learned, dbPath, db.TotalSymbols())
	}

	return learned
}

func handleScript(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(GetBrowserScript()))
}

func handleBestMove(w http.ResponseWriter, r *http.Request) {
	enableCORS(w)
	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	var req MoveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	resp := calculateBestMove(req)
	resp.TacticalPlan = engine.ExplainTacticalPlan(req.FEN, resp.BestMove, resp.SAN, "", resp.Explanation)

	var lastMove string
	if len(req.Moves) > 0 {
		lastMove = req.Moves[len(req.Moves)-1]
	}
	resp.Threat = engine.DetectOpponentThreat(lastMove, req.Moves)

	isCapture := strings.Contains(resp.SAN, "x")
	isCheck := strings.Contains(resp.SAN, "+")
	resp.HumanDelay = engine.CalculateHumanDelay(len(req.Moves), isCapture, isCheck)
	if req.Mode == "exotic" || req.Mode == "grob" {
		resp.HumanDelay = 0.35 + float64(time.Now().UnixNano()%300)/1000.0 // Turbo blitz 0.35s - 0.65s
	} else if req.Mode == "kingside" || req.Mode == "kia" || req.Mode == "kid" {
		resp.HumanDelay = 0.5 + float64(time.Now().UnixNano()%300)/1000.0 // Ritmo firme 0.5s - 0.8s
	}

	// Extrair avaliação numérica para a Eval Bar
	if strings.Contains(resp.Explanation, "+") || strings.Contains(resp.Explanation, "-") {
		for _, part := range strings.Fields(resp.Explanation) {
			if strings.HasPrefix(part, "+") || strings.HasPrefix(part, "-") {
				var s float64
				_, _ = fmt.Sscanf(part, "%f", &s)
				resp.EvalScore = s
				break
			}
		}
	}

	log.Printf("♟️ Lance Escolhido: %s (%s -> %s) [%s]\n   💡 Plano: %s | ⏱️ Delay: %.2fs",
		resp.SAN, resp.From, resp.To, resp.Source, resp.TacticalPlan, resp.HumanDelay)

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}


func getKingsidePlanMove(req MoveRequest) (MoveResponse, bool) {
	isWhite := (req.PlayingAs == "w" || (req.PlayingAs == "" && req.Turn == "w"))

	if isWhite {
		// Sequência King's Indian Attack (Bobby Fischer)
		switch len(req.Moves) {
		case 0:
			return MoveResponse{
				BestMove: "g1f3", From: "g1", To: "f3", SAN: "Nf3",
				Source: "👑 ATAQUE ALA DO REI (BOBBY FISCHER)",
				Explanation: "King's Indian Attack (1. Nf3): Início da fortaleza e ataque na Ala do Rei!",
			}, true
		case 2:
			return MoveResponse{
				BestMove: "g2g3", From: "g2", To: "g3", SAN: "g3",
				Source: "👑 ATAQUE ALA DO REI (BOBBY FISCHER)",
				Explanation: "Prepara o fianchetto do Bispo guarda-costas em g2!",
			}, true
		case 4:
			return MoveResponse{
				BestMove: "f1g2", From: "f1", To: "g2", SAN: "Bg2",
				Source: "👑 ATAQUE ALA DO REI (BOBBY FISCHER)",
				Explanation: "Bispo em g2 defendendo o Rei com raio-X na diagonal!",
			}, true
		case 6:
			return MoveResponse{
				BestMove: "e1g1", From: "e1", To: "g1", SAN: "O-O",
				Source: "👑 ATAQUE ALA DO REI (BOBBY FISCHER)",
				Explanation: "Roque Pequeno: Rei 100% blindado na fortaleza da direita!",
			}, true
		case 8:
			return MoveResponse{
				BestMove: "d2d3", From: "d2", To: "d3", SAN: "d3",
				Source: "👑 ATAQUE ALA DO REI (BOBBY FISCHER)",
				Explanation: "Apoia o centro e prepara o avanço da marcha de peões e4!",
			}, true
		case 10:
			return MoveResponse{
				BestMove: "e2e4", From: "e2", To: "e4", SAN: "e4",
				Source: "👑 ATAQUE ALA DO REI (BOBBY FISCHER)",
				Explanation: "Ruptura central e4! Tranca o centro e inicia o ataque no flanco do Rei!",
			}, true
		}
	} else {
		// Sequência King's Indian Defence (Garry Kasparov)
		switch len(req.Moves) {
		case 1:
			return MoveResponse{
				BestMove: "g8f6", From: "g8", To: "f6", SAN: "Nf6",
				Source: "👑 DEFESA ALA DO REI (GARRY KASPAROV)",
				Explanation: "King's Indian Defence (1... Nf6): Desenvolve o Cavalo para a Ala do Rei!",
			}, true
		case 3:
			return MoveResponse{
				BestMove: "g7g6", From: "g7", To: "g6", SAN: "g6",
				Source: "👑 DEFESA ALA DO REI (GARRY KASPAROV)",
				Explanation: "Prepara o fianchetto do Bispo na Ala do Rei!",
			}, true
		case 5:
			return MoveResponse{
				BestMove: "f8g7", From: "f8", To: "g7", SAN: "Bg7",
				Source: "👑 DEFESA ALA DO REI (GARRY KASPAROV)",
				Explanation: "Bispo em g7 guarda o Rei preto e mira na grande diagonal!",
			}, true
		case 7:
			return MoveResponse{
				BestMove: "e8g8", From: "e8", To: "g8", SAN: "O-O",
				Source: "👑 DEFESA ALA DO REI (GARRY KASPAROV)",
				Explanation: "Roque Pequeno: Rei preto em segurança absoluta na fortaleza!",
			}, true
		case 9:
			return MoveResponse{
				BestMove: "d7d6", From: "d7", To: "d6", SAN: "d6",
				Source: "👑 DEFESA ALA DO REI (GARRY KASPAROV)",
				Explanation: "Estrutura sólida controlando as casas centrais!",
			}, true
		case 11:
			return MoveResponse{
				BestMove: "e7e5", From: "e7", To: "e5", SAN: "e5",
				Source: "👑 DEFESA ALA DO REI (GARRY KASPAROV)",
				Explanation: "Contra-ataque central e5! Prepara f5 para marchar sobre o Rei branco!",
			}, true
		}
	}

	return MoveResponse{}, false
}

func calculateBestMove(req MoveRequest) MoveResponse {
	if (req.Mode == "kingside" || req.Mode == "kia" || req.Mode == "kid") && len(req.Moves) <= 12 {
		if ksMove, ok := getKingsidePlanMove(req); ok {
			return ksMove
		}
	}
	// 0. Estratégia de Vitória Rápida (Armadilhas Mortais / Mate Relâmpago em poucos lances)
	if trapMove, trapSAN, trapTheme, ok := chess.CheckTrapMove(req.Moves, req.PlayingAs); ok {
		if len(trapMove) >= 4 && isUCI(trapMove[:4]) {
			from := trapMove[0:2]
			to := trapMove[2:4]
			return MoveResponse{
				BestMove:    trapMove,
				From:        from,
				To:          to,
				SAN:         trapSAN,
				Source:      "⚡ ARMADILHA DE VITÓRIA RÁPIDA",
				Explanation: trapTheme,
			}
		}
	}

	// Lance 1 de Brancas (abertura inicial apenas se o FEN for o inicial ou vazio)
	isStartingPos := (req.FEN == "" || strings.HasPrefix(req.FEN, "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR"))
	if isStartingPos && len(req.Moves) == 0 && (req.PlayingAs == "w" || req.Turn == "w" || req.Turn == "") {
		if req.Mode == "exotic" || req.Mode == "grob" {
			return MoveResponse{
				BestMove:    "g2g4",
				From:        "g2",
				To:          "g4",
				SAN:         "g4",
				Source:      "👽 ATAQUE GROB (BIZARRO & CAOS)",
				Explanation: "Abertura Grob (1. g4): Isca bizarra no lance 1 que explode o centro com Bg2 e c4!",
			}
		}
		return MoveResponse{
			BestMove:    "e2e4",
			From:        "e2",
			To:          "e4",
			SAN:         "e4",
			Source:      "MBVM-DB Master Book",
			Explanation: "Abertura do Peão do Rei (1. e4 - Bobby Fischer: 'Best by test')",
		}
	}

	// 1. Consultar banco MBVM-DB para abertura / livro de mestres (apenas correspondência exata de sequência)
	if len(req.Moves) > 0 && len(req.Moves) <= 16 {
		moveKey := strings.Join(req.Moves, "_")
		slice := db.ExtractSlice(moveKey, 1)

		if len(slice.Symbols) > 0 {
			for _, sym := range slice.Symbols {
				if len(sym.OutgoingCalls) > 0 {
					cand := sym.OutgoingCalls[0]
					// Remove prefixes like "e4_c5 -> "
					if parts := strings.Split(cand, "->"); len(parts) > 1 {
						cand = strings.TrimSpace(parts[1])
					}
					cand = cleanSan(cand)
					from, to := parseSanToCoords(cand, req.LegalMoves, req.FEN)
					if from != "" && to != "" && isUCI(from+to) {
						src := "MBVM-DB Master Book"
						explanation := fmt.Sprintf("Abertura Mestre: %s", sym.Name)
						if sym.Package == "LRN" || strings.Contains(sym.Name, "Auto-Aprendizado") {
							src = "🧠 MBVM-DB Auto-Aprendizado"
							explanation = "Lance vencedor memorizado de vitória anterior!"
						}
						return MoveResponse{
							BestMove:    from + to,
							From:        from,
							To:          to,
							SAN:         cand,
							Source:      src,
							Explanation: explanation,
						}
					}
				}
			}
		}
	}

	// 2. Cálculo profundo com Stockfish 19 Local (Zero Latência, 100% Offline, Modo Ataque Total)
	if req.FEN != "" {
		moveTime := 180 // Modo Ataque Total (avaliação profunda para máxima pressão)
		if req.Mode == "solid" {
			moveTime = 120
		}
		if localEngine != nil {
			var sfMove, sfEval string
			var err error
			if req.Mode == "berserk" || req.Mode == "ultra" {
				sfMove, sfEval, err = localEngine.GetBestMoveBerserk(req.FEN, 240)
			} else if req.Mode == "exotic" || req.Mode == "grob" {
				sfMove, sfEval, err = localEngine.GetBestMoveBerserk(req.FEN, 140)
			} else if req.Mode == "kingside" || req.Mode == "kia" || req.Mode == "kid" {
				sfMove, sfEval, err = localEngine.GetBestMoveKingside(req.FEN, 180)
			} else if req.Mode == "tal" || req.Mode == "risk" {
				sfMove, sfEval, err = localEngine.GetBestMoveTal(req.FEN, moveTime)
			} else {
				sfMove, sfEval, err = localEngine.GetBestMoveWithMode(req.FEN, moveTime, req.Mode)
			}
			if err == nil {
				if sfMove == "(none)" || sfMove == "none" {
					return MoveResponse{
						BestMove:    "none",
						From:        "",
						To:          "",
						SAN:         "Fim de Jogo",
						Source:      "Stockfish 19 Local (NNUE)",
						Explanation: "Partida finalizada por Xeque-mate ou Empate",
					}
				}
				if len(sfMove) >= 4 && isUCI(sfMove[:4]) {
					from := sfMove[0:2]
					to := sfMove[2:4]
					sourceLabel := "Stockfish 19 Local (NNUE)"
					if req.Mode == "kingside" || req.Mode == "kia" || req.Mode == "kid" {
						sourceLabel = "👑 ALA DO REI (FISCHER & KASPAROV)"
					} else if req.Mode == "exotic" || req.Mode == "grob" {
						sourceLabel = "👽 EXÓTICO (GROB / CAOS)"
					} else if strings.Contains(sfEval, "Berserk") {
						sourceLabel = "💀 BERSERK (ALTÍSSIMO RISCO)"
					} else if strings.Contains(sfEval, "Mikhail Tal") {
						sourceLabel = "🔥 MIKHAIL TAL (ALTO RISCO)"
					}
					return MoveResponse{
						BestMove:    sfMove,
						From:        from,
						To:          to,
						SAN:         sfMove,
						Source:      sourceLabel,
						Explanation: fmt.Sprintf("Avaliação: %s", sfEval),
					}
				}
			}
		}

		// Fallback se o motor local não estiver disponível
		sfMove, sfEval, err := queryStockfishCloud(req.FEN)
		if err == nil {
			if sfMove == "(none)" || sfMove == "none" {
				return MoveResponse{
					BestMove:    "none",
					From:        "",
					To:          "",
					SAN:         "Fim de Jogo",
					Source:      "Stockfish Deep Engine (Cloud)",
					Explanation: "Partida finalizada por Xeque-mate ou Empate",
				}
			}
			if len(sfMove) >= 4 && isUCI(sfMove[:4]) {
				from := sfMove[0:2]
				to := sfMove[2:4]
				return MoveResponse{
					BestMove:    sfMove,
					From:        from,
					To:          to,
					SAN:         sfMove,
					Source:      "Stockfish Deep Engine (Cloud)",
					Explanation: fmt.Sprintf("Avaliação: %s", sfEval),
				}
			}
		}
	}

	// 3. Fallback: Primeiro lance legal válido
	if len(req.LegalMoves) > 0 {
		lm := req.LegalMoves[0]
		if len(lm) >= 4 {
			return MoveResponse{
				BestMove:    lm,
				From:        lm[0:2],
				To:          lm[2:4],
				SAN:         lm,
				Source:      "Legal Fallback",
				Explanation: "Lance legal de emergência",
			}
		}
	}

	if isStartingPos && (req.PlayingAs == "w" || req.Turn == "w" || req.Turn == "") {
		return MoveResponse{
			BestMove:    "e2e4",
			From:        "e2",
			To:          "e4",
			SAN:         "e4",
			Source:      "Abertura",
			Explanation: "Lance inicial (1. e4)",
		}
	}

	return MoveResponse{
		BestMove:    "",
		From:        "",
		To:          "",
		SAN:         "...",
		Source:      "Aguardando",
		Explanation: "Aguardando estabilização da posição na tela...",
	}
}

func queryStockfishCloud(fen string) (string, string, error) {
	apiURL := fmt.Sprintf("https://stockfish.online/api/s/v2.php?fen=%s&depth=12", url.QueryEscape(fen))
	resp, err := clientHTTP.Get(apiURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	var data struct {
		Success  bool    `json:"success"`
		BestMove string  `json:"bestmove"`
		Eval     float64 `json:"evaluation"`
		Mate     *int    `json:"mate"`
	}
	if err := json.Unmarshal(body, &data); err != nil {
		return "", "", err
	}

	if !data.Success || data.BestMove == "" {
		return "", "", fmt.Errorf("stockfish unsuccessful")
	}

	// Bestmove format: "bestmove e7e5 ponder g1f3" -> extract e7e5
	fields := strings.Fields(data.BestMove)
	move := fields[0]
	if move == "bestmove" && len(fields) > 1 {
		move = fields[1]
	}

	evalStr := fmt.Sprintf("%+.2f", data.Eval)
	if data.Mate != nil {
		evalStr = fmt.Sprintf("Mate em %d", *data.Mate)
	}

	return move, evalStr, nil
}

func isUCI(s string) bool {
	if len(s) < 4 {
		return false
	}
	return s[0] >= 'a' && s[0] <= 'h' && s[1] >= '1' && s[1] <= '8' &&
		s[2] >= 'a' && s[2] <= 'h' && s[3] >= '1' && s[3] <= '8'
}

func parseSanToCoords(san string, legalMoves []string, fen string) (string, string) {
	sanClean := cleanSan(san)
	// Coordenadas UCI estritas (ex: "e2e4", "g7g6")
	if isUCI(sanClean) {
		return sanClean[0:2], sanClean[2:4]
	}

	// Match com lances legais fornecidos pelo Chess.com
	for _, lm := range legalMoves {
		if strings.EqualFold(lm, sanClean) && isUCI(lm) {
			return lm[0:2], lm[2:4]
		}
	}

	if len(legalMoves) > 0 && isUCI(legalMoves[0]) {
		return legalMoves[0][0:2], legalMoves[0][2:4]
	}

	return "", ""
}

func cleanSan(s string) string {
	s = strings.TrimSpace(s)
	s = regexp.MustCompile(`^\d+\.+`).ReplaceAllString(s, "")
	s = strings.TrimSpace(s)
	fields := strings.Fields(s)
	if len(fields) > 0 {
		return fields[0]
	}
	return s
}

func enableCORS(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
}

func GetBrowserScript() string {
	paths := []string{
		filepath.Join("extension", "content.js"),
		filepath.Join("..", "extension", "content.js"),
		"content.js",
	}
	for _, p := range paths {
		if b, err := os.ReadFile(p); err == nil {
			return string(b)
		}
	}
	return `console.error("extension/content.js nao encontrado no disco");`
}
