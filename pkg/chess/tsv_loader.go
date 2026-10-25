package chess

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/kbelludoo/mbvm-db/pkg/mbvmdb"
)

// LoadAllECOOpenings reads all TSV files (a.tsv through e.tsv) from dir and converts them into ChessOpeningItems.
func LoadAllECOOpenings(dir string) ([]mbvmdb.ChessOpeningItem, error) {
	files := []string{"a.tsv", "b.tsv", "c.tsv", "d.tsv", "e.tsv"}
	var allItems []mbvmdb.ChessOpeningItem
	seen := make(map[string]bool)

	for _, f := range files {
		path := filepath.Join(dir, f)
		items, err := parseTSVFile(path)
		if err != nil {
			fmt.Printf("Aviso ao ler %s: %v\n", f, err)
			continue
		}
		for _, it := range items {
			key := fmt.Sprintf("%s:%s", it.ECO, it.Name)
			if !seen[key] {
				seen[key] = true
				allItems = append(allItems, it)
			}
		}
	}

	return allItems, nil
}

func parseTSVFile(filePath string) ([]mbvmdb.ChessOpeningItem, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var items []mbvmdb.ChessOpeningItem
	scanner := bufio.NewScanner(file)

	// First line is header: eco \t name \t pgn
	isHeader := true
	reCleanMoves := regexp.MustCompile(`\d+\.+`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if isHeader {
			isHeader = false
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}

		eco := strings.TrimSpace(parts[0])
		name := strings.TrimSpace(parts[1])
		pgn := strings.TrimSpace(parts[2])

		// Extract SAN moves
		clean := reCleanMoves.ReplaceAllString(pgn, "")
		movesTokens := strings.Fields(clean)

		var continuations []string
		if len(movesTokens) > 0 {
			lastMove := movesTokens[len(movesTokens)-1]
			continuations = append(continuations, lastMove)
		}

		item := mbvmdb.ChessOpeningItem{
			ECO:           eco,
			Name:          name,
			Moves:         pgn,
			WhiteWin:      38.0,
			DrawRate:      34.0,
			BlackWin:      28.0,
			Continuations: continuations,
			KeyThemes:     []string{fmt.Sprintf("Catálogo Oficial ECO Volume %s", eco[0:1])},
		}

		items = append(items, item)
	}

	return items, scanner.Err()
}
