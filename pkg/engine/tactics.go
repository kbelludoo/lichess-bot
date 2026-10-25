package engine

import (
	"fmt"
	"strings"
	"time"
)

// ExplainTacticalPlan generates an intuitive Grandmaster-level Portuguese explanation for a move.
func ExplainTacticalPlan(fen string, move string, san string, openingName string, eval string) string {
	if len(move) < 4 {
		if openingName != "" {
			return fmt.Sprintf("Linha teórica de elite: %s", openingName)
		}
		return "Lance posicional recomendado pelo motor."
	}

	from := move[0:2]
	to := move[2:4]
	_ = from
	toCol := to[0]
	toRow := to[1]

	// 1. Roque
	if (from == "e1" && to == "g1") || (from == "e8" && to == "g8") || san == "O-O" {
		return "Roque Pequeno: Protege o Rei na ala do Rei e centraliza a Torre para disputar o jogo."
	}
	if (from == "e1" && to == "c1") || (from == "e8" && to == "c8") || san == "O-O-O" {
		return "Roque Grande: Retira o Rei do centro e ativa a Torre diretamente na coluna aberta 'd'."
	}

	// 2. Promoção de Peão
	if len(move) == 5 {
		promo := strings.ToUpper(string(move[4]))
		return fmt.Sprintf("Coroação de Peão: Promove a %s garantindo vantagem material decisiva!", promo)
	}

	// 3. Identificar peça movida a partir do FEN
	piece := getPieceAt(fen, from)
	targetPiece := getPieceAt(fen, to)
	isCapture := targetPiece != '.'

	if isCapture {
		targetName := pieceName(targetPiece)
		if targetName == "Dama" {
			return "Golpe decisivo: Captura a DAMA adversária com ganho colossal de material (+9)!"
		}
		if targetName == "Torre" {
			return "Ganho de qualidade: Captura a Torre adversária conquistando vantagem material (+5)!"
		}
		if targetName == "Bispo" || targetName == "Cavalo" {
			return fmt.Sprintf("Captura vantajosa: Elimina o %s adversário (+3) e assume a liderança material.", targetName)
		}
		return fmt.Sprintf("Captura de peão em %s: Quebra a estrutura de peões do oponente e abre linhas de ataque.", to)
	}

	// 4. Lances por tipo de peça
	switch strings.ToLower(string(piece)) {
	case "p": // Peão
		if (from == "e2" && to == "e4") || (from == "e7" && to == "e5") {
			return "Domínio central: Ocupa o centro com o peão de Rei e abre diagonais para a Dama e o Bispo."
		}
		if (from == "d2" && to == "d4") || (from == "d7" && to == "d5") {
			return "Domínio central: Estabelece forte presença no centro com o peão de Dama e liberta o bispo."
		}
		if (from == "c2" && to == "c4") || (from == "c7" && to == "c5") {
			return "Contra-golpe na ala da Dama: Pressiona o centro adversário e cria assimetria tática."
		}
		if toRow == '7' || toRow == '2' {
			return fmt.Sprintf("Avanço agressivo: Peão em %s fica a um passo de coroar Dama!", to)
		}
		return fmt.Sprintf("Avanço de peão para %s: Conquista espaço e restringe as peças menores do oponente.", to)

	case "n": // Cavalo
		if to == "c6" || to == "c3" || to == "d5" || to == "e5" || to == "d6" || to == "e6" {
			return fmt.Sprintf("Invasão tática do Cavalo em %s: Posição dominante que sufoca o adversário e cria ameaças mortais de garfo e mate!", to)
		}
		if to == "f3" || to == "f6" {
			return "Desenvolvimento harmônico: Pressiona as casas centrais e prepara o roque."
		}
		return fmt.Sprintf("Manobra de Cavalo para %s: Reposiciona a peça para um posto tático mais agressivo.", to)

	case "b": // Bispo
		if to == "c4" || to == "c5" {
			return fmt.Sprintf("Bispo ativo em %s: Mira diretamente no ponto vulnerável do Rei inimigo.", to)
		}
		if to == "b5" || to == "b4" {
			return fmt.Sprintf("Cravada tática em %s: Imobiliza o Cavalo adversário e disputa o controle do centro.", to)
		}
		if to == "g5" || to == "g4" {
			return fmt.Sprintf("Cravada agressiva em %s: Pressiona a Dama/Cavalo do adversário e força concessões defensivas.", to)
		}
		return fmt.Sprintf("Desenvolvimento do Bispo para %s: Domina uma diagonal aberta crucial para a posição.", to)

	case "r": // Torre
		if toCol == 'd' || toCol == 'e' {
			return fmt.Sprintf("Centralização da Torre em %s: Domina a coluna central semi-aberta para infiltração.", to)
		}
		if toRow == '7' || toRow == '2' {
			return fmt.Sprintf("Torre na 7ª fila (%s): Posto devastador atacando a base de peões do oponente.", to)
		}
		return fmt.Sprintf("Ativação da Torre para %s: Coloca a peça pesada em uma coluna ativa.", to)

	case "q": // Dama
		if toRow == '7' || toRow == '2' || toRow == '8' || toRow == '1' {
			return fmt.Sprintf("Infiltração da Dama em %s: Ameaça direta de mate ou ganho massivo de material!", to)
		}
		return fmt.Sprintf("Manobra da Dama para %s: Coordena o ataque com as peças menores.", to)

	case "k": // Rei
		return fmt.Sprintf("Segurança do Rei: Move para %s saindo de cravadas e ameaças táticas.", to)
	}

	if openingName != "" {
		return fmt.Sprintf("Plano da %s: Lance teórico sólido de grande mestre.", openingName)
	}
	return fmt.Sprintf("Lance tático em %s otimizando a avaliação da posição.", to)
}

func getPieceAt(fen string, sq string) rune {
	if len(sq) < 2 {
		return '.'
	}
	parts := strings.Split(fen, " ")
	if len(parts) == 0 {
		return '.'
	}
	boardPart := parts[0]
	rows := strings.Split(boardPart, "/")
	if len(rows) != 8 {
		return '.'
	}

	targetCol := int(sq[0] - 'a')
	targetRow := 8 - int(sq[1]-'0')

	if targetRow < 0 || targetRow >= 8 || targetCol < 0 || targetCol >= 8 {
		return '.'
	}

	currCol := 0
	for _, ch := range rows[targetRow] {
		if ch >= '1' && ch <= '8' {
			currCol += int(ch - '0')
		} else {
			if currCol == targetCol {
				return ch
			}
			currCol++
		}
	}
	return '.'
}

func pieceName(p rune) string {
	switch strings.ToLower(string(p)) {
	case "p":
		return "Peão"
	case "n":
		return "Cavalo"
	case "b":
		return "Bispo"
	case "r":
		return "Torre"
	case "q":
		return "Dama"
	case "k":
		return "Rei"
	default:
		return "Peça"
	}
}

// DetectOpponentThreat analyzes the last move of the opponent and explains what danger they created.
func DetectOpponentThreat(lastMove string, moves []string) string {
	if lastMove == "" {
		return "Oponente acabou de iniciar. Nenhuma ameaça imediata no tabuleiro."
	}
	clean := strings.TrimSpace(lastMove)
	if strings.Contains(clean, "+") {
		return "⚠️ XEQUE DIRETO: O adversário atacou o seu Rei! A defesa ou fuga é prioritária."
	}
	if strings.HasPrefix(clean, "Qh5") || strings.HasPrefix(clean, "Bc4") {
		return "⚠️ AMEAÇA DE MATE: Oponente tentando montar o ataque do Mate do Pastor sobre f7/f2!"
	}
	if strings.HasPrefix(clean, "N") && (strings.Contains(clean, "g5") || strings.Contains(clean, "g4") || strings.Contains(clean, "e5")) {
		return "⚠️ PRESSÃO TÁTICA: Cavalo inimigo infiltrando em casa avançada com apoio de peças."
	}
	if strings.Contains(clean, "x") {
		return "⚠️ CAPTURA RECENTE: O oponente capturou uma peça e alterou a estrutura do jogo."
	}
	if strings.HasPrefix(clean, "B") {
		return "⚠️ DIAGONAL ATIVA: O Bispo adversário aponta para cravada ou pressão em suas peças."
	}
	return "Posicionamento do oponente: O oponente busca expansão e controle posicional de casas."
}

// CalculateHumanDelay returns a natural, randomized human delay in seconds.
func CalculateHumanDelay(plyCount int, isCapture bool, isCheck bool) float64 {
	var base float64
	if plyCount <= 6 {
		base = 0.8 + float64(time.Now().UnixNano()%600)/1000.0
	} else if isCheck || isCapture {
		base = 0.7 + float64(time.Now().UnixNano()%900)/1000.0
	} else if plyCount > 24 {
		base = 1.2 + float64(time.Now().UnixNano()%1300)/1000.0
	} else {
		base = 1.8 + float64(time.Now().UnixNano()%2000)/1000.0
	}
	return base
}
