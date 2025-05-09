# GoVPN Client

Cliente da aplicação GoVPN construído em Go com interface gráfica utilizando a biblioteca Fyne v2.

## Arquitetura

O cliente GoVPN segue uma arquitetura modular com os seguintes componentes principais:

### Componentes Core

1. **VPNClient**: Componente central que coordena todos os outros componentes do cliente.
   - Gerencia o ciclo de vida da aplicação
   - Integra todos os outros componentes

2. **NetworkManager**: Responsável pela gestão das conexões de rede.
   - Estabelece conexões com o servidor de sinalização
   - Gerencia a criação e entrada em salas
   - Coordena a conexão P2P com outros clientes

3. **SignalingClient**: Gerencia a comunicação WebSocket com o servidor.
   - Envia e recebe mensagens de sinalização
   - Processa notificações de eventos (novos peers, saída de peers)
   - Implementa o protocolo de comunicação definido em `models`

### Armazenamento de Dados

1. **DatabaseManager**: Gerencia o banco de dados SQLite local.
   - Armazena informações sobre salas salvas
   - Mantém registro de conexões anteriores
   - Armazena chaves criptográficas

2. **ConfigManager**: Gerencia as configurações do usuário.
   - Armazena preferências como idioma e tema
   - Lida com endereço do servidor e outras configurações

3. **RealtimeDataLayer**: Camada de dados em tempo real para a interface.
   - Fornece bindings de dados para widgets Fyne
   - Implementa o padrão Observer para notificação de mudanças
   - Centraliza o estado da aplicação

### Interface do Usuário

1. **UIManager**: Gerencia a interface gráfica do usuário.
   - Coordena a navegação entre telas
   - Integra os componentes de UI
   - Gerencia o ciclo de vida da UI

2. **Componentes de UI**:
   - **HeaderComponent**: Exibe o cabeçalho com status de conexão
   - **HomeTabComponent**: Tela principal com lista de salas e opções
   - **SettingsTabComponent**: Configurações do aplicativo
   - **NetworkListComponent**: Lista de salas disponíveis
   - **RoomItemComponent**: Representação visual de uma sala

3. **Diálogos**:
   - **ConnectDialog**: Diálogo para conectar a uma sala
   - **RoomDialog**: Diálogo para criar/entrar em salas

## Fluxo de Dados

```
UI Events → UIManager → NetworkManager → SignalingClient → WebSocket → Server
    ↑                      ↓                                   ↑
    └──── RealtimeData ← Events                                |
                                                              ↓
            Data Storage ← DatabaseManager ←→ ConfigManager
```

## Tecnologias Utilizadas

- **Go**: Linguagem de programação principal (Go 1.18+)
- **Fyne v2**: Framework de UI multiplataforma
  - Widgets nativos para diversas plataformas
  - Sistema de layout responsivo
  - Suporte a temas claro/escuro

- **SQLite**: Banco de dados local
  - Armazenamento de configurações e dados persistentes
  - Através do pacote `database/sql` e driver SQLite

- **WebSocket**: Comunicação em tempo real
  - Conexão persistente com o servidor de sinalização
  - Implementado através da biblioteca Gorilla WebSocket

- **Ed25519**: Criptografia de chave pública
  - Autenticação segura de mensagens
  - Geração e armazenamento de chaves

## Estrutura de Arquivos

- **main.go**: Ponto de entrada da aplicação
- **ui_manager.go**: Gerenciamento da interface do usuário
- **vpn_client.go**: Coordenação da lógica VPN
- **network_manager.go**: Gerenciamento de rede
- **signaling_client.go**: Cliente de sinalização WebSocket
- **data/**: Componentes da camada de dados em tempo real
- **storage/**: Gerenciamento de armazenamento persistente
- **dialogs/**: Componentes de diálogo da interface
- **icon/**: Recursos visuais e ícones

## Características Importantes

- **Tamanho fixo**: 300x600 pixels para compatibilidade entre plataformas
- **Design responsivo**: Layout adaptável dentro das dimensões fixas
- **Armazenamento local**: Todos os dados persistidos apenas localmente em SQLite
- **Comunicação segura**: Autenticação baseada em chaves públicas
- **Atualização em tempo real**: Interface reativa usando bindings do Fyne

## Executando o Cliente

```bash
cd cmd/client && go run .
```

## Dependências Principais

- fyne.io/fyne/v2
- github.com/mattn/go-sqlite3
- github.com/gorilla/websocket
- crypto/ed25519