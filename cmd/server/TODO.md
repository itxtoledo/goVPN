# GoVPN Server - TODO e Status de Implementação

Este documento detalha as funcionalidades já implementadas e as pendentes no servidor GoVPN.

## Componentes Implementados

### Core do Servidor WebSocket
- [x] **WebSocketServer**: Estrutura principal que gerencia as conexões
  - [x] Mapeamento de clientes para salas (`clients`)
  - [x] Mapeamento de salas para listas de conexões (`networks`)
  - [x] Mapeamento de conexões para chaves públicas (`clientToPublicKey`)
  - [x] Configuração do upgrader WebSocket
  - [x] Inicialização do servidor

### Gerenciamento de Mensagens
- [x] **HandleWebSocketEndpoint**: Roteamento de mensagens
  - [x] Upgrade de conexões HTTP para WebSocket
  - [x] Processamento de mensagens JSON
  - [x] Direcionamento para handlers específicos

### Operações de Sala
- [x] **Criação de Salas**
  - [x] Validação de parâmetros (nome, senha, chave pública)
  - [x] Geração de ID único para sala
  - [x] Persistência na base Supabase
  - [x] Resposta ao cliente com confirmação
  
- [x] **Entrada em Salas**
  - [x] Verificação de existência da sala
  - [x] Validação de senha
  - [x] Verificação de limites (número máximo de clientes)
  - [x] Notificação a outros participantes
  - [x] Atualização do timestamp de atividade

- [x] **Saída de Salas**
  - [x] Remoção do cliente da sala
  - [x] Notificação a outros participantes
  - [x] Limpeza de referências

- [x] **Renomeação de Salas**
  - [x] Verificação de permissão (apenas proprietário)
  - [x] Atualização na base Supabase
  - [x] Notificação aos participantes

- [x] **Expulsão de Usuários**
  - [x] Verificação de permissão (apenas proprietário)
  - [x] Notificação ao usuário expulso
  - [x] Remoção das estruturas de dados

### Gerenciamento de Desconexões
- [x] **HandleDisconnect**: Limpeza quando cliente desconecta
  - [x] Remoção do cliente das estruturas de dados
  - [x] Tratamento especial para proprietários de salas
  - [x] Notificação a outros participantes

### Integração com Supabase
- [x] **SupabaseManager**: Interface com banco de dados
  - [x] Criação de salas
  - [x] Busca de salas por ID
  - [x] Busca de salas por chave pública
  - [x] Atualização de nome e atividade
  - [x] Exclusão de salas
  - [x] Busca de salas expiradas

### Manutenção
- [x] **DeleteStaleNetworks**: Limpeza de salas inativas
  - [x] Busca periódica de salas expiradas
  - [x] Exclusão automática das salas
  - [x] Configuração de período de limpeza

### Utilitários
- [x] **Verificação de Senha**: Regex para validação
- [x] **Debugging**: Logs configuráveis por nível

## Componentes Pendentes

### Segurança e Autenticação
- [ ] **Validação Completa de Mensagens**
  - [ ] Verificação de limites de tamanho
  - [ ] Validação de formatos para todos os campos
  - [ ] Sanitização de entradas (nomes de sala, etc.)

- [ ] **Rate Limiting**
  - [ ] Limitação de criação de salas por IP/cliente
  - [ ] Limitação de mensagens por período de tempo
  - [ ] Proteção contra spam e flood

### Robustez
- [ ] **Reconexão Automática**
  - [ ] Mecanismo para reconectar clientes temporariamente desconectados
  - [ ] Manter estado da sala por um período após desconexão

- [x] **Graceful Shutdown**
  - [x] Tratamento de sinais (SIGTERM, SIGINT)
  - [x] Notificação a clientes sobre encerramento
  - [x] Persistência de estado para reinício

### Monitoramento e Estatísticas
- [x] **Endpoints de Status**
  - [x] Endpoint `/stats` para métricas básicas
  - [x] Contadores de conexões e salas ativas
  - [x] Tempo de atividade do servidor

- [x] **Logs Estruturados**
  - [x] Formato JSON para logs
  - [x] Níveis adequados para produção
  - [x] Rotação de arquivos de log

### Testes
- [ ] **Testes Unitários**
  - [ ] Cobertura para operações principais
  - [ ] Mocks para Supabase

- [ ] **Testes de Integração**
  - [ ] Simulação de clientes e salas
  - [ ] Cenários de falha e recuperação

### Performance
- [ ] **Otimização de Memória**
  - [ ] Revisão de estruturas para minimizar duplicação
  - [ ] Garbage collection apropriada de recursos

- [ ] **Otimização de Resposta**
  - [ ] Buffering adequado de mensagens
  - [ ] Configurações de timeout otimizadas

### Documentação
- [ ] **Comentários de Código**
  - [ ] Documentação GoDoc completa para todas as funções exportadas
  - [ ] Exemplos embutidos nos comentários

- [ ] **Diagramas**
  - [ ] Fluxo de mensagens
  - [ ] Arquitetura de componentes
  - [ ] Modelo de dados

## Limitações Conhecidas e Decisões Técnicas

- **Sem TLS Direto**: Conforme definido nos requisitos, não implementamos TLS. Em produção, recomenda-se colocar atrás de um proxy com TLS.

- **Escalabilidade Limitada**: O servidor mantém estado em memória, o que limita a escalabilidade horizontal. Para múltiplas instâncias, seria necessário implementar um mecanismo de compartilhamento de estado.

- **Supabase como Single Point**: A dependência do Supabase torna o sistema dependente da disponibilidade do serviço.

- **Segurança Básica**: O modelo de autenticação atual depende apenas da posse da chave privada correspondente à chave pública registrada.

## Próximos Passos Prioritários

1. Implementar rate limiting básico para evitar abusos
2. Adicionar endpoints de monitoramento
3. Melhorar validação e sanitização de entradas 
4. Desenvolver suite básica de testes unitários
5. Documentar completamente a API para integrações de terceiros

## Notas para Implementação

- Manter compatibilidade com o cliente atual
- Preferir mudanças incrementais para evitar regressões
- Documentar qualquer mudança na API em `docs/websocket_api.md`
- Seguir o padrão Go de tratamento de erros e logs