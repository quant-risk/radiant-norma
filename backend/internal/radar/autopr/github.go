// Package autopr implementa criação automática de GitHub Pull Requests
// quando o Radar detecta mudanças regulatórias.
//
// Usa GitHub REST API v3 (não GraphQL) para máxima compatibilidade.
package autopr

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Config é a configuração do GitHub Auto-PR.
type Config struct {
	Owner      string   // "quant-risk" (ou organização)
	Repo       string   // "radiant-norma"
	Token      string   // GitHub Personal Access Token
	BaseBranch string   // "main" (branch base do PR)
	Reviewers  []string // logins de reviewers (opcional)
	Assignee   string   // login do assignee (opcional)
	Labels     []string // labels do PR (opcional)
}

// Client é o cliente GitHub para Auto-PR.
type Client struct {
	cfg     Config
	hc      *http.Client
	baseURL string
}

// NewClient cria um novo GitHub Auto-PR client.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg: cfg,
		hc: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://api.github.com",
	}
}

// PRResult representa o resultado da criação de um PR.
type PRResult struct {
	Number     int
	URL        string
	CreatedAt  time.Time
	BranchName string
}

// RuleUpdatePRInput é o input para CreateRuleUpdatePR.
type RuleUpdatePRInput struct {
	CadocCode   string
	RuleCodes   []string
	DiffSummary string
	BranchName  string
	// FileChanges é um mapa de path → conteúdo do arquivo (opcional).
	FileChanges map[string]string
}

// CreateRuleUpdatePR cria um PR com as regras atualizadas a partir do diff.
//
// Fluxo:
//  1. Cria branch feature: radar/update/{cadoc}-{YYYYMMDD}
//  2. Faz commit com as mudanças (se houver arquivos)
//  3. Cria PR com título e body descritivos
func (c *Client) CreateRuleUpdatePR(ctx context.Context, input RuleUpdatePRInput) (*PRResult, error) {
	if c.cfg.Token == "" {
		return nil, fmt.Errorf("GitHub token não configurado")
	}

	branchName := input.BranchName
	if branchName == "" {
		branchName = fmt.Sprintf("radar/update/%s-%s", input.CadocCode, time.Now().Format("20060102"))
	}

	// 1. Cria branch.
	if err := c.createBranch(ctx, branchName); err != nil {
		return nil, fmt.Errorf("create branch: %w", err)
	}

	// 2. Commita arquivos (se houver).
	for path, content := range input.FileChanges {
		if err := c.commitFile(ctx, branchName, path, content, fmt.Sprintf("Update %s rules via Radar", input.CadocCode)); err != nil {
			return nil, fmt.Errorf("commit file %s: %w", path, err)
		}
	}

	// 3. Cria PR.
	pr, err := c.createPRFromInput(ctx, branchName, input)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}

	return pr, nil
}

// createBranch cria uma nova branch no repositório.
func (c *Client) createBranch(ctx context.Context, branchName string) error {
	// Primeiro: pega SHA do commit base da branch default.
	baseRef := c.cfg.BaseBranch
	getURL := fmt.Sprintf("%s/repos/%s/%s/git/ref/heads/%s", c.baseURL, c.cfg.Owner, c.cfg.Repo, baseRef)
	req, err := http.NewRequestWithContext(ctx, "GET", getURL, nil)
	if err != nil {
		return err
	}
	c.setAuth(req)

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("GET ref: status %d: %s", resp.StatusCode, string(body))
	}

	var refResp struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&refResp); err != nil {
		return err
	}

	// Cria a nova branch.
	postURL := fmt.Sprintf("%s/repos/%s/%s/git/refs", c.baseURL, c.cfg.Owner, c.cfg.Repo)
	body, _ := json.Marshal(map[string]string{
		"ref": "refs/heads/" + branchName,
		"sha": refResp.Object.SHA,
	})
	req, err = http.NewRequestWithContext(ctx, "POST", postURL, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err = c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("create ref: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// commitFile commita um arquivo na branch especificada.
func (c *Client) commitFile(ctx context.Context, branch, path, content, message string) error {
	// Usa API de contents para criar/atualizar arquivo.
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, c.cfg.Owner, c.cfg.Repo, path)

	// Prepara body.
	reqBody := map[string]string{
		"message": message,
		"content": toBase64(content),
		"branch":  branch,
	}
	body, err := json.Marshal(reqBody)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("commit file: status %d: %s", resp.StatusCode, string(respBody))
	}
	return nil
}

// createPRFromInput cria o Pull Request a partir de RuleUpdatePRInput.
func (c *Client) createPRFromInput(ctx context.Context, headBranch string, input RuleUpdatePRInput) (*PRResult, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, c.cfg.Owner, c.cfg.Repo)

	title := fmt.Sprintf("[Radar] %s atualizado — %d regra(s) afetada(s)", input.CadocCode, len(input.RuleCodes))
	body := c.buildPRBodyFromInput(input)

	prReq := map[string]any{
		"title":                 title,
		"body":                  body,
		"head":                  headBranch,
		"base":                  c.cfg.BaseBranch,
		"maintainer_can_modify": true,
	}

	if len(c.cfg.Reviewers) > 0 {
		prReq["reviewers"] = c.cfg.Reviewers
	}
	if c.cfg.Assignee != "" {
		prReq["assignees"] = []string{c.cfg.Assignee}
	}

	jsonBody, err := json.Marshal(prReq)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, err
	}
	c.setAuth(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("create PR: status %d: %s", resp.StatusCode, string(respBody))
	}

	var prResp struct {
		Number    int    `json:"number"`
		HTMLURL   string `json:"html_url"`
		CreatedAt string `json:"created_at"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&prResp); err != nil {
		return nil, err
	}

	createdAt, _ := time.Parse(time.RFC3339, prResp.CreatedAt)
	return &PRResult{
		Number:    prResp.Number,
		URL:       prResp.HTMLURL,
		CreatedAt: createdAt,
	}, nil
}

// buildPRBodyFromInput gera o corpo do PR em Markdown a partir de RuleUpdatePRInput.
func (c *Client) buildPRBodyFromInput(input RuleUpdatePRInput) string {
	body := fmt.Sprintf("## Radar detectou mudança em %s\n\n", input.CadocCode)
	body += fmt.Sprintf("**Detectado em:** %s\n\n", time.Now().Format("2006-01-02 15:04:05"))

	body += "## Resumo\n\n"
	if input.DiffSummary != "" {
		body += input.DiffSummary + "\n\n"
	}

	if len(input.RuleCodes) > 0 {
		body += "## Regras afetadas\n\n"
		for _, code := range input.RuleCodes {
			body += fmt.Sprintf("- `%s`\n", code)
		}
		body += "\n"
	}

	body += "---\n\n"
	body += "*Este PR foi criado automaticamente pelo Radiant Norma Radar.*\n"
	return body
}

func (c *Client) setAuth(req *http.Request) {
	req.Header.Set("Authorization", "Bearer "+c.cfg.Token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
}

// toBase64 codifica string para base64 (para conteúdo de arquivo GitHub).
func toBase64(content string) string {
	return base64.StdEncoding.EncodeToString([]byte(content))
}
