# GoVPN

Uma solução de Rede Local Virtual (VLAN) para jogos que permite que jogadores se conectem como se estivessem na mesma rede local.

## Funcionalidades

- Criação e participação em salas de jogo virtuais
- Atravessamento de NAT usando STUN/TURN para conexões P2P
- Criptografia ponta a ponta
- Suporte multiplataforma (Windows, macOS, Linux)
- Armazenamento local de dados usando SQLite

## Arquitetura do Sistema

O GoVPN é organizado em uma arquitetura cliente-servidor modular, com comunicação P2P entre os clientes para minimizar a latência. A arquitetura está estruturada da seguinte forma:

### Bibliotecas Compartilhadas

- **libs/crypto_utils**: Implementa funções criptográficas para garantir a segurança da comunicação, incluindo:
  - Geração de chaves Ed25519
  - Assinatura e verificação de mensagens
  - Criptografia para os dados transmitidos
  - Geração de identificadores seguros

- **libs/models**: Define as estruturas de dados compartilhadas entre cliente e servidor:
  - Room: Representa uma sala de jogo virtual
  - Message: Define o formato das mensagens trocadas via WebSocket
  - NetworkPacket: Estrutura para pacotes de rede tunelados pela VPN
  - ClientInfo: Informações sobre os clientes conectados

- **libs/network**: Gerencia a rede virtual entre os clientes:
  - VirtualNetwork: Classe principal que coordena a comunicação entre peers
  - Mapeamento de endereços IP virtuais
  - Encapsulamento e roteamento de pacotes entre clientes

### Componentes do Sistema

- **cmd/server**: Servidor de sinalização que facilita:
  - Criação e gerenciamento de salas
  - Autenticação das operações via chaves Ed25519
  - Estabelecimento de conexões entre clientes
  - Persistência de dados em Supabase
  - WebSocketServer: Gerencia conexões WebSocket com clientes
  - SupabaseManager: Interface com banco de dados Supabase

- **cmd/client**: Aplicação cliente com interface gráfica que permite:
  - Criação e entrada em salas de jogo
  - Gerenciamento de conexões P2P
  - Armazenamento local de configurações em SQLite
  - Interface gráfica construída com Fyne (v2.0+)
  - Componentes modulares como NetworkManager, SignalingClient, etc.

### Fluxo de Comunicação

1. **Fase de Sinalização**:
   - O servidor atua como intermediário para estabelecer conexões iniciais
   - Clientes se autenticam e trocam informações sobre salas disponíveis
   
2. **Estabelecimento de Conexão P2P**:
   - Troca de ofertas, respostas e candidatos via servidor
   - Uso de servidores STUN para descoberta de endereços públicos
   - Fallback para servidores TURN quando a conexão direta falha

3. **Comunicação Direta**:
   - Após estabelecida, a comunicação ocorre diretamente entre os clientes
   - Dados são criptografados ponta a ponta (chave derivada da senha da sala)

4. **Rede Virtual**:
   - Cada cliente recebe um endereço IP virtual (formato 10.0.0.x)
   - Pacotes de rede são encapsulados, criptografados e enviados pelo canal de dados

## Estrutura do Projeto Atual

A estrutura atual do projeto está organizada da seguinte forma:

```
README.md                        # Documentação principal
cmd/                             # Componentes principais
    client/                      # Cliente GoVPN
        data/                    # Camada de dados em tempo real
        dialogs/                 # Caixas de diálogo da UI
        icon/                    # Ícones e recursos gráficos
            assets/              # Arquivos de imagem 
        storage/                 # Gerenciamento de banco de dados e config
        *.go                     # Componentes da UI e lógica do cliente
    server/                      # Servidor de sinalização
        docs/                    # Documentação da API
        *.go                     # Implementação do servidor
libs/                            # Bibliotecas compartilhadas
    crypto_utils/                # Utilitários criptográficos
    models/                      # Definições de estruturas de dados
    network/                     # Gerenciamento de rede virtual
migrations/                      # Scripts SQL para o banco de dados
```

### Componentes Principais do Cliente

- **UIManager**: Gerencia toda a interface gráfica do usuário
- **VPNClient**: Controla toda a lógica de conexão VPN
- **NetworkManager**: Gerencia as conexões de rede e salas
- **SignalingClient**: Comunica-se com o servidor de sinalização
- **DatabaseManager**: Gerencia o armazenamento local usando SQLite
- **ConfigManager**: Gerencia configurações do aplicativo
- **RealtimeDataLayer**: Fornece binding de dados para atualizar a UI

### Componentes Principais do Servidor

- **WebSocketServer**: Gerencia conexões WebSocket, salas e clientes
- **SupabaseManager**: Interface com o banco de dados Supabase
- **Room Management**: Gerencia criação, exclusão e modificação de salas
- **Authentication**: Autenticação baseada em chaves Ed25519
- **Connection Management**: Gerencia o ciclo de vida das conexões

## API WebSocket do Servidor

O servidor implementa uma API WebSocket robusta para comunicação com os clientes. A documentação completa está disponível em `cmd/server/docs/websocket_api.md`.

### Principais Tipos de Mensagens

- **Cliente para Servidor**:
  - `CreateRoom`: Cria uma nova sala
  - `JoinRoom`: Entra em uma sala existente
  - `LeaveRoom`: Sai de uma sala
  - `Kick`: Expulsa um usuário de uma sala
  - `Rename`: Renomeia uma sala

- **Servidor para Cliente**:
  - `RoomCreated`: Confirmação de sala criada
  - `RoomJoined`: Confirmação de entrada em sala
  - `PeerJoined`: Notificação de novo par na sala
  - `PeerLeft`: Notificação de saída de par da sala
  - `RoomDeleted`: Notificação de sala excluída

## Variáveis de Ambiente do Servidor

| Variável | Descrição | Padrão |
|----------|-------------|---------|
| `PORT` | Porta para o servidor escutar | `8080` |
| `ALLOW_ALL_ORIGINS` | Permitir conexões WebSocket de qualquer origem | `true` |
| `PASSWORD_PATTERN` | Expressão regular para validar senhas de sala | `^\d{4}$` |
| `MAX_ROOMS` | Número máximo de salas permitidas | `100` |
| `MAX_CLIENTS_PER_ROOM` | Número máximo de clientes em uma sala | `10` |
| `LOG_LEVEL` | Nível de log (info, debug) | `info` |
| `IDLE_TIMEOUT_SECONDS` | Tempo limite para conexões inativas em segundos | `60` |
| `PING_INTERVAL_SECONDS` | Intervalo de ping WebSocket em segundos | `30` |
| `READ_BUFFER_SIZE` | Tamanho do buffer de leitura WebSocket | `1024` |
| `WRITE_BUFFER_SIZE` | Tamanho do buffer de escrita WebSocket | `1024` |
| `SUPABASE_URL` | URL do Supabase para persistência de salas (obrigatório) | `""` |
| `SUPABASE_KEY` | Chave API do Supabase para autenticação (obrigatório) | `""` |
| `ROOM_EXPIRY_DAYS` | Dias após os quais salas inativas são excluídas | `7` |
| `CLEANUP_INTERVAL_HOURS` | Intervalo para limpeza de salas expiradas em horas | `24` |

**Nota:** `SUPABASE_URL` e `SUPABASE_KEY` são necessários para o funcionamento adequado do servidor.

## Interface do Cliente

O cliente GoVPN apresenta uma interface gráfica construída com Fyne 2.0+ com tamanho fixo de 300x600 pixels. Principais características:

- **Home Tab**: Exibe salas salvas e opções de conexão
- **Settings Tab**: Configurações do aplicativo
- **Network List**: Lista de salas salvas com opções de conexão
- **Dialogs**: Para criar/entrar em salas e gerenciar conexões

### Armazenamento Local

O cliente armazena dados localmente usando SQLite, incluindo:

- Configurações do usuário
- Salas salvas e senhas
- Histórico de conexões
- Chaves criptográficas

## Processo de Lançamento

Este projeto usa GitHub Actions para construir e lançar automaticamente os componentes servidor e cliente.

### Criando uma Versão do Servidor

Para criar uma nova versão do servidor:

```bash
# Marcar o commit com uma versão do servidor
git tag server-v1.0.0
git push origin server-v1.0.0
```

Isso acionará o workflow de lançamento do servidor que constrói binários para:
- Linux (amd64)
- Windows (amd64)
- macOS (Intel/amd64)
- macOS (Apple Silicon/arm64)

### Criando uma Versão do Cliente

Para criar uma nova versão do cliente:

```bash
# Marcar o commit com uma versão do cliente
git tag client-v1.0.0
git push origin client-v1.0.0
```

Isso acionará o workflow de lançamento do cliente que constrói:
- Binários independentes para Linux, Windows e macOS
- Aplicativos empacotados quando possível

## Compilando Manualmente

### Servidor

```bash
# Compilar o executável do servidor
go build -o govpn-server ./cmd/server/main.go
```

### Cliente

```bash
# Compilar o executável do cliente
go build -o govpn-client ./cmd/client/main.go
```

Para aplicativos empacotados usando Fyne:

```bash
go install fyne.io/fyne/v2/cmd/fyne@latest
# Certifique-se de estar no diretório raiz do projeto
cd cmd/client
fyne package -os windows -icon icon/assets/app.png -name GoVPN
# Ou para outras plataformas: linux, darwin
```

## Executando a Aplicação

### Servidor

```bash
# Definir variáveis de ambiente necessárias
export SUPABASE_URL="seu-url-supabase"
export SUPABASE_KEY="sua-chave-supabase"

# Executar o servidor (binário compilado)
./govpn-server
```

### Cliente

```bash
# Executar o cliente (binário compilado)
./govpn-client
```

### Executando a partir do Código-Fonte (Desenvolvimento)

Para executar a aplicação diretamente do código-fonte sem compilar:

#### Servidor
```bash
# Definir variáveis de ambiente necessárias
export SUPABASE_URL="seu-url-supabase"
export SUPABASE_KEY="sua-chave-supabase"

# Executar usando go run (comando de linha única)
cd cmd/server && go run main.go
```

#### Cliente
```bash
# Executar usando go run (comando de linha única)
cd cmd/client && go run .
```

### Tarefas do VS Code

- Pressione `Ctrl+Shift+P` (ou `Cmd+Shift+P` no macOS)
- Digite "Tasks: Run Task"
- Selecione "Run GoVPN Server" ou "Run GoVPN Client"

## Características de Segurança

- Autenticação baseada em chaves Ed25519
- Senhas de sala validadas (padrão: 4 dígitos numéricos)
- Comunicação criptografada entre cliente e servidor
- Persistência segura de credenciais locais

## Contribuições e Desenvolvimento

Para contribuir com o projeto:

1. Fork o repositório
2. Crie uma branch para sua feature (`git checkout -b feature/nova-feature`)
3. Commit suas mudanças (`git commit -am 'Adiciona nova feature'`)
4. Push para a branch (`git push origin feature/nova-feature`)
5. Crie um Pull Request

## Notas de Implementação

- O cliente é construído com Fyne 2.0+ e tem tamanho fixo de 300x600
- O servidor usa Gorilla WebSocket para comunicação em tempo real
- O sistema utiliza Supabase para persistência de dados do servidor
- Comunicação P2P usa WebRTC para estabelecer conexões diretas entre clientes

## Resolução de Problemas

- **Erro de conexão**: Verifique se o servidor está rodando e as variáveis de ambiente estão configuradas
- **Problemas de compilação Fyne**: Certifique-se de que os requisitos do Fyne estão instalados (gcc, dependências gráficas)
- **Erros SQLite**: Verifique as permissões do diretório ~/.govpn

## Licença

[MIT License](LICENSE)