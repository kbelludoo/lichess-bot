package chess

import (
	"strings"
)

// OpeningTrap represents a fast-win trap with sequential moves, target blunder, and punishing response.
type OpeningTrap struct {
	Name        string
	PlayingAs   string   // "w" or "b"
	MovesPrefix []string // Sequence leading up to our trap move
	OurMove     string   // UCI move to play (e.g. "c6d4")
	SAN         string   // Standard algebraic notation (e.g. "Nd4")
	Theme       string   // What trap is being set
	IsFinish    bool     // True if this move delivers checkmate or wins the Queen
}

var killerTraps = []OpeningTrap{
	// ==========================================
	// JOGANDO DE BRANCAS (WHITE)
	// ==========================================
	// ==========================================
	// 0. ATAQUE GROB (1. g4 - ABERTURA MAIS ESTRANHA E BIZARRA DO XADREZ)
	// ==========================================
	{
		Name:        "Grob - Armadilha do Bispo",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "d5"},
		OurMove:     "f1g2",
		SAN:         "Bg2",
		Theme:       "👽 ATAQUE GROB: Bispo ataca d5 e mira na Torre de b7 com raio-X mortal!",
	},
	{
		Name:        "Grob - Contra-Golpe c4",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "d5", "Bg2", "Bxg4"},
		OurMove:     "c2c4",
		SAN:         "c4",
		Theme:       "🔥 O PEÃO DE g4 ERA ISCA! Ruptura c4 destrói o centro das pretas e abre caminho para a Dama!",
	},
	{
		Name:        "Grob - Ataque Duplo de Dama",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "d5", "Bg2", "Bxg4", "c4", "c6"},
		OurMove:     "d1b3",
		SAN:         "Qb3",
		Theme:       "👽 GROB - DUPLO FATAL: Dama mira em b7 e d5 simultaneamente! As pretas entram em colapso!",
	},
	{
		Name:        "Grob - Invasão em b7",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "d5", "Bg2", "Bxg4", "c4", "dxc4"},
		OurMove:     "g2b7",
		SAN:         "Bxb7",
		Theme:       "🏆 GROB VENCE MATERIAL: Bispo captura em b7 e a Torre preta de a8 está perdida!",
		IsFinish:    true,
	},
	{
		Name:        "Grob contra 1... e5",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "e5"},
		OurMove:     "f1g2",
		SAN:         "Bg2",
		Theme:       "👽 ATAQUE GROB: Bispo em g2 prepara c4 para estraçalhar o centro!",
	},
	{
		Name:        "Grob contra 1... e5 - Ruptura c4",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "e5", "Bg2", "d5"},
		OurMove:     "c2c4",
		SAN:         "c4",
		Theme:       "Grob: Ruptura c4 dinamitando os peões pretos!",
	},
	{
		Name:        "Grob contra 1... e5 - Ataque com c4",
		PlayingAs:   "w",
		MovesPrefix: []string{"g4", "e5", "Bg2", "Nc6"},
		OurMove:     "c2c4",
		SAN:         "c4",
		Theme:       "Grob: Ruptura c4 imediata dominando a ala da Dama!",
	},

	// ==========================================
	// 1. GAMBITO SMITH-MORRA (CONTRA A SICILIANA 1... c5)
	// ==========================================
	{
		Name:        "Smith-Morra - Ruptura d4",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5"},
		OurMove:     "d2d4",
		SAN:         "d4",
		Theme:       "🔥 GAMBITO SMITH-MORRA: Ruptura d4 imediata no lance 2! Destrói a Siciliana!",
	},
	{
		Name:        "Gambito Smith-Morra - Sacrifício de Peão",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4"},
		OurMove:     "c2c3",
		SAN:         "c3",
		Theme:       "🔥 GAMBITO SMITH-MORRA: Sacrifica o peão de c3 para abrir as colunas c e d para as Torres!",
	},
	{
		Name:        "Smith-Morra Aceito - Ativação Total",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3"},
		OurMove:     "b1c3",
		SAN:         "Nxc3",
		Theme:       "Smith-Morra: Cavalo entra com vigor! Desenvolvimento 3x mais rápido que as pretas!",
	},
	{
		Name:        "Smith-Morra - Bispo Mira f7",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3", "Nxc3", "Nc6"},
		OurMove:     "f1c4",
		SAN:         "Bc4",
		Theme:       "Smith-Morra: Bispo aponta como mira telescópica para o ponto f7 do Rei preto!",
	},
	{
		Name:        "Smith-Morra - Cavalo em f3",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3", "Nxc3", "Nc6", "Bc4", "d6"},
		OurMove:     "g1f3",
		SAN:         "Nf3",
		Theme:       "Smith-Morra: Desenvolve o Cavalo controlando as casas de invasão!",
	},
	{
		Name:        "Smith-Morra - O Golpe e5",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3", "Nxc3", "Nc6", "Bc4", "d6", "Nf3", "Nf6"},
		OurMove:     "e4e5",
		SAN:         "e5",
		Theme:       "🔥 GOLPE DO SMITH-MORRA: Ruptura e5 mortal! Se dxe5, segue sacrifício Bxf7+ que ganha a Dama!",
	},
	{
		Name:        "Smith-Morra - Sacrifício em f7",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3", "Nxc3", "Nc6", "Bc4", "d6", "Nf3", "Nf6", "e5", "dxe5"},
		OurMove:     "c4f7",
		SAN:         "Bxf7+",
		Theme:       "🏆 SACRIFÍCIO EM f7: O Rei preto é forçado a comer em f7!",
	},
	{
		Name:        "Smith-Morra - Captura da Dama",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3", "Nxc3", "Nc6", "Bc4", "d6", "Nf3", "Nf6", "e5", "dxe5", "Bxf7+", "Kxf7"},
		OurMove:     "d1d8",
		SAN:         "Qxd8",
		Theme:       "🏆 DAMA CAPTURADA NO LANCE 8! O Gambito Smith-Morra liquidou a Siciliana!",
		IsFinish:    true,
	},

	// ==========================================
	// 2. GAMBITO DANÊS / NÓRDICO (CONTRA 1... e5)
	// ==========================================
	{
		Name:        "Gambito Danês - Ruptura d4",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5"},
		OurMove:     "d2d4",
		SAN:         "d4",
		Theme:       "🔥 GAMBITO DANÊS: Ruptura d4 no lance 2! O gambito mais destrutivo contra 1... e5!",
	},
	{
		Name:        "Gambito Danês - Primeiro Sacrifício",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "d4", "exd4"},
		OurMove:     "c2c3",
		SAN:         "c3",
		Theme:       "🔥 GAMBITO DANÊS: Oferece peão central para abrir as grandes diagonais!",
	},
	{
		Name:        "Gambito Danês - Segundo Sacrifício",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "d4", "exd4", "c3", "dxc3"},
		OurMove:     "f1c4",
		SAN:         "Bc4",
		Theme:       "Gambito Danês: Entrega o segundo peão em b2 para mirar com fúria na casa f7!",
	},
	{
		Name:        "Gambito Danês - Bispos Monstros",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "d4", "exd4", "c3", "dxc3", "Bc4", "cxb2"},
		OurMove:     "c1b2",
		SAN:         "Bxb2",
		Theme:       "Gambito Danês Completo: Dois Bispos monstros cortam o tabuleiro direto no Rei preto!",
	},
	{
		Name:        "Armação do Mate do Pastor",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Qh5", "Nc6"},
		OurMove:     "f1c4",
		SAN:         "Bc4",
		Theme:       "Bispo mira f7! Se o oponente jogar Nf6 ou qualquer lance passivo, é XEQUE-MATE imediato!",
	},
	{
		Name:        "XEQUE-MATE DO PASTOR",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Qh5", "Nc6", "Bc4", "Nf6"},
		OurMove:     "h5f7",
		SAN:         "Qxf7#",
		Theme:       "🏆 XEQUE-MATE NO LANCE 4! A Dama apoiada pelo Bispo fulmina o Rei inimigo!",
		IsFinish:    true,
	},

	// 2. Mate de Légal (Legal's Mate - Mate em 7 lances)
	{
		Name:        "Mate de Légal (Sacrifício da Dama)",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "d6", "Bc4", "Bg4", "Nc3", "g6"},
		OurMove:     "f3e5",
		SAN:         "Nxe5",
		Theme:       "Sacrifício brutal da Dama! Se as pretas tomarem a Dama em d1, levam mate em 2 lances!",
	},
	{
		Name:        "Mate de Légal - O Golpe",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "d6", "Bc4", "Bg4", "Nc3", "g6", "Nxe5", "Bxd1"},
		OurMove:     "c4f7",
		SAN:         "Bxf7+",
		Theme:       "Xeque no Rei! O Rei preto é forçado para e7.",
	},
	{
		Name:        "XEQUE-MATE DE LÉGAL",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "d6", "Bc4", "Bg4", "Nc3", "g6", "Nxe5", "Bxd1", "Bxf7+", "Ke7"},
		OurMove:     "c3d5",
		SAN:         "Nd5#",
		Theme:       "🏆 XEQUE-MATE COM 3 PEÇAS MENORES! Uma das finalizações mais lendárias da história!",
		IsFinish:    true,
	},

	// 3. Ataque Fegatello (Fried Liver Attack)
	{
		Name:        "Ataque Fegatello (Fried Liver)",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nf6"},
		OurMove:     "f3g5",
		SAN:         "Ng5",
		Theme:       "Dupla pressão mortal sobre a casa fraca f7 do oponente!",
	},
	{
		Name:        "Fegatello - Sacrifício em f7",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nf6", "Ng5", "d5", "exd5", "Nxd5"},
		OurMove:     "g5f7",
		SAN:         "Nxf7",
		Theme:       "Sacrifício destruidor de Cavalo em f7! Arranca o Rei preto para o meio do tabuleiro!",
	},
	{
		Name:        "Fegatello - Ataque com Dama",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nf6", "Ng5", "d5", "exd5", "Nxd5", "Nxf7", "Kxf7"},
		OurMove:     "d1f3",
		SAN:         "Qf3+",
		Theme:       "Xeque duplo com Dama e Bispo! O Rei adversário está indefeso e a posição desmorona!",
	},

	// ==========================================
	// GAMBITOS AGRESSIVOS (ALTO RISCO / MIKHAIL TAL)
	// ==========================================
	// Gambito Evans (Sacrifício em b4 para demolição do centro)
	{
		Name:        "Gambito Evans (Sacrifício em b4)",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Bc5"},
		OurMove:     "b2b4",
		SAN:         "b4",
		Theme:       "🔥 GAMBITO EVANS: Sacrifício do peão de b4 para ganhar tempos rápidos e demolir o centro!",
	},
	{
		Name:        "Gambito Evans - Ataque ao Bispo",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Bc5", "b4", "Bxb4"},
		OurMove:     "c2c3",
		SAN:         "c3",
		Theme:       "Gambito Evans: Ganha tempo atacando o Bispo preto e preparando a ruptura d4!",
	},
	{
		Name:        "Gambito Evans - Ruptura Central",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Bc5", "b4", "Bxb4", "c3", "Bc5"},
		OurMove:     "d2d4",
		SAN:         "d4",
		Theme:       "Gambito Evans: Ruptura d4 com centro avassalador e ataque direto ao Rei!",
	},
	{
		Name:        "Gambito Evans - Variante Ba5",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Bc5", "b4", "Bxb4", "c3", "Ba5"},
		OurMove:     "d2d4",
		SAN:         "d4",
		Theme:       "Gambito Evans: Ruptura d4 esmagadora dominando todas as colunas!",
	},

	// Gambito Smith-Morra (Devastação contra a Defesa Siciliana)
	{
		Name:        "Gambito Smith-Morra",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4"},
		OurMove:     "c2c3",
		SAN:         "c3",
		Theme:       "🔥 GAMBITO SMITH-MORRA: Sacrifica o peão de c3 para abrir colunas c e d para as Torres!",
	},
	{
		Name:        "Smith-Morra Aceito - Ativação Total",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "c5", "d4", "cxd4", "c3", "dxc3"},
		OurMove:     "b1c3",
		SAN:         "Nxc3",
		Theme:       "Smith-Morra Aceito: Cavalo entra com vigor! Desenvolvimento 3x mais rápido que as pretas!",
	},

	// Gambito Danês / Nórdico (Sacrifício de 2 peões por ataque de mate)
	{
		Name:        "Gambito Danês - Primeiro Sacrifício",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "d4", "exd4"},
		OurMove:     "c2c3",
		SAN:         "c3",
		Theme:       "🔥 GAMBITO DANÊS: Oferece peão central para abrir as grandes diagonais!",
	},
	{
		Name:        "Gambito Danês - Segundo Sacrifício",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "d4", "exd4", "c3", "dxc3"},
		OurMove:     "f1c4",
		SAN:         "Bc4",
		Theme:       "Gambito Danês: Entrega o segundo peão em b2 para mirar com fúria na casa f7!",
	},
	{
		Name:        "Gambito Danês - Bispos Assassinos",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "d4", "exd4", "c3", "dxc3", "Bc4", "cxb2"},
		OurMove:     "c1b2",
		SAN:         "Bxb2",
		Theme:       "Gambito Danês Completo: Dois Bispos monstros cortam o tabuleiro direto no Rei preto!",
	},

	// Gambito do Rei
	{
		Name:        "Gambito do Rei",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5"},
		OurMove:     "f2f4",
		SAN:         "f4",
		Theme:       "🔥 GAMBITO DO REI: O ataque mais lendário e selvagem do xadrez romântico!",
	},

	// ==========================================
	// MODO BERSERK / ALTÍSSIMO RISCO (SACRIFÍCIOS DE PEÇA NO LANCE 4)
	// ==========================================
	// Gambito de Halloween (Sacrifício de Cavalo no lance 4 na Quatro Cavalos)
	{
		Name:        "Gambito de Halloween (Sacrifício de Cavalo)",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Nc3", "Nf6"},
		OurMove:     "f3e5",
		SAN:         "Nxe5",
		Theme:       "💀 GAMBITO DE HALLOWEEN: Sacrifício brutal de Cavalo no lance 4! Domina o centro com d4 e e5 para caçar o Rei preto!",
	},
	{
		Name:        "Halloween - Expulsão Central",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Nc3", "Nf6", "Nxe5", "Nxe5"},
		OurMove:     "d2d4",
		SAN:         "d4",
		Theme:       "Halloween: Ruptura d4 com ganho de tempo fulminante em cima do Cavalo preto!",
	},
	{
		Name:        "Halloween - Onda de Peões Mortais",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Nc3", "Nf6", "Nxe5", "Nxe5", "d4", "Ng6"},
		OurMove:     "e4e5",
		SAN:         "e5",
		Theme:       "Halloween: O rolo compressor avança! O Cavalo de f6 não tem casas e a posição preta entra em colapso!",
	},

	// Gambito Cochrane (Sacrifício em f7 contra a Petrov)
	{
		Name:        "Gambito Cochrane (Sacrifício em f7)",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "d6"},
		OurMove:     "e5f7",
		SAN:         "Nxf7",
		Theme:       "💀 GAMBITO COCHRANE: Sacrifício destrutivo de Cavalo em f7 no lance 4! Arranca o Rei preto para o centro do tabuleiro!",
	},
	{
		Name:        "Cochrane - Centro Dominante",
		PlayingAs:   "w",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "d6", "Nxf7", "Kxf7"},
		OurMove:     "d2d4",
		SAN:         "d4",
		Theme:       "Cochrane: Centro massivo com d4 e ataque implacável ao Rei desabrigado!",
	},

	// ==========================================
	// JOGANDO DE PRETAS (BLACK)
	// ==========================================
	// 4. Armadilha de Blackburne-Shilling (Mate sufocado em 7 lances)
	{
		Name:        "Armadilha de Blackburne-Shilling",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4"},
		OurMove:     "c6d4",
		SAN:         "Nd4",
		Theme:       "Isca mortal! O peão de e5 parece indefeso. Se as brancas tomarem em e5, caem na armadilha!",
	},
	{
		Name:        "Blackburne-Shilling - Ataque Duplo",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nd4", "Nxe5"},
		OurMove:     "d8g5",
		SAN:         "Qg5",
		Theme:       "Golpe fulminante! A Dama ataca simultaneamente o Cavalo de e5 e o peão de g2!",
	},
	{
		Name:        "Blackburne-Shilling - Invasão em g2",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nd4", "Nxe5", "Qg5", "Nxf7"},
		OurMove:     "g5g2",
		SAN:         "Qxg2",
		Theme:       "Captura em g2 ameaçando a Torre branca de h1!",
	},
	{
		Name:        "Blackburne-Shilling - Xeque Central",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nd4", "Nxe5", "Qg5", "Nxf7", "Qxg2", "Rf1"},
		OurMove:     "g2e4",
		SAN:         "Qxe4+",
		Theme:       "Xeque no Rei! O Bispo branco é obrigado a cobrir em e2.",
	},
	{
		Name:        "XEQUE-MATE DE BLACKBURNE",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bc4", "Nd4", "Nxe5", "Qg5", "Nxf7", "Qxg2", "Rf1", "Qxe4+", "Be2"},
		OurMove:     "d4f3",
		SAN:         "Nf3#",
		Theme:       "🏆 XEQUE-MATE SUFOCADO NO LANCE 7! O Cavalo crava o Rei adversário encurralado por suas próprias peças!",
		IsFinish:    true,
	},

	// 5. Gambito Stafford (Stafford Trap - Mate em 8 lances)
	{
		Name:        "Entrando no Gambito Stafford",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3"},
		OurMove:     "g8f6",
		SAN:         "Nf6",
		Theme:       "Defesa Petrov agressiva preparando o Gambito Stafford.",
	},
	{
		Name:        "Gambito Stafford (Isca tática)",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5"},
		OurMove:     "b8c6",
		SAN:         "Nc6",
		Theme:       "Gambito Stafford! Oferece o Cavalo para abrir diagonais mortais contra o Rei branco.",
	},
	{
		Name:        "Stafford - Abertura de Linhas",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "Nc6", "Nxc6"},
		OurMove:     "d7c6",
		SAN:         "dxc6",
		Theme:       "Retoma com peão de d, abrindo o caminho para a Dama e ambos os Bispos atacarem!",
	},
	{
		Name:        "Stafford - Mira no ponto f2",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "Nc6", "Nxc6", "dxc6", "d3"},
		OurMove:     "f8c5",
		SAN:         "Bc5",
		Theme:       "Bispo aponta direto para a casa f2 ao lado do Rei branco!",
	},
	{
		Name:        "Stafford - Sacrifício de Dama",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "Nc6", "Nxc6", "dxc6", "d3", "Bc5", "Bg5"},
		OurMove:     "f6e4",
		SAN:         "Nxe4",
		Theme:       "Sacrifício de Dama! Se as brancas tomarem Bxd8?, levam mate em 2 lances!",
	},
	{
		Name:        "Stafford - Xeque em f2",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "Nc6", "Nxc6", "dxc6", "d3", "Bc5", "Bg5", "Nxe4", "Bxd8"},
		OurMove:     "c5f2",
		SAN:         "Bxf2+",
		Theme:       "Xeque no Rei! O Rei branco é forçado a ir para e2.",
	},
	{
		Name:        "XEQUE-MATE DO GAMBITO STAFFORD",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nf6", "Nxe5", "Nc6", "Nxc6", "dxc6", "d3", "Bc5", "Bg5", "Nxe4", "Bxd8", "Bxf2+", "Ke2"},
		OurMove:     "c8g4",
		SAN:         "Bg4#",
		Theme:       "🏆 XEQUE-MATE NO LANCE 8! O Bispo de casas claras finaliza a partida com estilo!",
		IsFinish:    true,
	},

	// 6. Gambito Englund (Mate em 8 lances contra 1. d4)
	{
		Name:        "Gambito Englund",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4"},
		OurMove:     "e7e5",
		SAN:         "e5",
		Theme:       "Desafia 1. d4 imediatamente com a armadilha do Gambito Englund!",
	},
	{
		Name:        "Englund - Pressão em e5",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "e5", "dxe5"},
		OurMove:     "b8c6",
		SAN:         "Nc6",
		Theme:       "Desenvolve o Cavalo pressionando a recuperação do peão central.",
	},
	{
		Name:        "Englund - Tripla Ameaça",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "e5", "dxe5", "Nc6", "Nf3"},
		OurMove:     "d8e7",
		SAN:         "Qe7",
		Theme:       "Dama mira e5 e prepara o ataque fulminante de Dama em b4!",
	},
	{
		Name:        "Englund - Duplo no Rei e Bispo",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "e5", "dxe5", "Nc6", "Nf3", "Qe7", "Bf4"},
		OurMove:     "e7b4",
		SAN:         "Qb4+",
		Theme:       "Xeque duplo! Ataca o Rei e o Bispo de f4 simultaneamente!",
	},
	{
		Name:        "Englund - Invasão em b2",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "e5", "dxe5", "Nc6", "Nf3", "Qe7", "Bf4", "Qb4+", "Bd2"},
		OurMove:     "b4b2",
		SAN:         "Qxb2",
		Theme:       "Captura em b2 ameaçando a Torre branca de a1!",
	},
	{
		Name:        "Englund - Cravada fatal no Bispo",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "e5", "dxe5", "Nc6", "Nf3", "Qe7", "Bf4", "Qb4+", "Bd2", "Qxb2", "Bc3"},
		OurMove:     "f8b4",
		SAN:         "Bb4",
		Theme:       "Crava o Bispo de c3 contra o Rei branco! A Dama branca não pode cobrir.",
	},
	{
		Name:        "XEQUE-MATE DO GAMBITO ENGLUND",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "e5", "dxe5", "Nc6", "Nf3", "Qe7", "Bf4", "Qb4+", "Bd2", "Qxb2", "Bc3", "Bb4", "Qd2"},
		OurMove:     "b4c3",
		SAN:         "Bxc3",
		Theme:       "Elimina o Bispo e prepara mate com Qc1# no próximo lance!",
	},

	// 7. Armadilha de Kieninger (Gambito de Budapeste - Mate no lance 8)
	{
		Name:        "Gambito de Budapeste",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "Nf6", "c4"},
		OurMove:     "e7e5",
		SAN:         "e5",
		Theme:       "🔥 GAMBITO DE BUDAPESTE: Contra-ataque imediato sobre o centro branco!",
	},
	{
		Name:        "Budapeste - Salto de Cavalo",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "Nf6", "c4", "e5", "dxe5"},
		OurMove:     "f6g4",
		SAN:         "Ng4",
		Theme:       "Cavalo ataca o peão de e5 e prepara o ataque conjunto com o Bispo!",
	},
	{
		Name:        "Budapeste - Tripla Pressão",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "Nf6", "c4", "e5", "dxe5", "Ng4", "Bf4", "Nc6", "Nf3"},
		OurMove:     "f8b4+",
		SAN:         "Bb4+",
		Theme:       "Xeque no Rei branco com o Bispo, forçando Nbd2!",
	},
	{
		Name:        "Budapeste - Armação da Isca",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "Nf6", "c4", "e5", "dxe5", "Ng4", "Bf4", "Nc6", "Nf3", "Bb4+", "Nbd2"},
		OurMove:     "d8e7",
		SAN:         "Qe7",
		Theme:       "Dama adiciona terceira pressão ao peão de e5. Se as brancas jogarem a3 e tomarem o Bispo...",
	},
	{
		Name:        "Budapeste - Cavalo Central",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "Nf6", "c4", "e5", "dxe5", "Ng4", "Bf4", "Nc6", "Nf3", "Bb4+", "Nbd2", "Qe7", "a3"},
		OurMove:     "g4e5",
		SAN:         "Ngxe5",
		Theme:       "Recupera o peão! Se as brancas jogarem axb4??, levam XEQUE-MATE IMEDIATO!",
	},
	{
		Name:        "XEQUE-MATE DE KIENINGER",
		PlayingAs:   "b",
		MovesPrefix: []string{"d4", "Nf6", "c4", "e5", "dxe5", "Ng4", "Bf4", "Nc6", "Nf3", "Bb4+", "Nbd2", "Qe7", "a3", "Ngxe5", "axb4"},
		OurMove:     "e5d3",
		SAN:         "Nd3#",
		Theme:       "🏆 XEQUE-MATE SUFOCADO DE KIENINGER! O Cavalo crava mate em d3 porque o peão de e2 está cravado pela Dama!",
		IsFinish:    true,
	},

	// 8. Armadilha da Arca de Noé (Ruy Lopez - Encurralamento de Bispo)
	{
		Name:        "Arca de Noé - Preparação",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "d6", "d4", "b5", "Bb3"},
		OurMove:     "c6d4",
		SAN:         "Nxd4",
		Theme:       "Elimina o Cavalo defensor e prepara a armadilha de peões contra o Bispo espanhol!",
	},
	{
		Name:        "Arca de Noé - Avanço c5",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "d6", "d4", "b5", "Bb3", "Nxd4", "Nxd4", "exd4", "Qxd4"},
		OurMove:     "c7c5",
		SAN:         "c5",
		Theme:       "Ataca a Dama e prepara c4 para engaiolar o Bispo de b3!",
	},
	{
		Name:        "Arca de Noé - O Golpe Final",
		PlayingAs:   "b",
		MovesPrefix: []string{"e4", "e5", "Nf3", "Nc6", "Bb5", "a6", "Ba4", "d6", "d4", "b5", "Bb3", "Nxd4", "Nxd4", "exd4", "Qxd4", "c5", "Qd5", "Be6", "Qc6+", "Bd7", "Qd5"},
		OurMove:     "c5c4",
		SAN:         "c4",
		Theme:       "🏆 ARCA DE NOÉ CONCLUÍDA! O Bispo branco de b3 está totalmente preso e será capturado no próximo lance!",
		IsFinish:    true,
	},
}

// CheckTrapMove checks if the current sequence of moves matches a deadly trap line.
func CheckTrapMove(moves []string, playingAs string) (move string, san string, theme string, found bool) {
	cleanMoves := make([]string, 0, len(moves))
	for _, m := range moves {
		t := strings.TrimSpace(m)
		if t != "" && !strings.Contains(t, ".") {
			cleanMoves = append(cleanMoves, t)
		}
	}

	for _, trap := range killerTraps {
		if trap.PlayingAs != "" && trap.PlayingAs != playingAs {
			continue
		}

		if len(cleanMoves) == len(trap.MovesPrefix) {
			match := true
			for i := range cleanMoves {
				if !strings.EqualFold(cleanMoves[i], trap.MovesPrefix[i]) {
					match = false
					break
				}
			}
			if match {
				return trap.OurMove, trap.SAN, trap.Theme, true
			}
		}
	}

	return "", "", "", false
}
