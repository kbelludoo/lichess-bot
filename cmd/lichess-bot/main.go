package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/kbelludoo/chess-mbvmdb/pkg/chess"
	"github.com/kbelludoo/chess-mbvmdb/pkg/engine"
	"github.com/kbelludoo/mbvm-db/pkg/mbvmdb"
)

type Config struct {
	Token         string `json:"token"`
	EnginePath    string `json:"engine_path"`
	DBPath        string `json:"db_path"`
	Port          int    `json:"port"`
	MoveTimeMs    int    `json:"move_time_ms"`
	SkillLevel    int    `json:"skill_level"`
	Mode          string `json:"mode"`
	AcceptRated   bool   `json:"accept_rated"`
	AcceptCasual  bool   `json:"accept_casual"`
	AutoLearn     bool   `json:"auto_learn"`
	AutoMatch     bool   `json:"auto_match"`
	IdleLearn     bool   `json:"idle_learn"`
	IdleMinRating int    `json:"idle_min_rating"`
}

type BotApp struct {
	cfg         Config
	client      *http.Client
	engine      *engine.UCIEngine
	db          *mbvmdb.DB
	dbMu        sync.Mutex
	idleLearner *chess.IdleLearner
	botID       string
	botName     string
	activeLock  sync.Mutex
	activeGame  map[string]bool
	gameModes   map[string]string
	gameSkills  map[string]int
	modeLock    sync.RWMutex
	startTime   time.Time
	statsLock   sync.RWMutex
	gamesPlayed int
	wins        int
	losses      int
	movesPlayed int
	lastMove    string
	lastEval    string
	lastOpp     string
}

type UserAccount struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Title    string `json:"title"`
}

type StreamEvent struct {
	Type      string         `json:"type"`
	Challenge map[string]any `json:"challenge,omitempty"`
	Game      map[string]any `json:"game,omitempty"`
}

type GameFullEvent struct {
	Type       string         `json:"type"`
	ID         string         `json:"id"`
	InitialFen string         `json:"initialFen"`
	White      map[string]any `json:"white"`
	Black      map[string]any `json:"black"`
	State      GameState      `json:"state"`
}

type GameState struct {
	Type   string `json:"type"`
	Moves  string `json:"moves"`
	WTime  int64  `json:"wtime"`
	BTime  int64  `json:"btime"`
	Status string `json:"status"`
	Winner string `json:"winner"`
}

func main() {
	var (
		flagToken    = flag.String("token", "", "Lichess Personal Access Token (or env LICHESS_TOKEN)")
		flagEngine   = flag.String("engine", "", "Path to Stockfish binary")
		flagDB       = flag.String("db", "chess.mbvmdb", "Path to chess.mbvmdb database")
		flagPort     = flag.Int("port", 3016, "HTTP telemetry port (default 3016)")
		flagTime     = flag.Int("movetime", 250, "Time per move in milliseconds (default 250ms)")
		flagSkill    = flag.Int("skill", 20, "Stockfish skill level 0-20 (default 20)")
		flagMode     = flag.String("mode", "berserk", "Play style: berserk (default), attack, tal, solid")
		flagConfig   = flag.String("config", "bot-config.json", "JSON config file")
	)
	flag.Parse()

	log.Println("==================================================================")
	log.Println("  ♟️  LICHESS BOT 100% GO — MODO BERSERK + MBVM-DB AUTO-LEARN ♟️   ")
	log.Println("==================================================================")

	cfg := loadConfig(*flagConfig, *flagToken, *flagEngine, *flagDB, *flagPort, *flagTime, *flagSkill, *flagMode)
	if cfg.Token == "" {
		log.Fatal("❌ Erro: Token do Lichess não configurado! Use -token ou defina LICHESS_TOKEN.")
	}

	stockfishPath := resolveEnginePath(cfg.EnginePath)
	log.Printf("⚙️ Inicializando engine Stockfish em: %s", stockfishPath)
	eng, err := engine.NewUCIEngine(stockfishPath)
	if err != nil {
		log.Fatalf("❌ Falha ao iniciar Stockfish: %v", err)
	}
	defer eng.Close()

	if err := eng.SetSkillLevel(cfg.SkillLevel); err != nil {
		log.Printf("⚠️ Aviso ao ajustar Skill Level: %v", err)
	}

	// Carrega banco de dados MBVM-DB
	var chessDB *mbvmdb.DB
	if cfg.DBPath != "" {
		resolvedDB := resolveDBPath(cfg.DBPath)
		if loaded, err := mbvmdb.LoadFromFile(resolvedDB); err == nil {
			chessDB = loaded
			log.Printf("🧠 Banco MBVM-DB carregado com sucesso: %s (%d símbolos no grafo)", resolvedDB, chessDB.TotalSymbols())
		} else {
			log.Printf("ℹ️ Criando novo banco MBVM-DB em memória (%s não encontrado: %v)", resolvedDB, err)
			chessDB = mbvmdb.New()
		}
	} else {
		chessDB = mbvmdb.New()
	}

	// Ingestão de todas as lições mestres (Mates, Táticas, Finais de partida)
	lessonsCount := chess.IngestAllLessons(chessDB)
	log.Printf("📚 %d lições mestres ingeridas no MBVM-DB! Grafo expandido para %d símbolos", lessonsCount, chessDB.TotalSymbols())
	if cfg.DBPath != "" {
		_ = chessDB.SaveToFile(cfg.DBPath)
	}

	bot := &BotApp{
		cfg:        cfg,
		client:     &http.Client{Timeout: 0},
		engine:     eng,
		db:         chessDB,
		activeGame: make(map[string]bool),
		gameModes:  make(map[string]string),
		gameSkills: make(map[string]int),
		startTime:  time.Now(),
	}

	account, err := bot.fetchAccount()
	if err != nil {
		log.Fatalf("❌ Falha ao conectar na API do Lichess: %v", err)
	}

	bot.botID = strings.ToLower(account.ID)
	bot.botName = account.Username
	log.Printf("✅ Autenticado como: %s | Título: %s | Modo: 💀 %s", bot.botName, account.Title, strings.ToUpper(cfg.Mode))

	if account.Title != "BOT" {
		log.Println("⚠️ Conta não possui título BOT. Tentando upgrade automático...")
		if err := bot.upgradeToBot(); err != nil {
			log.Printf("⚠️ Não foi possível fazer upgrade automático: %v", err)
		} else {
			log.Println("🎉 Conta convertida para BOT com sucesso!")
		}
	}

	// Inicia servidor HTTP de telemetria e HUD
	go bot.startHTTPServer()

	// Inicia motor de busca e desafios 24/7 contra outros bots
	go bot.autoMatchmakerLoop()

	// Inicia motor de aprendizado contínuo em ociosidade (Lichess TV & Top GMs)
	idleCfg := chess.IdleLearnerConfig{
		MinRating:    cfg.IdleMinRating,
		PollInterval: 12 * time.Second,
		DBPath:       cfg.DBPath,
		Enabled:      cfg.AutoLearn || cfg.IdleLearn,
	}
	bot.idleLearner = chess.NewIdleLearner(
		idleCfg,
		bot.db,
		&bot.dbMu,
		func() bool {
			bot.activeLock.Lock()
			defer bot.activeLock.Unlock()
			return len(bot.activeGame) == 0
		},
		func() {
			bot.saveDB()
		},
	)
	bot.idleLearner.Start(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		log.Println("\n🛑 Encerrando bot com segurança...")
		if bot.idleLearner != nil {
			bot.idleLearner.Stop()
		}
		_ = eng.Close()
		bot.saveDB()
		os.Exit(0)
	}()

	for {
		log.Println("📡 Conectando ao stream de eventos do Lichess (/api/stream/event)...")
		err := bot.listenEvents()
		if err != nil {
			log.Printf("⚠️ Conexão perdida: %v. Reconectando em 5 segundos...", err)
		}
		time.Sleep(5 * time.Second)
	}
}

func loadConfig(configFile, flagToken, flagEngine, flagDB string, flagPort, moveTime, skill int, mode string) Config {
	cfg := Config{
		Token:        "",
		EnginePath:   "",
		DBPath:       "chess.mbvmdb",
		Port:         flagPort,
		MoveTimeMs:   moveTime,
		SkillLevel:   skill,
		Mode:         mode,
		AcceptRated:   true,
		AcceptCasual:  true,
		AutoLearn:     true,
		AutoMatch:     true,
		IdleLearn:     true,
		IdleMinRating: 2300,
	}

	if data, err := os.ReadFile(configFile); err == nil {
		_ = json.Unmarshal(data, &cfg)
	}

	if envPort := os.Getenv("PORT"); envPort != "" {
		if p, err := strconv.Atoi(envPort); err == nil && p > 0 {
			cfg.Port = p
		}
	}

	if envToken := os.Getenv("LICHESS_TOKEN"); envToken != "" {
		cfg.Token = envToken
	}
	if flagToken != "" {
		cfg.Token = flagToken
	}

	if envEngine := os.Getenv("STOCKFISH_PATH"); envEngine != "" {
		cfg.EnginePath = envEngine
	}
	if flagEngine != "" {
		cfg.EnginePath = flagEngine
	}

	if flagDB != "" {
		cfg.DBPath = flagDB
	}

	if cfg.Port <= 0 {
		cfg.Port = 3016
	}

	return cfg
}

func (b *BotApp) startHTTPServer() {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/status", b.handleStatusJSON)
	mux.HandleFunc("/status.json", b.handleStatusJSON)
	mux.HandleFunc("/", b.handleDashboard)

	addr := fmt.Sprintf(":%d", b.cfg.Port)
	log.Printf("🌐 Servidor de Telemetria & HUD rodando em http://0.0.0.0%s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil && err != http.ErrServerClosed {
		log.Printf("⚠️ Erro no servidor HTTP: %v", err)
	}
}

func (b *BotApp) handleStatusJSON(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	b.statsLock.RLock()
	moves := b.movesPlayed
	games := b.gamesPlayed
	wins := b.wins
	losses := b.losses
	lastMove := b.lastMove
	lastEval := b.lastEval
	lastOpp := b.lastOpp
	b.statsLock.RUnlock()

	b.activeLock.Lock()
	activeCount := len(b.activeGame)
	b.activeLock.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ramMB := float64(m.Alloc) / 1024.0 / 1024.0

	dbCount := 0
	if b.db != nil {
		dbCount = b.db.TotalSymbols()
	}

	var idleStats chess.IdleLearnerStats
	if b.idleLearner != nil {
		idleStats = b.idleLearner.GetStats()
	}

	status := map[string]any{
		"online":       true,
		"key":          "lichess",
		"name":         "Lichess Bot (Berserk)",
		"game":         "Lichess.org • Go + Stockfish NNUE",
		"port":         b.cfg.Port,
		"bot_user":     b.botName,
		"title":        "BOT",
		"mode":         b.cfg.Mode,
		"active_games": activeCount,
		"games_played": games,
		"wins":         wins,
		"losses":       losses,
		"moves_played": moves,
		"last_move":    lastMove,
		"last_eval":    lastEval,
		"last_opp":     lastOpp,
		"db_symbols":   dbCount,
		"idle_learner": idleStats,
		"uptime_sec":   int(time.Since(b.startTime).Seconds()),
		"ram_mb":       fmt.Sprintf("%.2f MB", ramMB),
		"dashboard":    fmt.Sprintf("https://lichess.org/@/%s", b.botName),
	}

	_ = json.NewEncoder(w).Encode(status)
}

func (b *BotApp) handleDashboard(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	b.statsLock.RLock()
	moves := b.movesPlayed
	games := b.gamesPlayed
	wins := b.wins
	losses := b.losses
	lastMove := b.lastMove
	lastEval := b.lastEval
	lastOpp := b.lastOpp
	b.statsLock.RUnlock()

	b.activeLock.Lock()
	activeCount := len(b.activeGame)
	b.activeLock.Unlock()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	ramMB := float64(m.Alloc) / 1024.0 / 1024.0

	dbCount := 0
	if b.db != nil {
		dbCount = b.db.TotalSymbols()
	}

	var idleStats chess.IdleLearnerStats
	if b.idleLearner != nil {
		idleStats = b.idleLearner.GetStats()
	}

	idleBadgeColor := "#10b981"
	if !idleStats.IsIdle {
		idleBadgeColor = "#f59e0b"
	}

	html := fmt.Sprintf(`<!DOCTYPE html>
<html lang="pt-BR">
<head>
  <meta charset="UTF-8">
  <title>♟️ Lichess Bot Dashboard — %s</title>
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <style>
    body { background: #0c0d14; color: #e5e7eb; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif; margin: 0; padding: 24px; }
    .container { max-width: 900px; margin: 0 auto; }
    .header { display: flex; align-items: center; justify-content: space-between; margin-bottom: 24px; border-bottom: 1px solid #1f2937; padding-bottom: 16px; }
    .title { font-size: 24px; font-weight: bold; color: #fff; display: flex; align-items: center; gap: 10px; }
    .badge { background: #10b981; color: #000; font-size: 11px; font-weight: bold; padding: 3px 8px; border-radius: 4px; }
    .badge-berserk { background: #ef4444; color: #fff; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 16px; margin-bottom: 24px; }
    .card { background: #161822; border: 1px solid #242938; border-radius: 10px; padding: 18px; }
    .lbl { color: #9ca3af; font-size: 12px; text-transform: uppercase; letter-spacing: 0.5px; }
    .val { font-size: 26px; font-weight: bold; color: #f3f4f6; margin-top: 6px; }
    .banner { background: #1a1e2e; border: 1px solid #3b82f6; border-radius: 10px; padding: 16px; margin-bottom: 24px; }
    .banner-title { font-weight: bold; color: #60a5fa; font-size: 14px; margin-bottom: 6px; display: flex; align-items: center; gap: 8px; }
    .banner-desc { font-size: 13px; color: #d1d5db; }
    .link-btn { display: inline-block; background: #2563eb; color: #fff; padding: 10px 18px; border-radius: 6px; text-decoration: none; font-weight: 500; margin-top: 8px; }
    .link-btn:hover { background: #1d4ed8; }
  </style>
</head>
<body>
  <div class="container">
    <div class="header">
      <div class="title">
        <span>♟️ Lichess Bot (Go 1.27)</span>
        <span class="badge">ONLINE</span>
        <span class="badge badge-berserk">💀 BERSERK</span>
      </div>
      <div>
        <a class="link-btn" href="https://lichess.org/@/%s" target="_blank">Ver Perfil Lichess ↗</a>
      </div>
    </div>

    <!-- Banner de Aprendizado Contínuo em Ociosidade -->
    <div class="banner">
      <div class="banner-title">
        <span style="display:inline-block; width:10px; height:10px; border-radius:50%%; background:%s;"></span>
        <span>🧠 Modo Aprendizado em Ociosidade (Lichess TV & Melhores GMs)</span>
      </div>
      <div class="banner-desc">
        <strong>Status:</strong> %s &nbsp;|&nbsp; 
        <strong>Última Partida:</strong> %s
      </div>
    </div>

    <div class="grid">
      <div class="card">
        <div class="lbl">Partidas em Andamento</div>
        <div class="val" style="color: #60a5fa;">%d</div>
      </div>
      <div class="card">
        <div class="lbl">Partidas Jogadas</div>
        <div class="val">%d</div>
      </div>
      <div class="card">
        <div class="lbl">Vitórias / Derrotas</div>
        <div class="val" style="color: #34d399;">%d <span style="font-size:16px; color:#ef4444;">/ %d</span></div>
      </div>
      <div class="card">
        <div class="lbl">Símbolos no Grafo MBVM-DB</div>
        <div class="val" style="color: #fbbf24;">%d</div>
      </div>
      <div class="card">
        <div class="lbl">GMs Analisados (Ocioso)</div>
        <div class="val" style="color: #a78bfa;">%d</div>
      </div>
      <div class="card">
        <div class="lbl">Lances Mestres Memorizados</div>
        <div class="val" style="color: #38bdf8;">%d</div>
      </div>
      <div class="card">
        <div class="lbl">Total de Lances Calculados</div>
        <div class="val">%d</div>
      </div>
      <div class="card">
        <div class="lbl">Último Lance / Eval</div>
        <div class="val" style="font-size: 18px; color: #c084fc;">%s <span style="font-size: 13px; color: #9ca3af;">(%s)</span></div>
      </div>
      <div class="card">
        <div class="lbl">Último Oponente</div>
        <div class="val" style="font-size: 18px;">%s</div>
      </div>
      <div class="card">
        <div class="lbl">Consumo de Memória RAM</div>
        <div class="val" style="font-size: 20px; color: #38bdf8;">%.2f MB</div>
      </div>
    </div>
  </div>
</body>
</html>`, b.botName, b.botName, idleBadgeColor, idleStats.Status, idleStats.LastGameDesc, activeCount, games, wins, losses, dbCount, idleStats.GamesLearned, idleStats.MovesLearned, moves, lastMove, lastEval, lastOpp, ramMB)

	_, _ = w.Write([]byte(html))
}

func resolveDBPath(p string) string {
	candidates := []string{
		p,
		"chess.mbvmdb",
		filepath.Join("..", "..", "chess.mbvmdb"),
	}
	for _, cand := range candidates {
		if abs, err := filepath.Abs(cand); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}
	return p
}

func resolveEnginePath(configured string) string {
	candidates := []string{}
	if configured != "" {
		candidates = append(candidates, configured)
	}

	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join("bin", "stockfish.exe"),
			filepath.Join("..", "..", "bin", "stockfish.exe"),
			"stockfish.exe",
		)
	} else {
		candidates = append(candidates,
			"/usr/games/stockfish",
			"/usr/bin/stockfish",
			"/usr/local/bin/stockfish",
			"./stockfish",
			"stockfish",
		)
	}

	for _, p := range candidates {
		if abs, err := filepath.Abs(p); err == nil {
			if _, err := os.Stat(abs); err == nil {
				return abs
			}
		}
	}

	if len(candidates) > 0 {
		return candidates[0]
	}
	return "stockfish"
}

func (b *BotApp) fetchAccount() (*UserAccount, error) {
	req, err := http.NewRequest("GET", "https://lichess.org/api/account", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var acc UserAccount
	if err := json.NewDecoder(resp.Body).Decode(&acc); err != nil {
		return nil, err
	}
	return &acc, nil
}

func (b *BotApp) upgradeToBot() error {
	req, err := http.NewRequest("POST", "https://lichess.org/api/bot/account/upgrade", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (b *BotApp) listenEvents() error {
	req, err := http.NewRequest("GET", "https://lichess.org/api/stream/event", nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("erro HTTP %d: %s", resp.StatusCode, string(body))
	}

	log.Println("🟢 Conectado ao stream de eventos! Aguardando desafios e partidas...")

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var ev StreamEvent
		if err := json.Unmarshal([]byte(line), &ev); err != nil {
			continue
		}

		switch ev.Type {
		case "challenge":
			b.handleChallenge(ev.Challenge)
		case "gameStart":
			if game, ok := ev.Game["id"].(string); ok {
				b.activeLock.Lock()
				alreadyRunning := b.activeGame[game]
				b.activeGame[game] = true
				b.activeLock.Unlock()

				if !alreadyRunning {
					log.Printf("⚔️ Nova partida iniciada! ID: %s", game)
					go b.playGame(game)
				}
			}
		case "gameFinish":
			if game, ok := ev.Game["id"].(string); ok {
				log.Printf("🏁 Partida finalizada: %s", game)
				b.activeLock.Lock()
				delete(b.activeGame, game)
				b.activeLock.Unlock()
			}
		}
	}

	return scanner.Err()
}

func (b *BotApp) handleChallenge(ch map[string]any) {
	chID, _ := ch["id"].(string)
	if chID == "" {
		return
	}

	challenger, _ := ch["challenger"].(map[string]any)
	challengerName, _ := challenger["name"].(string)
	challengerID, _ := challenger["id"].(string)
	challengerTitle, _ := challenger["title"].(string)

	// Não aceita o próprio desafio de saída
	if strings.EqualFold(challengerName, b.botName) || strings.EqualFold(challengerID, b.botID) {
		return
	}

	isHuman := (challengerTitle != "BOT")
	rated, _ := ch["rated"].(bool)

	variant, _ := ch["variant"].(map[string]any)
	variantKey, _ := variant["key"].(string)

	if isHuman {
		log.Printf("🌟 [JOGADOR HUMANO DETECTADO] Desafio recebido de %s! Aceitando com PRIORIDADE MÁXIMA!", challengerName)
	} else {
		log.Printf("📩 Desafio de [%s] (ID: %s, Variante: %s, Rating: %v)", challengerName, chID, variantKey, rated)
	}

	if variantKey != "standard" && variantKey != "" {
		log.Printf("⛔ Recusando desafio %s (variante: %s)", chID, variantKey)
		b.declineChallenge(chID, "variant")
		return
	}

	log.Printf("✅ Aceitando desafio de %s...", challengerName)
	b.acceptChallenge(chID)
}

func (b *BotApp) acceptChallenge(id string) {
	url := fmt.Sprintf("https://lichess.org/api/challenge/%s/accept", id)
	req, _ := http.NewRequest("POST", url, nil)
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)
	resp, err := b.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
}

func (b *BotApp) declineChallenge(id, reason string) {
	url := fmt.Sprintf("https://lichess.org/api/challenge/%s/decline", id)
	payload := fmt.Sprintf(`{"reason":"%s"}`, reason)
	req, _ := http.NewRequest("POST", url, bytes.NewBufferString(payload))
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := b.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
}

func (b *BotApp) playGame(gameID string) {
	defer func() {
		b.activeLock.Lock()
		delete(b.activeGame, gameID)
		b.activeLock.Unlock()

		b.modeLock.Lock()
		delete(b.gameModes, gameID)
		delete(b.gameSkills, gameID)
		b.modeLock.Unlock()
	}()

	url := fmt.Sprintf("https://lichess.org/api/bot/game/stream/%s", gameID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("❌ Erro ao criar stream de jogo: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		log.Printf("❌ Erro ao conectar ao jogo %s: %v", gameID, err)
		return
	}
	defer resp.Body.Close()

	var (
		isWhite         bool
		isHumanOpponent bool
		initialFen      = "startpos"
		lastMoveCount = -1
		whitePlayer   = "Brancas"
		blackPlayer   = "Pretas"
		latestMoves   = ""
	)

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var generic map[string]any
		if err := json.Unmarshal([]byte(line), &generic); err != nil {
			continue
		}

		eventType, _ := generic["type"].(string)

		switch eventType {
		case "gameFull":
			var gf GameFullEvent
			if err := json.Unmarshal([]byte(line), &gf); err != nil {
				continue
			}

			if gf.InitialFen != "" && gf.InitialFen != "startpos" {
				initialFen = gf.InitialFen
			}

			wID, _ := gf.White["id"].(string)
			if name, ok := gf.White["name"].(string); ok {
				whitePlayer = name
			}
			if name, ok := gf.Black["name"].(string); ok {
				blackPlayer = name
			}

			isWhite = strings.EqualFold(wID, b.botID)

			oppTitle := ""
			oppUser := blackPlayer
			if isWhite {
				if t, ok := gf.Black["title"].(string); ok {
					oppTitle = t
				}
			} else {
				oppUser = whitePlayer
				if t, ok := gf.White["title"].(string); ok {
					oppTitle = t
				}
			}
			isHumanOpponent = (oppTitle != "BOT")

			if isHumanOpponent {
				log.Printf("🌟 Partida contra JOGADOR HUMANO [%s] | %s vs %s (Bot: %s)",
					gameID, whitePlayer, blackPlayer,
					map[bool]string{true: "Brancas ⚪", false: "Pretas ⚫"}[isWhite])
				welcomeMsg := fmt.Sprintf("Olá %s! Sou um bot especialista em ATAQUE BERSERK 💀, mas você pode escolher o estilo para treinar!\nDigite no chat:\n• !tal (Mikhail Tal - sacrifícios e tática)\n• !solido ou !karpov (Posicional e profilático)\n• !fischer (Ataque na Ala do Rei)\n• !berserk (Ataque total kamikaze - padrão)\n• !facil / !medio / !dificil (Dificuldade)", oppUser)
				b.sendChatMessage(gameID, "player", welcomeMsg)
			} else {
				log.Printf("🎮 Partida [%s] | %s vs %s (Bot: %s, Modo: 💀 %s)",
					gameID, whitePlayer, blackPlayer,
					map[bool]string{true: "Brancas ⚪", false: "Pretas ⚫"}[isWhite],
					strings.ToUpper(b.cfg.Mode))
				b.sendChatMessage(gameID, "player", "Good luck! 100% Go Bot in BERSERK mode + MBVM-DB.")
			}

			latestMoves = gf.State.Moves
			myClock := gf.State.WTime
			if !isWhite {
				myClock = gf.State.BTime
			}
			b.processTurn(gameID, isWhite, initialFen, latestMoves, &lastMoveCount, myClock)

		case "gameState":
			var state GameState
			if err := json.Unmarshal([]byte(line), &state); err != nil {
				continue
			}

			latestMoves = state.Moves

			if state.Status != "started" && state.Status != "" {
				log.Printf("🏁 Fim de jogo [%s]. Status: %s | Vencedor: %s", gameID, state.Status, state.Winner)

				b.statsLock.Lock()
				b.gamesPlayed++
				botWon := (isWhite && state.Winner == "white") || (!isWhite && state.Winner == "black")
				isDraw := (state.Winner == "")
				if botWon {
					b.wins++
				} else if !isDraw {
					b.losses++
				}
				opponent := blackPlayer
				if !isWhite {
					opponent = whitePlayer
				}
				b.lastOpp = opponent
				b.statsLock.Unlock()

				// Mensagem de encerramento para humanos
				if isHumanOpponent {
					b.sendChatMessage(gameID, "player", fmt.Sprintf("Obrigado pela partida, %s! Foi uma honra jogar com você! 🤝", opponent))
				}

				// Auto-aprendizado contínuo (Vitórias E Derrotas!)
				if b.cfg.AutoLearn {
					if botWon {
						log.Printf("🏆 [VITÓRIA] O bot venceu %s! Gravando aprendizado no banco MBVM-DB...", opponent)
						b.learnFromVictory(latestMoves, isWhite, opponent, state.Status)
					} else if !isDraw {
						log.Printf("🧠 [DERROTA] O bot perdeu para %s. Analisando e aprendendo com a derrota no MBVM-DB...", opponent)
						b.learnFromDefeat(latestMoves, isWhite, opponent, state.Status)
					}
				}
				return
			}

			myClockState := state.WTime
			if !isWhite {
				myClockState = state.BTime
			}
			b.processTurn(gameID, isWhite, initialFen, latestMoves, &lastMoveCount, myClockState)

		case "chatLine":
			sender, _ := generic["username"].(string)
			text, _ := generic["text"].(string)
			if sender != b.botName && sender != "lichess" {
				log.Printf("💬 Chat [%s] %s: %s", gameID, sender, text)
				b.handleChatCommand(gameID, sender, text)
			}
		}
	}
}

func (b *BotApp) processTurn(gameID string, isWhite bool, initialFen, moves string, lastMoveCount *int, clockMs int64) {
	moveList := strings.Fields(strings.TrimSpace(moves))
	currentCount := len(moveList)

	isWhiteTurn := (currentCount % 2 == 0)
	isMyTurn := (isWhite && isWhiteTurn) || (!isWhite && !isWhiteTurn)

	if !isMyTurn {
		return
	}

	if currentCount == *lastMoveCount {
		return
	}
	*lastMoveCount = currentCount

	turnNumber := (currentCount / 2) + 1
	playerColor := "Brancas ⚪"
	if !isWhiteTurn {
		playerColor = "Pretas ⚫"
	}

	currentMode := b.getGameMode(gameID)
	currentSkill := b.getGameSkill(gameID)
	_ = b.engine.SetSkillLevel(currentSkill)

	// Estratégia de 3 Segundos: Força Bruta Monstro (Depth 26+) com relógio alto,
	// adaptando a estratégia dinamicamente conforme o tempo diminui
	calcTimeMs := 3000
	if clockMs > 0 {
		if clockMs > 90000 { // Mais de 1m30s no relógio
			calcTimeMs = 3000 // 3.0s -> Profundidade Máxima de Supercomputador (Depth 26+)
		} else if clockMs > 45000 { // 45s a 90s
			calcTimeMs = 2000 // 2.0s -> Modo Forte (Depth 23-25)
		} else if clockMs > 20000 { // 20s a 45s
			calcTimeMs = 1000 // 1.0s -> Transição rápida
		} else if clockMs > 10000 { // 10s a 20s
			calcTimeMs = 450  // 0.45s -> Modo aceleração
		} else { // Menos de 10s (aperto total de tempo)
			calcTimeMs = 180  // 0.18s -> Blitz relâmpago anti-queda de tempo
		}
	}

	log.Printf("🤔 Lance %d (%s) - Calculando com %dms (Relógio: %.1fs) em modo 💀 %s (Skill: %d)...",
		turnNumber, playerColor, calcTimeMs, float64(clockMs)/1000.0, strings.ToUpper(currentMode), currentSkill)

	var bestMove, eval string

	// 1. Consulta o livro de aberturas e transições de mestres no MBVM-DB (0ms)
	if currentCount <= 12 && b.db != nil {
		prefix := strings.Join(moveList, "_")
		if prefix == "" {
			prefix = "start"
		}
		b.dbMu.Lock()
		slice := b.db.ExtractSlice(prefix, 1)
		b.dbMu.Unlock()

		if len(slice.Symbols) > 0 {
			for _, sym := range slice.Symbols {
				if len(sym.OutgoingCalls) > 0 {
					cand := sym.OutgoingCalls[0]
					if parts := strings.Split(cand, "->"); len(parts) > 1 {
						cand = strings.TrimSpace(parts[1])
					}
					cand = strings.TrimSpace(cand)
					if isUCI(cand) {
						bestMove = cand
						eval = "🧠 MBVM-DB Livro Mestre (0ms)"
						log.Printf("🧠 [LIVRO MBVM-DB] Lance mestre recuperado com 0ms: [%s] (%s)", bestMove, sym.Name)
						break
					}
				}
			}
		}
	}

	// 2. Se não houver lance de livro mestre no MBVM-DB, calcula via Stockfish com o modo ativo
	if bestMove == "" {
		var err error
		bestMove, eval, err = b.engine.GetBestMoveFromPosition(initialFen, moves, calcTimeMs, currentMode)
		if err != nil {
			log.Printf("❌ Erro da engine no jogo %s: %v", gameID, err)
			return
		}
	}

	b.statsLock.Lock()
	b.movesPlayed++
	b.lastMove = bestMove
	b.lastEval = eval
	b.statsLock.Unlock()

	log.Printf("💡 Lance escolhido: [%s] (Eval: %s)", bestMove, eval)

	if err := b.sendMove(gameID, bestMove); err != nil {
		log.Printf("❌ Falha ao enviar lance %s: %v", bestMove, err)
	} else {
		log.Printf("⚡ Lance [%s] enviado com sucesso para %s!", bestMove, gameID)
	}
}

func (b *BotApp) setGameMode(gameID, mode string) {
	b.modeLock.Lock()
	defer b.modeLock.Unlock()
	b.gameModes[gameID] = mode
}

func (b *BotApp) getGameMode(gameID string) string {
	b.modeLock.RLock()
	defer b.modeLock.RUnlock()
	if m, ok := b.gameModes[gameID]; ok && m != "" {
		return m
	}
	return b.cfg.Mode
}

func (b *BotApp) setGameSkill(gameID string, skill int) {
	b.modeLock.Lock()
	defer b.modeLock.Unlock()
	b.gameSkills[gameID] = skill
}

func (b *BotApp) getGameSkill(gameID string) int {
	b.modeLock.RLock()
	defer b.modeLock.RUnlock()
	if s, ok := b.gameSkills[gameID]; ok && s >= 0 {
		return s
	}
	return b.cfg.SkillLevel
}

func (b *BotApp) handleChatCommand(gameID, sender, text string) {
	cmd := strings.ToLower(strings.TrimSpace(text))

	switch {
	case strings.Contains(cmd, "!tal") || cmd == "tal":
		b.setGameMode(gameID, "tal")
		b.sendChatMessage(gameID, "player", "⚔️ [ESTILO MIKHAIL TAL] Prepare-se para complicações táticas e sacrifícios agressivos! Boa sorte! 🔥")
		log.Printf("🎯 Jogo %s: modo alterado para TAL a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!solido") || strings.Contains(cmd, "!karpov") || strings.Contains(cmd, "!petrosian") || cmd == "solido" || cmd == "karpov":
		b.setGameMode(gameID, "solid")
		b.sendChatMessage(gameID, "player", "🛡️ [ESTILO POSICIONAL SÓLIDO (KARPOV)] Foco em profilaxia, controle de casas fracas e finais técnicos. Bom jogo!")
		log.Printf("🎯 Jogo %s: modo alterado para SÓLIDO a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!fischer") || strings.Contains(cmd, "!kingside") || cmd == "fischer":
		b.setGameMode(gameID, "kingside")
		b.sendChatMessage(gameID, "player", "🏹 [ESTILO BOBBY FISCHER] Pressão veloz, dinamismo e ataque direto na Ala do Rei!")
		log.Printf("🎯 Jogo %s: modo alterado para FISCHER a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!berserk") || strings.Contains(cmd, "!ultra") || cmd == "berserk":
		b.setGameMode(gameID, "berserk")
		b.sendChatMessage(gameID, "player", "💀 [ESTILO BERSERK MÁXIMO] Ataque kamikaze total e caça implacável ao Rei ativados!")
		log.Printf("🎯 Jogo %s: modo alterado para BERSERK a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!facil") || strings.Contains(cmd, "!easy") || cmd == "facil":
		b.setGameSkill(gameID, 6)
		_ = b.engine.SetSkillLevel(6)
		b.sendChatMessage(gameID, "player", "🌱 [NÍVEL FÁCIL] Força ajustada para treino casual (~1500 ELO). Divirta-se!")
		log.Printf("🎯 Jogo %s: nível alterado para FÁCIL (Skill 6) a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!medio") || strings.Contains(cmd, "!medium") || cmd == "medio":
		b.setGameSkill(gameID, 12)
		_ = b.engine.SetSkillLevel(12)
		b.sendChatMessage(gameID, "player", "⚖️ [NÍVEL MÉDIO] Força ajustada para treino intermediário (~2000 ELO). Bom jogo!")
		log.Printf("🎯 Jogo %s: nível alterado para MÉDIO (Skill 12) a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!dificil") || strings.Contains(cmd, "!hard") || cmd == "dificil":
		b.setGameSkill(gameID, 20)
		_ = b.engine.SetSkillLevel(20)
		b.sendChatMessage(gameID, "player", "🔥 [NÍVEL DIFÍCIL] Stockfish NNUE em força máxima GM (+3500 ELO)!")
		log.Printf("🎯 Jogo %s: nível alterado para DIFÍCIL (Skill 20) a pedido de %s", gameID, sender)

	case strings.Contains(cmd, "!ajuda") || strings.Contains(cmd, "!help") || strings.Contains(cmd, "!estilos") || cmd == "ajuda" || cmd == "help":
		b.sendChatMessage(gameID, "player", "Estilos: !tal, !solido, !karpov, !fischer, !berserk | Dificuldade: !facil, !medio, !dificil")
	}
}

func isUCI(m string) bool {
	if len(m) != 4 && len(m) != 5 {
		return false
	}
	return m[0] >= 'a' && m[0] <= 'h' && m[1] >= '1' && m[1] <= '8' &&
		m[2] >= 'a' && m[2] <= 'h' && m[3] >= '1' && m[3] <= '8'
}

func (b *BotApp) learnFromVictory(moves string, isWhite bool, opponent, status string) {
	b.dbMu.Lock()
	defer b.dbMu.Unlock()

	moveList := strings.Fields(strings.TrimSpace(moves))
	if len(moveList) < 2 || b.db == nil {
		return
	}

	myColor := "w"
	if !isWhite {
		myColor = "b"
	}

	gameItem := mbvmdb.ChessGameItem{
		ID:        fmt.Sprintf("lichess_%d", time.Now().Unix()),
		White:     "kbelludoo (Bot)",
		Black:     opponent,
		Result:    "1-0",
		Date:      time.Now().Format("2006.01.02"),
		Moves:     moveList,
		TimeClass: "lichess-live",
	}
	if !isWhite {
		gameItem.White = opponent
		gameItem.Black = "kbelludoo (Bot)"
		gameItem.Result = "0-1"
	}

	b.db.IngestChessGames([]mbvmdb.ChessGameItem{gameItem})

	// Indexa transições vitoriosas
	learned := 0
	for i := 0; i < len(moveList); i++ {
		isOurTurn := (myColor == "w" && i%2 == 0) || (myColor == "b" && i%2 == 1)
		if isOurTurn {
			prefix := strings.Join(moveList[:i], "_")
			winningMove := moveList[i]
			cand := fmt.Sprintf("%s -> %s", prefix, winningMove)
			if prefix == "" {
				prefix = "start"
			}

			b.db.IngestChessOpenings([]mbvmdb.ChessOpeningItem{
				{
					ECO:           "LRN",
					Name:          fmt.Sprintf("Auto-Aprendizado: %s", winningMove),
					Moves:         prefix,
					WhiteWin:      100.0,
					Continuations: []string{winningMove, cand},
					KeyThemes:     []string{fmt.Sprintf("Vitória contra %s (%s)", opponent, status)},
				},
			})
			learned++
		}
	}

	b.db.CompileChessBytecode()
	b.saveDB()

	log.Printf("🧠 [AUTO-APRENDIZADO CONCLUÍDO] %d lances vencedores gravados no banco MBVM-DB! Total de símbolos: %d",
		learned, b.db.TotalSymbols())
}

func (b *BotApp) learnFromDefeat(moves string, isWhite bool, opponent, status string) {
	b.dbMu.Lock()
	defer b.dbMu.Unlock()

	moveList := strings.Fields(strings.TrimSpace(moves))
	if len(moveList) < 2 || b.db == nil {
		return
	}

	oppColor := "b"
	whiteName := "kbelludoo (Bot)"
	blackName := opponent
	if !isWhite {
		oppColor = "w"
		whiteName = opponent
		blackName = "kbelludoo (Bot)"
	}

	// 1. Registra a partida completa no histórico
	gameItem := mbvmdb.ChessGameItem{
		ID:        fmt.Sprintf("lichess_loss_%d", time.Now().Unix()),
		White:     whiteName,
		Black:     blackName,
		Result:    "0-1",
		Date:      time.Now().Format("2006.01.02"),
		Moves:     moveList,
		TimeClass: "lichess-live",
	}
	if !isWhite {
		gameItem.Result = "1-0"
	}
	b.db.IngestChessGames([]mbvmdb.ChessGameItem{gameItem})

	// 2. Aprende as respostas vitoriosas do oponente (técnicas de contra-ataque e punição)
	learned := 0
	for i := 0; i < len(moveList); i++ {
		isOppTurn := (oppColor == "w" && i%2 == 0) || (oppColor == "b" && i%2 == 1)
		if isOppTurn {
			prefix := strings.Join(moveList[:i], "_")
			winningMove := moveList[i]
			cand := fmt.Sprintf("%s -> %s", prefix, winningMove)
			if prefix == "" {
				prefix = "start"
			}

			b.db.IngestChessOpenings([]mbvmdb.ChessOpeningItem{
				{
					ECO:           "DEF",
					Name:          fmt.Sprintf("Defesa/Ataque de %s: %s", opponent, winningMove),
					Moves:         prefix,
					WhiteWin:      map[bool]float64{true: 100.0, false: 0.0}[oppColor == "w"],
					Continuations: []string{winningMove, cand},
					KeyThemes:     []string{fmt.Sprintf("Lição aprendida na derrota contra %s (%s)", opponent, status)},
				},
			})
			learned++
		}
	}

	b.db.CompileChessBytecode()
	b.saveDB()

	log.Printf("🧠 [APRENDIZADO DE DERROTA CONCLUÍDO] %d lances vencedores de %s assimilados no MBVM-DB! Grafo expandido para %d símbolos",
		learned, opponent, b.db.TotalSymbols())
}

func (b *BotApp) saveDB() {
	if b.db == nil || b.cfg.DBPath == "" {
		return
	}
	targetPath := b.cfg.DBPath
	if err := b.db.SaveToFile(targetPath); err != nil {
		log.Printf("⚠️ Erro ao salvar banco MBVM-DB em %s: %v", targetPath, err)
	} else {
		log.Printf("💾 Banco MBVM-DB persistido com sucesso em %s", targetPath)
	}
}

func (b *BotApp) sendMove(gameID, move string) error {
	url := fmt.Sprintf("https://lichess.org/api/bot/game/%s/move/%s", gameID, move)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

func (b *BotApp) sendChatMessage(gameID, room, text string) {
	url := fmt.Sprintf("https://lichess.org/api/bot/game/%s/chat", gameID)
	payload := fmt.Sprintf("room=%s&text=%s", room, text)
	req, _ := http.NewRequest("POST", url, strings.NewReader(payload))
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := b.client.Do(req)
	if err == nil {
		defer resp.Body.Close()
	}
}

type BotTarget struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Rating   int    `json:"rating"`
	PerfType string `json:"perf_type"`
}

// challengeBot envia um desafio direto a outro bot no Lichess e retorna o ID do desafio.
func (b *BotApp) challengeBot(username string, limitSec, incSec int, rated bool) (string, error) {
	endpoint := fmt.Sprintf("https://lichess.org/api/challenge/%s", url.PathEscape(username))
	form := url.Values{}
	form.Set("rated", strconv.FormatBool(rated))
	form.Set("clock.limit", strconv.Itoa(limitSec))
	form.Set("clock.increment", strconv.Itoa(incSec))
	form.Set("color", "random")
	form.Set("variant", "standard")

	req, err := http.NewRequest("POST", endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := b.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, string(body))
	}

	var chResp struct {
		ID string `json:"id"`
	}
	_ = json.Unmarshal(body, &chResp)
	return chResp.ID, nil
}

// cancelChallenge cancela um desafio pendente não aceito.
func (b *BotApp) cancelChallenge(id string) {
	if id == "" {
		return
	}
	endpoint := fmt.Sprintf("https://lichess.org/api/challenge/%s/cancel", id)
	req, err := http.NewRequest("POST", endpoint, nil)
	if err != nil {
		return
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)
	resp, err := b.client.Do(req)
	if err == nil {
		_ = resp.Body.Close()
	}
}

// fetchOnlineBots descobre bots ativos no Lichess e ordena por pontuação (mais pontos primeiro).
func (b *BotApp) fetchOnlineBots(limit int) ([]BotTarget, error) {
	if limit <= 0 {
		limit = 100
	}
	endpoint := fmt.Sprintf("https://lichess.org/api/bot/online?nb=%d", limit)
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+b.cfg.Token)

	resp, err := b.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("status %d", resp.StatusCode)
	}

	type rawBot struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Perfs    struct {
			Blitz struct {
				Rating int `json:"rating"`
			} `json:"blitz"`
			Rapid struct {
				Rating int `json:"rating"`
			} `json:"rapid"`
			Bullet struct {
				Rating int `json:"rating"`
			} `json:"bullet"`
		} `json:"perfs"`
		TosViolation bool `json:"tosViolation"`
		Disabled     bool `json:"disabled"`
	}

	var bots []BotTarget
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		var r rawBot
		if err := json.Unmarshal([]byte(line), &r); err != nil || r.Username == "" {
			continue
		}
		if r.TosViolation || r.Disabled {
			continue
		}
		// Ignora o próprio bot
		if strings.EqualFold(r.ID, b.botID) || strings.EqualFold(r.Username, b.botName) {
			continue
		}
		// Ignora bots com peças de vantagem / odds que requerem FEN específico
		lowerName := strings.ToLower(r.Username)
		if strings.Contains(lowerName, "odds") {
			continue
		}

		// Prioriza rating blitz, depois rapid, depois bullet
		rating := r.Perfs.Blitz.Rating
		perfType := "Blitz"
		if rating == 0 {
			rating = r.Perfs.Rapid.Rating
			perfType = "Rapid"
		}
		if rating == 0 {
			rating = r.Perfs.Bullet.Rating
			perfType = "Bullet"
		}

		bots = append(bots, BotTarget{
			ID:       r.ID,
			Username: r.Username,
			Rating:   rating,
			PerfType: perfType,
		})
	}

	// Ordena do que tem MAIS PONTOS para o que tem menos pontos (Rating Decrescente)
	sort.Slice(bots, func(i, j int) bool {
		return bots[i].Rating > bots[j].Rating
	})

	return bots, scanner.Err()
}

// autoMatchmakerLoop executa o ciclo 24/7 desafiando sempre os bots com mais pontos primeiro.
func (b *BotApp) autoMatchmakerLoop() {
	if !b.cfg.AutoMatch {
		log.Println("ℹ️ Auto-Matchmaker 24/7 desativado nas configurações.")
		return
	}

	log.Println("🤖 [MATCHMAKER 24/7] Iniciando motor autônomo focado nos BOTS COM MAIS PONTOS (Rating Alto)...")
	time.Sleep(5 * time.Second)

	// Mapa de cooldown: Username -> Expiração do bloqueio temporário
	cooldowns := make(map[string]time.Time)

	for {
		b.activeLock.Lock()
		activeCount := len(b.activeGame)
		b.activeLock.Unlock()

		if activeCount > 0 {
			time.Sleep(4 * time.Second)
			continue
		}

		now := time.Now()
		// Limpa cooldowns expirados
		for name, exp := range cooldowns {
			if now.After(exp) {
				delete(cooldowns, name)
			}
		}

		// Busca até 100 bots online ordenados pelos que têm MAIS PONTOS
		candidates, err := b.fetchOnlineBots(100)
		if err != nil || len(candidates) == 0 {
			log.Printf("⚠️ [MATCHMAKER] Erro ao listar bots online ou nenhum bot: %v. Tentando em 8s...", err)
			time.Sleep(8 * time.Second)
			continue
		}

		var target *BotTarget
		for i := range candidates {
			cand := &candidates[i]
			if exp, blocked := cooldowns[cand.Username]; blocked && now.Before(exp) {
				continue
			}
			target = cand
			break
		}

		if target == nil {
			log.Println("ℹ️ [MATCHMAKER] Todos os bots de pontuação alta estão em cooldown temporário. Aguardando 12s...")
			time.Sleep(12 * time.Second)
			continue
		}

		rated := b.cfg.AcceptRated
		log.Printf("🎯 [MATCHMAKER 24/7] Desafiando BOT COM MAIS PONTOS: %s (Rating: %d %s | Valendo Pontos: %v)...",
			target.Username, target.Rating, target.PerfType, rated)

		chID, err := b.challengeBot(target.Username, 180, 2, rated)
		if err != nil {
			errStr := strings.ToLower(err.Error())
			if strings.Contains(errStr, "429") || strings.Contains(errStr, "too many requests") {
				log.Printf("⚠️ [MATCHMAKER] Rate limit 429 do Lichess! Pausando por 45s...")
				time.Sleep(45 * time.Second)
				continue
			} else if strings.Contains(errStr, "100 games") || strings.Contains(errStr, "ratelimit") || strings.Contains(errStr, "rate limit") {
				log.Printf("⚠️ [MATCHMAKER] %s atingiu o limite de partidas do Lichess. Cooldown de 1h.", target.Username)
				cooldowns[target.Username] = time.Now().Add(1 * time.Hour)
			} else if strings.Contains(errStr, "rated") && rated {
				log.Printf("ℹ️ [MATCHMAKER] %s não joga valendo pontos (rated). Tentando casual em 2s...", target.Username)
				if b.cfg.AcceptCasual {
					chID, err = b.challengeBot(target.Username, 180, 2, false)
				}
				if err != nil {
					log.Printf("⚠️ [MATCHMAKER] Falha ao desafiar %s mesmo casual: %v. Cooldown 5m.", target.Username, err)
					cooldowns[target.Username] = time.Now().Add(5 * time.Minute)
					time.Sleep(2 * time.Second)
					continue
				}
			} else {
				log.Printf("⚠️ [MATCHMAKER] Erro ao desafiar %s: %v. Cooldown 5m e avançando para o próximo bot...", target.Username, err)
				cooldowns[target.Username] = time.Now().Add(5 * time.Minute)
				time.Sleep(2 * time.Second)
				continue
			}
		}

		log.Printf("⏳ [MATCHMAKER 24/7] Desafio enviado para %s (Rating: %d, ID: %s)! Aguardando aceite...", target.Username, target.Rating, chID)

		// Aguarda até 20s verificando se a partida iniciou
		started := false
		for i := 0; i < 7; i++ {
			time.Sleep(3 * time.Second)
			b.activeLock.Lock()
			if len(b.activeGame) > 0 {
				started = true
			}
			b.activeLock.Unlock()
			if started {
				break
			}
		}

		if !started {
			log.Printf("ℹ️ [MATCHMAKER 24/7] %s não aceitou o desafio a tempo. Cancelando desafio e buscando próximo bot com mais pontos...", target.Username)
			if chID != "" {
				b.cancelChallenge(chID)
			}
			cooldowns[target.Username] = time.Now().Add(3 * time.Minute)
			time.Sleep(2 * time.Second)
		} else {
			log.Printf("⚔️ [MATCHMAKER 24/7] Partida contra %s (Rating: %d) iniciada com sucesso!", target.Username, target.Rating)
			// Monitora até a partida terminar
			for {
				time.Sleep(3 * time.Second)
				b.activeLock.Lock()
				inGame := len(b.activeGame) > 0
				b.activeLock.Unlock()
				if !inGame {
					break
				}
			}
			log.Println("🧘 [MATCHMAKER 24/7] Partida encerrada! Cooldown de 5s antes de buscar o próximo bot com mais pontos...")
			time.Sleep(5 * time.Second)
		}
	}
}
