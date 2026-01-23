# LocalAIStack

[English README](README.md)

**LocalAIStack** 是一个开放、模块化的软件栈，用于构建和运营 **本地 AI 工作站**。

它提供统一的控制层，用于在本地硬件上安装、管理、升级并运行 AI 开发环境、推理运行时、模型和应用 —— 无需依赖云服务或厂商专有平台。

LocalAIStack 旨在 **硬件感知**、**可复现**、**可扩展**，作为本地 AI 计算的长期基础。

---

## 为什么选择 LocalAIStack

本地运行 AI 工作负载不再是小众需求。
然而，本地 AI 软件生态仍然高度碎片化：

* 推理引擎、框架与应用彼此独立演进
* CUDA、驱动、Python 与系统依赖高度耦合
* 不同硬件配置的安装路径不一致
* 环境漂移导致难以复现与维护
* 许多工具默认面向云端部署模型

LocalAIStack 通过将 **本地 AI 工作站本身视为基础设施** 来解决这些问题。

---

## 设计目标

LocalAIStack 围绕以下原则构建：

* **本地优先**
  不强制依赖云端，必要时可完全离线运行。

* **硬件感知**
  自动基于 CPU、GPU、内存和互联能力适配可用软件能力。

* **模块化与可组合**
  所有组件可选且可独立管理。

* **默认可复现**
  安装与运行行为可确定且可版本化。

* **开放且厂商中立**
  不锁定特定硬件厂商、模型或框架。

---

## LocalAIStack 提供什么

LocalAIStack 不是单一应用。
它是一个由多层协同组成的 **堆叠式系统**。

### 1. 系统与环境管理

* 支持的操作系统：

  * Ubuntu 22.04 LTS
  * Ubuntu 24.04 LTS
* GPU 驱动与 CUDA 兼容性管理
* 系统级包管理与镜像配置
* 安全升级与回滚机制

---

### 2. 编程语言环境（按需）

* Python（多版本，隔离环境）
* Java（OpenJDK 8 / 11 / 17）
* Node.js（LTS，版本管理）
* Ruby
* PHP
* Rust

所有语言环境均为：

* 可选
* 隔离
* 可升级
* 可在不污染系统的情况下移除

---

### 3. 本地 AI 推理运行时

支持的推理引擎包括：

* Ollama
* llama.cpp
* vLLM
* SGLang

可用性会根据硬件能力（如 GPU 显存、互联带宽）自动限制。

---

### 4. AI 开发框架

* PyTorch
* TensorFlow（可选）
* Hugging Face Transformers
* LangChain
* LangGraph

框架版本与已安装运行时和 CUDA 配置保持一致。

---

### 5. 数据与基础设施服务

用于 AI 开发和 RAG 工作流的可选本地服务：

* PostgreSQL
* MySQL
* Redis
* ClickHouse
* Nginx

所有服务支持：

* 一键启动/停止
* 持久化数据目录
* 仅本地或可网络访问模式

---

### 6. AI 应用

精选的开源 AI 应用，以受管服务形式部署：

* RAGFlow
* ComfyUI
* open-deep-research
* （可通过清单扩展）

每个应用包含：

* 依赖隔离
* 端口管理
* 统一访问入口

---

### 7. 开发者工具

* VS Code（本地服务器模式）
* Aider
* OpenCode
* RooCode

工具已集成但非强制使用。

---

### 8. 模型管理

LocalAIStack 提供统一的模型管理层：

* 模型来源：

  * Hugging Face
  * ModelScope
* 支持格式：

  * GGUF
  * safetensors
* 能力：

  * 搜索
  * 下载
  * 完整性校验
  * 硬件兼容性检查

---

## 硬件能力感知

LocalAIStack 将硬件划分为能力等级，并自动适配可用功能。

示例等级：

* **Tier 1**：入门级（≤14B 推理）
* **Tier 2**：中端（≈30B 推理）
* **Tier 3**：高端（≥70B，多 GPU，NVLink）

用户不会安装其硬件无法可靠运行的软件。

---

## 用户界面

LocalAIStack 提供：

* 基于 Web 的管理界面
* 面向高级用户的 CLI

### 国际化

* 内置多语言 UI 支持
* 可选 AI 辅助界面翻译
* 无硬编码语言假设

---

## 架构概览

```
LocalAIStack
├── Control Layer
│   ├── Hardware Detection
│   ├── Capability Policy Engine
│   ├── Package & Version Management
│
├── Runtime Layer
│   ├── Container-based execution
│   ├── Native high-performance paths
│
├── Software Modules
│   ├── Languages
│   ├── Inference Engines
│   ├── Frameworks
│   ├── Services
│   └── Applications
│
└── Interfaces
    ├── Web UI
    └── CLI
```

---

## 典型使用场景

* 本地 LLM 推理与实验
* RAG 与智能体开发
* AI 教育与教学实验室
* 研究复现性
* 企业私有 AI 环境
* 硬件评估与基准测试

---

## 开源

LocalAIStack 是一个开源项目。

* 许可证：Apache 2.0（或 MIT，待定）
* 欢迎贡献
* 设计上保持厂商中立

---

## 项目状态

LocalAIStack 正在积极开发中。

当前初期重点为：

* 稳定的 Tier 2（≈30B）本地推理流程
* 可确定的安装路径
* 清晰的硬件到能力映射

随着项目演进，将发布路线图与里程碑。

---

## 快速开始

文档、安装指南和清单位于 `docs/` 目录。

---

## 理念

LocalAIStack 将 **本地 AI 计算视为基础设施**，而不是一组工具。

它希望让本地 AI 系统：

* 可预测
* 易维护
* 易理解
* 可长期使用
