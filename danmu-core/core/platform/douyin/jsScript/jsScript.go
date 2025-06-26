package jsScript

import (
	_ "embed"
	"github.com/dop251/goja"
	"sync"
)

// 嵌入的 JavaScript 文件来源于开源项目，感谢贡献者们的努力
//
//go:embed webmssdk.js
var jsScript string

type GojaDouyin struct {
	ua       string
	vm       *goja.Runtime
	mu       sync.Mutex
	fGetSign func(signature string) string
}

// LoadGoja 加载 JavaScript 到 Goja 运行时中，并设置必要的环境
func LoadGoja(ua string) (*GojaDouyin, error) {
	gd := &GojaDouyin{
		ua: ua,
		vm: goja.New(),
	}
	// 创建一个新的 Goja VM 实例

	// 构建 JavaScript 环境，模拟浏览器的 navigator 和 window 对象
	jsdom := `
		navigator = { userAgent: '` + ua + `' };
		window = this;
		document = {};
		window.navigator = navigator;
		setTimeout = function() {};
	`

	// 运行 JavaScript 环境设置和嵌入的 JavaScript 代码
	if _, err := gd.vm.RunString(jsdom + jsScript); err != nil {
		return nil, err
	}

	// 将 JavaScript 函数 get_sign 导出为 Go 函数 fGetSign
	if err := gd.vm.ExportTo(gd.vm.Get("get_sign"), &gd.fGetSign); err != nil {
		return nil, err
	}
	return gd, nil
}

// ExecuteJS 执行 JavaScript 中的 get_sign 函数
func (gd *GojaDouyin) GetSign(signature string) string {
	gd.mu.Lock()
	defer gd.mu.Unlock()
	return gd.fGetSign(signature)
}
