# di
**中文** | [English](README.md)

[![Go Reference](https://pkg.go.dev/badge/github.com/shanjunmei/di.svg)](https://pkg.go.dev/github.com/shanjunmei/di)
[![Go Report Card](https://goreportcard.com/badge/github.com/shanjunmei/di)](https://goreportcard.com/report/github.com/shanjunmei/di)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**di** 是一个极简的 Go 依赖注入容器。  
**无代码生成**、**零外部依赖**（仅标准库），API 极其简单：`Provide`、`Invoke`、`Module`。  
设计哲学：**保持简单，忠于 Go 惯用法**。

## 特性

- ✅ **纯运行时** – 使用反射，无需代码生成。
- ✅ **单例模式** – 每个构造函数只调用一次，结果被缓存。
- ✅ **支持错误返回** – 构造函数可返回 `(T, error)`，`Invoke` 函数也可返回 `error`。
- ✅ **模块化** – 通过 `Module` 组合多个 `Provide` / `Invoke`。
- ✅ **零外部依赖** – 仅依赖 Go 标准库。
- ✅ **二进制体积极小** – 仅几十 KB。

## 安装

    go get github.com/shanjunmei/di

要求 Go 1.22+（如需支持低版本，请将 `for i := range n` 改为传统 `for i := 0; i < n; i++`）。

## 快速开始

    package main

    import (
        "context"
        "fmt"
        "log"

        "github.com/shanjunmei/di"
    )

    type Config struct {
        Addr string
    }

    func NewConfig() *Config {
        return &Config{Addr: ":8080"}
    }

    type Server struct {
        cfg *Config
    }

    func NewServer(cfg *Config) *Server {
        return &Server{cfg: cfg}
    }

    func (s *Server) Run() error {
        fmt.Printf("Server listening on %s\n", s.cfg.Addr)
        return nil
    }

    func main() {
        app := di.New(
            di.Provide(NewConfig),
            di.Provide(NewServer),
            di.Invoke(func(srv *Server) error {
                return srv.Run()
            }),
        )

        if err := app.Run(context.Background()); err != nil {
            log.Fatal(err)
        }
    }

## 模块组合

    // user/module.go
    func Module() di.Option {
        return di.Module(
            di.Provide(repository.New),
            di.Provide(service.New),
            di.Provide(handler.New),
            di.Invoke(RegisterRoutes),
        )
    }

    // main.go
    app := di.New(
        user.Module(),
        order.Module(),
        di.Invoke(startServer),
    )

> 完整可运行示例见 [examples/main.go](./examples/main.go)。

## 与主流 DI 框架对比

| 特性                     | di            | Google Wire           | Uber Fx                |
| ------------------------ | ------------- | --------------------- | ---------------------- |
| 代码生成                 | ❌ 无         | ✅ 需要 `wire gen`     | ❌ 无                  |
| 运行时反射               | ✅ 是         | ❌ 否（编译期生成）     | ✅ 是                  |
| 启动性能                 | 中等          | 极快                  | 较慢                   |
| 编译时类型安全           | ❌ 否         | ✅ 是                 | ❌ 否                  |
| 生命周期钩子             | ❌ 无         | ❌ 无                 | ✅ `OnStart`/`OnStop` |
| 依赖顺序保证             | 注册顺序      | 拓扑排序              | 排序 + 生命周期       |
| 外部依赖                 | 0            | 0（仅生成代码）       | 多个                  |
| 学习成本                 | 极低          | 中等                  | 较高                  |
| 模块组合                 | `Module`      | `wire.NewSet`         | `fx.Module`           |
| 最终二进制大小           | 极小（~50KB） | 极小                  | 较大                  |
| 适用场景                 | 中小型项目、快速原型 | 大型项目、启动性能敏感 | 大型项目、需要生命周期管理 |

### 如何选择？

- **追求极简、零依赖、快速上手** → `di`
- **需要编译时类型安全、极致启动性能** → Wire（原版已归档，建议使用社区 fork）
- **需要复杂生命周期管理（启停钩子、插件收集）** → Fx

## 为什么不用代码生成？

- 避免引入额外的 `go generate` 步骤。
- 运行时更灵活：可根据配置动态决定是否注册某个组件。
- 体积小，便于嵌入其他项目。

## 许可证

MIT
