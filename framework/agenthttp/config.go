package agenthttp

import (
	"net/http"
	"strings"

	"github.com/gohade/hade/framework/contract"
	"github.com/gohade/hade/framework/gin"
	"github.com/pkg/errors"
)

const llmUnavailable = "configurable llm service unavailable"

func configurableLLMFrom(c *gin.Context) (contract.ConfigurableLLM, error) {
	instance, err := c.Make(contract.LLMKey)
	if err != nil {
		return nil, errors.New(llmUnavailable)
	}
	llm, ok := instance.(contract.ConfigurableLLM)
	if !ok || llm == nil {
		return nil, errors.New(llmUnavailable)
	}
	return llm, nil
}

func writeConfig(c *gin.Context, config contract.LLMConfig) {
	c.Header("Cache-Control", "no-store")
	c.JSON(http.StatusOK, gin.H{
		"base_url":           config.BaseURL,
		"model":              config.Model,
		"api_key_configured": strings.TrimSpace(config.APIKey) != "",
	})
}

// GetConfig 返回非敏感模型配置。API Key 只返回是否存在。
func GetConfig(c *gin.Context) {
	llm, err := configurableLLMFrom(c)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": llmUnavailable})
		return
	}
	writeConfig(c, llm.Configuration())
}

// UpdateConfig 更新当前 Agent 进程的模型配置。省略 api_key 会保留现有值，
// 显式传空字符串则清空，配置不会写回 yaml 或浏览器存储。
func UpdateConfig(c *gin.Context) {
	llm, err := configurableLLMFrom(c)
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": llmUnavailable})
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, contract.RequestBodyMaxBytes)
	var request struct {
		BaseURL string  `json:"base_url"`
		Model   string  `json:"model"`
		APIKey  *string `json:"api_key"`
	}
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(bindErrorStatus(err), gin.H{"error": err.Error()})
		return
	}
	config := llm.Configuration()
	config.BaseURL = request.BaseURL
	config.Model = request.Model
	if request.APIKey != nil {
		config.APIKey = *request.APIKey
	}
	if err := llm.Configure(config); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	writeConfig(c, llm.Configuration())
}

// ListTools 返回当前 Agent 暴露给模型的工具定义。
func ListTools(c *gin.Context) {
	agent, err := agentFrom(c)
	if err != nil {
		writeUnavailable(c, err)
		return
	}
	tools := agent.ListTools()
	if tools == nil {
		tools = []contract.ToolSpec{}
	}
	c.JSON(http.StatusOK, gin.H{"tools": tools})
}
