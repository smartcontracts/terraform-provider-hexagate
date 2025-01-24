package provider

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type HexagateClient struct {
	APIToken string
	BaseURL  string
	Client   *http.Client
}

type Monitor struct {
	ID           int                    `json:"id,omitempty"`
	Name         string                 `json:"name"`
	MonitorID    int                    `json:"monitor_id"`
	Description  string                 `json:"description,omitempty"`
	CreatedBy    string                 `json:"created_by,omitempty"`
	CreatedAt    string                 `json:"created_at,omitempty"`
	UpdatedAt    string                 `json:"updated_at,omitempty"`
	Disabled     bool                   `json:"disabled,omitempty"`
	Entities     []interface{}          `json:"entities,omitempty"`
	MonitorTags  []string               `json:"monitor_tags,omitempty"`
	MonitorRules []interface{}          `json:"monitor_rules"`
	Params       map[string]interface{} `json:"params,omitempty"`
}

type CreateMonitorResponse struct {
	ID int `json:"id"`
}

func (c *HexagateClient) CreateMonitor(monitor map[string]interface{}) (*CreateMonitorResponse, error) {
	body, err := json.Marshal(monitor)
	if err != nil {
		return nil, err
	}

	log.Printf("[DEBUG] Creating monitor: %s", string(body))

	req, err := http.NewRequest("POST", fmt.Sprintf("%s/monitoring/user_monitors/", c.BaseURL), bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Hexagate-Api-Key", c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var result CreateMonitorResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (c *HexagateClient) GetMonitor(id int) (*Monitor, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/monitoring/user_monitors/%d", c.BaseURL, id), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Hexagate-Api-Key", c.APIToken)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var monitor Monitor
	if err := json.NewDecoder(resp.Body).Decode(&monitor); err != nil {
		return nil, err
	}

	return &monitor, nil
}

func (c *HexagateClient) UpdateMonitor(id int, monitor map[string]interface{}) error {
	body, err := json.Marshal(monitor)
	if err != nil {
		return err
	}

	// log the monitor so I can see it
	log.Printf("[DEBUG] Updating monitor: %s", string(body))

	req, err := http.NewRequest("PUT", fmt.Sprintf("%s/monitoring/user_monitors/%d", c.BaseURL, id), bytes.NewBuffer(body))
	if err != nil {
		return err
	}

	req.Header.Set("X-Hexagate-Api-Key", c.APIToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *HexagateClient) DeleteMonitor(id int) error {
	req, err := http.NewRequest("DELETE", fmt.Sprintf("%s/monitoring/user_monitors/%d", c.BaseURL, id), nil)
	if err != nil {
		return err
	}

	req.Header.Set("X-Hexagate-Api-Key", c.APIToken)

	resp, err := c.Client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

func (c *HexagateClient) GetAllMonitors() ([]*Monitor, error) {
	req, err := http.NewRequest("GET", fmt.Sprintf("%s/monitoring/user_monitors/", c.BaseURL), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-Hexagate-Api-Key", c.APIToken)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	var response struct {
		Items []*Monitor `json:"items"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, err
	}

	return response.Items, nil
}
