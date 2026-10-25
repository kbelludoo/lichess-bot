package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStockfishLocalEngine(t *testing.T) {
	binPath := filepath.Join("..", "..", "bin", "stockfish.exe")
	if _, err := os.Stat(binPath); err != nil {
		t.Skipf("Stockfish binary not found at %s, skipping test", binPath)
	}

	eng, err := NewUCIEngine(binPath)
	if err != nil {
		t.Fatalf("Failed to initialize Stockfish: %v", err)
	}
	defer eng.Close()

	// Initial position
	fen := "rnbqkbnr/pppppppp/8/8/8/8/PPPPPPPP/RNBQKBNR w KQkq - 0 1"
	bestMove, eval, err := eng.GetBestMove(fen, 100)
	if err != nil {
		t.Fatalf("Failed to get best move: %v", err)
	}

	if bestMove == "" {
		t.Fatalf("Expected valid best move, got empty")
	}

	t.Logf("Stockfish BestMove: %s | Eval: %s", bestMove, eval)
}

func TestStockfishTalEngine(t *testing.T) {
	binPath := filepath.Join("..", "..", "bin", "stockfish.exe")
	if _, err := os.Stat(binPath); err != nil {
		t.Skipf("Stockfish binary not found at %s, skipping test", binPath)
	}

	eng, err := NewUCIEngine(binPath)
	if err != nil {
		t.Fatalf("Failed to initialize Stockfish: %v", err)
	}
	defer eng.Close()

	// Italian Game position where b2b4 (Evans Gambit) is a candidate:
	// 1. e4 e5 2. Nf3 Nc6 3. Bc4 Bc5
	fen := "r1bqk1nr/pppp1ppp/2n5/2b1p3/2B1P3/5N2/PPPP1PPP/RNBQK2R w KQkq - 4 4"
	bestMove, eval, err := eng.GetBestMoveTal(fen, 200)
	if err != nil {
		t.Fatalf("Failed to get best move in Tal mode: %v", err)
	}

	if bestMove == "" {
		t.Fatalf("Expected valid best move, got empty")
	}

	t.Logf("Mikhail Tal BestMove: %s | Eval: %s", bestMove, eval)
}

func TestStockfishBerserkEngine(t *testing.T) {
	binPath := filepath.Join("..", "..", "bin", "stockfish.exe")
	if _, err := os.Stat(binPath); err != nil {
		t.Skipf("Stockfish binary not found at %s, skipping test", binPath)
	}

	eng, err := NewUCIEngine(binPath)
	if err != nil {
		t.Fatalf("Failed to initialize Stockfish: %v", err)
	}
	defer eng.Close()

	// 1. e4 e5 2. Nf3 Nc6 3. Nc3 Nf6 (Four Knights position where Halloween sacrifice Nxe5 is tested)
	fen := "r1bqkb1r/pppp1ppp/2n2n2/4p3/4P3/2N2N2/PPPP1PPP/R1BQKB1R w KQkq - 4 4"
	bestMove, eval, err := eng.GetBestMoveBerserk(fen, 200)
	if err != nil {
		t.Fatalf("Failed to get best move in Berserk mode: %v", err)
	}

	if bestMove == "" {
		t.Fatalf("Expected valid best move, got empty")
	}

	t.Logf("Berserk BestMove: %s | Eval: %s", bestMove, eval)
}
