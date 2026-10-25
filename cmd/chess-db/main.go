package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/kbelludoo/chess-mbvmdb/pkg/chess"
	"github.com/kbelludoo/mbvm-db/pkg/mbvmdb"
)

const defaultDBPath = "chess.mbvmdb"

func printUsage() {
	fmt.Println(`⚡ CHESS-MBVM-DB — Micro-Bytecode Virtual Machine Chess Database (Go 1.27)
Integrado com Chess.com PubAPI e Catálogo Mestre ECO (A00-E99)

COMANDOS:
  chess-db build [saida.mbvmdb]
      Compila o catálogo ECO Mestre e partidas de GMs do Chess.com para banco binário.

  chess-db fetch-user <usuario> [max_jogos] [banco.mbvmdb]
      Baixa partidas recentes de qualquer jogador do Chess.com e indexa no banco.

  chess-db ingest-pgn <arquivo.pgn> [banco.mbvmdb]
      Ingere arquivo PGN local no grafo de lances e aberturas do banco.

  chess-db slice <banco.mbvmdb> <termo_ou_lance> [profundidade]
      Extrai subgrafo de lances e variações em microssegundos (otimizado para LLM).

  chess-db slice-budget <banco.mbvmdb> <termo> <teto_tokens>
      Extrai subgrafo com limite rígido de tokens para prompts de IA sem ruído.

  chess-db suggest <abertura_ou_lance> [banco.mbvmdb]
      Consulta respostas táticas recomendadas, taxas de vitória e temas estratégicos.

  chess-db inspect [banco.mbvmdb]
      Mostra estatísticas do banco binário (símbolos, string pool, bytecode, CRC32).

  chess-db verify <banco.mbvmdb>
      Valida cabeçalho mágico MBVM\x01\x00 e integridade CRC32 do arquivo.`)
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		return
	}

	cmd := os.Args[1]
	switch cmd {
	case "build":
		runBuild()
	case "fetch-user":
		runFetchUser()
	case "ingest-pgn":
		runIngestPGN()
	case "slice":
		runSlice()
	case "slice-budget":
		runSliceBudget()
	case "suggest":
		runSuggest()
	case "inspect":
		runInspect()
	case "verify":
		runVerify()
	case "help", "-h", "--help":
		printUsage()
	default:
		fmt.Printf("Comando desconhecido: %s\n\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func runBuild() {
	outPath := defaultDBPath
	if len(os.Args) >= 3 {
		outPath = os.Args[2]
	}

	fmt.Printf("🚀 Compilando Banco de Xadrez MBVM-DB em: %s\n", outPath)
	start := time.Now()

	db := mbvmdb.New()

	// 1. Ingest Master ECO Openings
	var openings []mbvmdb.ChessOpeningItem
	tsvOpenings, err := chess.LoadAllECOOpenings("data/openings")
	if err == nil && len(tsvOpenings) > 0 {
		fmt.Printf("  • Ingerindo %d aberturas mestres da Enciclopédia ECO Completa (A00-E99)...\n", len(tsvOpenings))
		openings = tsvOpenings
	} else {
		openings = chess.MasterOpenings()
		fmt.Printf("  • Ingerindo %d aberturas mestres ECO (A00-E99)...\n", len(openings))
	}
	db.IngestChessOpenings(openings)

	// 2. Fetch Sample Grandmaster Games from Chess.com
	client := chess.NewChessComClient()
	fmt.Printf("  • Conectando à API do Chess.com (api.chess.com/pub)...\n")
	gms := []string{"hikaru", "magnuscarlsen"}
	totalGames := 0

	for _, gm := range gms {
		fmt.Printf("    - Baixando partidas recentes de GM: %s...", gm)
		games, err := client.FetchRecentGames(gm, 15)
		if err != nil {
			fmt.Printf(" (aviso: %v, prosseguindo)\n", err)
			continue
		}
		fmt.Printf(" %d partidas obtidas\n", len(games))
		db.IngestChessGames(games)
		totalGames += len(games)
	}

	// 3. Compile Bytecode and CSR Graph
	fmt.Printf("  • Gerando bytecode compacto e arrays CSR...\n")
	db.CompileChessBytecode()

	// 4. Atomic Save
	if err := db.SaveToFile(outPath); err != nil {
		fmt.Printf("❌ Erro ao salvar banco: %v\n", err)
		os.Exit(1)
	}

	dur := time.Since(start)
	info, _ := os.Stat(outPath)
	var fSize int64
	if info != nil {
		fSize = info.Size()
	}

	fmt.Println("\n✅ Banco de Xadrez MBVM-DB gerado com sucesso!")
	fmt.Printf("   Arquivo:       %s\n", outPath)
	fmt.Printf("   Tamanho:       %.2f KB (%d bytes)\n", float64(fSize)/1024.0, fSize)
	fmt.Printf("   Símbolos:      %d nós indexados\n", db.TotalSymbols())
	fmt.Printf("   Aberturas ECO: %d\n", len(openings))
	fmt.Printf("   Partidas GM:   %d\n", totalGames)
	fmt.Printf("   Tempo total:   %v\n", dur)
}

func runFetchUser() {
	if len(os.Args) < 3 {
		fmt.Println("Uso: chess-db fetch-user <usuario_chess.com> [max_jogos] [banco.mbvmdb]")
		return
	}

	user := os.Args[2]
	maxGames := 25
	dbPath := defaultDBPath

	if len(os.Args) >= 4 {
		if n, err := strconv.Atoi(os.Args[3]); err == nil && n > 0 {
			maxGames = n
		}
	}
	if len(os.Args) >= 5 {
		dbPath = os.Args[4]
	}

	client := chess.NewChessComClient()
	fmt.Printf("🌐 Consultando perfil de '%s' no Chess.com...\n", user)
	profile, err := client.GetPlayerProfile(user)
	if err != nil {
		fmt.Printf("⚠️ Erro ao consultar perfil: %v\n", err)
	} else {
		fmt.Printf("   Jogador: %s (%s) | Status: %s | URL: %s\n", profile.Username, profile.Name, profile.Status, profile.URL)
	}

	fmt.Printf("📥 Baixando até %d partidas recentes de '%s'...\n", maxGames, user)
	games, err := client.FetchRecentGames(user, maxGames)
	if err != nil {
		fmt.Printf("❌ Falha ao buscar partidas: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("✓ %d partidas recuperadas com sucesso!\n", len(games))

	var db *mbvmdb.DB
	if _, err := os.Stat(dbPath); err == nil {
		fmt.Printf("📂 Carregando banco existente: %s\n", dbPath)
		db, err = mbvmdb.LoadFromFile(dbPath)
		if err != nil {
			fmt.Printf("❌ Falha ao abrir banco existente: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("🌱 Criando novo banco com catálogo ECO mestre...")
		db = mbvmdb.New()
		db.IngestChessOpenings(chess.MasterOpenings())
	}

	db.IngestChessGames(games)
	db.CompileChessBytecode()

	if err := db.SaveToFile(dbPath); err != nil {
		fmt.Printf("❌ Erro ao salvar banco: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Banco atualizado! Total de símbolos indexados: %d\n", db.TotalSymbols())
}

func runIngestPGN() {
	if len(os.Args) < 3 {
		fmt.Println("Uso: chess-db ingest-pgn <arquivo.pgn> [banco.mbvmdb]")
		return
	}

	pgnPath := os.Args[2]
	dbPath := defaultDBPath
	if len(os.Args) >= 4 {
		dbPath = os.Args[3]
	}

	data, err := os.ReadFile(pgnPath)
	if err != nil {
		fmt.Printf("❌ Não foi possível ler %s: %v\n", pgnPath, err)
		os.Exit(1)
	}

	var db *mbvmdb.DB
	if _, err := os.Stat(dbPath); err == nil {
		db, err = mbvmdb.LoadFromFile(dbPath)
		if err != nil {
			fmt.Printf("❌ Falha ao abrir banco: %v\n", err)
			os.Exit(1)
		}
	} else {
		db = mbvmdb.New()
		db.IngestChessOpenings(chess.MasterOpenings())
	}

	fmt.Printf("📖 Ingerindo PGN %s...\n", filepath.Base(pgnPath))
	if err := db.IngestChessPGN(string(data)); err != nil {
		fmt.Printf("❌ Erro ao ingerir PGN: %v\n", err)
		os.Exit(1)
	}

	db.CompileChessBytecode()
	if err := db.SaveToFile(dbPath); err != nil {
		fmt.Printf("❌ Erro ao salvar: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ PGN incorporado! Total de símbolos: %d\n", db.TotalSymbols())
}

func runSlice() {
	if len(os.Args) < 4 {
		fmt.Println("Uso: chess-db slice <banco.mbvmdb> <abertura_ou_lance> [profundidade]")
		return
	}

	dbPath := os.Args[2]
	target := os.Args[3]
	depth := 1
	if len(os.Args) >= 5 {
		if d, err := strconv.Atoi(os.Args[4]); err == nil && d >= 0 {
			depth = d
		}
	}

	db, err := mbvmdb.LoadFromFile(dbPath)
	if err != nil {
		fmt.Printf("❌ Erro ao carregar banco %s: %v\n", dbPath, err)
		os.Exit(1)
	}

	start := time.Now()
	slice := db.ExtractSlice(target, depth)
	dur := time.Since(start)

	fmt.Printf("🎯 [MBVM-DB SLICE] Alvo: '%s' | Profundidade: %d | Tempo de busca: %v\n", target, depth, dur)
	fmt.Printf("   Nós conectados encontrados: %d | Estimativa de tokens: ~%d\n\n", len(slice.Symbols), slice.TokensEstimate)

	if len(slice.Symbols) == 0 {
		fmt.Printf("Nenhum lance ou abertura correspondente a '%s' encontrado.\n", target)
		return
	}

	for i, s := range slice.Symbols {
		fmt.Printf("[%d] %s (%s)\n", i+1, s.Name, s.Kind)
		if s.Signature != "" {
			fmt.Printf("    Estatística: %s\n", s.Signature)
		}
		if s.Receiver != "" {
			fmt.Printf("    Origem/ECO:  %s\n", s.Receiver)
		}
		for _, f := range s.Fields {
			fmt.Printf("    • %s\n", f)
		}
		if len(s.OutgoingCalls) > 0 {
			fmt.Printf("    ↳ Próximos lances: %s\n", strings.Join(s.OutgoingCalls, ", "))
		}
		fmt.Println()
	}
}

func runSliceBudget() {
	if len(os.Args) < 5 {
		fmt.Println("Uso: chess-db slice-budget <banco.mbvmdb> <termo> <teto_tokens>")
		return
	}

	dbPath := os.Args[2]
	target := os.Args[3]
	budget, err := strconv.Atoi(os.Args[4])
	if err != nil || budget <= 0 {
		fmt.Println("Teto de tokens inválido.")
		return
	}

	db, err := mbvmdb.LoadFromFile(dbPath)
	if err != nil {
		fmt.Printf("❌ Erro ao abrir: %v\n", err)
		os.Exit(1)
	}

	slice := db.ExtractBudgetSlice(target, 2, budget)
	fmt.Printf("🛡️ [MBVM-DB SLICE-BUDGET] Alvo: '%s' | Teto: %d tokens | Retornados: ~%d tokens (%d nós)\n\n",
		target, budget, slice.TokensEstimate, len(slice.Symbols))

	for i, s := range slice.Symbols {
		fmt.Printf("[%d] %s [%s]\n", i+1, s.Name, s.Kind)
		for _, f := range s.Fields {
			fmt.Printf("    • %s\n", f)
		}
	}
}

func runSuggest() {
	if len(os.Args) < 3 {
		fmt.Println("Uso: chess-db suggest <abertura_ou_lance> [banco.mbvmdb]")
		return
	}

	target := os.Args[2]
	dbPath := defaultDBPath
	if len(os.Args) >= 4 {
		dbPath = os.Args[3]
	}

	db, err := mbvmdb.LoadFromFile(dbPath)
	if err != nil {
		fmt.Printf("❌ Erro ao abrir banco %s: %v\n", dbPath, err)
		os.Exit(1)
	}

	slice := db.ExtractSlice(target, 2)
	fmt.Printf("♟️ Sugestões Táticas e Estratégicas para: '%s'\n", target)
	fmt.Printf("=====================================================\n\n")

	if len(slice.Symbols) == 0 {
		fmt.Println("Nenhum dado encontrado para esse lance/abertura.")
		return
	}

	for _, s := range slice.Symbols {
		if s.Kind == "chess_opening" {
			fmt.Printf("📖 ABERTURA: %s [%s]\n", s.Name, s.Package)
			fmt.Printf("   Taxas: %s\n", s.Signature)
			for _, f := range s.Fields {
				fmt.Printf("   %s\n", f)
			}
			if len(s.OutgoingCalls) > 0 {
				fmt.Printf("   ⚡ Respostas Recomendadas: %s\n", strings.Join(s.OutgoingCalls, ", "))
			}
			fmt.Println()
		} else if s.Kind == "chess_move" {
			fmt.Printf("♞ TRANSIÇÃO DE LANCE: %s\n", s.Name)
			fmt.Printf("   Frequência e Desempenho: %s\n", s.Signature)
			for _, f := range s.Fields {
				fmt.Printf("   • %s\n", f)
			}
			fmt.Println()
		}
	}
}

func runInspect() {
	dbPath := defaultDBPath
	if len(os.Args) >= 3 {
		dbPath = os.Args[2]
	}

	info, err := os.Stat(dbPath)
	if err != nil {
		fmt.Printf("❌ Arquivo não encontrado: %s\n", dbPath)
		os.Exit(1)
	}

	db, err := mbvmdb.LoadFromFile(dbPath)
	if err != nil {
		fmt.Printf("❌ Erro ao carregar banco: %v\n", err)
		os.Exit(1)
	}

	openings := 0
	moves := 0
	games := 0
	for _, s := range db.Symbols() {
		switch s.Kind {
		case "chess_opening":
			openings++
		case "chess_move":
			moves++
		case "chess_game":
			games++
		}
	}

	fmt.Printf("🔍 INFORMAÇÕES DO BANCO BINÁRIO: %s\n", dbPath)
	fmt.Printf("   Tamanho em Disco:   %.2f KB (%d bytes)\n", float64(info.Size())/1024.0, info.Size())
	fmt.Printf("   Total de Símbolos:  %d nós indexados\n", db.TotalSymbols())
	fmt.Printf("   • Aberturas ECO:    %d\n", openings)
	fmt.Printf("   • Lances & Ramos:   %d\n", moves)
	fmt.Printf("   • Partidas Indexadas: %d\n", games)
	fmt.Printf("   Tamanho Bytecode:   %d bytes\n", db.BytecodeSize())
}

func runVerify() {
	if len(os.Args) < 3 {
		fmt.Println("Uso: chess-db verify <banco.mbvmdb>")
		return
	}
	dbPath := os.Args[2]
	db, err := mbvmdb.LoadFromFile(dbPath)
	if err != nil {
		fmt.Printf("❌ Falha na verificação de %s: %v\n", dbPath, err)
		os.Exit(1)
	}
	fmt.Printf("✅ [MBVM-DB] Arquivo '%s' 100%% íntegro!\n", dbPath)
	fmt.Printf("   Magic Header MBVM\\x01\\x00 confirmado, versão 1.0 e CRC32 validado.\n")
	fmt.Printf("   Total de Símbolos: %d nós | Bytecode: %d bytes\n", db.TotalSymbols(), db.BytecodeSize())
}
