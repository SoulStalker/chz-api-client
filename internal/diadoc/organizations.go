package diadoc

import (
	"encoding/json"
	"fmt"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

type orgListResponse struct {
	Organizations []struct {
		OrgId     string `json:"OrgId"`
		FullName  string `json:"FullName"`
		ShortName string `json:"ShortName"`
		Inn       string `json:"Inn"`
		Boxes     []struct {
			BoxId   string `json:"BoxId"`
			BoxName string `json:"BoxName"`
		} `json:"Boxes"`
	} `json:"Organizations"`
}

func (c *Client) GetMyOrganizations() ([]model.Organization, error) {
	resp, err := c.http.R().Get("/V3/GetMyOrganizations")
	if err != nil {
		return nil, fmt.Errorf("diadoc get orgs: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("diadoc get orgs: status %d: %s", resp.StatusCode(), resp.String())
	}

	var result orgListResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("diadoc get orgs: parse: %w", err)
	}

	var orgs []model.Organization
	for _, org := range result.Organizations {
		name := org.FullName
		if name == "" {
			name = org.ShortName
		}
		for _, box := range org.Boxes {
			boxName := box.BoxName
			if boxName == "" {
				boxName = name
			}
			orgs = append(orgs, model.Organization{
				BoxId: box.BoxId,
				Name:  boxName,
				Inn:   org.Inn,
			})
		}
	}
	return orgs, nil
}
