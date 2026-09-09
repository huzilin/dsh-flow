// T6 service 单测：document href 解析的路径穿越防护（../、绝对路径、symlink
// 逃逸/内链、缺失叶子）——resolveDocsPath 读写两路共用同一防线，无 DB 依赖。
package service

import (
	"os"
	"path/filepath"
	"testing"

	"gotest.tools/v3/assert"

	"github.com/huzilin/wbflow/rd/pkg/errs"
)

// newDocsRootSvc 临时 docs root + 待测 service（repo 未触及，传 nil）。
func newDocsRootSvc(t *testing.T) (*ArtifactService, string) {
	t.Helper()
	root := t.TempDir()
	return NewArtifactService(nil, root), root
}

// rootReal root 真实路径（darwin 下 /tmp 为 symlink，EvalSymlinks 归一后比较）。
func rootReal(t *testing.T, root string) string {
	t.Helper()
	real, err := filepath.EvalSymlinks(root)
	assert.NilError(t, err)
	return real
}

// TestUnit_ResolveDocsPath_InsideRoot 根内相对路径正常解析（存在/缺失两态）。
func TestUnit_ResolveDocsPath_InsideRoot(t *testing.T) {
	svc, root := newDocsRootSvc(t)
	assert.NilError(t, os.WriteFile(filepath.Join(root, "brd.md"), []byte("x"), 0o644))
	assert.NilError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(root, "sub", "b.md"), []byte("y"), 0o644))

	real, err := filepath.EvalSymlinks(root)
	assert.NilError(t, err)

	got, exists, resolveErr := svc.resolveDocsPath("brd.md")
	assert.NilError(t, resolveErr)
	assert.Equal(t, exists, true)
	assert.Equal(t, got, filepath.Join(real, "brd.md"))

	// ./ 与内层 ../ 归一后仍在根内
	got, exists, resolveErr = svc.resolveDocsPath("./sub/../brd.md")
	assert.NilError(t, resolveErr)
	assert.Equal(t, exists, true)
	assert.Equal(t, got, filepath.Join(real, "brd.md"))

	got, exists, resolveErr = svc.resolveDocsPath("sub/b.md")
	assert.NilError(t, resolveErr)
	assert.Equal(t, exists, true)
	assert.Equal(t, got, filepath.Join(real, "sub", "b.md"))

	// 缺失叶子：不报错，exists=false（调用方区分 404 分支）；父目录缺失同理
	got, exists, resolveErr = svc.resolveDocsPath("missing.md")
	assert.NilError(t, resolveErr)
	assert.Equal(t, exists, false)
	assert.Equal(t, got, filepath.Join(real, "missing.md"))

	_, exists, resolveErr = svc.resolveDocsPath("no/such/dir/x.md")
	assert.NilError(t, resolveErr)
	assert.Equal(t, exists, false)
}

// TestUnit_ResolveDocsPath_TraversalRejected 路径穿越全形态拦截：
// ../ 上跳、绝对路径、symlink 逃逸（叶子与中间目录），越界一律 ErrPolicyRejected。
func TestUnit_ResolveDocsPath_TraversalRejected(t *testing.T) {
	svc, root := newDocsRootSvc(t)
	assert.NilError(t, os.MkdirAll(filepath.Join(root, "sub"), 0o755))

	// ../ 上跳（清洗后越出 root；目标存在与否语义一致）
	_, _, err := svc.resolveDocsPath("../outside.md")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected), "../ 越界应拦截，实测 %v", err)
	_, _, err = svc.resolveDocsPath("../missing/outside.md")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected), "缺失叶子的 ../ 越界同样拦截")
	_, _, err = svc.resolveDocsPath("sub/../../etc.md")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected))

	// 绝对路径
	_, _, err = svc.resolveDocsPath("/etc/passwd")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected), "绝对路径应拦截")
	_, _, err = svc.resolveDocsPath("")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected), "空 href 应拦截")

	// symlink 逃逸：根内入口指向根外文件
	outside := filepath.Join(t.TempDir(), "secret.md")
	assert.NilError(t, os.WriteFile(outside, []byte("s"), 0o644))
	assert.NilError(t, os.Symlink(outside, filepath.Join(root, "innocent.md")))
	_, _, err = svc.resolveDocsPath("innocent.md")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected), "symlink 叶子逃逸应拦截")

	// symlink 逃逸：中间目录为根外 symlink
	outDir := filepath.Join(t.TempDir(), "outdir")
	assert.NilError(t, os.MkdirAll(outDir, 0o755))
	assert.NilError(t, os.WriteFile(filepath.Join(outDir, "c.md"), []byte("c"), 0o644))
	assert.NilError(t, os.Symlink(outDir, filepath.Join(root, "jump")))
	_, _, err = svc.resolveDocsPath("jump/c.md")
	assert.Assert(t, errs.Is(err, errs.ErrPolicyRejected), "中间目录 symlink 逃逸应拦截")

	// symlink 内链（根内互指）：放行
	assert.NilError(t, os.WriteFile(filepath.Join(root, "real.md"), []byte("r"), 0o644))
	assert.NilError(t, os.Symlink(filepath.Join(root, "real.md"), filepath.Join(root, "alias.md")))
	_, exists, err := svc.resolveDocsPath("alias.md")
	assert.NilError(t, err)
	assert.Equal(t, exists, true, "根内 symlink 应放行")
}

// TestUnit_ResolveDocsPath_RootUnavailable docs root 不存在/非目录 →
// ErrInvalidArgument（部署配置问题，非入参越界）。
func TestUnit_ResolveDocsPath_RootUnavailable(t *testing.T) {
	svc := NewArtifactService(nil, filepath.Join(t.TempDir(), "not-exist"))
	_, _, err := svc.resolveDocsPath("a.md")
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "root 缺失应报配置非法，实测 %v", err)

	plainFile := filepath.Join(t.TempDir(), "file.md")
	assert.NilError(t, os.WriteFile(plainFile, []byte("x"), 0o644))
	svc = NewArtifactService(nil, plainFile)
	_, _, err = svc.resolveDocsPath("a.md")
	assert.Assert(t, errs.Is(err, errs.ErrInvalidArgument), "root 为文件应报配置非法")
}

// TestUnit_ValidateExternalHref link href：须为带 host 的 http(s) URL。
func TestUnit_ValidateExternalHref(t *testing.T) {
	assert.NilError(t, validateExternalHref("https://example.com/a/b?x=1"))
	assert.NilError(t, validateExternalHref("http://localhost:8080/x"))
	assert.Assert(t, errs.Is(validateExternalHref("ftp://example.com/a"), errs.ErrInvalidArgument), "非 http(s) 拒绝")
	assert.Assert(t, errs.Is(validateExternalHref("https:///no-host"), errs.ErrInvalidArgument), "缺 host 拒绝")
	assert.Assert(t, errs.Is(validateExternalHref("docs root 下 a.md"), errs.ErrInvalidArgument), "相对路径非外链")
}
