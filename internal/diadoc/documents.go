package diadoc

import (
	"encoding/json"
	"fmt"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

type documentListResponse struct {
	Documents []struct {
		MessageId      string `json:"MessageId"`
		EntityId       string `json:"EntityId"`
		DocumentNumber string `json:"DocumentNumber"`
		DocumentDate   string `json:"DocumentDate"`
		Counteragent   struct {
			OrgId   string `json:"OrgId"`
			OrgName string `json:"OrgName"`
		} `json:"Counteragent"`
		DocflowStatus struct {
			PrimaryStatus struct {
				StatusText string `json:"StatusText"`
			} `json:"PrimaryStatus"`
		} `json:"DocflowStatus"`
	} `json:"Documents"`
}

func (c *Client) GetDocuments(boxId string) ([]model.Document, error) {
	resp, err := c.http.R().
		SetQueryParams(map[string]string{
			"filterCategory": "UniversalTransferDocument.InboundNotFinished",
			"boxId":          boxId,
			"count":          "50",
		}).
		Get("/V3/GetDocuments")
	if err != nil {
		return nil, fmt.Errorf("diadoc get docs: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("diadoc get docs: status %d: %s", resp.StatusCode(), resp.String())
	}

	var result documentListResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		return nil, fmt.Errorf("diadoc get docs: parse: %w", err)
	}

	docs := make([]model.Document, 0, len(result.Documents))
	for _, d := range result.Documents {
		docs = append(docs, model.Document{
			MessageId:        d.MessageId,
			EntityId:         d.EntityId,
			DocumentNumber:   d.DocumentNumber,
			DocumentDate:     d.DocumentDate,
			CounteragentId:   d.Counteragent.OrgId,
			CounteragentName: d.Counteragent.OrgName,
			Status:           d.DocflowStatus.PrimaryStatus.StatusText,
		})
	}
	return docs, nil
}
