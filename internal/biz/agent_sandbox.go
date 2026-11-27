package biz

import (
	"context"
	"path/filepath"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/agentinstall"
	"momoko/pkg/agentsandbox"
	"momoko/pkg/response"
)

// ———————————————————————————— 沙箱 ————————————————————————————
//
// 配了代理的 Agent 在沙箱里跑（只有 Linux，见 pkg/agentsandbox），沙箱的网络由 sing-box 接管。
// sing-box 由 Momoko 下载并更新，放在数据目录下，和运行时的「Momoko 下载并更新」一样走 pkg/agentinstall。

// singBoxInstallKey sing-box 下载中时在 installing 里的标记。
const singBoxInstallKey = "@sing-box"

func (uc *AgentUsecase) singBoxDir() string {
	return filepath.Join(uc.root, "runtimes", "sing-box")
}

// sandboxPB 这台服务器的沙箱：能不能用、建不了的原因、sing-box 下载了哪个版本、是不是正在下载。
func (uc *AgentUsecase) sandboxPB() *v1.AgentSandbox {
	out := &v1.AgentSandbox{Supported: agentsandbox.Supported()}
	if !out.Supported {
		return out
	}
	if err := agentsandbox.Check(); err != nil {
		out.Unavailable = err.Error()
	}
	if launch, ok := agentinstall.Installed(uc.singBoxDir()); ok {
		out.SingBoxVersion = launch.Version
	}
	uc.installMu.Lock()
	out.Installing = uc.installing[singBoxInstallKey]
	uc.installMu.Unlock()
	return out
}

// UpdateSandbox 沙箱用的 sing-box：没下载时下载最新版本，下载过时更新到最新版本（可能要几分钟，不受请求超时影响）。
// 正在跑的沙箱用的是旧的那份，下次起沙箱时换成新的。
func (uc *AgentUsecase) UpdateSandbox(ctx context.Context) (*v1.UpdateAgentSandboxResponse, error) {
	rel, ok := agentsandbox.SingBoxRelease()
	if !ok {
		return nil, response.BadRequest(agentsandbox.ErrUnsupported.Error())
	}
	uc.installMu.Lock()
	if uc.installing[singBoxInstallKey] {
		uc.installMu.Unlock()
		return nil, response.BadRequest("sing-box 正在下载，稍等一下")
	}
	uc.installing[singBoxInstallKey] = true
	uc.installMu.Unlock()
	defer func() {
		uc.installMu.Lock()
		delete(uc.installing, singBoxInstallKey)
		uc.installMu.Unlock()
	}()
	bg := context.WithoutCancel(ctx)
	resp := &v1.UpdateAgentSandboxResponse{}
	if launch, ok := agentinstall.Installed(uc.singBoxDir()); ok {
		resp.Current = launch.Version
	}
	latest, err := agentinstall.Latest(bg, rel)
	if err != nil {
		return nil, agentErr(err)
	}
	resp.Latest = latest
	if resp.Current == "" || agentinstall.Newer(latest, resp.Current) {
		if _, err := agentinstall.Install(bg, rel, uc.singBoxDir(), nil); err != nil {
			return nil, agentErr(err)
		}
		resp.Updated = true
	}
	uc.installMu.Lock()
	delete(uc.installing, singBoxInstallKey)
	uc.installMu.Unlock()
	resp.Sandbox = uc.sandboxPB()
	uc.configChanged()
	return resp, nil
}
