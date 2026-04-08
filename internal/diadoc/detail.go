package diadoc

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"strings"

	"github.com/SoulStalker/chz-api-client/internal/model"
)

type messageResponse struct {
	MessageId string `json:"MessageId"`
	Entities  []struct {
		EntityId       string `json:"EntityId"`
		EntityType     string `json:"EntityType"`
		AttachmentType string `json:"AttachmentType"`
		Content        struct {
			Data string `json:"Data"` // base64
		} `json:"Content"`
	} `json:"Entities"`
}

func (c *Client) GetDocument(boxId, messageId, entityId string) (*model.DocumentDetail, error) {
	resp, err := c.http.R().
		SetQueryParams(map[string]string{
			"boxId":     boxId,
			"messageId": messageId,
			"entityId":  entityId,
		}).
		Get("/V3/GetMessage")
	if err != nil {
		return nil, fmt.Errorf("diadoc get message: %w", err)
	}
	if resp.IsError() {
		return nil, fmt.Errorf("diadoc get message: status %d: %s", resp.StatusCode(), resp.String())
	}

	var msg messageResponse
	if err := json.Unmarshal(resp.Body(), &msg); err != nil {
		return nil, fmt.Errorf("diadoc get message: parse: %w", err)
	}

	detail := &model.DocumentDetail{
		Document: model.Document{MessageId: messageId, EntityId: entityId},
	}

	for _, entity := range msg.Entities {
		if entity.Content.Data == "" {
			continue
		}
		raw, err := base64.StdEncoding.DecodeString(entity.Content.Data)
		if err != nil {
			continue
		}
		trimmed := strings.TrimSpace(string(raw))
		if strings.HasPrefix(trimmed, "<") || strings.HasPrefix(trimmed, "<?xml") {
			if detail.XML == "" {
				detail.XML = trimmed
			}
			codes := extractKIZ(raw)
			detail.MarkingCodes = append(detail.MarkingCodes, codes...)
		}
	}

	if len(detail.MarkingCodes) > 0 {
		detail.HasMarkingCodes = true
	}

	return detail, nil
}

func extractKIZ(data []byte) []string {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	var codes []string
	inKIZ := false

	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "КИЗ" {
				inKIZ = true
			}
		case xml.EndElement:
			if t.Name.Local == "КИЗ" {
				inKIZ = false
			}
		case xml.CharData:
			if inKIZ {
				val := strings.TrimSpace(string(t))
				if val != "" {
					codes = append(codes, val)
				}
			}
		}
	}
	return codes
}
