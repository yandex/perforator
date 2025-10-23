package agent

import (
	"github.com/yandex/perforator/perforator/agent/collector/pkg/agent/custom_profiling_operation"
	"github.com/yandex/perforator/perforator/agent/collector/pkg/config"
	agent_gateway_client "github.com/yandex/perforator/perforator/internal/agent_gateway/client"
)

type Config struct {
	AgentGateway     *agent_gateway_client.Config              `yaml:"agent_gateway"`
	DebugModeToggler *DebugModeTogglerConfig                   `yaml:"debug_mode_toggler"`
	Profiler         config.Config                             `yaml:",inline"`
	CPOService       *custom_profiling_operation.ServiceConfig `yaml:"custom_profiling_operation"`
}
