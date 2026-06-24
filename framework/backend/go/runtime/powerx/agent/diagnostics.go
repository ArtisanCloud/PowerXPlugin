package agent

type Diagnostics struct {
	Mode       string `json:"mode"`
	BaseURL    string `json:"base_url"`
	AuthScheme string `json:"auth_scheme"`
	STSPresent bool   `json:"sts_present"`
}

func (c PowerXAgentClientConfig) Diagnostics() Diagnostics {
	c = c.WithDefaults()
	return Diagnostics{
		Mode:       c.Mode,
		BaseURL:    c.BaseURL,
		AuthScheme: c.AuthScheme,
		STSPresent: c.STSClientID != "" && c.STSClientSecret != "" && c.STSTokenURL != "",
	}
}

func (c *Client) Diagnostics() Diagnostics {
	if c == nil {
		return Diagnostics{}
	}
	return c.cfg.Diagnostics()
}
