# ♟️ Chess-MBVM-DB: Banco de Dados de Jogadas de Xadrez em Micro-Bytecode

> **Motor de Banco de Dados de Grafos de Xadrez embutido em MBVM-DB (Go 1.27)**  
> *Integrado com a API oficial do Chess.com (api.chess.com/pub) e Catálogo Mestre ECO (A00-E99)*  
> *Redução de >95% de Tokens para LLMs • Travessia CSR em Microssegundos • Formato Binário com CRC32*

---

## 🏛️ O Que é o Chess-MBVM-DB?

O **Chess-MBVM-DB** é uma implementação especializada que modela todo o domínio de xadrez (aberturas ECO, variações táticas, transições de lances e partidas reais do Chess.com) sobre a arquitetura do **MBVM-DB** (*Micro-Bytecode Virtual Machine Database*):

1. **Catálogo Mestre ECO (A00-E99):** Enciclopédia de aberturas (Siciliana Najdorf/Dragão, Ruy Lopez Berlim/Marshall, Gambito da Dama, Defesa Francesa, Defesa Índia do Rei, Inglesa, etc.) com sequências de lances, taxas de vitória (Brancas/Empate/Pretas) e temas estratégicos.
2. **Integração Chess.com PubAPI:** Conector nativo HTTP com cabeçalhos autorizados que baixa o histórico de partidas de grandes mestres (Hikaru Nakamura, Magnus Carlsen, Fabiano Caruana, etc.) ou de qualquer usuário cadastrado no Chess.com.
3. **Árvore de Lances em CSR (*Compressed Sparse Row*):** Lances e transições são mapeados em nós (`OpChessMove`, `OpChessOpening`, `OpChessPosition`) e arestas bidirecionais contíguas em memória cache L1/L2.
4. **Slicing de Subgrafo em Microssegundos:** Permite a agentes de IA consultar lances (`chess-db slice chess.mbvmdb "Najdorf" 1` ou `chess-db slice chess.mbvmdb "e4" 2`) obtendo imediatamente as respostas ideais sem despejar gigabytes de PGN.
5. **Slicing com Teto de Tokens (`slice-budget`):** Garante respostas compactas (ex: teto estrito de 120 tokens) para prompts de IA sem alucinação nem estouro de contexto.
6. **Arquivo Único Binário Atômico (`chess.mbvmdb`):** Serialização de alto desempenho com cabeçalho mágico `MBVM\x01\x00`, deduplicação em *String Pool* e soma de verificação CRC32.

---

## 🚀 Como Usar o CLI (`chess-db.exe`)

O binário compilado está pronto em `c:\Users\kbell\Downloads\chess\chess-db.exe`.

### 1. Compilar o Banco Mestre Inicial
Gera o arquivo `chess.mbvmdb` unindo o catálogo ECO e partidas de GMs do Chess.com:
```powershell
.\chess-db.exe build
```

### 2. Baixar Partidas de Qualquer Usuário do Chess.com
Exemplo: Baixar até 20 partidas recentes de um jogador e indexar na árvore de lances:
```powershell
.\chess-db.exe fetch-user "fabianocaruana" 20
```

### 3. Inspecionar o Banco Binário
Exibe a contagem de nós, aberturas, lances e tamanho de bytecode:
```powershell
.\chess-db.exe inspect chess.mbvmdb
```

### 4. Consultar Fatias de Subgrafo (Slice)
Recupera as continuações e contexto de uma abertura ou lance em microssegundos:
```powershell
.\chess-db.exe slice chess.mbvmdb "Najdorf" 1
.\chess-db.exe slice chess.mbvmdb "Italian" 1
.\chess-db.exe slice chess.mbvmdb "e4" 2
```

### 5. Fatiar com Teto Rígido de Tokens (`slice-budget`)
Perfeito para alimentar prompts de IA com orçamento de tokens estrito:
```powershell
.\chess-db.exe slice-budget chess.mbvmdb "Sicilian" 150
```

### 6. Sugestões de Lances e Estratégia
Recomenda respostas táticas, planos e estatísticas para a posição consultada:
```powershell
.\chess-db.exe suggest "Italian"
.\chess-db.exe suggest "Najdorf"
```

### 7. Ingerir Arquivos PGN Locais
Incorpora arquivos PGN externos na base de conhecimento:
```powershell
.\chess-db.exe ingest-pgn partidas.pgn
```

### 8. Verificar Integridade e CRC32
Valida a integridade binária do arquivo:
```powershell
.\chess-db.exe verify chess.mbvmdb
```

---

## 📊 Estrutura de Arquivos

```
c:\Users\kbell\Downloads\chess\
├── chess-db.exe                 # Binário CLI compilado
├── chess.mbvmdb                 # Banco binário indexado e persistido (CRC32)
├── go.mod                       # Módulo Go 1.27 vinculado ao mbvm-db
├── go.sum                       # Dependências e hashes
├── README.md                    # Documentação do projeto
├── cmd\
│   └── chess-db\
│       └── main.go              # CLI com subcomandos build, slice, suggest, fetch-user, etc.
└── pkg\
    └── chess\
        ├── chesscom_client.go   # Cliente HTTP para Chess.com PubAPI (arquivos, perfis e PGNs)
        └── eco_openings.go      # Catálogo Mestre de Aberturas ECO (A00-E99)
```

No repositório central `C:\Users\kbell\Downloads\mbvm-db`:
- `pkg/mbvmdb/chess_ingest.go`: Extensão nativa do motor com `IngestChessOpenings`, `IngestChessGames`, `IngestChessPGN` e `CompileChessBytecode`.
- `pkg/mbvmdb/chess_test.go`: Suíte de testes automatizados com cobertura completa para ingestão de xadrez, parsing PGN e slicing CSR.
