# GoVPN Server

Servidor de sinalização para a aplicação GoVPN, construído em Go. Este servidor gerencia a comunicação entre clientes, permitindo que estabeleçam conexões P2P.

## Arquitetura

O servidor GoVPN segue uma arquitetura simples, focada em sinalização, onde o principal objetivo é facilitar a comunicação inicial entre clientes e gerenciar salas. Não há implementação de TLS.

### Componentes Principais

1. **WebSocketServer**: Componente central responsável por:
   - Gerenciar conexões WebSocket com clientes
   - Rotear mensagens entre clientes
   - Administrar salas e seus participantes
   - Implementar lógica de autenticação básica
   - Lidar com situações de desconexão

2. **SupabaseManager**: Interface para o banco de dados Supabase:
   - Armazenamento persistente de informações das salas
   - Consulta e modificação de dados das salas
   - Gerenciamento do ciclo de vida das salas (expiração, limpeza)

### Estruturas de Dados Principais

1. **Mapeamentos em Memória**:
   - `clients`: Mapeia conexões WebSocket para IDs de salas
   - `networks`: Mapeia IDs de salas para listas de conexões
   - `clientToPublicKey`: Associa cada conexão à sua chave pública

2. **ServerRoom**: Extensão da estrutura de sala básica:
   - Dados fundamentais da sala (ID, nome, senha)
   - Chave pública do proprietário
   - Metadados (criação, última atividade)

## Fluxo de Operação

```
Cliente WebSocket → WebSocketServer → [Processamento de Mensagem]
                                       ↓
                                   [Validação]
                                       ↓
                            [Ação Específica por Tipo]
                           /       |        \        \
                     Criar Sala  Entrar    Sair    Outras Ações
                         |         |         |          |
                         v         v         v          v
                     [Supabase] [Notificar] [Limpar] [Processar]
                                  Outros    Recursos
```

## Gestão de Salas

1. **Criação de Sala**:
   - Validação de dados (senha, nome)
   - Geração de ID único
   - Persistência em Supabase
   - Associação do proprietário

2. **Entrada em Sala**:
   - Validação de credenciais
   - Verificação de limites
   - Notificação a peers existentes
   - Rastreamento de conexão

3. **Saída de Sala**:
   - Remoção do cliente da sala
   - Limpeza condicional baseada em propriedade
   - Notificação a outros participantes

4. **Propriedade de Sala**:
   - Chave pública como identificador do proprietário
   - Permissões especiais (renomear, expulsar)
   - Exclusão automática quando proprietário sai

## Sistema de Mensagens

O servidor implementa um protocolo de mensagens baseado em JSON sobre WebSocket:

- **SignalingMessage**: Estrutura envelope para todas as mensagens
- **Tipos de Mensagens**: CreateRoom, JoinRoom, LeaveRoom, etc.
- **Identificação**: Cada mensagem possui um ID único para tracking

Detalhes completos da API podem ser encontrados em `docs/websocket_api.md`.

## Tecnologias Utilizadas

- **Go**: Linguagem de programação principal (Go 1.18+)
- **Gorilla WebSocket**: Biblioteca para gerenciamento de conexões WebSocket
- **Supabase-Go**: Cliente Supabase para Go
- **Ed25519**: Para verificação de assinaturas e autenticação

## Persistência de Dados

Os dados das salas são armazenados no Supabase com os seguintes campos:
- ID da sala
- Nome da sala
- Senha (hash)
- Chave pública do proprietário
- Timestamp de criação
- Timestamp de última atividade

## Características Importantes

- **Stateful**: Mantém estado das conexões e salas em memória
- **Baixa sobrecarga**: Apenas relaying de mensagens, sem processamento pesado
- **Limpeza automática**: Remoção de salas inativas após período configurável
- **Escalabilidade horizontal limitada**: Sem compartilhamento de estado entre instâncias

## Configuração

O servidor é configurado através de variáveis de ambiente:

```bash
# Obrigatórias
export SUPABASE_URL="seu-url-supabase"
export SUPABASE_KEY="sua-chave-supabase"

# Opcionais
export PORT="8080"
export MAX_CLIENTS_PER_ROOM="50"
export ROOM_EXPIRY_DAYS="7"
export LOG_LEVEL="info"
```

## Executando o Servidor

```bash
cd cmd/server && go run .
```

## Limitações

- Não implementa TLS diretamente (recomendado uso atrás de proxy)
- Escala verticalmente, não horizontalmente
- Sem banco de dados em cluster (usa apenas Supabase)
- Sem balanceamento de carga integrado

## Dependências Principais

- github.com/gorilla/websocket
- github.com/supabase-community/supabase-go
- crypto/ed25519