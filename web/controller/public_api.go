package controller

import (
	"encoding/json"
	"net/http"

	"github.com/mhsanaei/3x-ui/v2/database/model"
	"github.com/mhsanaei/3x-ui/v2/web/middleware"
	"github.com/mhsanaei/3x-ui/v2/web/service"

	"github.com/gin-gonic/gin"
)

// PublicAPIController exposes a subset of API without the '/panel' prefix.
// It proxies to existing handlers and adds lightweight aggregate endpoints.
type PublicAPIController struct {
	BaseController
	inboundController *InboundController
	inboundService    service.InboundService
}

func NewPublicAPIController(g *gin.RouterGroup) *PublicAPIController {
	a := &PublicAPIController{}
	a.initRouter(g)
	return a
}

func (a *PublicAPIController) initRouter(g *gin.RouterGroup) {
	// Base is root (no '/panel' prefix)
	g = g.Group("/api")

	// Protect ONLY this new API with API key middleware
	g.Use(middleware.ApiKeyAuthMiddleware())

	// Create inbound controller instance WITHOUT auto-registering routes
	inbound := &InboundController{}
	a.inboundController = inbound

	// Manually register only the endpoints we need
	g.GET("/list", inbound.getInbounds)
	g.GET("/get/:id", inbound.getInbound)
	g.POST("/addClient", inbound.addInboundClient)
	g.POST("/:id/delClient/:clientId", inbound.delInboundClient)
	g.POST("/updateClient/:clientId", inbound.updateInboundClient)

	// Aggregation endpoints
	g.GET("/ping", func(c *gin.Context) { c.String(http.StatusOK, "ok") })
	g.GET("/stats/users", a.countUsers)
	g.GET("/stats/online", a.countOnline)
}

// countUsers returns total number of users across all inbounds (by clients in settings)
func (a *PublicAPIController) countUsers(c *gin.Context) {
	inbounds, err := a.inboundService.GetAllInbounds()
	if err != nil {
		pureJsonMsg(c, http.StatusOK, false, err.Error())
		return
	}
	total := 0
	for _, ib := range inbounds {
		switch ib.Protocol {
		case model.VMESS, model.VLESS, model.Trojan, model.Shadowsocks:
			clients, _ := a.inboundService.GetClients(ib)
			total += len(clients)
		case model.WireGuard:
			var settings map[string]any
			if err := json.Unmarshal([]byte(ib.Settings), &settings); err == nil {
				if arr, ok := settings["peers"].([]any); ok {
					total += len(arr)
				}
			}
		}
	}
	jsonObj(c, gin.H{"count": total}, nil)
}

// countOnline returns count of online users based on current online clients
func (a *PublicAPIController) countOnline(c *gin.Context) {
	inbounds, err := a.inboundService.GetAllInbounds()
	if err != nil {
		pureJsonMsg(c, http.StatusOK, false, err.Error())
		return
	}
	allowed := map[model.Protocol]bool{
		model.VMESS:       true,
		model.VLESS:       true,
		model.Trojan:      true,
		model.Shadowsocks: true,
		model.WireGuard:   true,
	}
	onlineSlice := a.inboundService.GetOnlineClients()
	onlineSet := map[string]bool{}
	for _, e := range onlineSlice {
		onlineSet[e] = true
	}
	uniq := map[string]bool{}
	for _, ib := range inbounds {
		if !allowed[ib.Protocol] || !ib.Enable {
			continue
		}
		switch ib.Protocol {
		case model.WireGuard:
			// We do not track wireguard peers by email; treat any traffic as online per-peer is non-trivial.
			// For now, approximate as 0 added (requires deeper integration to map peers to traffic).
			continue
		default:
			clients, _ := a.inboundService.GetClients(ib)
			for _, cl := range clients {
				if cl.Enable && onlineSet[cl.Email] {
					uniq[cl.Email] = true
				}
			}
		}
	}
	jsonObj(c, gin.H{"count": len(uniq)}, nil)
}
