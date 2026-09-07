package agentcore

// BaseAgent 基础代理接口
type BaseAgent interface {
	Run(request any, context any) (any, error)
	GetName() string
}

// AgentRegistry 代理注册表
type AgentRegistry struct {
	agents map[string]BaseAgent
}

// NewAgentRegistry 创建代理注册表
func NewAgentRegistry() *AgentRegistry {
	return &AgentRegistry{
		agents: make(map[string]BaseAgent),
	}
}

// Register 注册代理
func (r *AgentRegistry) Register(agent BaseAgent) {
	r.agents[agent.GetName()] = agent
}

// Get 获取代理
func (r *AgentRegistry) Get(name string) (BaseAgent, bool) {
	agent, ok := r.agents[name]
	return agent, ok
}

// List 获取所有代理
func (r *AgentRegistry) List() []BaseAgent {
	var result []BaseAgent
	for _, agent := range r.agents {
		result = append(result, agent)
	}
	return result
}
