package loops

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

type ThemeStyles struct {
	BackgroundColor       string  `json:"backgroundColor"`
	BackgroundXPadding    float64 `json:"backgroundXPadding"`
	BackgroundYPadding    float64 `json:"backgroundYPadding"`
	BodyColor             string  `json:"bodyColor"`
	BodyXPadding          float64 `json:"bodyXPadding"`
	BodyYPadding          float64 `json:"bodyYPadding"`
	BodyFontFamily        string  `json:"bodyFontFamily"`
	BodyFontCategory      string  `json:"bodyFontCategory"`
	BorderColor           string  `json:"borderColor"`
	BorderWidth           float64 `json:"borderWidth"`
	BorderRadius          float64 `json:"borderRadius"`
	ButtonBodyColor       string  `json:"buttonBodyColor"`
	ButtonBodyXPadding    float64 `json:"buttonBodyXPadding"`
	ButtonBodyYPadding    float64 `json:"buttonBodyYPadding"`
	ButtonBorderColor     string  `json:"buttonBorderColor"`
	ButtonBorderWidth     float64 `json:"buttonBorderWidth"`
	ButtonBorderRadius    float64 `json:"buttonBorderRadius"`
	ButtonTextColor       string  `json:"buttonTextColor"`
	ButtonTextFormat      float64 `json:"buttonTextFormat"`
	ButtonTextFontSize    float64 `json:"buttonTextFontSize"`
	DividerColor          string  `json:"dividerColor"`
	DividerBorderWidth    float64 `json:"dividerBorderWidth"`
	TextBaseColor         string  `json:"textBaseColor"`
	TextBaseFontSize      float64 `json:"textBaseFontSize"`
	TextBaseLineHeight    float64 `json:"textBaseLineHeight"`
	TextBaseLetterSpacing float64 `json:"textBaseLetterSpacing"`
	TextLinkColor         string  `json:"textLinkColor"`
	Heading1Color         string  `json:"heading1Color"`
	Heading1FontSize      float64 `json:"heading1FontSize"`
	Heading1LineHeight    float64 `json:"heading1LineHeight"`
	Heading1LetterSpacing float64 `json:"heading1LetterSpacing"`
	Heading2Color         string  `json:"heading2Color"`
	Heading2FontSize      float64 `json:"heading2FontSize"`
	Heading2LineHeight    float64 `json:"heading2LineHeight"`
	Heading2LetterSpacing float64 `json:"heading2LetterSpacing"`
	Heading3Color         string  `json:"heading3Color"`
	Heading3FontSize      float64 `json:"heading3FontSize"`
	Heading3LineHeight    float64 `json:"heading3LineHeight"`
	Heading3LetterSpacing float64 `json:"heading3LetterSpacing"`
}

type Theme struct {
	ThemeID   string      `json:"themeId"`
	Name      string      `json:"name"`
	Styles    ThemeStyles `json:"styles"`
	IsDefault bool        `json:"isDefault"`
	CreatedAt string      `json:"createdAt"`
	UpdatedAt string      `json:"updatedAt"`
}

func (c *Client) GetTheme(id string) (*Theme, error) {
	req, err := c.newRequest(http.MethodGet, "/themes/"+id, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errorFromResponse(resp)
	}

	var result Theme
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

func (c *Client) ListThemes(params PaginationParams) ([]Theme, *Pagination, error) {
	q := url.Values{}
	if params.PerPage != "" {
		q.Set("perPage", params.PerPage)
	}
	if params.Cursor != "" {
		q.Set("cursor", params.Cursor)
	}

	path := "/themes"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	req, err := c.newRequest(http.MethodGet, path, nil)
	if err != nil {
		return nil, nil, err
	}

	resp, err := c.do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, nil, errorFromResponse(resp)
	}

	var result struct {
		Pagination Pagination `json:"pagination"`
		Data       []Theme    `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Data, &result.Pagination, nil
}
