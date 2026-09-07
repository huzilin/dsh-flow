// T6 handler 单测：产物两端点路由绑定（路径 id 优先）与错误集中映射
// （越集拦截 400 + blocked:true / 资源缺失 404）——httptest 打桩 service。
package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/api"
	"github.com/huzilin/wbflow/rd/pkg/errs"
	"github.com/huzilin/wbflow/rd/service"
)

// fakeArtifactAPI ArtifactAPI 打桩。
type fakeArtifactAPI struct {
	addReq      *api.AddArtifactReq
	addResp     *api.AddArtifactResp
	addErr      error
	contentReq  *api.GetArtifactContentReq
	contentResp *api.GetArtifactContentResp
	contentErr  error
}

func (f *fakeArtifactAPI) Add(ctx context.Context, req *api.AddArtifactReq) (*api.AddArtifactResp, error) {
	f.addReq = req
	return f.addResp, f.addErr
}

func (f *fakeArtifactAPI) GetContent(ctx context.Context, req *api.GetArtifactContentReq) (*api.GetArtifactContentResp, error) {
	f.contentReq = req
	return f.contentResp, f.contentErr
}

func newArtifactTestRouter(t *testing.T, fake *fakeArtifactAPI) *gin.Engine {
	t.Helper()
	return NewRouter(&Deps{
		Logger:    zap.NewNop(),
		Health:    &fakeHealth{status: &service.HealthStatus{}},
		Token:     "t-token",
		Artifacts: fake,
	})
}

// TestUnit_ArtifactRoutes_BindAndRespond 路径 id 覆盖 body 同名字段 + 成功出参。
func TestUnit_ArtifactRoutes_BindAndRespond(t *testing.T) {
	fake := &fakeArtifactAPI{addResp: &api.AddArtifactResp{
		Ok: true,
		Artifact: &api.ArtifactItem{
			ArtifactId: "art-1", NodeId: "n-01", ProductType: "brd",
			Label: "《BRD》", Href: "brd.md", Kind: "document", AddedBy: "pm",
		},
	}}
	router := newArtifactTestRouter(t, fake)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/requirements/req-9/nodes/n-01/artifacts",
		strings.NewReader(`{"requirementId":"WRONG","nodeId":"WRONG","productType":"brd","label":"《BRD》","href":"brd.md","kind":"document","addedBy":"pm"}`))
	req.Header.Set("Authorization", "Bearer t-token")
	router.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusOK)
	// 路径 id 优先于 body 同名字段
	assert.Equal(t, fake.addReq.RequirementId, "req-9")
	assert.Equal(t, fake.addReq.NodeId, "n-01")
	assert.Equal(t, fake.addReq.ProductType, "brd")
	assert.Equal(t, fake.addReq.AddedBy, "pm")

	fake.contentResp = &api.GetArtifactContentResp{Ok: true, ArtifactId: "art-1", Kind: "link", Href: "https://x.example.com/a"}
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/requirements/req-9/artifacts/art-1/content", nil)
	req.Header.Set("Authorization", "Bearer t-token")
	router.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusOK)
	assert.Equal(t, fake.contentReq.RequirementId, "req-9")
	assert.Equal(t, fake.contentReq.ArtifactId, "art-1")
	assert.Assert(t, strings.Contains(resp.Body.String(), "\"content\":"), "出参应含 %s", "\"content\":")
}

// TestUnit_ArtifactRoutes_ErrorMapping 越集拦截（400 + blocked:true）、
// 缺失（404 + blocked:false）、未认证（401）集中映射。
func TestUnit_ArtifactRoutes_ErrorMapping(t *testing.T) {
	fake := &fakeArtifactAPI{addErr: errs.Trace(fmt.Errorf("%w: product_type 越集", errs.ErrPolicyRejected))}
	router := newArtifactTestRouter(t, fake)

	resp := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/requirements/req-9/nodes/n-01/artifacts",
		strings.NewReader(`{"productType":"case_design","label":"x","href":"a.md","addedBy":"pm"}`))
	req.Header.Set("Authorization", "Bearer t-token")
	router.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusBadRequest)
	assert.Assert(t, strings.Contains(resp.Body.String(), "\"blocked\":true"), "出参应含 %s", "\"blocked\":true")

	fake.addErr = nil
	fake.contentErr = errs.Trace(fmt.Errorf("%w: 产物缺失", errs.ErrNotFound))
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/requirements/req-9/artifacts/art-x/content", nil)
	req.Header.Set("Authorization", "Bearer t-token")
	router.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusNotFound)
	assert.Assert(t, strings.Contains(resp.Body.String(), "\"blocked\":false"), "出参应含 %s", "\"blocked\":false")

	// 未认证 fail closed
	resp = httptest.NewRecorder()
	req = httptest.NewRequest(http.MethodGet, "/api/requirements/req-9/artifacts/art-1/content", nil)
	router.ServeHTTP(resp, req)
	assert.Equal(t, resp.Code, http.StatusUnauthorized)
}
