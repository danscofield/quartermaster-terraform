package client

import (
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Client struct {
	Endpoint   string
	Token      string
	HTTPClient *http.Client
}

func New(endpoint, token string, insecure bool) *Client {
	transport := &http.Transport{}
	if insecure {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}
	return &Client{
		Endpoint:   endpoint,
		Token:      token,
		HTTPClient: &http.Client{Transport: transport},
	}
}

type APIError struct {
	StatusCode int
	Message    string
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.Message)
}

func (c *Client) do(method, path string, body, result interface{}) error {
	var reqBody io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reqBody = bytes.NewReader(b)
	}

	req, err := http.NewRequest(method, c.Endpoint+path, reqBody)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if resp.StatusCode >= 400 {
		return &APIError{StatusCode: resp.StatusCode, Message: string(respBody)}
	}

	if result != nil && len(respBody) > 0 {
		return json.Unmarshal(respBody, result)
	}
	return nil
}

// Billet types

type Billet struct {
	Name              string   `json:"name"`
	Description       string   `json:"description,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	AssociatedAWSRoles []string `json:"associated_aws_roles,omitempty"`
	AssociatedGCPSAs  []string `json:"associated_gcp_sas,omitempty"`
	UpdatedAt         string   `json:"updated_at,omitempty"`
}

func (c *Client) CreateBillet(b *Billet) (*Billet, error) {
	var result Billet
	err := c.do("POST", "/admin/billets", b, &result)
	return &result, err
}

func (c *Client) GetBillet(name string) (*Billet, error) {
	var result Billet
	err := c.do("GET", "/admin/billets/"+name, nil, &result)
	return &result, err
}

func (c *Client) UpdateBillet(name string, b *Billet) (*Billet, error) {
	var result Billet
	err := c.do("PUT", "/admin/billets/"+name, b, &result)
	return &result, err
}

func (c *Client) DeleteBillet(name string) error {
	return c.do("DELETE", "/admin/billets/"+name, nil, nil)
}

// Policy types

type Policy struct {
	ID          string `json:"id,omitempty"`
	Billet      string `json:"billet,omitempty"`
	Statement   string `json:"statement"`
	Description string `json:"description,omitempty"`
	CreatedAt   string `json:"created_at,omitempty"`
	UpdatedAt   string `json:"updated_at,omitempty"`
}

func (c *Client) CreatePolicy(billet string, p *Policy) (*Policy, error) {
	var result Policy
	err := c.do("POST", "/admin/billets/"+billet+"/policies", p, &result)
	return &result, err
}

func (c *Client) GetPolicy(billet, id string) (*Policy, error) {
	var result Policy
	err := c.do("GET", "/admin/billets/"+billet+"/policies/"+id, nil, &result)
	return &result, err
}

func (c *Client) UpdatePolicy(billet, id string, p *Policy) (*Policy, error) {
	var result Policy
	err := c.do("PUT", "/admin/billets/"+billet+"/policies/"+id, p, &result)
	return &result, err
}

func (c *Client) DeletePolicy(billet, id string) error {
	return c.do("DELETE", "/admin/billets/"+billet+"/policies/"+id, nil, nil)
}
