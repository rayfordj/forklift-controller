package ovirt

import (
	"errors"
	"github.com/gin-gonic/gin"
	libmodel "github.com/konveyor/controller/pkg/inventory/model"
	api "github.com/konveyor/forklift-controller/pkg/apis/forklift/v1beta1"
	model "github.com/konveyor/forklift-controller/pkg/controller/provider/model/ovirt"
	"github.com/konveyor/forklift-controller/pkg/controller/provider/web/base"
	"net/http"
)

//
// Routes
const (
	HostParam      = "host"
	HostCollection = "hosts"
	HostsRoot      = ProviderRoot + "/" + HostCollection
	HostRoot       = HostsRoot + "/:" + HostParam
)

//
// Host handler.
type HostHandler struct {
	Handler
}

//
// Add routes to the `gin` router.
func (h *HostHandler) AddRoutes(e *gin.Engine) {
	e.GET(HostsRoot, h.List)
	e.GET(HostsRoot+"/", h.List)
	e.GET(HostRoot, h.Get)
}

//
// List resources in a REST collection.
// A GET onn the collection that includes the `X-Watch`
// header will negotiate an upgrade of the connection
// to a websocket and push watch events.
func (h HostHandler) List(ctx *gin.Context) {
	status := h.Prepare(ctx)
	if status != http.StatusOK {
		ctx.Status(status)
		return
	}
	if h.WatchRequest {
		h.watch(ctx)
		return
	}
	db := h.Collector.DB()
	list := []model.Host{}
	err := db.List(&list, h.ListOptions(ctx))
	if err != nil {
		log.Trace(
			err,
			"url",
			ctx.Request.URL)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	content := []interface{}{}
	for _, m := range list {
		r := &Host{}
		r.With(&m)
		r.Link(h.Provider)
		content = append(content, r.Content(h.Detail))
	}

	ctx.JSON(http.StatusOK, content)
}

//
// Get a specific REST resource.
func (h HostHandler) Get(ctx *gin.Context) {
	status := h.Prepare(ctx)
	if status != http.StatusOK {
		ctx.Status(status)
		return
	}
	h.Detail = true
	m := &model.Host{
		Base: model.Base{
			ID: ctx.Param(HostParam),
		},
	}
	db := h.Collector.DB()
	err := db.Get(m)
	if errors.Is(err, model.NotFound) {
		ctx.Status(http.StatusNotFound)
		return
	}
	if err != nil {
		log.Trace(
			err,
			"url",
			ctx.Request.URL)
		ctx.Status(http.StatusInternalServerError)
		return
	}
	r := &Host{}
	r.With(m)
	r.Link(h.Provider)
	content := r.Content(true)

	ctx.JSON(http.StatusOK, content)
}

//
// Watch.
func (h HostHandler) watch(ctx *gin.Context) {
	db := h.Collector.DB()
	err := h.Watch(
		ctx,
		db,
		&model.Host{},
		func(in libmodel.Model) (r interface{}) {
			m := in.(*model.Host)
			host := &Host{}
			host.With(m)
			host.Link(h.Provider)
			r = host
			return
		})
	if err != nil {
		log.Trace(
			err,
			"url",
			ctx.Request.URL)
		ctx.Status(http.StatusInternalServerError)
	}
}

//
// REST Resource.
type Host struct {
	Resource
	Cluster            string              `json:"cluster"`
	ProductName        string              `json:"productName"`
	ProductVersion     string              `json:"productVersion"`
	InMaintenance      bool                `json:"inMaintenance"`
	CpuSockets         int16               `json:"cpuSockets"`
	CpuCores           int16               `json:"cpuCores"`
	NetworkAttachments []NetworkAttachment `json:"networkAttachments"`
	NICs               []hNIC              `json:"nics"`
}

type NetworkAttachment = model.NetworkAttachment
type hNIC = model.HostNIC

//
// Build the resource using the model.
func (r *Host) With(m *model.Host) {
	r.Resource.With(&m.Base)
	r.Cluster = m.Cluster
	r.ProductName = m.ProductName
	r.ProductVersion = m.ProductVersion
	r.InMaintenance = m.InMaintenance
	r.CpuSockets = m.CpuSockets
	r.CpuCores = m.CpuCores
	r.NetworkAttachments = m.NetworkAttachments
	r.NICs = m.NICs
}

//
// Build self link (URI).
func (r *Host) Link(p *api.Provider) {
	r.SelfLink = base.Link(
		HostRoot,
		base.Params{
			base.ProviderParam: string(p.UID),
			HostParam:          r.ID,
		})
}

//
// As content.
func (r *Host) Content(detail bool) interface{} {
	if !detail {
		return r.Resource
	}

	return r
}
