package chess

import (
	"github.com/kbelludoo/mbvm-db/pkg/mbvmdb"
)

// MasterLessons contains all fundamental chess tactics, checkmate patterns, and endgame techniques.
var MasterLessons = []mbvmdb.ChessOpeningItem{
	// ==================== CHECKMATE PATTERNS ====================
	{
		ECO:           "MAT01",
		Name:          "Anastasia's Mate",
		Moves:         "e4_e5_Nf3_Nc6_Bc4_Nf6_Ng5_d5_exd5_Nxd5_Nxf7_Kxf7_Qf3+_Ke6_Nc3_Ncb4",
		WhiteWin:      100.0,
		Continuations: []string{"Qe4", "a3", "O-O"},
		KeyThemes:     []string{"Anastasia Mate", "Knight and Rook coordination", "King trapped on edge"},
	},
	{
		ECO:           "MAT02",
		Name:          "Arabian Mate",
		Moves:         "e4_e5_Nf3_Nc6_d4_exd4_Nxd4_Nf6_Nc3_Bb4_Nxc6_bxc6_Bd3_d5_exd5_cxd5_O-O_O-O",
		WhiteWin:      100.0,
		Continuations: []string{"Bg5", "Qf3", "Re1"},
		KeyThemes:     []string{"Arabian Mate", "Knight and Rook in the corner", "f7/h7 pressure"},
	},
	{
		ECO:           "MAT03",
		Name:          "Smothered Mate (Philidor Legacy)",
		Moves:         "e4_e5_Nf3_Nc6_Bc4_Bc5_c3_Nf6_d4_exd4_cxd4_Bb4+_Bd2_Bxd2+_Nbxd2_d5_exd5_Nxd5_Qb3_Nce7_O-O_O-O",
		WhiteWin:      100.0,
		Continuations: []string{"Nf7+", "Nh6+", "Qg8+", "Nf7#"},
		KeyThemes:     []string{"Smothered Mate", "Queen sacrifice on g8", "Knight smother on f7"},
	},
	{
		ECO:           "MAT04",
		Name:          "Boden's Mate (Criss-Cross Bishops)",
		Moves:         "e4_e5_Nf3_d6_d4_Nd7_Bc4_c6_O-O_Be7_dxe5_dxe5_Qe2_Ngf6_Rd1_Qc7",
		WhiteWin:      100.0,
		Continuations: []string{"Ng5", "Bxf7+", "Ne6"},
		KeyThemes:     []string{"Boden Mate", "Intersecting diagonal bishops", "Uncastled King"},
	},
	{
		ECO:           "MAT05",
		Name:          "Legal's Mate (Queen Sacrifice Trap)",
		Moves:         "e4_e5_Nf3_d6_Bc4_Bg4_Nc3_g6_Nxe5_Bxd1_Bxf7+_Ke7_Nd5#",
		WhiteWin:      100.0,
		Continuations: []string{"Nd5#"},
		KeyThemes:     []string{"Legal Mate", "Pseudo-queen sacrifice", "Fast checkmate on move 7"},
	},
	{
		ECO:           "MAT06",
		Name:          "Opera Mate (Morphy's Masterpiece)",
		Moves:         "e4_e5_Nf3_d6_d4_Bg4_dxe5_Bxf3_Qxf3_dxe5_Bc4_Nf6_Qb3_Qe7_Nc3_c6_Bg5_b5_Nxb5_cxb5_Bxb5+_Nbd7_O-O-O_Rd8_Rxd7_Rxd7_Rd1_Qe6_Bxd7+_Nxd7_Qb8+_Nxb8_Rd8#",
		WhiteWin:      100.0,
		Continuations: []string{"Qb8+", "Rd8#"},
		KeyThemes:     []string{"Opera Mate", "Rook and Bishop mate", "Pin exploitation"},
	},
	{
		ECO:           "MAT07",
		Name:          "Back Rank Mate (Corredor)",
		Moves:         "e4_e5_Nf3_Nc6_d4_exd4_c3_dxc3_Bc4_cxb2_Bxb2_d6_O-O_Be6_Bxe6_fxe6_Qb3_Qd7_Ng5_Nd8_f4_Nf6",
		WhiteWin:      100.0,
		Continuations: []string{"Re1", "Rad1", "e5"},
		KeyThemes:     []string{"Back Rank", "Trapped King behind pawns", "Heavy piece invasion"},
	},
	{
		ECO:           "MAT08",
		Name:          "Scholar's Mate (Mate do Pastor)",
		Moves:         "e4_e5_Qh5_Nc6_Bc4_Nf6_Qxf7#",
		WhiteWin:      100.0,
		Continuations: []string{"Qxf7#"},
		KeyThemes:     []string{"Scholar Mate", "Weak f7 square", "Early Queen attack"},
	},
	{
		ECO:           "MAT09",
		Name:          "Fool's Mate (Mate do Louco)",
		Moves:         "f3_e5_g4_Qh4#",
		WhiteWin:      0.0,
		Continuations: []string{"Qh4#"},
		KeyThemes:     []string{"Fool Mate", "Weak e1-h4 diagonal", "Fastest checkmate"},
	},
	{
		ECO:           "MAT10",
		Name:          "Blackburne Shilling Mate",
		Moves:         "e4_e5_Nf3_Nc6_Bc4_Nd4_Nxe5_Qg5_Nxf7_Qxg2_Rf1_Qxe4+_Be2_Nf3#",
		WhiteWin:      0.0,
		Continuations: []string{"Qxg2", "Qxe4+", "Nf3#"},
		KeyThemes:     []string{"Blackburne Mate", "Knight fork counter-trap", "Smothered mate on f3"},
	},

	// ==================== BASIC & ADVANCED TACTICS ====================
	{
		ECO:           "TAC01",
		Name:          "The Absolute Pin (Cravada Absoluta)",
		Moves:         "e4_e5_Nf3_Nc6_Bb5_a6_Ba4_Nf6_O-O_Be7_Re1_b5_Bb3_d6_c3_O-O_h3_Nb8_d4_Nbd7_Nbd2_Bb7_Bc2_Re8",
		WhiteWin:      65.0,
		Continuations: []string{"Nf1", "Ng3", "d5"},
		KeyThemes:     []string{"Pin on d7/f6", "Ruy Lopez pin structure", "Tension maintenance"},
	},
	{
		ECO:           "TAC02",
		Name:          "The Knight Fork (Garfo de Cavalo)",
		Moves:         "e4_e5_Nf3_Nc6_Bc4_Nf6_Ng5_d5_exd5_Na5_Bb5+_c6_dxc6_bxc6_Bd3_h6_Ne4_Nd5_O-O_Be7_Re1_O-O",
		WhiteWin:      60.0,
		Continuations: []string{"Bf1", "Nbc3", "c4"},
		KeyThemes:     []string{"Fork threats on c7 and f7", "Tactical deflection"},
	},
	{
		ECO:           "TAC03",
		Name:          "The Skewer (Espeto / Raio-X)",
		Moves:         "d4_d5_c4_c6_Nf3_Nf6_Nc3_e6_Bg5_dxc4_e4_b5_e5_h6_Bh4_g5_Nxg5_hxg5_Bxg5_Nbd7_exf6_Bb7",
		WhiteWin:      70.0,
		Continuations: []string{"g3", "Bg2", "Qe2"},
		KeyThemes:     []string{"Skewer on h4-d8 diagonal", "Botvinnik Semi-Slav tactic"},
	},
	{
		ECO:           "TAC04",
		Name:          "Discovered Attack & Double Check (Ataque a Descoberto)",
		Moves:         "e4_e6_d4_d5_Nc3_Nf6_Bg5_Be7_e5_Nfd7_h4_Bxg5_hxg5_Qxg5_Nh3_Qe7_Nf4_a6_Qg4_g6",
		WhiteWin:      75.0,
		Continuations: []string{"Bd3", "O-O-O", "Nfxd5"},
		KeyThemes:     []string{"Alekhine-Chatard Attack", "Discovered check on h-file", "Sacrifice on d5/g6"},
	},
	{
		ECO:           "TAC05",
		Name:          "Deflection & Decoy (Desvio e Atração)",
		Moves:         "d4_Nf6_c4_g6_Nc3_Bg7_e4_d6_Nf3_O-O_Be2_e5_O-O_Nc6_d5_Ne7_Ne1_Nd7_Nd3_f5_f3_f4",
		WhiteWin:      60.0,
		Continuations: []string{"Bd2", "c5", "b4"},
		KeyThemes:     []string{"King's Indian Classical", "Queenside break vs Kingside attack", "Deflection on c5"},
	},
	{
		ECO:           "TAC06",
		Name:          "Greek Gift Sacrifice (Sacrifício Clássico de Bispo em h7)",
		Moves:         "e4_e6_d4_d5_e5_c5_c3_Nc6_Nf3_Qb6_Bd3_cxd4_cxd4_Bd7_O-O_Nxd4_Nxd4_Qxd4_Nc3_a6_Qe2_Ne7_Kh1_Nc6_f4_Bc5_a3_O-O",
		WhiteWin:      90.0,
		Continuations: []string{"Bxh7+", "Kxh7", "Qh5+", "Kg8", "Rf3", "Rh3#"},
		KeyThemes:     []string{"Greek Gift", "Bxh7+ sacrifice", "Queen & Rook mating net"},
	},
	{
		ECO:           "TAC07",
		Name:          "Interference (Interferência de Linhas)",
		Moves:         "e4_c5_Nf3_d6_d4_cxd4_Nxd4_Nf6_Nc3_a6_Bg5_e6_f4_Qb6_Qd2_Qxb2_Rb1_Qa3_e5_dxe5_fxe5_Nfd7_Ne4_h6_Bh4_Qxa2_Rd1",
		WhiteWin:      80.0,
		Continuations: []string{"Be2", "O-O", "Nd6"},
		KeyThemes:     []string{"Sicilian Najdorf Poisoned Pawn", "Line interference", "Dynamic piece activity"},
	},
	{
		ECO:           "TAC08",
		Name:          "Clearance Sacrifice (Sacrifício de Liberação)",
		Moves:         "e4_c5_Nf3_Nc6_d4_cxd4_Nxd4_Nf6_Nc3_e5_Ndb5_d6_Bg5_a6_Na3_b5_Nd5_Be7_Bxf6_Bxf6_c3_O-O_Nc2_Bg5_a4_bxa4_Rxa4_a5_Bc4_Rb8",
		WhiteWin:      65.0,
		Continuations: []string{"b3", "Nce3", "O-O"},
		KeyThemes:     []string{"Sveshnikov Sicilian", "Outpost on d5", "Square clearance on c2-e3"},
	},

	// ==================== ENDGAME ESSENTIALS ====================
	{
		ECO:           "END01",
		Name:          "Lucena Position (Building a Bridge)",
		Moves:         "e4_e5_d4_exd4_Qxd4_Nc6_Qe3_Nf6_Nc3_d5_exd5_Nxd5_Nxd5_Qxd5_Bd2_Bc5_Qg3_O-O_O-O-O_Qxa2_Bc3_Qa1+_Kd2_Rd8+_Bd3",
		WhiteWin:      100.0,
		Continuations: []string{"Ke2", "Rxd3", "Rf1"},
		KeyThemes:     []string{"Lucena technique", "Bridge with Rook", "Pawn promotion safeguard"},
	},
	{
		ECO:           "END02",
		Name:          "Philidor Defense (Rook 3rd Rank Cutoff)",
		Moves:         "e4_e5_Nf3_d6_d4_Nd7_Bc4_Be7_O-O_Ngf6_dxe5_dxe5_Ng5_O-O_Bxf7+_Rxf7_Ne6_Qe8_Nxc7_Qd8_Nxa8_b6",
		WhiteWin:      65.0,
		Continuations: []string{"Nc3", "Qe2", "Be3"},
		KeyThemes:     []string{"Philidor Cutoff", "Passive defense breakthrough", "Rook behind passed pawn"},
	},
	{
		ECO:           "END03",
		Name:          "The Opposition & Key Squares (Oposição de Reis)",
		Moves:         "e4_e5_Nf3_Nc6_d4_exd4_Nxd4_Bc5_Be3_Qf6_c3_Nge7_Bc4_O-O_O-O_d6_f4_Qg6_Qf3_Bg4_Qg3_Qxe4_Nd2_Qg6",
		WhiteWin:      65.0,
		Continuations: []string{"Rae1", "Bf2", "h3"},
		KeyThemes:     []string{"Direct opposition", "King penetration into enemy territory"},
	},
}

// IngestAllLessons ingests all fundamental lessons into the given MBVM-DB instance.
func IngestAllLessons(db *mbvmdb.DB) int {
	if db == nil {
		return 0
	}
	db.IngestChessOpenings(MasterLessons)
	db.CompileChessBytecode()
	return len(MasterLessons)
}
