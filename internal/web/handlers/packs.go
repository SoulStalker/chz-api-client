package handlers

import (
	"errors"
	"log/slog"

	"github.com/SoulStalker/chz-api-client/internal/crpt"
	"github.com/SoulStalker/chz-api-client/internal/model"
	"github.com/SoulStalker/chz-api-client/internal/session"
	"github.com/SoulStalker/chz-api-client/views"
	"github.com/gofiber/fiber/v2"
)

type PacksHandler struct {
	crpt     *crpt.Client
	sessions *session.Store
}

func NewPacksHandler(c *crpt.Client, sessions *session.Store) *PacksHandler {
	return &PacksHandler{crpt: c, sessions: sessions}
}

func (h *PacksHandler) token(c *fiber.Ctx) (string, bool) {
	sessionID := c.Cookies("crpt_session")
	if sessionID == "" {
		return "", false
	}
	return h.sessions.Get(sessionID)
}

// enrichChildren fetches details for child codes and populates cis.Child.
func (h *PacksHandler) enrichChildren(c *fiber.Ctx, token string, cis *model.CisInfo) {
	if len(cis.ChildCodes) == 0 {
		return
	}
	childInfos, err := h.crpt.GetCisInfo(c.Context(), token, cis.ChildCodes)
	if err != nil {
		slog.Warn("cises/info for children failed", "err", err)
		// Fall back: populate Child with just the codes.
		cis.Child = make([]model.CisChild, len(cis.ChildCodes))
		for i, code := range cis.ChildCodes {
			cis.Child[i] = model.CisChild{CisKey: code}
		}
		return
	}
	infoMap := make(map[string]model.CisInfo, len(childInfos))
	for _, ci := range childInfos {
		infoMap[ci.CisKey] = ci
	}
	cis.Child = make([]model.CisChild, 0, len(cis.ChildCodes))
	for _, code := range cis.ChildCodes {
		child := model.CisChild{CisKey: code}
		if ci, ok := infoMap[code]; ok {
			child.GTIN = ci.GTIN
			child.Name = ci.Name
			child.PackType = ci.PackType
			child.ChildCount = ci.ChildCount
		}
		cis.Child = append(cis.Child, child)
	}
}

func (h *PacksHandler) ShowPack(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")
	packCode := c.Params("code")
	invoiceNumber := c.Query("invoice_number", docID)

	cisInfos, err := h.crpt.GetCisInfo(c.Context(), token, []string{packCode})
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.PackDetail(docID, packCode, invoiceNumber, nil, errMsg(err)).Render(c.Context(), c.Response().BodyWriter())
	}
	if len(cisInfos) == 0 {
		c.Type("html")
		return views.PackDetail(docID, packCode, invoiceNumber, nil, "КИ не найдено").Render(c.Context(), c.Response().BodyWriter())
	}

	pack := cisInfos[0]
	h.enrichChildren(c, token, &pack)

	c.Type("html")
	return views.PackDetail(docID, packCode, invoiceNumber, &pack, "").Render(c.Context(), c.Response().BodyWriter())
}

func (h *PacksHandler) ShowGroup(c *fiber.Ctx) error {
	token, ok := h.token(c)
	if !ok {
		return c.Redirect("/auth")
	}

	docID := c.Params("id")
	packCode := c.Params("code")
	groupCode := c.Params("childCode")
	invoiceNumber := c.Query("invoice_number", docID)

	cisInfos, err := h.crpt.GetCisInfo(c.Context(), token, []string{groupCode})
	if err != nil {
		if errors.Is(err, crpt.ErrUnauthorized) {
			return c.Redirect("/auth")
		}
		c.Type("html")
		return views.GroupDetail(docID, packCode, groupCode, invoiceNumber, nil, errMsg(err)).Render(c.Context(), c.Response().BodyWriter())
	}
	if len(cisInfos) == 0 {
		c.Type("html")
		return views.GroupDetail(docID, packCode, groupCode, invoiceNumber, nil, "КИ не найдено").Render(c.Context(), c.Response().BodyWriter())
	}

	group := cisInfos[0]
	h.enrichChildren(c, token, &group)

	c.Type("html")
	return views.GroupDetail(docID, packCode, groupCode, invoiceNumber, &group, "").Render(c.Context(), c.Response().BodyWriter())
}
