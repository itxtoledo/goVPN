# GoVPN Server

Servidor de sinalização para a aplicação GoVPN, construído em Go. Este servidor gerencia a comunicação entre clientes, permitindo que estabeleçam conexões P2P seguras e eficientes.

## Arquitetura

O servidor GoVPN segue uma arquitetura simples, focada em sinalização, onde o principal objetivo é facilitar a comunicação inicial entre clientes e gerenciar salas. Não há implementação de TLS.

### Componentes Principais

1. **WebSocketServer**: Componente central responsável por:
   - Gerenciar conexões WebSocket com clientes
   - Rotear mensagens entre clientes
   - Administrar salas e seus participantes
   - Implementar lógica de autenticação básica
   - Lidar com situações de desconexão
   - Monitorar estatísticas de uso
   - Gerenciar o ciclo de vida das salas

2. **SupabaseManager**: Interface para o banco de dados Supabase:
   - Armazenamento persistente de informações das salas
   - Consulta e modificação de dados das salas
   - Gerenciamento do ciclo de vida das salas (expiração, limpeza)
   - Verificação de propriedade por chave pública

3. **StatsManager**: Componente de monitoramento:
   - Acompanhamento de conexões ativas
   - Estatísticas de salas e mensagens
   - Métricas de performance
   - Endpoint para monitoramento

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
   - Verificação de unicidade de chave pública

2. **Entrada em Sala**:
   - Validação de credenciais
   - Verificação de limites
   - Notificação a peers existentes
   - Rastreamento de conexão
   - Atualização da atividade da sala

3. **Saída de Sala**:
   - Remoção do cliente da sala
   - Limpeza condicional baseada em propriedade
   - Notificação a outros participantes
   - Preservação opcional da sala

4. **Propriedade de Sala**:
   - Chave pública como identificador do proprietário
   - Permissões especiais (renomear, expulsar)
   - Exclusão automática quando proprietário sai (se não configurado para preservar)

## Sistema de Mensagens

O servidor implementa um protocolo de mensagens baseado em JSON sobre WebSocket:

- **SignalingMessage**: Estrutura envelope para todas as mensagens
- **Tipos de Mensagens**: CreateRoom, JoinRoom, LeaveRoom, Ping, Rename, Kick, etc.
- **Identificação**: Cada mensagem possui um ID único para tracking e correlação de respostas

Detalhes completos da API podem ser encontrados em `docs/websocket_api.md`.

## Características de Segurança

- **Verificação de Chave Pública**: Validação de identidade através de chaves Ed25519
- **Autenticação de Sala**: Proteção por senha para acesso às salas
- **Isolamento de Salas**: Mensagens são roteadas apenas dentro das salas corretas
- **Validação de Dados**: Verificação rigorosa de entradas de usuário
- **Controle de Acesso**: Apenas proprietários podem executar ações administrativas
- **Timeouts**: Desconexão automática de clientes inativos

## Monitoramento e Métricas

O servidor fornece um endpoint `/stats` que retorna métricas em tempo real:

- Número total de conexões
- Conexões ativas
- Mensagens processadas
- Salas ativas
- Estatísticas de limpeza
- Tempo de atividade

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

## Características de Performance

- **Uso Eficiente de Memória**: Estruturas de dados otimizadas
- **Concorrência**: Aproveitamento de goroutines para operações paralelas
- **Limpeza Automática**: Remoção programada de salas inativas para liberar recursos
- **Desligamento Gracioso**: Notificação aos clientes e persistência de estado durante reinicializações
- **Timeouts**: Prevenção de vazamentos de recursos por conexões pendentes

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
export READ_BUFFER_SIZE="4096"
export WRITE_BUFFER_SIZE="4096"
export CLEANUP_INTERVAL="24h"
export SUPABASE_ROOMS_TABLE="govpn_rooms"
export ALLOW_ALL_ORIGINS="true"
```

## Endpoints

- `/ws`: Endpoint principal para conexões WebSocket
- `/health`: Verificação de saúde do servidor (retorna status 200 se operacional)
- `/stats`: Retorna estatísticas em tempo real do servidor em formato JSON

## Executando o Servidor

```bash
cd cmd/server && go run .
```

## Desligamento Gracioso

O servidor suporta desligamento gracioso, onde:

1. Todos os clientes conectados são notificados sobre a iminente paralisação
2. O estado atual das salas é preservado no Supabase
3. As conexões são fechadas ordenadamente
4. Os recursos são liberados antes do encerramento

## Limitações

- Não implementa TLS diretamente (recomendado uso atrás de proxy como Nginx ou Traefik)
- Escala verticalmente, não horizontalmente
- Sem banco de dados em cluster (usa apenas Supabase)
- Sem balanceamento de carga integrado

## Dependências Principais

- github.com/gorilla/websocket
- github.com/supabase-community/supabase-go
- crypto/ed25519